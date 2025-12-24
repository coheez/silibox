package container

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/coheez/silibox/internal/lima"
	"github.com/coheez/silibox/internal/state"
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
	return state.WithLockedState(func(s *state.State) error {
		// Ensure VM is running
		vm := s.GetVM()
		if vm == nil || vm.Status != "running" {
			return fmt.Errorf("VM is not running. Run 'sili vm up' first")
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
		if err := createContainer(cfg, uid, gid); err != nil {
			return err
		}

		// Get absolute project path
		projectPath, err := filepath.Abs(cfg.ProjectDir)
		if err != nil {
			return fmt.Errorf("failed to get absolute project path: %w", err)
		}

		// Create environment info
		envInfo := &state.EnvInfo{
			Name:        cfg.Name,
			Image:       cfg.Image,
			Runtime:     "podman",
			ProjectPath: projectPath,
			ContainerID: cfg.Name, // Using name as container ID for now
			Volumes:     make(map[string]string),
			Mounts: map[string]state.Mount{
				"work": {
					Host:  projectPath,
					Guest: cfg.WorkingDir,
					RW:    true,
				},
			},
			Ports: make(map[string]int),
			User: state.UserInfo{
				UID:  uid,
				GID:  gid,
				Name: cfg.User,
			},
			Status:        "running",
			Persistent:    false,
			LastActive:    time.Now(),
			ExportedShims: make([]string, 0),
		}

		// Update state
		s.UpsertEnv(envInfo)
		s.TouchVMActivity()

		return nil
	})
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

// Stop stops a named container and updates state
func Stop(name string) error {
	return state.WithLockedState(func(s *state.State) error {
		// Check if environment exists in state
		env := s.GetEnv(name)
		if env == nil {
			return fmt.Errorf("environment %s not found in state", name)
		}

		// Stop the container
		cmd := exec.Command("limactl", "shell", lima.Instance, "--", "podman", "stop", name)
		var stderr bytes.Buffer
		cmd.Stdout = os.Stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			// Check if container doesn't exist (desync)
			if strings.Contains(stderr.String(), "no such container") {
				// Container doesn't exist but is in state - update state as stopped
				fmt.Fprintf(os.Stderr, "Warning: container %s not found in Podman, updating state\n", name)
				s.UpdateEnvStatus(name, "stopped")
				s.TouchVMActivity()
				return nil
			}
			return fmt.Errorf("failed to stop container: %w", err)
		}

		// Update state
		s.UpdateEnvStatus(name, "stopped")
		s.TouchVMActivity()

		return nil
	})
}

// Remove removes a named container and cleans up state
func Remove(name string, force bool) error {
	return state.WithLockedState(func(s *state.State) error {
		// Check if environment exists in state
		env := s.GetEnv(name)
		if env == nil {
			return fmt.Errorf("environment %s not found in state", name)
		}

		// Build podman rm command
		args := []string{"shell", lima.Instance, "--", "podman", "rm"}
		if force {
			args = append(args, "-f") // Force remove even if running
		}
		args = append(args, name)

		// Remove the container
		cmd := exec.Command("limactl", args...)
		var stderr bytes.Buffer
		cmd.Stdout = os.Stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			stderrStr := stderr.String()
			// Check if the error is because container doesn't exist
			if strings.Contains(stderrStr, "no such container") {
				// Container doesn't exist in Podman but is in state - clean up state
				fmt.Fprintf(os.Stderr, "Warning: container %s not found in Podman, cleaning up state\n", name)
			} else if strings.Contains(stderrStr, "cannot be removed without force") {
				// Container is running and force flag not used
				return fmt.Errorf("container %s is running. Stop it first with 'sili stop --name %s' or use --force (-f) to remove it", name, name)
			} else {
				return fmt.Errorf("failed to remove container: %w", err)
			}
		}

		// Remove from state (this also releases ports)
		s.RemoveEnv(name)
		s.TouchVMActivity()

		return nil
	})
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

// RunResult contains the result of a non-interactive command execution
type RunResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Run executes a command in a named container non-interactively and returns the result
func Run(name string, command []string) (RunResult, error) {
	// Check if environment exists in state
	st, err := state.Load()
	if err != nil {
		return RunResult{}, fmt.Errorf("failed to load state: %w", err)
	}

	env := st.GetEnv(name)
	if env == nil {
		return RunResult{}, fmt.Errorf("environment %s not found. Create it with 'sili create --name %s'", name, name)
	}

	// Check if container exists and is running
	running, err := isContainerRunning(name)
	if err != nil {
		return RunResult{}, fmt.Errorf("failed to check container status: %w", err)
	}

	if !running {
		if env.Status == "stopped" {
			return RunResult{}, fmt.Errorf("container %s is stopped. Start it first (it will auto-start on 'sili enter')", name)
		}
		return RunResult{}, fmt.Errorf("container %s not found or not running. It may have been manually deleted - recreate it with 'sili create'", name)
	}

	args := append([]string{"shell", lima.Instance, "--", "podman", "exec", name}, command...)
	cmd := exec.Command("limactl", args...)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command and capture exit code
	err = cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			return RunResult{}, fmt.Errorf("failed to run command: %w", err)
		}
	}

	return RunResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

// Enter starts an interactive shell in a named container
func Enter(name string, shell string) error {
	// Check if environment exists in state
	st, err := state.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	env := st.GetEnv(name)
	if env == nil {
		return fmt.Errorf("environment %s not found. Create it with 'sili create --name %s'", name, name)
	}

	// Check if container exists and is running
	running, err := isContainerRunning(name)
	if err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	}

	if !running {
		if env.Status == "stopped" {
			return fmt.Errorf("container %s is stopped. Start it with 'podman start' or recreate it with 'sili rm --name %s && sili create --name %s --image %s'", name, name, name, env.Image)
		}
		return fmt.Errorf("container %s not found. It may have been manually deleted - recreate it with 'sili create --name %s --image %s'", name, name, env.Image)
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
