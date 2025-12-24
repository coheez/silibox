package container

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/coheez/silibox/internal/lima"
)

// MigrateDirToVolume migrates a directory from the host to a Podman volume
// This is necessary because we can't mount volumes inside host-mounted directories
// Solution: move the directory to a volume, create backup on host, volume mount fills the gap
func MigrateDirToVolume(envName, projectPath, dirName, volumeName string) error {
	hostPath := filepath.Join(projectPath, dirName)

	// Verify directory exists and is not empty
	entries, err := os.ReadDir(hostPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}
	if len(entries) == 0 {
		// Empty directory, no need to migrate
		return nil
	}

	fmt.Printf("Migrating %s to volume %s...\n", dirName, volumeName)

	// Step 1: Create backup on host with timestamp
	timestamp := time.Now().Unix()
	backupPath := fmt.Sprintf("%s.silibox-backup-%d", hostPath, timestamp)
	
	fmt.Printf("Creating backup at %s\n", filepath.Base(backupPath))
	if err := os.Rename(hostPath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Step 2: Copy contents to volume using a temporary container
	// We mount both the backup directory and the volume, then copy
	fmt.Printf("Copying contents to volume (this may take a moment)...\n")
	
	// Use alpine for the copy operation (small, fast)
	copyCmd := exec.Command(
		"limactl", "shell", lima.Instance, "--", "podman", "run", "--rm",
		"-v", fmt.Sprintf("%s:/src:ro", backupPath), // Backup dir as read-only source
		"-v", fmt.Sprintf("%s:/dest", volumeName),   // Volume as destination
		"alpine:latest",
		"sh", "-c", "cp -a /src/. /dest/", // Copy all contents including hidden files
	)
	copyCmd.Stdout = os.Stdout
	copyCmd.Stderr = os.Stderr
	
	if err := copyCmd.Run(); err != nil {
		// Copy failed - restore backup
		fmt.Fprintf(os.Stderr, "Migration failed, restoring backup...\n")
		if restoreErr := os.Rename(backupPath, hostPath); restoreErr != nil {
			return fmt.Errorf("migration failed and backup restore failed: %w (original error: %v)", restoreErr, err)
		}
		return fmt.Errorf("failed to copy to volume: %w", err)
	}

	fmt.Printf("✓ Successfully migrated %s to volume\n", dirName)
	fmt.Printf("  Backup kept at: %s\n", filepath.Base(backupPath))
	fmt.Printf("  You can delete the backup once you verify everything works\n")

	return nil
}

// GetDirSize calculates the size of a directory in bytes
func GetDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip errors (e.g., permission denied)
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// FormatBytes formats bytes as human-readable string
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
