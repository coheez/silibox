package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/coheez/silibox/internal/container"
	"github.com/coheez/silibox/internal/lima"
	runtimex "github.com/coheez/silibox/internal/runtime"
	"github.com/spf13/cobra"
)

var (
	cpus        int
	memory      string
	disk        string
	createName  string
	createImage string
	createDir   string
	createWork  string
	createUser  string
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

var vmCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a named Podman container in the VM",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := container.CreateConfig{
			Name:       createName,
			Image:      createImage,
			ProjectDir: createDir,
			WorkingDir: createWork,
			User:       createUser,
		}
		return container.Create(cfg)
	},
}

var outputJSON bool

func init() {
	vmCmd.AddCommand(vmUpCmd, vmStatusCmd, vmStopCmd, vmProbeCmd, vmCreateCmd)
	vmUpCmd.Flags().IntVar(&cpus, "cpus", 4, "vCPUs")
	vmUpCmd.Flags().StringVar(&memory, "memory", "8GiB", "RAM (e.g., 8GiB)")
	vmUpCmd.Flags().StringVar(&disk, "disk", "60GiB", "Disk size")
	vmStatusCmd.Flags().BoolVarP(&outputJSON, "json", "j", false, "Output JSON")
	vmCreateCmd.Flags().StringVarP(&createName, "name", "n", "silibox-dev", "Container name")
	vmCreateCmd.Flags().StringVarP(&createImage, "image", "i", "ubuntu:22.04", "Container image")
	vmCreateCmd.Flags().StringVarP(&createDir, "dir", "d", ".", "Project directory to bind mount")
	vmCreateCmd.Flags().StringVarP(&createWork, "workdir", "w", "/workspace", "Working directory inside container")
	vmCreateCmd.Flags().StringVarP(&createUser, "user", "u", "", "User to run as (default: current user)")
}
