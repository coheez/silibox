package cli

import (
	"fmt"
	"github.com/coheez/silibox/internal/lima"
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
		s, err := lima.Status()
		if err != nil {
			return err
		}
		fmt.Println(s)
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

func init() {
	vmCmd.AddCommand(vmUpCmd, vmStatusCmd, vmStopCmd)
	vmUpCmd.Flags().IntVar(&cpus, "cpus", 4, "vCPUs")
	vmUpCmd.Flags().StringVar(&memory, "memory", "8GiB", "RAM (e.g., 8GiB)")
	vmUpCmd.Flags().StringVar(&disk, "disk", "60GiB", "Disk size")
}