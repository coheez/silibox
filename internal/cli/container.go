package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/coheez/silibox/internal/container"
	"github.com/coheez/silibox/internal/state"
	"github.com/coheez/silibox/internal/vm"
	"github.com/spf13/cobra"
)

var (
	createName          string
	createImage         string
	createDir           string
	createWork          string
	createUser          string
	createPorts         []string
	createDetectVolumes bool
	createNoMigrate     bool
	enterName           string
	enterShell          string
	runName             string
	runNoPolling        bool
	runForcePolling     bool
	stopName            string
	rmName              string
	rmForce             bool
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a named Podman container in the VM",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Pass through common environment variables
		env := make(map[string]string)
		for _, key := range []string{"PATH", "HOME", "USER", "SHELL", "TERM", "LANG", "LC_ALL"} {
			if value := os.Getenv(key); value != "" {
				env[key] = value
			}
		}

		cfg := container.CreateConfig{
			Name:                    createName,
			Image:                   createImage,
			ProjectDir:              createDir,
			WorkingDir:              createWork,
			User:                    createUser,
			Environment:             env,
			Ports:                   createPorts,
			DetectAndPrepareVolumes: createDetectVolumes,
			NoMigrate:               createNoMigrate,
		}
		return container.Create(cfg)
	},
}

var enterCmd = &cobra.Command{
	Use:   "enter",
	Short: "Enter an interactive shell in a running container",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Ensure VM is running (auto-wake)
		if err := vm.EnsureVMRunning(); err != nil {
			return err
		}
		return container.Enter(enterName, enterShell)
	},
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run arbitrary commands inside a container (non-interactive)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("no command specified")
		}

		// Ensure VM is running (auto-wake)
		if err := vm.EnsureVMRunning(); err != nil {
			return err
		}

		runOpts := container.RunOptions{
			EnablePolling: !runNoPolling, // Enabled by default unless --no-polling
			ForcePolling:  runForcePolling,
		}

		result, err := container.RunWithOptions(runName, args, runOpts)
		if err != nil {
			return err
		}

		// Print stdout and stderr
		if result.Stdout != "" {
			fmt.Print(result.Stdout)
		}
		if result.Stderr != "" {
			fmt.Fprint(os.Stderr, result.Stderr)
		}

		// Exit with the same code as the command
		os.Exit(result.ExitCode)
		return nil // This line won't be reached due to os.Exit above
	},
}

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all known environments",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load state to get all environments
		st, err := state.Load()
		if err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}

		envs := st.ListEnvs()
		if len(envs) == 0 {
			fmt.Println("No environments found. Create one with 'sili create'.")
			return nil
		}

		// Sort environments by name for consistent output
		sort.Slice(envs, func(i, j int) bool {
			return envs[i].Name < envs[j].Name
		})

		// Get actual running containers from Podman
		runningContainers, err := container.List()
		if err != nil {
			// If we can't get running containers, we'll just use state info
			runningContainers = []string{}
		}
		runningMap := make(map[string]bool)
		for _, name := range runningContainers {
			runningMap[name] = true
		}

		// Print header
		fmt.Printf("%-20s %-15s %-30s %s\n", "NAME", "STATUS", "IMAGE", "LAST ACTIVE")
		fmt.Println(strings.Repeat("-", 90))

		// Print each environment
		for _, env := range envs {
			// Determine actual status
			status := "stopped"
			if runningMap[env.Name] {
				status = "running"
			}

			// Format last active time
			lastActive := formatRelativeTime(env.LastActive)

			// Truncate image if too long
			image := env.Image
			if len(image) > 30 {
				image = image[:27] + "..."
			}

			fmt.Printf("%-20s %-15s %-30s %s\n", env.Name, status, image, lastActive)
		}

		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a running container",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Ensure VM is running (auto-wake)
		if err := vm.EnsureVMRunning(); err != nil {
			return err
		}
		if err := container.Stop(stopName); err != nil {
			return err
		}
		fmt.Printf("Stopped environment: %s\n", stopName)
		return nil
	},
}

var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove a container and clean up resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Ensure VM is running (auto-wake)
		if err := vm.EnsureVMRunning(); err != nil {
			return err
		}
		if err := container.Remove(rmName, rmForce); err != nil {
			return err
		}
		fmt.Printf("Removed environment: %s\n", rmName)
		return nil
	},
}

// formatRelativeTime formats a time as a relative string (e.g., "2 hours ago")
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}

	duration := time.Since(t)
	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	days := int(duration.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	if days < 7 {
		return fmt.Sprintf("%d days ago", days)
	}
	weeks := days / 7
	if weeks == 1 {
		return "1 week ago"
	}
	if weeks < 4 {
		return fmt.Sprintf("%d weeks ago", weeks)
	}
	return t.Format("Jan 2, 2006")
}

func init() {
	rootCmd.AddCommand(createCmd, enterCmd, runCmd, lsCmd, stopCmd, rmCmd)
	createCmd.Flags().StringVarP(&createName, "name", "n", "silibox-dev", "Container name")
	createCmd.Flags().StringVarP(&createImage, "image", "i", "ubuntu:22.04", "Container image")
	createCmd.Flags().StringVarP(&createDir, "dir", "d", ".", "Project directory to bind mount")
	createCmd.Flags().StringVarP(&createWork, "workdir", "w", "/workspace", "Working directory inside container")
	createCmd.Flags().StringVarP(&createUser, "user", "u", "", "User to run as (default: current user)")
	createCmd.Flags().StringArrayVarP(&createPorts, "ports", "p", []string{}, "Port mappings (format: 3000 or 8080:80 or 8080:80/tcp)")
	createCmd.Flags().BoolVar(&createDetectVolumes, "detect-volumes", false, "[Experimental] Enable automatic project stack detection and volume creation")
	createCmd.Flags().BoolVar(&createNoMigrate, "no-migrate", false, "Skip migration prompts for existing directories when using --detect-volumes")
	enterCmd.Flags().StringVarP(&enterName, "name", "n", "silibox-dev", "Container name to enter")
	enterCmd.Flags().StringVarP(&enterShell, "shell", "s", "bash", "Shell to use (bash, sh, zsh, etc.)")
	runCmd.Flags().StringVarP(&runName, "name", "n", "silibox-dev", "Container name to run command in")
	runCmd.Flags().BoolVar(&runNoPolling, "no-polling", false, "Disable automatic polling mode for file watchers")
	runCmd.Flags().BoolVar(&runForcePolling, "force-polling", false, "Force polling mode even if not detected as watcher")
	stopCmd.Flags().StringVarP(&stopName, "name", "n", "silibox-dev", "Container name to stop")
	rmCmd.Flags().StringVarP(&rmName, "name", "n", "silibox-dev", "Container name to remove")
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Force remove even if running")
}
