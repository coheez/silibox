package cli

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/coheez/silibox/internal/container"
	"github.com/coheez/silibox/internal/lima"
	"github.com/coheez/silibox/internal/state"
	"github.com/spf13/cobra"
)

var (
	doctorFix bool
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose environment and dependencies",
	Long:  "Diagnose environment and dependencies. Use --fix to automatically repair common issues.",
	RunE: func(cmd *cobra.Command, args []string) error {
		issues := []string{}
		warnings := []string{}

		fmt.Println("🔍 Silibox Doctor - Environment Diagnostics")
		fmt.Println(strings.Repeat("=", 50))

		// Check system info
		fmt.Printf("System: %s %s\n", runtime.GOOS, runtime.GOARCH)

		// Check Lima installation
		if err := checkLimaInstallation(); err != nil {
			issues = append(issues, err.Error())
		} else {
			fmt.Println("✓ Lima is installed")
		}

		// Check VM status
		if err := checkVMStatus(); err != nil {
			issues = append(issues, err.Error())
		}

		// Check Podman inside VM (if VM is running)
		if err := checkPodmanInVM(); err != nil {
			warnings = append(warnings, err.Error())
		}

		// Check state consistency
		if err := checkStateConsistency(); err != nil {
			warnings = append(warnings, err.Error())
		}

		// Check for orphaned or desynced containers
		if desyncWarnings := checkContainerDesync(); len(desyncWarnings) > 0 {
			warnings = append(warnings, desyncWarnings...)
		}

		// Check Apple Silicon specific requirements
		if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
			fmt.Println("• Apple Silicon detected - Virtualization.framework (vz) required")
		}

		// Print results
		fmt.Println("\n" + strings.Repeat("=", 50))
		if len(issues) > 0 {
			fmt.Println("❌ Issues found:")
			for _, issue := range issues {
				fmt.Printf("  • %s\n", issue)
			}
		}

		if len(warnings) > 0 {
			fmt.Println("⚠️  Warnings:")
			for _, warning := range warnings {
				fmt.Printf("  • %s\n", warning)
			}
		}

		if len(issues) == 0 && len(warnings) == 0 {
			fmt.Println("✅ All checks passed! Silibox is ready to use.")
		} else if len(issues) == 0 {
			fmt.Println("✅ No critical issues found. Silibox should work.")
		}

		if len(issues) > 0 {
			return fmt.Errorf("doctor found %d issue(s) that need to be fixed", len(issues))
		}

		return nil
	},
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "Automatically fix common issues")
}

func checkLimaInstallation() error {
	if _, err := exec.LookPath("limactl"); err != nil {
		return fmt.Errorf("lima not found - install with: brew install lima")
	}
	return nil
}

func checkVMStatus() error {
	// Check if VM exists and is running
	inst, found, err := lima.GetInstance()
	if err != nil {
		return fmt.Errorf("failed to check VM status: %v", err)
	}

	if !found {
		fmt.Println("• VM not found - Run 'sili vm up' to create it")
		return nil
	}

	switch inst.Status {
	case "Running":
		fmt.Println("✓ VM is running")
		return nil
	case "Stopped":
		fmt.Println("• VM exists but is stopped - Run 'sili vm up' to start it")
		return nil
	case "Error", "Broken":
		return fmt.Errorf("VM is in %s state - try 'sili vm stop' then 'sili vm up' to recreate", inst.Status)
	default:
		fmt.Printf("• VM status: %s\n", inst.Status)
		return nil
	}
}

func checkPodmanInVM() error {
	// Only check if VM is running
	inst, found, err := lima.GetInstance()
	if err != nil || !found || inst.Status != "Running" {
		return nil // Skip check if VM not running
	}

	// Check if podman is installed inside VM
	cmd := exec.Command("limactl", "shell", "silibox", "--", "which", "podman")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("podman not found in VM - run 'sili vm up' to install it")
	}

	// Check if podman works
	cmd = exec.Command("limactl", "shell", "silibox", "--", "podman", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("podman in VM is not working - run 'sili vm up' to reinstall")
	}

	fmt.Println("✓ Podman is installed and working in VM")
	return nil
}

