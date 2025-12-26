package shim

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	shimTemplate = `#!/bin/bash
# Silibox shim for %s in environment %s
exec sili run --name %s -- %s "$@"
`
)

// ShimDir returns the directory where shims are stored
func ShimDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".sili", "bin"), nil
}

// EnsureShimDir creates the shim directory if it doesn't exist
func EnsureShimDir() error {
	shimDir, err := ShimDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(shimDir, 0755)
}

// GenerateShim creates a shim script for a command in an environment
func GenerateShim(envName, command string, force bool) error {
	if err := EnsureShimDir(); err != nil {
		return fmt.Errorf("failed to create shim directory: %w", err)
	}

	shimDir, err := ShimDir()
	if err != nil {
		return err
	}

	shimPath := filepath.Join(shimDir, command)

	// Check if shim already exists
	if _, err := os.Stat(shimPath); err == nil && !force {
		return fmt.Errorf("shim %s already exists (use --force to overwrite)", command)
	}

	// Generate shim content
	content := fmt.Sprintf(shimTemplate, command, envName, envName, command)

	// Write shim file
	if err := os.WriteFile(shimPath, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to write shim file: %w", err)
	}

	return nil
}

// RemoveShim deletes a shim script
func RemoveShim(command string) error {
	shimDir, err := ShimDir()
	if err != nil {
		return err
	}

	shimPath := filepath.Join(shimDir, command)

	if err := os.Remove(shimPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("shim %s does not exist", command)
		}
		return fmt.Errorf("failed to remove shim: %w", err)
	}

	return nil
}

// IsInPATH checks if the shim directory is in the user's PATH
func IsInPATH() (bool, error) {
	shimDir, err := ShimDir()
	if err != nil {
		return false, err
	}

	pathEnv := os.Getenv("PATH")
	pathParts := strings.Split(pathEnv, ":")

	for _, part := range pathParts {
		// Expand ~ to home directory for comparison
		expandedPart := part
		if strings.HasPrefix(part, "~/") {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				expandedPart = filepath.Join(homeDir, part[2:])
			}
		}

		// Compare absolute paths
		if absPath, err := filepath.Abs(expandedPart); err == nil {
			if absShimDir, err := filepath.Abs(shimDir); err == nil {
				if absPath == absShimDir {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// GetPATHInstructions returns shell-specific instructions for adding shim dir to PATH
func GetPATHInstructions() (string, error) {
	shimDir, err := ShimDir()
	if err != nil {
		return "", err
	}

	shell := os.Getenv("SHELL")
	var instructions string

	if strings.Contains(shell, "zsh") {
		instructions = fmt.Sprintf(`Add this line to your ~/.zshrc:
    export PATH="%s:$PATH"

Then run: source ~/.zshrc`, shimDir)
	} else if strings.Contains(shell, "bash") {
		instructions = fmt.Sprintf(`Add this line to your ~/.bashrc or ~/.bash_profile:
    export PATH="%s:$PATH"

Then run: source ~/.bashrc`, shimDir)
	} else if strings.Contains(shell, "fish") {
		instructions = fmt.Sprintf(`Run this command:
    fish_add_path %s`, shimDir)
	} else {
		instructions = fmt.Sprintf(`Add the shim directory to your PATH:
    export PATH="%s:$PATH"`, shimDir)
	}

	return instructions, nil
}
