package stack

import (
	"fmt"
	"os"
	"path/filepath"
)

// StackType represents the type of project detected
type StackType int

const (
	Unknown StackType = iota
	Node
	Python
	Rust
	Go
	Mixed
)

func (s StackType) String() string {
	switch s {
	case Node:
		return "Node"
	case Python:
		return "Python"
	case Rust:
		return "Rust"
	case Go:
		return "Go"
	case Mixed:
		return "Mixed"
	default:
		return "Unknown"
	}
}

// WatcherInfo describes a file watcher command and its polling configuration
type WatcherInfo struct {
	Command string            // Command pattern to match (e.g., "vite", "npm run dev")
	EnvVars map[string]string // Environment variables needed for polling
}

// ProjectInfo contains information about a detected project
type ProjectInfo struct {
	Type            StackType         // Primary stack type
	Types           []StackType       // All detected stack types (for mixed projects)
	HotDirs         []string          // Directories to move into volumes for performance
	ConfigFiles     map[string]bool   // Detected configuration files
	WatcherCommands []string          // Known watcher commands for this stack (deprecated, use Watchers)
	Watchers        []WatcherInfo     // File watchers with polling configuration
	PackageManager  string            // Detected package manager (npm, yarn, pnpm, bun, etc.)
}

// DetectStack analyzes a project directory and determines the language stack
func DetectStack(projectPath string) (*ProjectInfo, error) {
	// Verify project path exists
	info, err := os.Stat(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access project path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project path is not a directory: %s", projectPath)
	}

	// Initialize project info
	projectInfo := &ProjectInfo{
		Type:            Unknown,
		Types:           make([]StackType, 0),
		HotDirs:         make([]string, 0),
		ConfigFiles:     make(map[string]bool),
		WatcherCommands: make([]string, 0),
		Watchers:        make([]WatcherInfo, 0),
	}

	// Detect each stack type
	detectedTypes := make([]StackType, 0)

	if nodeInfo := detectNode(projectPath); nodeInfo != nil {
		detectedTypes = append(detectedTypes, Node)
		projectInfo.HotDirs = append(projectInfo.HotDirs, nodeInfo.HotDirs...)
		projectInfo.WatcherCommands = append(projectInfo.WatcherCommands, nodeInfo.WatcherCommands...)
		projectInfo.Watchers = append(projectInfo.Watchers, nodeInfo.Watchers...)
		projectInfo.PackageManager = nodeInfo.PackageManager
		for k, v := range nodeInfo.ConfigFiles {
			projectInfo.ConfigFiles[k] = v
		}
	}

	if pythonInfo := detectPython(projectPath); pythonInfo != nil {
		detectedTypes = append(detectedTypes, Python)
		projectInfo.HotDirs = append(projectInfo.HotDirs, pythonInfo.HotDirs...)
		projectInfo.WatcherCommands = append(projectInfo.WatcherCommands, pythonInfo.WatcherCommands...)
		projectInfo.Watchers = append(projectInfo.Watchers, pythonInfo.Watchers...)
		for k, v := range pythonInfo.ConfigFiles {
			projectInfo.ConfigFiles[k] = v
		}
	}

	if rustInfo := detectRust(projectPath); rustInfo != nil {
		detectedTypes = append(detectedTypes, Rust)
		projectInfo.HotDirs = append(projectInfo.HotDirs, rustInfo.HotDirs...)
		projectInfo.WatcherCommands = append(projectInfo.WatcherCommands, rustInfo.WatcherCommands...)
		projectInfo.Watchers = append(projectInfo.Watchers, rustInfo.Watchers...)
		for k, v := range rustInfo.ConfigFiles {
			projectInfo.ConfigFiles[k] = v
		}
	}

	if goInfo := detectGo(projectPath); goInfo != nil {
		detectedTypes = append(detectedTypes, Go)
		projectInfo.HotDirs = append(projectInfo.HotDirs, goInfo.HotDirs...)
		projectInfo.WatcherCommands = append(projectInfo.WatcherCommands, goInfo.WatcherCommands...)
		projectInfo.Watchers = append(projectInfo.Watchers, goInfo.Watchers...)
		for k, v := range goInfo.ConfigFiles {
			projectInfo.ConfigFiles[k] = v
		}
	}

	// Set the type based on what was detected
	if len(detectedTypes) == 0 {
		projectInfo.Type = Unknown
	} else if len(detectedTypes) == 1 {
		projectInfo.Type = detectedTypes[0]
	} else {
		projectInfo.Type = Mixed
	}
	projectInfo.Types = detectedTypes

	return projectInfo, nil
}

