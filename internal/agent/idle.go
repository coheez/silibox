package agent

import (
	"time"

	"github.com/coheez/silibox/internal/state"
)

// GetIdleEnvironments returns environments that have been idle longer than the threshold
// Persistent environments are never considered idle
func GetIdleEnvironments(threshold time.Duration) ([]*state.EnvInfo, error) {
	st, err := state.Load()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	idleEnvs := make([]*state.EnvInfo, 0)

	for _, env := range st.ListEnvs() {
		// Skip persistent environments (databases, long-running services)
		if env.Persistent {
			continue
		}

		// Skip if already stopped
		if env.Status == "stopped" {
			continue
		}

		// Check if idle
		idleDuration := now.Sub(env.LastActive)
		if idleDuration > threshold {
			idleEnvs = append(idleEnvs, env)
		}
	}

	return idleEnvs, nil
}

// IsVMIdle checks if the VM has been idle longer than the threshold
// VM is considered idle if:
// - All environments are stopped OR
// - VM LastActive exceeds threshold
func IsVMIdle(threshold time.Duration) (bool, error) {
	st, err := state.Load()
	if err != nil {
		return false, err
	}

	vm := st.GetVM()
	if vm == nil {
		return true, nil // No VM = idle
	}

	// If VM is already stopped, it's idle
	if vm.Status == "stopped" {
		return true, nil
	}

	// Check if all environments are stopped
	allStopped := true
	for _, env := range st.ListEnvs() {
		if env.Status != "stopped" {
			allStopped = false
			break
		}
	}

	if allStopped {
		// All containers stopped - check VM idle time
		now := time.Now()
		idleDuration := now.Sub(vm.LastActive)
		return idleDuration > threshold, nil
	}

	// Some containers still running - VM not idle
	return false, nil
}

// GetIdleDuration returns how long an environment has been idle
func GetIdleDuration(env *state.EnvInfo) time.Duration {
	return time.Since(env.LastActive)
}

// GetVMIdleDuration returns how long the VM has been idle
func GetVMIdleDuration(vm *state.VMInfo) time.Duration {
	return time.Since(vm.LastActive)
}
