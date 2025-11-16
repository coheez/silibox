package cli

import (
	"fmt"
	"os"

	"github.com/coheez/silibox/internal/container"
	"github.com/spf13/cobra"
)

var (
	createName  string
	createImage string
	createDir   string
	createWork  string
	createUser  string
	enterName   string
	enterShell  string
	runName     string
	stopName    string
	rmName      string
	rmForce     bool
	startName   string
	logsName    string
	logsFollow  bool
	logsTail    int
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
			Name:        createName,
			Image:       createImage,
			ProjectDir:  createDir,
			WorkingDir:  createWork,
			User:        createUser,
			Environment: env,
		}
		return container.Create(cfg)
	},
}

var enterCmd = &cobra.Command{
	Use:   "enter",
	Short: "Enter an interactive shell in a running container",
	RunE: func(cmd *cobra.Command, args []string) error {
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

		result, err := container.Run(runName, args)
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

var psCmd = &cobra.Command{
	Use:     "ps",
	Aliases: []string{"list"},
	Short:   "List running containers in the VM",
	RunE: func(cmd *cobra.Command, args []string) error {
		containers, err := container.List()
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}

		if len(containers) == 0 {
			fmt.Println("No containers running")
			return nil
		}

		fmt.Println("CONTAINER NAME")
		for _, name := range containers {
			fmt.Println(name)
		}
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a running container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := container.Stop(name); err != nil {
			return fmt.Errorf("failed to stop container %s: %w", name, err)
		}
		fmt.Printf("Container %s stopped\n", name)
		return nil
	},
}

var rmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Remove a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// If force flag is set, stop the container first
		if rmForce {
			container.Stop(name) // Ignore errors if already stopped
		}

		if err := container.Remove(name); err != nil {
			return fmt.Errorf("failed to remove container %s: %w", name, err)
		}
		fmt.Printf("Container %s removed\n", name)
		return nil
	},
}

var startCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start a stopped container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := container.Start(name); err != nil {
			return fmt.Errorf("failed to start container %s: %w", name, err)
		}
		fmt.Printf("Container %s started\n", name)
		return nil
	},
}

var logsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "View container logs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := container.Logs(name, logsFollow, logsTail); err != nil {
			return fmt.Errorf("failed to get logs for container %s: %w", name, err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd, enterCmd, runCmd, psCmd, stopCmd, rmCmd, startCmd, logsCmd)

	createCmd.Flags().StringVarP(&createName, "name", "n", "silibox-dev", "Container name")
	createCmd.Flags().StringVarP(&createImage, "image", "i", "ubuntu:22.04", "Container image")
	createCmd.Flags().StringVarP(&createDir, "dir", "d", ".", "Project directory to bind mount")
	createCmd.Flags().StringVarP(&createWork, "workdir", "w", "/workspace", "Working directory inside container")
	createCmd.Flags().StringVarP(&createUser, "user", "u", "", "User to run as (default: current user)")

	enterCmd.Flags().StringVarP(&enterName, "name", "n", "silibox-dev", "Container name to enter")
	enterCmd.Flags().StringVarP(&enterShell, "shell", "s", "bash", "Shell to use (bash, sh, zsh, etc.)")

	runCmd.Flags().StringVarP(&runName, "name", "n", "silibox-dev", "Container name to run command in")

	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Force remove (stop if running)")

	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVarP(&logsTail, "tail", "n", 50, "Number of lines to show from the end of the logs")
}
