package vm

import (
	"fmt"

	"github.com/coheez/silibox/internal/lima"
	"github.com/coheez/silibox/internal/state"
)

// EnsureVMRunning checks if the VM is running and starts it if stopped
// This enables auto-wake functionality for all commands
func EnsureVMRunning() error {
	// First check state (fast path)
	st, err := state.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	vm := st.GetVM()
	if vm == nil {
		// No VM in state - need to create it
		return fmt.Errorf("VM not found. Run 'sili vm up' to create it")
	}

	// If state says running, check actual status to be sure
	if vm.Status == "running" {
		// Verify with Lima that it's actually running
		inst, found, err := lima.GetInstance()
		if err != nil {
			return fmt.Errorf("failed to check VM status: %w", err)
		}
		if found && inst.Status == "Running" {
			// VM is running
			return nil
		}
		// State is stale - fall through to start VM
	}

	// VM is stopped or state is stale - start it
	fmt.Println("⏳ VM is stopped. Starting VM...")
	
	// Use lima.Up() with default config
	// This will read existing config and start the VM
	cfg := lima.Config{
		CPUs:   4,  // Default values - lima.Up will use existing VM config
		Memory: "4GiB",
		Disk:   "100GiB",
	}
	
	if err := lima.Up(cfg); err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}

	fmt.Println("✅ VM started successfully")
	return nil
}

// EnsureContainerRunning checks if a container is running and starts it if stopped
// Returns true if the container was started, false if it was already running
func EnsureContainerRunning(name string) (bool, error) {
	st, err := state.Load()
	if err != nil {
		return false, fmt.Errorf("failed to load state: %w", err)
	}

	env := st.GetEnv(name)
	if env == nil {
		return false, fmt.Errorf("environment %s not found", name)
	}

	// If status is running, assume it's running (podman check happens in caller)
	// This is a quick check - the actual container operations will verify
	if env.Status == "running" {
		return false, nil
	}

	// Container is stopped - start it
	fmt.Printf("⏳ Container '%s' is stopped. Starting...\n", name)
	
	// Start the container via limactl/podman
	if err := startContainer(name); err != nil {
		return false, fmt.Errorf("failed to start container: %w", err)
	}

	// Update state
	if err := state.WithLockedState(func(s *state.State) error {
		s.UpdateEnvStatus(name, "running")
		s.TouchEnvActivity(name)
		s.TouchVMActivity()
		return nil
	}); err != nil {
		// Don't fail if state update fails
		fmt.Printf("Warning: failed to update state: %v\n", err)
	}

	fmt.Printf("✅ Container '%s' started\n", name)
	return true, nil
}

// startContainer starts a stopped container
func startContainer(name string) error {
	// For now, we'll skip auto-starting containers in MVP
	// Containers that are stopped need to be explicitly started with 'podman start'
	// or recreated. This is because:
	// 1. We don't track enough info to auto-start (need to know if it was intentionally stopped)
	// 2. Auto-starting containers could be surprising behavior
	// 3. Most use cases have containers running continuously (sleep infinity)
	
	// Return a helpful error for MVP
	return fmt.Errorf("container is stopped. Start it with 'sili rm --name %s --force && sili create --name %s' or manually with 'podman start %s'", name, name, name)
}
