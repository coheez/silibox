package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/coheez/silibox/internal/agent"
	"github.com/coheez/silibox/internal/config"
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

Configuration:
  Settings can be configured in ~/.sili/config.yaml:
    autosleep:
      container_timeout: 15m
      vm_timeout: 30m
      poll_interval: 30s
      no_stop_vm: false

  Command-line flags override config file settings.

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
		// Load config file (defaults if not found)
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Override with flags if they were explicitly set
		if cmd.Flags().Changed("container-timeout") {
			cfg.Autosleep.ContainerTimeout = agentContainerTimeout
		}
		if cmd.Flags().Changed("vm-timeout") {
			cfg.Autosleep.VMTimeout = agentVMTimeout
		}
		if cmd.Flags().Changed("poll-interval") {
			cfg.Autosleep.PollInterval = agentPollInterval
		}
		if cmd.Flags().Changed("no-stop-vm") {
			cfg.Autosleep.NoStopVM = agentNoStopVM
		}

		// Build agent config
		agentCfg := agent.AutosleepConfig{
			ContainerIdleTimeout: cfg.Autosleep.ContainerTimeout,
			VMIdleTimeout:        cfg.Autosleep.VMTimeout,
			PollInterval:         cfg.Autosleep.PollInterval,
			StopVM:               !cfg.Autosleep.NoStopVM,
		}

		// Run the agent (blocks until interrupted)
		ctx := context.Background()
		return agent.RunAutosleep(ctx, agentCfg)
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
