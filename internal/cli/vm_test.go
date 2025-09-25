package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestVMUpCommandFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected struct {
			cpus   int
			memory string
			disk   string
		}
	}{
		{
			name: "default values",
			args: []string{},
			expected: struct {
				cpus   int
				memory string
				disk   string
			}{
				cpus:   4,
				memory: "8GiB",
				disk:   "60GiB",
			},
		},
		{
			name: "custom values",
			args: []string{"--cpus", "2", "--memory", "4GiB", "--disk", "30GiB"},
			expected: struct {
				cpus   int
				memory string
				disk   string
			}{
				cpus:   2,
				memory: "4GiB",
				disk:   "30GiB",
			},
		},
		{
			name: "partial custom values",
			args: []string{"--cpus", "8", "--memory", "16GiB"},
			expected: struct {
				cpus   int
				memory string
				disk   string
			}{
				cpus:   8,
				memory: "16GiB",
				disk:   "60GiB", // default
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags for each test
			cpus = 0
			memory = ""
			disk = ""

			// Create a new command for each test
			cmd := &cobra.Command{}
			cmd.Flags().IntVar(&cpus, "cpus", 4, "vCPUs")
			cmd.Flags().StringVar(&memory, "memory", "8GiB", "RAM (e.g., 8GiB)")
			cmd.Flags().StringVar(&disk, "disk", "60GiB", "Disk size")

			// Parse the arguments
			cmd.SetArgs(tt.args)
			cmd.ParseFlags(tt.args)

			// Check the parsed values
			if cpus != tt.expected.cpus {
				t.Errorf("expected cpus %d, got %d", tt.expected.cpus, cpus)
			}
			if memory != tt.expected.memory {
				t.Errorf("expected memory %s, got %s", tt.expected.memory, memory)
			}
			if disk != tt.expected.disk {
				t.Errorf("expected disk %s, got %s", tt.expected.disk, disk)
			}
		})
	}
}

func TestVMCommandStructure(t *testing.T) {
	// Test that all subcommands are properly registered
	expectedSubcommands := []string{"up", "status", "stop"}
	
	for _, subcmd := range expectedSubcommands {
		found := false
		for _, cmd := range vmCmd.Commands() {
			if cmd.Name() == subcmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %s not found", subcmd)
		}
	}
}

func TestVMUpCommandHasFlags(t *testing.T) {
	// Test that vmUpCmd has the expected flags
	expectedFlags := []string{"cpus", "memory", "disk"}
	
	for _, flag := range expectedFlags {
		if !vmUpCmd.Flags().HasFlags() {
			t.Error("vmUpCmd should have flags")
		}
		
		// Check if the flag exists
foundFlag := vmUpCmd.Flags().Lookup(flag)
if foundFlag == nil {
			t.Errorf("expected flag %s not found", flag)
		}
	}
}
