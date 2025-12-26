package stack

import (
	"strings"
)

// DetectWatcher checks if a command is a known file watcher and returns its configuration
func DetectWatcher(command []string, projectPath string) *WatcherInfo {
	if len(command) == 0 {
		return nil
	}

	// Get project info to access stack-specific watchers
	projectInfo, err := DetectStack(projectPath)
	if err != nil || projectInfo == nil {
		return nil
	}

	// Build command string for matching
	cmdStr := strings.Join(command, " ")

	// Check against known watchers
	for _, watcher := range projectInfo.Watchers {
		if isWatcherMatch(cmdStr, watcher.Command) {
			return &watcher
		}
	}

	// Check for common watcher flags as fallback
	if hasWatcherFlags(command) {
		// Return generic polling env vars
		return &WatcherInfo{
			Command: cmdStr,
			EnvVars: map[string]string{
				"CHOKIDAR_USEPOLLING": "true",
				"WATCHPACK_POLLING":   "true",
			},
		}
	}

	return nil
}

// isWatcherMatch checks if a command matches a watcher pattern
func isWatcherMatch(cmdStr, pattern string) bool {
	// Normalize both strings
	cmdStr = strings.ToLower(strings.TrimSpace(cmdStr))
	pattern = strings.ToLower(strings.TrimSpace(pattern))

	// Exact match
	if cmdStr == pattern {
		return true
	}

	// Prefix match (e.g., "npm run dev --port 3000" matches "npm run dev")
	if strings.HasPrefix(cmdStr, pattern+" ") {
		return true
	}

	// For "npm run X" commands, check if it's the same script
	// e.g., "npm run dev" should match "npm run dev" but not "npm run build"
	if strings.HasPrefix(pattern, "npm run ") || strings.HasPrefix(pattern, "yarn ") ||
		strings.HasPrefix(pattern, "pnpm ") || strings.HasPrefix(pattern, "bun run ") {
		// Already handled by prefix match above
		return false
	}

	// For direct commands (vite, webpack, etc.), match command name
	patternParts := strings.Fields(pattern)
	cmdParts := strings.Fields(cmdStr)
	if len(patternParts) > 0 && len(cmdParts) > 0 {
		return patternParts[0] == cmdParts[0]
	}

	return false
}

// hasWatcherFlags checks if command has common watcher flags
func hasWatcherFlags(command []string) bool {
	for _, arg := range command {
		arg = strings.ToLower(arg)
		if arg == "--watch" || arg == "-w" || arg == "--reload" || 
		   arg == "-f" || arg == "--follow" || strings.Contains(arg, "watch") {
			return true
		}
	}
	return false
}
