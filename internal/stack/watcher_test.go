package stack

import (
	"testing"
)

func TestIsWatcherMatch(t *testing.T) {
	tests := []struct {
		name    string
		cmdStr  string
		pattern string
		want    bool
	}{
		// Exact matches
		{
			name:    "exact match - vite",
			cmdStr:  "vite",
			pattern: "vite",
			want:    true,
		},
		{
			name:    "exact match - npm run dev",
			cmdStr:  "npm run dev",
			pattern: "npm run dev",
			want:    true,
		},
		// Prefix matches (command with additional args)
		{
			name:    "prefix match - vite with port",
			cmdStr:  "vite --port 3000",
			pattern: "vite",
			want:    true,
		},
		{
			name:    "prefix match - npm run dev with args",
			cmdStr:  "npm run dev --host 0.0.0.0",
			pattern: "npm run dev",
			want:    true,
		},
		// Case insensitive
		{
			name:    "case insensitive - VITE",
			cmdStr:  "VITE",
			pattern: "vite",
			want:    true,
		},
		// Different npm scripts should NOT match
		{
			name:    "different npm script",
			cmdStr:  "npm run build",
			pattern: "npm run dev",
			want:    false,
		},
		// Command name matches
		{
			name:    "webpack with args matches webpack pattern",
			cmdStr:  "webpack serve --mode development",
			pattern: "webpack serve",
			want:    true,
		},
		// Non-matches
		{
			name:    "different command",
			cmdStr:  "node app.js",
			pattern: "vite",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWatcherMatch(tt.cmdStr, tt.pattern)
			if got != tt.want {
				t.Errorf("isWatcherMatch(%q, %q) = %v, want %v", tt.cmdStr, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestHasWatcherFlags(t *testing.T) {
	tests := []struct {
		name    string
		command []string
		want    bool
	}{
		{
			name:    "has --watch flag",
			command: []string{"pytest", "--watch"},
			want:    true,
		},
		{
			name:    "has -w flag",
			command: []string{"jest", "-w"},
			want:    true,
		},
		{
			name:    "has --reload flag",
			command: []string{"flask", "run", "--reload"},
			want:    true,
		},
		{
			name:    "has -f flag",
			command: []string{"pytest", "-f"},
			want:    true,
		},
		{
			name:    "has watch in command name",
			command: []string{"cargo-watch"},
			want:    true,
		},
		{
			name:    "no watcher flags",
			command: []string{"npm", "run", "build"},
			want:    false,
		},
		{
			name:    "empty command",
			command: []string{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasWatcherFlags(tt.command)
			if got != tt.want {
				t.Errorf("hasWatcherFlags(%v) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestDetectWatcher(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		command     []string
		projectPath string
		wantNil     bool
		wantEnvVars bool
	}{
		{
			name:        "empty command",
			command:     []string{},
			projectPath: tmpDir,
			wantNil:     true,
		},
		{
			name:        "non-watcher command",
			command:     []string{"echo", "hello"},
			projectPath: tmpDir,
			wantNil:     true,
		},
		{
			name:        "command with --watch flag",
			command:     []string{"pytest", "--watch"},
			projectPath: tmpDir,
			wantNil:     false,
			wantEnvVars: true,
		},
		{
			name:        "command with --reload flag",
			command:     []string{"uvicorn", "main:app", "--reload"},
			projectPath: tmpDir,
			wantNil:     false,
			wantEnvVars: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectWatcher(tt.command, tt.projectPath)
			if tt.wantNil {
				if got != nil {
					t.Errorf("DetectWatcher() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Error("DetectWatcher() = nil, want non-nil")
					return
				}
				if tt.wantEnvVars && len(got.EnvVars) == 0 {
					t.Error("DetectWatcher() returned watcher with no env vars")
				}
			}
		})
	}
}
