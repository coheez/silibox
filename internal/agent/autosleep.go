package agent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coheez/silibox/internal/container"
)

// AutosleepConfig configures the autosleep agent behavior
type AutosleepConfig struct {
	ContainerIdleTimeout time.Duration // How long before stopping idle containers
	VMIdleTimeout        time.Duration // How long before stopping idle VM
	PollInterval         time.Duration // How often to check for idle resources
	StopVM               bool          // Whether to stop VM when fully idle
}

// DefaultAutosleepConfig returns sensible defaults for autosleep
func DefaultAutosleepConfig() AutosleepConfig {
	return AutosleepConfig{
		ContainerIdleTimeout: 15 * time.Minute,
		VMIdleTimeout:        30 * time.Minute,
		PollInterval:         30 * time.Second,
		StopVM:               true,
	}
}

// RunAutosleep runs the autosleep agent with the given configuration
// It polls periodically and stops idle containers (and optionally the VM)
// The agent runs until the context is cancelled or a signal is received
func RunAutosleep(ctx context.Context, cfg AutosleepConfig) error {
	fmt.Fprintf(os.Stderr, "🌙 Autosleep agent starting...\n")
	fmt.Fprintf(os.Stderr, "   Container idle timeout: %s\n", cfg.ContainerIdleTimeout)
	fmt.Fprintf(os.Stderr, "   VM idle timeout: %s\n", cfg.VMIdleTimeout)
	fmt.Fprintf(os.Stderr, "   Poll interval: %s\n", cfg.PollInterval)
	fmt.Fprintf(os.Stderr, "   Auto-stop VM: %v\n", cfg.StopVM)
	fmt.Fprintf(os.Stderr, "\n")

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	// Run initial check immediately
	if err := checkAndStopIdle(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: initial check failed: %v\n", err)
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "\n🛑 Autosleep agent stopping (context cancelled)...\n")
			return ctx.Err()

		case <-sigChan:
			fmt.Fprintf(os.Stderr, "\n🛑 Autosleep agent stopping (received signal)...\n")
			return nil

		case <-ticker.C:
			if err := checkAndStopIdle(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: check failed: %v\n", err)
			}
		}
	}
}

// checkAndStopIdle checks for idle containers and stops them
func checkAndStopIdle(cfg AutosleepConfig) error {
	// Get idle environments
	idleEnvs, err := GetIdleEnvironments(cfg.ContainerIdleTimeout)
	if err != nil {
		return fmt.Errorf("failed to get idle environments: %w", err)
	}

	if len(idleEnvs) == 0 {
		// No idle environments
		return nil
	}

	// Stop each idle environment
	for _, env := range idleEnvs {
		idleDuration := GetIdleDuration(env)
		fmt.Fprintf(os.Stderr, "💤 Stopping idle container '%s' (idle for %s)...\n", 
			env.Name, formatDuration(idleDuration))

		if err := container.Stop(env.Name); err != nil {
			fmt.Fprintf(os.Stderr, "   ⚠️  Failed to stop '%s': %v\n", env.Name, err)
			continue
		}

		fmt.Fprintf(os.Stderr, "   ✅ Stopped '%s'\n", env.Name)
	}

	return nil
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f seconds", d.Seconds())
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if hours == 1 {
		if minutes == 0 {
			return "1 hour"
		}
		return fmt.Sprintf("1 hour %d minutes", minutes)
	}
	if minutes == 0 {
		return fmt.Sprintf("%d hours", hours)
	}
	return fmt.Sprintf("%d hours %d minutes", hours, minutes)
}
