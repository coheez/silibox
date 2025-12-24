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
	"github.com/coheez/silibox/internal/stack"
	"github.com/coheez/silibox/internal/state"
)

type CreateConfig struct {
	Name                    string
	Image                   string
	ProjectDir              string
	WorkingDir              string
	User                    string
	Environment             map[string]string
	DetectAndPrepareVolumes bool // Auto-detect project stack and create volumes for hot dirs
	NoMigrate               bool // Skip migration prompts for existing directories
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

	// Get absolute project path
	projectPath, err := filepath.Abs(cfg.ProjectDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute project path: %w", err)
	}

	// Detect project stack and prepare volumes if requested
	volumes := make(map[string]string)
	migratedDirs := make(map[string]string) // Track migrations for state
	
	if cfg.DetectAndPrepareVolumes {
		projectInfo, err := stack.DetectStack(projectPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to detect project stack: %v\n", err)
		} else if projectInfo.Type != stack.Unknown {
			fmt.Printf("Detected %s project\n", projectInfo.Type)
			
			// Create volumes for hot directories
			// Strategy: Only create volumes for directories that:
			// 1. Already exist and user wants to migrate (for performance)
			// 2. Don't create volumes pre-emptively - let them be created on host first
			//    This avoids the mount conflict issue with directories that don't exist yet
			for _, hotDir := range projectInfo.HotDirs {
				// Skip wildcard patterns (e.g., *.egg-info)
				if strings.Contains(hotDir, "*") {
					continue
				}
				
				// Generate volume name
				volumeName := sanitizeVolumeName(fmt.Sprintf("%s-%s", cfg.Name, hotDir))
				
				// Check if directory already exists on host
				hostPath := filepath.Join(projectPath, hotDir)
				if stat, err := os.Stat(hostPath); err == nil && stat.IsDir() {
					// Directory exists - check if empty
					entries, err := os.ReadDir(hostPath)
					if err != nil || len(entries) == 0 {
						// Empty or unreadable - just remove it and create volume
						os.Remove(hostPath)
					} else if !cfg.NoMigrate {
						// Directory has contents - offer migration
						size, _ := GetDirSize(hostPath)
						fmt.Printf("\nFound existing %s (%s)\n", hotDir, FormatBytes(size))
						fmt.Printf("Migrate to volume for better performance? [Y/n]: ")
						
						var response string
						fmt.Scanln(&response)
						
						response = strings.ToLower(strings.TrimSpace(response))
						if response == "" || response == "y" || response == "yes" {
							// Create volume first
							if err := createVolume(volumeName); err != nil {
								fmt.Fprintf(os.Stderr, "Warning: Failed to create volume: %v\n", err)
								continue
							}
							
							// Migrate directory to volume
							if err := MigrateDirToVolume(cfg.Name, projectPath, hotDir, volumeName); err != nil {
								fmt.Fprintf(os.Stderr, "Warning: Migration failed: %v\n", err)
								continue
							}
							
							// Track migration
							backupPath := fmt.Sprintf("%s.silibox-backup-%d", hostPath, time.Now().Unix())
							migratedDirs[hotDir] = filepath.Base(backupPath)
							volumes[hotDir] = volumeName
							continue
						} else {
							fmt.Printf("Skipping migration for %s\n", hotDir)
							continue // Don't create volume, use host directory
						}
					} else {
						// NoMigrate flag set - skip
						continue
					}
				}
				// Directory doesn't exist on host - don't create volume pre-emptively
				// This avoids mount conflicts. The directory will be created on the host
				// when needed (e.g., npm install creates node_modules)
			}
		}
	}

	// Pull the image
	if err := pullImage(cfg.Image); err != nil {
		return fmt.Errorf("failed to pull image %s: %w", cfg.Image, err)
	}

	// Create the container with volumes
	if err := createContainer(cfg, uid, gid, volumes); err != nil {
		return err
	}

	// Create environment info
	envInfo := &state.EnvInfo{
		Name:        cfg.Name,
		Image:       cfg.Image,
		Runtime:     "podman",
		ProjectPath: projectPath,
		ContainerID: cfg.Name, // Using name as container ID for now
		Volumes:     volumes,
			Mounts: map[string]state.Mount{
				"work": {
					Host:  projectPath,
					Guest: cfg.WorkingDir,
					RW:    true,
				},
			},
			Ports:         make(map[string]int),
			User: state.UserInfo{
				UID:  uid,
				GID:  gid,
				Name: cfg.User,
			},
			Status:        "running",
			Persistent:    false,
			LastActive:    time.Now(),
			ExportedShims: make([]string, 0),
			MigratedDirs:  migratedDirs,
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

func createContainer(cfg CreateConfig, uid, gid int, volumes map[string]string) error {
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
	}

	// CRITICAL: Mount volumes for hot directories FIRST using --mount syntax
	// The --mount syntax creates the mount point if it doesn't exist
	// When we mount the project directory at /workspace later, these volume mounts
	// will take precedence for their specific paths (e.g., /workspace/node_modules)
	for hotDir, volumeName := range volumes {
		mountPath := filepath.Join("/workspace", hotDir)
		// Use --mount instead of -v for better control
		args = append(args, "--mount", fmt.Sprintf("type=volume,source=%s,destination=%s", volumeName, mountPath))
	}

	// Now mount the project directory at /workspace
	// The volume mounts above will "punch through" and remain visible
	args = append(args, "-v", fmt.Sprintf("%s:/workspace", projectDir)) // project dir (writable)
	args = append(args, "-v", fmt.Sprintf("%s:/home/host:ro", homeDir)) // home dir (read-only)
	args = append(args, "-w", cfg.WorkingDir)

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

// createVolume creates a Podman volume inside the Lima VM
func createVolume(volumeName string) error {
	cmd := exec.Command("limactl", "shell", lima.Instance, "--", "podman", "volume", "create", volumeName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create volume: %w (output: %s)", err, string(output))
	}
	return nil
}

// sanitizeVolumeName converts a directory path into a valid volume name
// Replaces problematic characters with hyphens
func sanitizeVolumeName(name string) string {
	// Replace slashes, dots at start, and other special chars
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.ReplaceAll(name, "_", "-")
	// Remove leading/trailing hyphens
	name = strings.Trim(name, "-")
	// Convert to lowercase for consistency
	name = strings.ToLower(name)
	return name
}
