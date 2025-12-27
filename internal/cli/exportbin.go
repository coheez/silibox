package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/coheez/silibox/internal/shim"
	"github.com/coheez/silibox/internal/state"
	"github.com/coheez/silibox/internal/vm"
	"github.com/spf13/cobra"
)

var (
	exportBinName   string
	exportBinBins   []string
	exportBinList   bool
	exportBinRemove []string
	exportBinForce  bool
)

var exportBinCmd = &cobra.Command{
	Use:   "export-bin",
	Short: "Export container commands as host shims",
	Long: `Export commands from a container environment to run natively on the host.

Creates executable shims in ~/.sili/bin that proxy to container commands.
This allows you to run 'node' instead of 'sili run --name dev -- node'.

Examples:
  # Export multiple commands from an environment
  sili export-bin --name dev --bin node --bin npm --bin npx

  # List all exported shims
  sili export-bin --list

  # Remove shims
  sili export-bin --remove node --remove npm`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle --list flag
		if exportBinList {
			return listShims()
		}

		// Handle --remove flag
		if len(exportBinRemove) > 0 {
			return removeShims(exportBinRemove)
		}

		// Create shims (default behavior)
		if exportBinName == "" {
			return fmt.Errorf("--name is required when creating shims")
		}
		if len(exportBinBins) == 0 {
			return fmt.Errorf("--bin is required (specify at least one command to export)")
		}

		return createShims(exportBinName, exportBinBins, exportBinForce)
	},
}

func createShims(envName string, commands []string, force bool) error {
	// Ensure VM is running (needed to verify commands exist)
	if err := vm.EnsureVMRunning(); err != nil {
		return err
	}

	// Verify environment exists
	st, err := state.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	env := st.GetEnv(envName)
	if env == nil {
		return fmt.Errorf("environment %s not found", envName)
	}

	// Track created shims
	createdShims := make([]string, 0)
	var firstError error

	// Create each shim
	for _, cmdName := range commands {
		if err := shim.GenerateShim(envName, cmdName, force); err != nil {
			if firstError == nil {
				firstError = err
			}
			fmt.Printf("Warning: failed to create shim for %s: %v\n", cmdName, err)
			continue
		}

		createdShims = append(createdShims, cmdName)
		fmt.Printf("Created shim: %s -> %s\n", cmdName, envName)
	}

	// If we created any shims, update state
	if len(createdShims) > 0 {
		if err := state.WithLockedState(func(s *state.State) error {
			env := s.GetEnv(envName)
			if env == nil {
				return fmt.Errorf("environment %s not found", envName)
			}

			// Update exported shims list in env
			existingShims := make(map[string]bool)
			for _, shimName := range env.ExportedShims {
				existingShims[shimName] = true
			}
			for _, cmdName := range createdShims {
				if !existingShims[cmdName] {
					env.ExportedShims = append(env.ExportedShims, cmdName)
				}
			}

			// Register shims globally
			for _, cmdName := range createdShims {
				s.RegisterShim(cmdName, envName, cmdName)
			}

			return nil
		}); err != nil {
			return fmt.Errorf("failed to update state: %w", err)
		}

		// Check if shim directory is in PATH
		inPath, err := shim.IsInPATH()
		if err != nil {
			fmt.Printf("Warning: failed to check PATH: %v\n", err)
		} else if !inPath {
			fmt.Println("\n⚠️  Shim directory is not in your PATH!")
			if instructions, err := shim.GetPATHInstructions(); err == nil {
				fmt.Println(instructions)
			}
		}
	}

	if firstError != nil {
		return firstError
	}

	return nil
}

func listShims() error {
	st, err := state.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	shims := st.ListShims()
	if len(shims) == 0 {
		fmt.Println("No shims registered.")
		fmt.Println("Create shims with: sili export-bin --name <env> --bin <command>")
		return nil
	}

	// Convert map to sorted slice
	type shimEntry struct {
		name   string
		env    string
		target string
	}
	entries := make([]shimEntry, 0, len(shims))
	for name, info := range shims {
		entries = append(entries, shimEntry{
			name:   name,
			env:    info.Env,
			target: info.Target,
		})
	}

	// Sort by name
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	// Print header
	fmt.Printf("%-20s %-15s %s\n", "SHIM", "ENV", "TARGET")
	fmt.Println(strings.Repeat("-", 60))

	// Print each shim
	for _, entry := range entries {
		fmt.Printf("%-20s %-15s %s\n", entry.name, entry.env, entry.target)
	}

	return nil
}

func removeShims(commands []string) error {
	var firstError error

	for _, cmdName := range commands {
		if err := shim.RemoveShim(cmdName); err != nil {
			if firstError == nil {
				firstError = err
			}
			fmt.Printf("Warning: %v\n", err)
			continue
		}

		fmt.Printf("Removed shim: %s\n", cmdName)
	}

	// Update state to remove shim registrations
	if err := state.WithLockedState(func(s *state.State) error {
		for _, cmdName := range commands {
			// Get shim info to find associated environment
			shimInfo := s.ListShims()[cmdName]
			if shimInfo != nil {
				// Remove from environment's exported shims list
				if env := s.GetEnv(shimInfo.Env); env != nil {
					newShims := make([]string, 0)
					for _, shimName := range env.ExportedShims {
						if shimName != cmdName {
							newShims = append(newShims, shimName)
						}
					}
					env.ExportedShims = newShims
				}
			}

			// Unregister from global shim map
			s.UnregisterShim(cmdName)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	if firstError != nil {
		return firstError
	}

	return nil
}

func init() {
	rootCmd.AddCommand(exportBinCmd)
	exportBinCmd.Flags().StringVarP(&exportBinName, "name", "n", "", "Environment name to export commands from")
	exportBinCmd.Flags().StringArrayVarP(&exportBinBins, "bin", "b", []string{}, "Command to export (repeatable)")
	exportBinCmd.Flags().BoolVarP(&exportBinList, "list", "l", false, "List all registered shims")
	exportBinCmd.Flags().StringArrayVarP(&exportBinRemove, "remove", "r", []string{}, "Remove shims (repeatable)")
	exportBinCmd.Flags().BoolVarP(&exportBinForce, "force", "f", false, "Overwrite existing shims")
}
