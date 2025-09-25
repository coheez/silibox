package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/coheez/silibox/internal/state"
	"github.com/spf13/cobra"
)

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "View or manage silibox state",
}

var stateShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current state",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := state.Load()
		if err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(s)
	},
}

func init() {
	rootCmd.AddCommand(stateCmd)
	stateCmd.AddCommand(stateShowCmd)
}