// detectNode checks for Node.js project indicators
func detectNode(projectPath string) *ProjectInfo {
	configFiles := map[string]bool{
		"package.json":     false,
		"bun.lockb":        false,
		"yarn.lock":        false,
		"pnpm-lock.yaml":   false,
		"package-lock.json": false,
	}

	// Check which files exist
	foundAny := false
	for file := range configFiles {
		if fileExists(filepath.Join(projectPath, file)) {
			configFiles[file] = true
			foundAny = true
		}
	}

	if !foundAny {
		return nil
	}

	// Determine package manager
	packageManager := "npm" // default
	if configFiles["bun.lockb"] {
		packageManager = "bun"
	} else if configFiles["pnpm-lock.yaml"] {
		packageManager = "pnpm"
	} else if configFiles["yarn.lock"] {
		packageManager = "yarn"
	}

	// Common Node.js watcher env vars for polling
	pollingEnvVars := map[string]string{
		"CHOKIDAR_USEPOLLING": "true",
		"WATCHPACK_POLLING":   "true",
	}

	return &ProjectInfo{
		Type:        Node,
		ConfigFiles: configFiles,
		HotDirs: []string{
			"node_modules",
			".pnpm-store",
			"node_modules/.cache",
			".next",
			".nuxt",
			".vite",
			"dist",
			"build",
		},
		WatcherCommands: []string{
			"npm run dev",
			"npm start",
			"yarn dev",
			"yarn start",
			"pnpm dev",
			"pnpm start",
			"bun dev",
			"bun run dev",
			"next dev",
			"vite",
			"vite dev",
			"webpack serve",
			"webpack-dev-server",
			"nodemon",
			"ts-node-dev",
		},
		Watchers: []WatcherInfo{
			{Command: "npm run dev", EnvVars: pollingEnvVars},
			{Command: "npm start", EnvVars: pollingEnvVars},
			{Command: "npm run start", EnvVars: pollingEnvVars},
			{Command: "yarn dev", EnvVars: pollingEnvVars},
			{Command: "yarn start", EnvVars: pollingEnvVars},
			{Command: "pnpm dev", EnvVars: pollingEnvVars},
			{Command: "pnpm start", EnvVars: pollingEnvVars},
			{Command: "bun dev", EnvVars: pollingEnvVars},
			{Command: "bun run dev", EnvVars: pollingEnvVars},
			{Command: "next dev", EnvVars: pollingEnvVars},
			{Command: "vite", EnvVars: pollingEnvVars},
			{Command: "vite dev", EnvVars: pollingEnvVars},
			{Command: "webpack serve", EnvVars: pollingEnvVars},
			{Command: "webpack-dev-server", EnvVars: pollingEnvVars},
			{Command: "nodemon", EnvVars: pollingEnvVars},
			{Command: "ts-node-dev", EnvVars: pollingEnvVars},
		},
		PackageManager: packageManager,
	}
}

// detectPython checks for Python project indicators
func detectPython(projectPath string) *ProjectInfo {
	configFiles := map[string]bool{
		"pyproject.toml":   false,
		"requirements.txt": false,
		"setup.py":         false,
		"Pipfile":          false,
		"poetry.lock":      false,
		"setup.cfg":        false,
	}

	// Check which files exist
	foundAny := false
	for file := range configFiles {
		if fileExists(filepath.Join(projectPath, file)) {
			configFiles[file] = true
			foundAny = true
		}
	}

	if !foundAny {
		return nil
	}

	// Python watcher env vars (most Python watchers use polling by default or have --poll flag)
	pollingEnvVars := map[string]string{
		"WATCHDOG_FORCE_POLLING": "true",
	}

	return &ProjectInfo{
		Type:        Python,
		ConfigFiles: configFiles,
		HotDirs: []string{
			".venv",
			"venv",
			".tox",
			"__pycache__",
			".pytest_cache",
			".mypy_cache",
			".ruff_cache",
			"*.egg-info",
			".eggs",
			"build",
			"dist",
		},
		WatcherCommands: []string{
			"pytest -f",
			"pytest --watch",
			"ptw",
			"watchdog",
			"flask run",
			"flask run --reload",
			"uvicorn --reload",
			"fastapi dev",
		},
		Watchers: []WatcherInfo{
			{Command: "flask run", EnvVars: pollingEnvVars},
			{Command: "flask run --reload", EnvVars: pollingEnvVars},
			{Command: "uvicorn --reload", EnvVars: pollingEnvVars},
			{Command: "fastapi dev", EnvVars: pollingEnvVars},
			{Command: "watchdog", EnvVars: pollingEnvVars},
		},
	}
}

// detectRust checks for Rust project indicators
func detectRust(projectPath string) *ProjectInfo {
	configFiles := map[string]bool{
		"Cargo.toml": false,
		"Cargo.lock": false,
	}

	// Check which files exist
	foundAny := false
	for file := range configFiles {
		if fileExists(filepath.Join(projectPath, file)) {
			configFiles[file] = true
			foundAny = true
		}
	}

	if !foundAny {
		return nil
	}

	// Rust watcher env vars (cargo watch uses notify which may need polling)
	pollingEnvVars := map[string]string{
		"CARGO_WATCH_POLL": "1",
	}

	return &ProjectInfo{
		Type:        Rust,
		ConfigFiles: configFiles,
		HotDirs: []string{
			"target",
			".cargo/registry",
			".cargo/git",
		},
		WatcherCommands: []string{
			"cargo watch",
			"cargo watch -x run",
			"cargo watch -x test",
			"cargo run --watch",
		},
		Watchers: []WatcherInfo{
			{Command: "cargo watch", EnvVars: pollingEnvVars},
		},
	}
}

// detectGo checks for Go project indicators
func detectGo(projectPath string) *ProjectInfo {
	configFiles := map[string]bool{
		"go.mod": false,
		"go.sum": false,
	}

	// Check which files exist
	foundAny := false
	for file := range configFiles {
		if fileExists(filepath.Join(projectPath, file)) {
			configFiles[file] = true
			foundAny = true
		}
	}

	if !foundAny {
		return nil
	}

	// Check if vendor directory exists
	hotDirs := []string{}
	if dirExists(filepath.Join(projectPath, "vendor")) {
		hotDirs = append(hotDirs, "vendor")
	}

	// Go watcher env vars (most Go watchers handle polling internally)
	pollingEnvVars := map[string]string{
		"POLLING": "true",
	}

	return &ProjectInfo{
		Type:        Go,
		ConfigFiles: configFiles,
		HotDirs:     hotDirs,
		WatcherCommands: []string{
			"air",
			"fresh",
			"realize start",
			"gow run",
			"reflex",
		},
		Watchers: []WatcherInfo{
			{Command: "air", EnvVars: pollingEnvVars},
			{Command: "gow run", EnvVars: pollingEnvVars},
		},
	}
}

// fileExists checks if a file exists and is not a directory
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
