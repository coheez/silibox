package cli

import (
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

func init() {
	rootCmd.AddCommand(createCmd, enterCmd)
	createCmd.Flags().StringVarP(&createName, "name", "n", "silibox-dev", "Container name")
	createCmd.Flags().StringVarP(&createImage, "image", "i", "ubuntu:22.04", "Container image")
	createCmd.Flags().StringVarP(&createDir, "dir", "d", ".", "Project directory to bind mount")
	createCmd.Flags().StringVarP(&createWork, "workdir", "w", "/workspace", "Working directory inside container")
	createCmd.Flags().StringVarP(&createUser, "user", "u", "", "User to run as (default: current user)")
	enterCmd.Flags().StringVarP(&enterName, "name", "n", "silibox-dev", "Container name to enter")
	enterCmd.Flags().StringVarP(&enterShell, "shell", "s", "bash", "Shell to use (bash, sh, zsh, etc.)")
}
