package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coheez/silibox/internal/state"
)

func setupTestState(t *testing.T) (string, func()) {
	// Create temporary directory for state
	tmpDir := t.TempDir()
	
	// Override state paths
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	
	// Reinitialize state paths with new HOME
	state.ResetForTesting()
	
	// Create state directory
	stateDir := filepath.Join(tmpDir, state.StateDir)
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}

	cleanup := func() {
		os.Setenv("HOME", oldHome)
		state.ResetForTesting()
	}

	return tmpDir, cleanup
}

func TestGetIdleEnvironments(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name      string
		envs      []*state.EnvInfo
		threshold time.Duration
		wantCount int
		wantNames []string
	}{
		{
			name: "no environments",
			envs: []*state.EnvInfo{},
			threshold: 15 * time.Minute,
			wantCount: 0,
			wantNames: []string{},
		},
		{
			name: "all active",
			envs: []*state.EnvInfo{
				{
					Name:       "env1",
					Status:     "running",
					LastActive: now,
					Persistent: false,
				},
				{
					Name:       "env2",
					Status:     "running",
					LastActive: now.Add(-5 * time.Minute),
					Persistent: false,
				},
			},
			threshold: 15 * time.Minute,
			wantCount: 0,
			wantNames: []string{},
		},
		{
			name: "one idle",
			envs: []*state.EnvInfo{
				{
					Name:       "active",
					Status:     "running",
					LastActive: now,
					Persistent: false,
				},
				{
					Name:       "idle",
					Status:     "running",
					LastActive: now.Add(-20 * time.Minute),
					Persistent: false,
				},
			},
			threshold: 15 * time.Minute,
			wantCount: 1,
			wantNames: []string{"idle"},
		},
		{
			name: "skip persistent",
			envs: []*state.EnvInfo{
				{
					Name:       "postgres",
					Status:     "running",
					LastActive: now.Add(-30 * time.Minute),
					Persistent: true, // Should be skipped
				},
				{
					Name:       "dev",
					Status:     "running",
					LastActive: now.Add(-20 * time.Minute),
					Persistent: false,
				},
			},
			threshold: 15 * time.Minute,
			wantCount: 1,
			wantNames: []string{"dev"},
		},
		{
			name: "skip stopped",
			envs: []*state.EnvInfo{
				{
					Name:       "stopped",
					Status:     "stopped",
					LastActive: now.Add(-30 * time.Minute),
					Persistent: false,
				},
				{
					Name:       "idle",
					Status:     "running",
					LastActive: now.Add(-20 * time.Minute),
					Persistent: false,
				},
			},
			threshold: 15 * time.Minute,
			wantCount: 1,
			wantNames: []string{"idle"},
		},
		{
			name: "multiple idle",
			envs: []*state.EnvInfo{
				{
					Name:       "idle1",
					Status:     "running",
					LastActive: now.Add(-20 * time.Minute),
					Persistent: false,
				},
				{
					Name:       "idle2",
					Status:     "running",
					LastActive: now.Add(-25 * time.Minute),
					Persistent: false,
				},
				{
					Name:       "active",
					Status:     "running",
					LastActive: now,
					Persistent: false,
				},
			},
			threshold: 15 * time.Minute,
			wantCount: 2,
			wantNames: []string{"idle1", "idle2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh state for each test
			_, cleanup := setupTestState(t)
			defer cleanup()
			
			// Create state with test environments
			err := state.WithLockedState(func(s *state.State) error {
				for _, env := range tt.envs {
					s.UpsertEnv(env)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("failed to setup state: %v", err)
			}

			// Get idle environments
			idleEnvs, err := GetIdleEnvironments(tt.threshold)
			if err != nil {
				t.Fatalf("GetIdleEnvironments() error = %v", err)
			}

			if len(idleEnvs) != tt.wantCount {
				t.Errorf("GetIdleEnvironments() count = %d, want %d", len(idleEnvs), tt.wantCount)
			}

			// Check names match
			gotNames := make(map[string]bool)
			for _, env := range idleEnvs {
				gotNames[env.Name] = true
			}

			for _, wantName := range tt.wantNames {
				if !gotNames[wantName] {
					t.Errorf("GetIdleEnvironments() missing expected env %q", wantName)
				}
			}
		})
	}
}

func TestIsVMIdle(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		vm        *state.VMInfo
		envs      []*state.EnvInfo
		threshold time.Duration
		wantIdle  bool
	}{
		{
			name:      "no VM",
			vm:        nil,
			envs:      []*state.EnvInfo{},
			threshold: 30 * time.Minute,
			wantIdle:  true,
		},
		{
			name: "VM stopped",
			vm: &state.VMInfo{
				Name:       "silibox",
				Status:     "stopped",
				LastActive: now.Add(-1 * time.Hour),
			},
			envs:      []*state.EnvInfo{},
			threshold: 30 * time.Minute,
			wantIdle:  true,
		},
		{
			name: "VM running with active containers",
			vm: &state.VMInfo{
				Name:       "silibox",
				Status:     "running",
				LastActive: now.Add(-40 * time.Minute),
			},
			envs: []*state.EnvInfo{
				{
					Name:       "dev",
					Status:     "running",
					LastActive: now,
				},
			},
			threshold: 30 * time.Minute,
			wantIdle:  false,
		},
		{
			name: "VM running, all containers stopped, VM idle",
			vm: &state.VMInfo{
				Name:       "silibox",
				Status:     "running",
				LastActive: now.Add(-40 * time.Minute),
			},
			envs: []*state.EnvInfo{
				{
					Name:       "dev",
					Status:     "stopped",
					LastActive: now.Add(-1 * time.Hour),
				},
			},
			threshold: 30 * time.Minute,
			wantIdle:  true,
		},
		{
			name: "VM running, all containers stopped, VM recently active",
			vm: &state.VMInfo{
				Name:       "silibox",
				Status:     "running",
				LastActive: now.Add(-10 * time.Minute),
			},
			envs: []*state.EnvInfo{
				{
					Name:       "dev",
					Status:     "stopped",
					LastActive: now.Add(-1 * time.Hour),
				},
			},
			threshold: 30 * time.Minute,
			wantIdle:  false,
		},
		{
			name: "VM running, no containers",
			vm: &state.VMInfo{
				Name:       "silibox",
				Status:     "running",
				LastActive: now.Add(-40 * time.Minute),
			},
			envs:      []*state.EnvInfo{},
			threshold: 30 * time.Minute,
			wantIdle:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh state for each test
			_, cleanup := setupTestState(t)
			defer cleanup()
			
			// Create state
			err := state.WithLockedState(func(s *state.State) error {
				if tt.vm != nil {
					s.SetVM(tt.vm)
				}
				for _, env := range tt.envs {
					s.UpsertEnv(env)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("failed to setup state: %v", err)
			}

			// Check if VM is idle
			idle, err := IsVMIdle(tt.threshold)
			if err != nil {
				t.Fatalf("IsVMIdle() error = %v", err)
			}

			if idle != tt.wantIdle {
				t.Errorf("IsVMIdle() = %v, want %v", idle, tt.wantIdle)
			}
		})
	}
}

func TestGetIdleDuration(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name       string
		lastActive time.Time
		wantMin    time.Duration
		wantMax    time.Duration
	}{
		{
			name:       "just now",
			lastActive: now,
			wantMin:    0,
			wantMax:    1 * time.Second,
		},
		{
			name:       "15 minutes ago",
			lastActive: now.Add(-15 * time.Minute),
			wantMin:    14*time.Minute + 59*time.Second,
			wantMax:    15*time.Minute + 1*time.Second,
		},
		{
			name:       "1 hour ago",
			lastActive: now.Add(-1 * time.Hour),
			wantMin:    59*time.Minute + 59*time.Second,
			wantMax:    1*time.Hour + 1*time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &state.EnvInfo{
				LastActive: tt.lastActive,
			}

			duration := GetIdleDuration(env)

			if duration < tt.wantMin || duration > tt.wantMax {
				t.Errorf("GetIdleDuration() = %v, want between %v and %v", duration, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestGetVMIdleDuration(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name       string
		lastActive time.Time
		wantMin    time.Duration
		wantMax    time.Duration
	}{
		{
			name:       "recently active",
			lastActive: now.Add(-5 * time.Minute),
			wantMin:    4*time.Minute + 59*time.Second,
			wantMax:    5*time.Minute + 1*time.Second,
		},
		{
			name:       "long idle",
			lastActive: now.Add(-2 * time.Hour),
			wantMin:    119*time.Minute + 59*time.Second,
			wantMax:    2*time.Hour + 1*time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := &state.VMInfo{
				LastActive: tt.lastActive,
			}

			duration := GetVMIdleDuration(vm)

			if duration < tt.wantMin || duration > tt.wantMax {
				t.Errorf("GetVMIdleDuration() = %v, want between %v and %v", duration, tt.wantMin, tt.wantMax)
			}
		})
	}
}
