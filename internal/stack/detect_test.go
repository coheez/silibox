package stack

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDetectStack_Node tests Node.js project detection
func TestDetectStack_Node(t *testing.T) {
	tests := []struct {
		name           string
		files          []string
		expectedType   StackType
		expectedPkgMgr string
		minHotDirs     int
	}{
		{
			name:           "npm project",
			files:          []string{"package.json", "package-lock.json"},
			expectedType:   Node,
			expectedPkgMgr: "npm",
			minHotDirs:     3,
		},
		{
			name:           "yarn project",
			files:          []string{"package.json", "yarn.lock"},
			expectedType:   Node,
			expectedPkgMgr: "yarn",
			minHotDirs:     3,
		},
		{
			name:           "pnpm project",
			files:          []string{"package.json", "pnpm-lock.yaml"},
			expectedType:   Node,
			expectedPkgMgr: "pnpm",
			minHotDirs:     3,
		},
		{
			name:           "bun project",
			files:          []string{"package.json", "bun.lockb"},
			expectedType:   Node,
			expectedPkgMgr: "bun",
			minHotDirs:     3,
		},
		{
			name:           "package.json only",
			files:          []string{"package.json"},
			expectedType:   Node,
			expectedPkgMgr: "npm",
			minHotDirs:     3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := createTempProject(t, tt.files)
			defer os.RemoveAll(dir)

			info, err := DetectStack(dir)
			if err != nil {
				t.Fatalf("DetectStack() error = %v", err)
			}

			if info.Type != tt.expectedType {
				t.Errorf("Type = %v, want %v", info.Type, tt.expectedType)
			}

			if info.PackageManager != tt.expectedPkgMgr {
				t.Errorf("PackageManager = %v, want %v", info.PackageManager, tt.expectedPkgMgr)
			}

			if len(info.HotDirs) < tt.minHotDirs {
				t.Errorf("HotDirs count = %d, want at least %d", len(info.HotDirs), tt.minHotDirs)
			}

			// Verify node_modules is in hot dirs
			foundNodeModules := false
			for _, dir := range info.HotDirs {
				if dir == "node_modules" {
					foundNodeModules = true
					break
				}
			}
			if !foundNodeModules {
				t.Errorf("HotDirs missing 'node_modules'")
			}

			// Verify watcher commands are populated
			if len(info.WatcherCommands) == 0 {
				t.Errorf("WatcherCommands is empty")
			}
		})
	}
}

// TestDetectStack_Python tests Python project detection
func TestDetectStack_Python(t *testing.T) {
	tests := []struct {
		name         string
		files        []string
		expectedType StackType
		minHotDirs   int
	}{
		{
			name:         "poetry project",
			files:        []string{"pyproject.toml", "poetry.lock"},
			expectedType: Python,
			minHotDirs:   3,
		},
		{
			name:         "pip project",
			files:        []string{"requirements.txt"},
			expectedType: Python,
			minHotDirs:   3,
		},
		{
			name:         "setuptools project",
			files:        []string{"setup.py", "setup.cfg"},
			expectedType: Python,
			minHotDirs:   3,
		},
		{
			name:         "pipenv project",
			files:        []string{"Pipfile"},
			expectedType: Python,
			minHotDirs:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := createTempProject(t, tt.files)
			defer os.RemoveAll(dir)

			info, err := DetectStack(dir)
			if err != nil {
				t.Fatalf("DetectStack() error = %v", err)
			}

			if info.Type != tt.expectedType {
				t.Errorf("Type = %v, want %v", info.Type, tt.expectedType)
			}

			if len(info.HotDirs) < tt.minHotDirs {
				t.Errorf("HotDirs count = %d, want at least %d", len(info.HotDirs), tt.minHotDirs)
			}

			// Verify .venv is in hot dirs
			foundVenv := false
			for _, dir := range info.HotDirs {
				if dir == ".venv" || dir == "venv" {
					foundVenv = true
					break
				}
			}
			if !foundVenv {
				t.Errorf("HotDirs missing '.venv' or 'venv'")
			}

			// Verify watcher commands are populated
			if len(info.WatcherCommands) == 0 {
				t.Errorf("WatcherCommands is empty")
			}
		})
	}
}

