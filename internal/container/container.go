package container

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/coheez/silibox/internal/lima"
)

type CreateConfig struct {
	Name        string
	Image       string
	ProjectDir  string
	WorkingDir  string
	User        string
	Environment map[string]string
}

// Create pulls the image and starts a named Podman container with proper bind mounts and UID/GID mapping
func Create(cfg CreateConfig) error {
	// Ensure VM is running
	inst, found, err := lima.GetInstance()
	if err != nil {
		return fmt.Errorf("failed to check VM status: %w", err)
	}
	if !found || inst.Status != "Running" {
		return fmt.Errorf("VM is not running (status: %s). Run 'sili vm up' first", inst.Status)
	}

	// Get current user UID/GID for mapping
	uid, gid, err := getCurrentUserIDs()
	if err != nil {
		return fmt.Errorf("failed to get user IDs: %w", err)
	}

	// Pull the image
	if err := pullImage(cfg.Image); err != nil {
		return fmt.Errorf("failed to pull image %s: %w", cfg.Image, err)
	}

	// Create the container
	return createContainer(cfg, uid, gid)
}

func getCurrentUserIDs() (int, int, error) {
	currentUser, err := user.Current()
	if err != nil {
		return 0, 0, err
	}

	uid, err := strconv.Atoi(currentUser.Uid)
	if err != nil {
		return 0, 0, err
	}

	gid, err := strconv.Atoi(currentUser.Gid)
	if err != nil {
		return 0, 0, err
	}

	return uid, gid, nil
}

func pullImage(image string) error {
	cmd := exec.Command("limactl", "shell", lima.Instance, "--", "podman", "pull", image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func createContainer(cfg CreateConfig, uid, gid int) error {
	// Get absolute paths
	projectDir, err := filepath.Abs(cfg.ProjectDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute project path: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Build podman run command
	args := []string{
		"shell", lima.Instance, "--", "podman", "run",
		"-d", // detached
		"--name", cfg.Name,
		"--user", fmt.Sprintf("%d:%d", uid, gid),
		"-v", fmt.Sprintf("%s:/workspace", projectDir), // project dir (writable)
		"-v", fmt.Sprintf("%s:/home/host:ro", homeDir), // home dir (read-only)
		"-w", cfg.WorkingDir,
	}

	// Add environment variables
	for key, value := range cfg.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Add the image and a command to keep it running
	args = append(args, cfg.Image, "sleep", "infinity")

	cmd := exec.Command("limactl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// List returns all running containers
func List() ([]string, error) {
	cmd := exec.Command("limactl", "shell", lima.Instance, "--", "podman", "ps", "--format", "{{.Names}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	names := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(names) == 1 && names[0] == "" {
		return []string{}, nil
	}

	return names, nil
}

// Stop stops a named container
func Stop(name string) error {
	cmd := exec.Command("limactl", "shell", lima.Instance, "--", "podman", "stop", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Remove removes a named container
func Remove(name string) error {
	cmd := exec.Command("limactl", "shell", lima.Instance, "--", "podman", "rm", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Exec runs a command in a named container
func Exec(name string, command []string) error {
	args := append([]string{"shell", lima.Instance, "--", "podman", "exec", name}, command...)
	cmd := exec.Command("limactl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// Enter starts an interactive shell in a named container
func Enter(name string, shell string) error {
	// Check if container exists and is running
	running, err := isContainerRunning(name)
	if err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	}

	if !running {
		return fmt.Errorf("container %s not found or not running", name)
	}

	// Use the specified shell or default to bash
	if shell == "" {
		shell = "bash"
	}

	// Start interactive shell with proper terminal settings
	args := []string{
		"shell", lima.Instance, "--", "podman", "exec",
		"-it", // interactive + allocate pseudo-TTY
		name,
		shell,
	}

	cmd := exec.Command("limactl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Set terminal to raw mode for proper interactive behavior
	return cmd.Run()
}

// isContainerRunning checks if a container is running
func isContainerRunning(name string) (bool, error) {
	cmd := exec.Command("limactl", "shell", lima.Instance, "--", "podman", "ps", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Names}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(output)) == name, nil
}
