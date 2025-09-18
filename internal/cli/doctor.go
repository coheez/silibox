package cli

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose environment and dependencies",
	RunE: func(cmd *cobra.Command, args []string) error {
		ok := true

		fmt.Println("OS:", runtime.GOOS, "Arch:", runtime.GOARCH)

		if _, err := exec.LookPath("limactl"); err != nil {
			ok = false
			fmt.Println("✗ lima not found. Install via: brew install lima")
		} else {
			fmt.Println("✓ lima found")
		}

		// We’ll install podman inside VM, but check if present on host for devs.
		if _, err := exec.LookPath("podman"); err != nil {
			fmt.Println("• podman not found on host (ok). Will install inside VM.")
		} else {
			fmt.Println("✓ podman found on host (optional)")
		}

		// Check Apple virtualization support (vz)
		// lima itself will fail clearly if vz not supported; we just hint here.
		fmt.Println("• Apple Virtualization.framework (vz) is required on Apple Silicon")

		if !ok {
			return fmt.Errorf("doctor found issues (see above)")
		}
		fmt.Println("All good. You're ready to `sili vm up`.")
		return nil
	},
}