// TestDetectStack_Rust tests Rust project detection
func TestDetectStack_Rust(t *testing.T) {
	tests := []struct {
		name         string
		files        []string
		expectedType StackType
		minHotDirs   int
	}{
		{
			name:         "cargo project",
			files:        []string{"Cargo.toml", "Cargo.lock"},
			expectedType: Rust,
			minHotDirs:   1,
		},
		{
			name:         "cargo toml only",
			files:        []string{"Cargo.toml"},
			expectedType: Rust,
			minHotDirs:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := createTempProject(t, tt.files)
			defer os.RemoveAll(dir)

			info, err := DetectStack(dir)
			if err != nil {
				t.Fatalf("DetectStack() error = %v", err)
			}

			if info.Type != tt.expectedType {
				t.Errorf("Type = %v, want %v", info.Type, tt.expectedType)
			}

			if len(info.HotDirs) < tt.minHotDirs {
				t.Errorf("HotDirs count = %d, want at least %d", len(info.HotDirs), tt.minHotDirs)
			}

			// Verify target is in hot dirs
			foundTarget := false
			for _, dir := range info.HotDirs {
				if dir == "target" {
					foundTarget = true
					break
				}
			}
			if !foundTarget {
				t.Errorf("HotDirs missing 'target'")
			}

			// Verify watcher commands are populated
			if len(info.WatcherCommands) == 0 {
				t.Errorf("WatcherCommands is empty")
			}
		})
	}
}

// TestDetectStack_Go tests Go project detection
func TestDetectStack_Go(t *testing.T) {
	tests := []struct {
		name         string
		files        []string
		dirs         []string
		expectedType StackType
		expectVendor bool
	}{
		{
			name:         "go mod project",
			files:        []string{"go.mod", "go.sum"},
			dirs:         []string{},
			expectedType: Go,
			expectVendor: false,
		},
		{
			name:         "go mod with vendor",
			files:        []string{"go.mod", "go.sum"},
			dirs:         []string{"vendor"},
			expectedType: Go,
			expectVendor: true,
		},
		{
			name:         "go mod only",
			files:        []string{"go.mod"},
			dirs:         []string{},
			expectedType: Go,
			expectVendor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := createTempProjectWithDirs(t, tt.files, tt.dirs)
			defer os.RemoveAll(dir)

			info, err := DetectStack(dir)
			if err != nil {
				t.Fatalf("DetectStack() error = %v", err)
			}

			if info.Type != tt.expectedType {
				t.Errorf("Type = %v, want %v", info.Type, tt.expectedType)
			}

			// Check if vendor is in hot dirs
			foundVendor := false
			for _, dir := range info.HotDirs {
				if dir == "vendor" {
					foundVendor = true
					break
				}
			}

			if tt.expectVendor && !foundVendor {
				t.Errorf("Expected vendor in HotDirs, but not found")
			}
			if !tt.expectVendor && foundVendor {
				t.Errorf("Did not expect vendor in HotDirs, but found it")
			}

			// Verify watcher commands are populated
			if len(info.WatcherCommands) == 0 {
				t.Errorf("WatcherCommands is empty")
			}
		})
	}
}