func checkStateConsistency() error {
	// Load state and check for consistency
	s, err := state.Load()
	if err != nil {
		return fmt.Errorf("state file corrupted - run 'sili state show' to check")
	}

	vm := s.GetVM()
	if vm == nil {
		return nil // No VM in state, that's ok
	}

	// Check if state matches actual VM status
	inst, found, err := lima.GetInstance()
	if err != nil {
		return fmt.Errorf("cannot verify state consistency - lima error: %v", err)
	}

	if !found {
		if vm.Status == "running" {
			if doctorFix {
				fmt.Println("🔧 Fixing stale state (VM not found, updating state to stopped)...")
				if err := state.WithLockedState(func(s *state.State) error {
					s.UpdateVMStatus("stopped")
					return nil
				}); err != nil {
					return fmt.Errorf("failed to fix state: %w", err)
				}
				fmt.Println("   ✅ State updated")
				return nil
			}
			return fmt.Errorf("state says VM is running but lima shows no VM - state may be stale (run with --fix to repair)")
		}
		return nil
	}

	// Normalize status for comparison
	stateStatus := strings.ToLower(vm.Status)
	actualStatus := strings.ToLower(inst.Status)

	if stateStatus != actualStatus {
		if doctorFix {
			fmt.Printf("🔧 Fixing state inconsistency (updating state from '%s' to '%s')...\n", vm.Status, inst.Status)
			if err := state.WithLockedState(func(s *state.State) error {
				s.UpdateVMStatus(strings.ToLower(inst.Status))
				return nil
			}); err != nil {
				return fmt.Errorf("failed to fix state: %w", err)
			}
			fmt.Println("   ✅ State updated")
			return nil
		}
		return fmt.Errorf("state inconsistency - state says '%s' but lima shows '%s' (run with --fix to repair)", vm.Status, inst.Status)
	}

	fmt.Println("✓ State is consistent with lima")
	return nil
}

func checkContainerDesync() []string {
	warnings := []string{}

	// Only check if VM is running
	inst, found, err := lima.GetInstance()
	if err != nil || !found || inst.Status != "Running" {
		return warnings // Skip check if VM not running
	}

	// Load state
	s, err := state.Load()
	if err != nil {
		return warnings // Can't check without state
	}

	// Get all environments from state
	envs := s.ListEnvs()
	if len(envs) == 0 {
		fmt.Println("✓ No environments to check")
		return warnings
	}

	// Get all running containers from Podman
	runningContainers, err := container.List()
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("failed to list containers: %v", err))
		return warnings
	}

	// Create map for quick lookup
	runningMap := make(map[string]bool)
	for _, name := range runningContainers {
		runningMap[name] = true
	}

	// Check each environment for desync
	desyncCount := 0
	fixedCount := 0
	for _, env := range envs {
		isRunning := runningMap[env.Name]

		// Case 1: State says running but container doesn't exist or is stopped
		if env.Status == "running" && !isRunning {
			if doctorFix {
				fmt.Printf("🔧 Fixing desync: '%s' marked as running but not found (updating to stopped)...\n", env.Name)
				if err := state.WithLockedState(func(s *state.State) error {
					env := s.GetEnv(env.Name)
					if env != nil {
						env.Status = "stopped"
						s.UpsertEnv(env)
					}
					return nil
				}); err != nil {
					warnings = append(warnings, fmt.Sprintf("Failed to fix '%s': %v", env.Name, err))
				} else {
					fmt.Println("   ✅ Fixed")
					fixedCount++
				}
			} else {
				warnings = append(warnings, fmt.Sprintf("Environment '%s' marked as running in state but not found in Podman (run with --fix to repair)", env.Name))
			}
			desyncCount++
		}

		// Case 2: Container is running but state says stopped (less critical)
		if env.Status == "stopped" && isRunning {
			if doctorFix {
				fmt.Printf("🔧 Fixing desync: '%s' is running but marked as stopped (updating to running)...\n", env.Name)
				if err := state.WithLockedState(func(s *state.State) error {
					env := s.GetEnv(env.Name)
					if env != nil {
						env.Status = "running"
						s.UpsertEnv(env)
					}
					return nil
				}); err != nil {
					warnings = append(warnings, fmt.Sprintf("Failed to fix '%s': %v", env.Name, err))
				} else {
					fmt.Println("   ✅ Fixed")
					fixedCount++
				}
			} else {
				warnings = append(warnings, fmt.Sprintf("Environment '%s' is running but marked as stopped in state (run with --fix to repair)", env.Name))
			}
			desyncCount++
		}
	}

	if desyncCount == 0 {
		fmt.Printf("✓ All %d environment(s) in sync with Podman\n", len(envs))
	} else if doctorFix && fixedCount > 0 {
		fmt.Printf("✅ Fixed %d/%d desync issue(s)\n", fixedCount, desyncCount)
	}

	return warnings
}
