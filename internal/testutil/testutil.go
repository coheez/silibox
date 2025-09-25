package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TempHomeDir creates a temporary directory and sets it as HOME
// Returns a cleanup function that should be called in defer
func TempHomeDir(t *testing.T) (string, func()) {
	t.Helper()
	
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	
	os.Setenv("HOME", tmpDir)
	
	cleanup := func() {
		os.Setenv("HOME", originalHome)
	}
	
	return tmpDir, cleanup
}

// CreateTestLimaTemplate creates a test Lima template file
func CreateTestLimaTemplate(t *testing.T, dir string, content string) string {
	t.Helper()
	
	templatePath := filepath.Join(dir, "test-template.yaml")
	err := os.WriteFile(templatePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test template: %v", err)
	}
	
	return templatePath
}

// AssertFileExists checks if a file exists and fails the test if it doesn't
func AssertFileExists(t *testing.T, filePath string) {
	t.Helper()
	
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", filePath)
	}
}

// AssertFileContains checks if a file contains the expected content
func AssertFileContains(t *testing.T, filePath string, expectedContent string) {
	t.Helper()
	
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", filePath, err)
	}
	
	if !contains(string(content), expectedContent) {
		t.Errorf("expected file %s to contain %s", filePath, expectedContent)
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr || 
			 contains(s[1:], substr))))
}