// TestDetectStack_Mixed tests mixed project detection
func TestDetectStack_Mixed(t *testing.T) {
	tests := []struct {
		name          string
		files         []string
		expectedType  StackType
		expectedTypes []StackType
	}{
		{
			name:          "node and python",
			files:         []string{"package.json", "requirements.txt"},
			expectedType:  Mixed,
			expectedTypes: []StackType{Node, Python},
		},
		{
			name:          "rust and node",
			files:         []string{"Cargo.toml", "package.json"},
			expectedType:  Mixed,
			expectedTypes: []StackType{Node, Rust},
		},
		{
			name:          "go and python",
			files:         []string{"go.mod", "pyproject.toml"},
			expectedType:  Mixed,
			expectedTypes: []StackType{Python, Go},
		},
		{
			name:          "all four",
			files:         []string{"package.json", "requirements.txt", "Cargo.toml", "go.mod"},
			expectedType:  Mixed,
			expectedTypes: []StackType{Node, Python, Rust, Go},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := createTempProject(t, tt.files)
			defer os.RemoveAll(dir)

			info, err := DetectStack(dir)
			if err != nil {
				t.Fatalf("DetectStack() error = %v", err)
			}

			if info.Type != tt.expectedType {
				t.Errorf("Type = %v, want %v", info.Type, tt.expectedType)
			}

			if len(info.Types) != len(tt.expectedTypes) {
				t.Errorf("Types count = %d, want %d", len(info.Types), len(tt.expectedTypes))
			}

			// Verify all expected types are present
			for _, expectedType := range tt.expectedTypes {
				found := false
				for _, actualType := range info.Types {
					if actualType == expectedType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected type %v not found in Types", expectedType)
				}
			}

			// Verify hot dirs from multiple stacks are combined
			if len(info.HotDirs) == 0 {
				t.Errorf("HotDirs is empty for mixed project")
			}

			// Verify watcher commands from multiple stacks are combined
			if len(info.WatcherCommands) == 0 {
				t.Errorf("WatcherCommands is empty for mixed project")
			}
		})
	}
}

// TestDetectStack_Unknown tests detection of unknown projects
func TestDetectStack_Unknown(t *testing.T) {
	tests := []struct {
		name         string
		files        []string
		expectedType StackType
	}{
		{
			name:         "empty directory",
			files:        []string{},
			expectedType: Unknown,
		},
		{
			name:         "only readme",
			files:        []string{"README.md"},
			expectedType: Unknown,
		},
		{
			name:         "random files",
			files:        []string{"data.csv", "notes.txt"},
			expectedType: Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := createTempProject(t, tt.files)
			defer os.RemoveAll(dir)

			info, err := DetectStack(dir)
			if err != nil {
				t.Fatalf("DetectStack() error = %v", err)
			}

			if info.Type != tt.expectedType {
				t.Errorf("Type = %v, want %v", info.Type, tt.expectedType)
			}

			if len(info.Types) != 0 {
				t.Errorf("Types should be empty for unknown project, got %d types", len(info.Types))
			}

			if len(info.HotDirs) != 0 {
				t.Errorf("HotDirs should be empty for unknown project, got %d", len(info.HotDirs))
			}
		})
	}
}

// TestDetectStack_InvalidPath tests error handling for invalid paths
func TestDetectStack_InvalidPath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "non-existent directory",
			path:        "/tmp/silibox-test-nonexistent-12345",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DetectStack(tt.path)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for path %s, got nil", tt.path)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for path %s: %v", tt.path, err)
			}
		})
	}
}

// TestDetectStack_FileAsPath tests that a file path returns an error
func TestDetectStack_FileAsPath(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "silibox-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	_, err = DetectStack(tmpfile.Name())
	if err == nil {
		t.Error("Expected error when passing file path, got nil")
	}
}

// TestStackType_String tests the String() method
func TestStackType_String(t *testing.T) {
	tests := []struct {
		stackType StackType
		expected  string
	}{
		{Unknown, "Unknown"},
		{Node, "Node"},
		{Python, "Python"},
		{Rust, "Rust"},
		{Go, "Go"},
		{Mixed, "Mixed"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.stackType.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Helper function to create a temporary project with files
func createTempProject(t *testing.T, files []string) string {
	t.Helper()
	dir := t.TempDir()

	for _, file := range files {
		fullPath := filepath.Join(dir, file)
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	return dir
}

// Helper function to create a temporary project with files and directories
func createTempProjectWithDirs(t *testing.T, files []string, dirs []string) string {
	t.Helper()
	dir := t.TempDir()

	for _, file := range files {
		fullPath := filepath.Join(dir, file)
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	for _, d := range dirs {
		fullPath := filepath.Join(dir, d)
		if err := os.Mkdir(fullPath, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", d, err)
		}
	}

	return dir
}
