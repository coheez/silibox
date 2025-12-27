package cli

import (
	"context"
	"time"

	"github.com/coheez/silibox/internal/agent"
	"github.com/spf13/cobra"
)

var (
	agentContainerTimeout time.Duration
	agentVMTimeout        time.Duration
	agentPollInterval     time.Duration
	agentNoStopVM         bool
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage background agent processes",
	Long:  "Manage background agent processes like autosleep",
}

var agentAutosleepCmd = &cobra.Command{
	Use:   "autosleep",
	Short: "Run autosleep agent to automatically stop idle containers and VM",
	Long: `Run the autosleep agent in the foreground.

The agent polls periodically and stops containers that have been idle longer than the
configured timeout. Persistent containers (marked with --persistent) are never stopped.

If all containers are stopped and the VM has been idle, it can also be stopped to save
resources.

Examples:
  # Run with default settings (15m container timeout, 30m VM timeout)
  sili agent autosleep

  # Run with custom timeouts
  sili agent autosleep --container-timeout 10m --vm-timeout 20m

  # Run with faster polling
  sili agent autosleep --poll-interval 10s

  # Don't stop the VM, only containers
  sili agent autosleep --no-stop-vm`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Build config from flags
		cfg := agent.AutosleepConfig{
			ContainerIdleTimeout: agentContainerTimeout,
			VMIdleTimeout:        agentVMTimeout,
			PollInterval:         agentPollInterval,
			StopVM:               !agentNoStopVM,
		}

		// Run the agent (blocks until interrupted)
		ctx := context.Background()
		return agent.RunAutosleep(ctx, cfg)
	},
}

func init() {
	// Add agent command to root
	rootCmd.AddCommand(agentCmd)
	agentCmd.AddCommand(agentAutosleepCmd)

	// Flags for autosleep
	agentAutosleepCmd.Flags().DurationVar(&agentContainerTimeout, "container-timeout", 15*time.Minute,
		"How long a container can be idle before being stopped")
	agentAutosleepCmd.Flags().DurationVar(&agentVMTimeout, "vm-timeout", 30*time.Minute,
		"How long the VM can be idle before being stopped")
	agentAutosleepCmd.Flags().DurationVar(&agentPollInterval, "poll-interval", 30*time.Second,
		"How often to check for idle resources")
	agentAutosleepCmd.Flags().BoolVar(&agentNoStopVM, "no-stop-vm", false,
		"Don't stop the VM, only stop idle containers")
}
