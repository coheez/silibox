package lima

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestLimaTemplateGeneration tests the complete template generation process
func TestLimaTemplateGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
	}()

	// Set HOME to our temp directory
	os.Setenv("HOME", tmpDir)

	// Point to repo template if running from package dir
	if _, err := os.Stat("build/lima/templates/ubuntu-lts.yaml.tmpl"); err != nil {
		// Try to find template two levels up (package tests often run from pkg dir)
		alt := filepath.Clean(filepath.Join("..", "..", "build", "lima", "templates", "ubuntu-lts.yaml.tmpl"))
		if _, err := os.Stat(alt); err == nil {
			os.Setenv("SILI_LIMA_TEMPLATE", alt)
		}
	}

	// Test different configurations
	configs := []Config{
		{CPUs: 1, Memory: "2GiB", Disk: "20GiB"},
		{CPUs: 4, Memory: "8GiB", Disk: "60GiB"},
		{CPUs: 8, Memory: "16GiB", Disk: "100GiB"},
	}

	for i, cfg := range configs {
		t.Run(fmt.Sprintf("config_%d", i), func(t *testing.T) {
			err := ensureTemplate(cfg)
			if err != nil {
				t.Fatalf("ensureTemplate failed: %v", err)
			}

			// Verify the generated file
			yamlPath := filepath.Join(tmpDir, ".sili", "lima.yaml")
			content, err := os.ReadFile(yamlPath)
			if err != nil {
				t.Fatalf("failed to read generated yaml: %v", err)
			}

			// Validate YAML syntax (basic check)
			if !isValidYAML(string(content)) {
				t.Error("generated YAML is not valid")
			}

			// Check specific values
			contentStr := string(content)
			expectedCPUs := fmt.Sprintf("cpus: %d", cfg.CPUs)
			expectedMemory := fmt.Sprintf("memory: \"%s\"", cfg.Memory)
			expectedDisk := fmt.Sprintf("disk: \"%s\"", cfg.Disk)

			if !contains(contentStr, expectedCPUs) {
				t.Errorf("expected %s in yaml", expectedCPUs)
			}
			if !contains(contentStr, expectedMemory) {
				t.Errorf("expected %s in yaml", expectedMemory)
			}
			if !contains(contentStr, expectedDisk) {
				t.Errorf("expected %s in yaml", expectedDisk)
			}
		})
	}
}

// TestLimaCommandAvailability tests if limactl is available
func TestLimaCommandAvailability(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Check if limactl is available
	_, err := exec.LookPath("limactl")
	if err != nil {
		t.Skip("limactl not available, skipping test")
	}

	// Test limactl version command (some versions use --version)
	cmd := exec.Command("limactl", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fall back to 'limactl -v'
		cmd = exec.Command("limactl", "-v")
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Errorf("limactl version failed: %v, output: %s", err, string(output))
		}
	}

	// Should contain version information
	if len(output) == 0 {
		t.Error("limactl version should return some output")
	}
}

// TestLimaTemplateValidation tests that the generated template is valid Lima YAML
func TestLimaTemplateValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Skip if limactl is not available
	_, err := exec.LookPath("limactl")
	if err != nil {
		t.Skip("limactl not available, skipping test")
	}

	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
	}()

	os.Setenv("HOME", tmpDir)

	// Point to repo template if running from package dir
	if _, err := os.Stat("build/lima/templates/ubuntu-lts.yaml.tmpl"); err != nil {
		alt := filepath.Clean(filepath.Join("..", "..", "build", "lima", "templates", "ubuntu-lts.yaml.tmpl"))
		if _, err := os.Stat(alt); err == nil {
			os.Setenv("SILI_LIMA_TEMPLATE", alt)
		}
	}

	cfg := Config{CPUs: 2, Memory: "4GiB", Disk: "30GiB"}
	err = ensureTemplate(cfg)
	if err != nil {
		t.Fatalf("ensureTemplate failed: %v", err)
	}

	// Try to validate the YAML with limactl
	yamlPath := filepath.Join(tmpDir, ".sili", "lima.yaml")
	cmd := exec.Command("limactl", "validate", yamlPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("limactl validate failed: %v, output: %s", err, string(output))
	}
}

// Helper functions
func isValidYAML(content string) bool {
	// Basic YAML validation - check for common YAML structure
	lines := strings.Split(content, "\n")
	if len(lines) < 3 {
		return false
	}

	// Check for key YAML indicators
	hasKeyValue := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			if strings.Contains(line, ":") {
				hasKeyValue = true
				break
			}
		}
	}

	return hasKeyValue
}
