package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/coheez/silibox/internal/lima"
	runtimex "github.com/coheez/silibox/internal/runtime"
	"github.com/spf13/cobra"
)

var (
	cpus   int
	memory string
	disk   string
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
		info, err := lima.GetStatus()
		if err != nil {
			return err
		}
		if outputJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(info)
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(tw, "NAME\tSTATUS\n")
		fmt.Fprintf(tw, "%s\t%s\n", info.Name, info.Status)
		return tw.Flush()
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
}
