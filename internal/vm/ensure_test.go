package vm

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coheez/silibox/internal/state"
)

func setupTestState(t *testing.T) func() {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	state.ResetForTesting()

	stateDir := filepath.Join(tmpDir, state.StateDir)
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}

	return func() {
		os.Setenv("HOME", oldHome)
		state.ResetForTesting()
	}
}

func TestEnsureVMRunning_NoVM(t *testing.T) {
	cleanup := setupTestState(t)
	defer cleanup()

	// Create empty state (no VM)
	err := state.WithLockedState(func(s *state.State) error {
		return nil
	})
	if err != nil {
		t.Fatalf("failed to setup state: %v", err)
	}

	// Should fail with "VM not found" error
	err = EnsureVMRunning()
	if err == nil {
		t.Errorf("EnsureVMRunning() should fail when VM doesn't exist")
	}
	if err != nil && err.Error() != "VM not found. Run 'sili vm up' to create it" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestEnsureContainerRunning_NotFound(t *testing.T) {
	cleanup := setupTestState(t)
	defer cleanup()

	// Create state without environments
	err := state.WithLockedState(func(s *state.State) error {
		return nil
	})
	if err != nil {
		t.Fatalf("failed to setup state: %v", err)
	}

	// Should fail with "environment not found" error
	_, err = EnsureContainerRunning("nonexistent")
	if err == nil {
		t.Errorf("EnsureContainerRunning() should fail when environment doesn't exist")
	}
	if err != nil && err.Error() != "environment nonexistent not found" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestEnsureContainerRunning_AlreadyRunning(t *testing.T) {
	cleanup := setupTestState(t)
	defer cleanup()

	// Create state with running environment
	err := state.WithLockedState(func(s *state.State) error {
		env := &state.EnvInfo{
			Name:       "test",
			Status:     "running",
			LastActive: time.Now(),
		}
		s.UpsertEnv(env)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to setup state: %v", err)
	}

	// Should return false (not started) and no error
	started, err := EnsureContainerRunning("test")
	if err != nil {
		t.Errorf("EnsureContainerRunning() unexpected error: %v", err)
	}
	if started {
		t.Errorf("EnsureContainerRunning() should return false for already running container")
	}
}

func TestEnsureContainerRunning_Stopped(t *testing.T) {
	cleanup := setupTestState(t)
	defer cleanup()

	// Create state with stopped environment
	err := state.WithLockedState(func(s *state.State) error {
		env := &state.EnvInfo{
			Name:       "test",
			Status:     "stopped",
			LastActive: time.Now().Add(-1 * time.Hour),
		}
		s.UpsertEnv(env)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to setup state: %v", err)
	}

	// Should return error explaining how to restart (MVP behavior)
	_, err = EnsureContainerRunning("test")
	if err == nil {
		t.Errorf("EnsureContainerRunning() should fail for stopped container in MVP")
	}
	// Check error message contains helpful instructions
	if err != nil && err.Error() != "failed to start container: container is stopped. Start it with 'sili rm --name test --force && sili create --name test' or manually with 'podman start test'" {
		t.Errorf("unexpected error message: %v", err)
	}
}
