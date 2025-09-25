package runtime

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/coheez/silibox/internal/lima"
)

// Probe verifies that podman is available inside the Silibox VM and can run a container.
// It runs a simple hello-world container to warm the image cache and validate networking.
func Probe() error {
	// First, check podman presence
	if err := runInVM("podman", "--version"); err != nil {
		return fmt.Errorf("podman not available in VM: %w", err)
	}

	// Pull and run a tiny hello container. Using docker.io/library/hello-world ensures availability.
	// --rm ensures the container is cleaned up after exit.
	if err := runInVM("podman", "run", "--rm", "--pull=always", "docker.io/library/hello-world:latest"); err != nil {
		return fmt.Errorf("failed to run hello-world via podman in VM: %w", err)
	}
	return nil
}

func runInVM(cmd string, args ...string) error {
	fullArgs := append([]string{"shell", lima.Instance, "--", cmd}, args...)
	c := exec.Command("limactl", fullArgs...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
