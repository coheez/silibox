package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/coheez/silibox/internal/lima"
	runtimex "github.com/coheez/silibox/internal/runtime"
	"github.com/spf13/cobra"
)

var (
	cpus       int
	memory     string
	disk       string
	statusLive bool
)

var vmCmd = &cobra.Command{
	Use:   "vm",
	Short: "Manage Silibox VM",
}

var vmUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Create/Start the Silibox VM",
	RunE: func(cmd *cobra.Command, args []string) error {
		return lima.Up(lima.Config{CPUs: cpus, Memory: memory, Disk: disk})
	},
}

var vmStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Silibox VM status",
	RunE: func(cmd *cobra.Command, args []string) error {
		var status string
		var err error

		if statusLive {
			status, err = lima.StatusLive()
		} else {
			status, err = lima.Status()
		}

		if err != nil {
			return err
		}

		if outputJSON {
			// For JSON output, we need structured data
			info, err := lima.GetStatus()
			if err != nil {
				return err
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(info)
		}

		// Simple text output
		fmt.Println(status)
		return nil
	},
}

var vmStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Silibox VM",
	RunE: func(cmd *cobra.Command, args []string) error {
		return lima.Stop()
	},
}

var vmProbeCmd = &cobra.Command{
	Use:   "probe",
	Short: "Run runtime probe inside VM (podman hello)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runtimex.Probe()
	},
}

var outputJSON bool

func init() {
	vmCmd.AddCommand(vmUpCmd, vmStatusCmd, vmStopCmd, vmProbeCmd)
	vmUpCmd.Flags().IntVar(&cpus, "cpus", 4, "vCPUs")
	vmUpCmd.Flags().StringVar(&memory, "memory", "8GiB", "RAM (e.g., 8GiB)")
	vmUpCmd.Flags().StringVar(&disk, "disk", "60GiB", "Disk size")
	vmStatusCmd.Flags().BoolVarP(&outputJSON, "json", "j", false, "Output JSON")
	vmStatusCmd.Flags().BoolVarP(&statusLive, "live", "l", false, "Get live status from lima (slower but always current)")
}
