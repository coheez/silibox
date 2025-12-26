package shim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShimDir(t *testing.T) {
	shimDir, err := ShimDir()
	if err != nil {
		t.Fatalf("ShimDir() error = %v", err)
	}

	// Should end with .sili/bin
	if !strings.HasSuffix(shimDir, filepath.Join(".sili", "bin")) {
		t.Errorf("ShimDir() = %v, want path ending with .sili/bin", shimDir)
	}

	// Should be an absolute path
	if !filepath.IsAbs(shimDir) {
		t.Errorf("ShimDir() = %v, want absolute path", shimDir)
	}
}

func TestGenerateAndRemoveShim(t *testing.T) {
	// Create a temporary shim directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	
	// Override HOME to use temp directory
	testHome := filepath.Join(tmpDir, "home")
	os.Setenv("HOME", testHome)
	defer os.Setenv("HOME", originalHome)

	envName := "test-env"
	command := "test-command"

	// Generate shim
	err := GenerateShim(envName, command, false)
	if err != nil {
		t.Fatalf("GenerateShim() error = %v", err)
	}

	// Check that shim file was created
	shimDir, err := ShimDir()
	if err != nil {
		t.Fatalf("ShimDir() error = %v", err)
	}

	shimPath := filepath.Join(shimDir, command)
	if _, err := os.Stat(shimPath); os.IsNotExist(err) {
		t.Errorf("Shim file was not created at %s", shimPath)
	}

	// Check that shim is executable
	info, err := os.Stat(shimPath)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Errorf("Shim file is not executable: %o", info.Mode())
	}

	// Check shim content
	content, err := os.ReadFile(shimPath)
	if err != nil {
		t.Fatalf("Failed to read shim file: %v", err)
	}

	expectedContent := []string{
		"#!/bin/bash",
		"exec sili run --name test-env -- test-command",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(string(content), expected) {
			t.Errorf("Shim content does not contain expected string: %s", expected)
		}
	}

	// Try to generate again without force (should fail)
	err = GenerateShim(envName, command, false)
	if err == nil {
		t.Error("GenerateShim() should fail when shim already exists without force flag")
	}

	// Generate again with force (should succeed)
	err = GenerateShim(envName, command, true)
	if err != nil {
		t.Errorf("GenerateShim() with force should succeed: %v", err)
	}

	// Remove shim
	err = RemoveShim(command)
	if err != nil {
		t.Fatalf("RemoveShim() error = %v", err)
	}

	// Check that shim file was removed
	if _, err := os.Stat(shimPath); !os.IsNotExist(err) {
		t.Errorf("Shim file still exists after removal")
	}

	// Try to remove again (should fail)
	err = RemoveShim(command)
	if err == nil {
		t.Error("RemoveShim() should fail when shim doesn't exist")
	}
}

func TestIsInPATH(t *testing.T) {
	// Create a temporary shim directory
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalPath := os.Getenv("PATH")
	
	// Override HOME to use temp directory
	testHome := filepath.Join(tmpDir, "home")
	os.Setenv("HOME", testHome)
	defer os.Setenv("HOME", originalHome)
	defer os.Setenv("PATH", originalPath)

	shimDir, err := ShimDir()
	if err != nil {
		t.Fatalf("ShimDir() error = %v", err)
	}

	// Initially, shim dir should NOT be in PATH
	inPath, err := IsInPATH()
	if err != nil {
		t.Fatalf("IsInPATH() error = %v", err)
	}
	if inPath {
		t.Error("IsInPATH() = true, want false (shim dir not in PATH yet)")
	}

	// Add shim dir to PATH
	os.Setenv("PATH", shimDir+":"+originalPath)

	// Now it should be in PATH
	inPath, err = IsInPATH()
	if err != nil {
		t.Fatalf("IsInPATH() error = %v", err)
	}
	if !inPath {
		t.Error("IsInPATH() = false, want true (shim dir should be in PATH)")
	}
}

func TestGetPATHInstructions(t *testing.T) {
	shells := []string{"bash", "zsh", "fish"}
	
	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			originalShell := os.Getenv("SHELL")
			os.Setenv("SHELL", "/bin/"+shell)
			defer os.Setenv("SHELL", originalShell)

			instructions, err := GetPATHInstructions()
			if err != nil {
				t.Fatalf("GetPATHInstructions() error = %v", err)
			}

			if instructions == "" {
				t.Error("GetPATHInstructions() returned empty string")
			}

			// Check that instructions contain shell-specific keywords
			lowerInstructions := strings.ToLower(instructions)
			if shell == "fish" {
				if !strings.Contains(lowerInstructions, "fish") {
					t.Errorf("Instructions for fish shell should mention fish")
				}
			} else {
				// bash and zsh should mention export PATH
				if !strings.Contains(lowerInstructions, "export path") {
					t.Errorf("Instructions for %s shell should mention 'export PATH'", shell)
				}
			}
		})
	}
}
