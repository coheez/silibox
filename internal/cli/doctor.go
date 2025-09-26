package cli

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/coheez/silibox/internal/lima"
	"github.com/coheez/silibox/internal/state"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose environment and dependencies",
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
			return fmt.Errorf("state says VM is running but lima shows no VM - state may be stale")
		}
		return nil
	}

	// Normalize status for comparison
	stateStatus := strings.ToLower(vm.Status)
	actualStatus := strings.ToLower(inst.Status)

	if stateStatus != actualStatus {
		return fmt.Errorf("state inconsistency - state says '%s' but lima shows '%s' - run 'sili vm status --live' to check", vm.Status, inst.Status)
	}

	fmt.Println("✓ State is consistent with lima")
	return nil
}
