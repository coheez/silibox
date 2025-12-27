package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/coheez/silibox/internal/lima"
	"github.com/spf13/cobra"
)

var (
	uninstallAll bool
	uninstallYes bool
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall sili (optionally purge VM and state)",
	Long: `Removes the sili binary. With --all, also deletes the Lima VM (\"silibox\") and ~/.sili state/shims.

By default this only removes the binary. Use --all to purge everything.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !uninstallYes {
			if !confirm(uninstallAll) {
				fmt.Println("aborted")
				return nil
			}
		}

		if uninstallAll {
			// Stop VM if present (ignore errors)
			_ = runSilent("limactl", "stop", lima.Instance)
			// Delete VM
			_ = runSilent("limactl", "delete", lima.Instance)
			// Remove ~/.sili directory
			home, _ := os.UserHomeDir()
			_ = os.RemoveAll(filepath.Join(home, ".sili"))
			fmt.Println("✓ removed VM and ~/.sili state")
		}

		// Remove the current binary (schedule removal after exit)
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("unable to determine executable path: %w", err)
		}

		if err := scheduleSelfRemove(execPath); err != nil {
			return fmt.Errorf("failed to remove binary: %w", err)
		}
		fmt.Printf("✓ scheduled removal of binary: %s\n", execPath)
		fmt.Println("sili has been uninstalled. If the file still exists, remove it manually with: sudo rm -f /usr/local/bin/sili")
		return nil
	},
}

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallAll, "all", false, "Also delete VM and ~/.sili state (purge)")
	uninstallCmd.Flags().BoolVar(&uninstallYes, "yes", false, "Do not prompt for confirmation")
}

func confirm(all bool) bool {
	reader := bufio.NewReader(os.Stdin)
	if all {
		fmt.Print("This will delete the VM, all environments, and ~/.sili state. Continue? [y/N]: ")
	} else {
		fmt.Print("This will remove the sili binary. Continue? [y/N]: ")
	}
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

func runSilent(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func scheduleSelfRemove(path string) error {
	// Try normal removal (works on Unix for running binaries), otherwise fallback to sudo via background shell
	if err := os.Remove(path); err == nil {
		return nil
	}
	// Background shell removes after process exits
	script := fmt.Sprintf("sh -c 'sleep 0.2; rm -f %q || sudo rm -f %q' &", path, path)
	cmd := exec.Command("sh", "-c", script)
	return cmd.Start()
}