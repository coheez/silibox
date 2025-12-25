package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/coheez/silibox/internal/state"
	"github.com/spf13/cobra"
)

var (
	portsEnv string
)

var portsCmd = &cobra.Command{
	Use:   "ports",
	Short: "List active port mappings",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load state
		st, err := state.Load()
		if err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}

		// Collect all port mappings
		type portInfo struct {
			envName       string
			hostPort      int
			containerPort int
			protocol      string
			url           string
		}

		var allPorts []portInfo

		for envName, env := range st.Envs {
			// Filter by environment if specified
			if portsEnv != "" && envName != portsEnv {
				continue
			}

			for _, pm := range env.Ports {
				// Generate URL
				url := generateURL(pm.HostPort, pm.Protocol)

				allPorts = append(allPorts, portInfo{
					envName:       envName,
					hostPort:      pm.HostPort,
					containerPort: pm.ContainerPort,
					protocol:      pm.Protocol,
					url:           url,
				})
			}
		}

		// Check if no ports found
		if len(allPorts) == 0 {
			if portsEnv != "" {
				fmt.Printf("No port mappings found for environment '%s'.\n", portsEnv)
			} else {
				fmt.Println("No port mappings found.")
				fmt.Println("Add ports with: sili create --name <env> --ports <port> ...")
			}
			return nil
		}

		// Sort by environment name, then by host port
		sort.Slice(allPorts, func(i, j int) bool {
			if allPorts[i].envName != allPorts[j].envName {
				return allPorts[i].envName < allPorts[j].envName
			}
			return allPorts[i].hostPort < allPorts[j].hostPort
		})

		// Print header
		fmt.Printf("%-20s %-12s %-16s %-10s %s\n", "ENV", "HOST PORT", "CONTAINER PORT", "PROTOCOL", "URL")
		fmt.Println(strings.Repeat("-", 90))

		// Print each port mapping
		for _, port := range allPorts {
			fmt.Printf("%-20s %-12d %-16d %-10s %s\n",
				port.envName,
				port.hostPort,
				port.containerPort,
				port.protocol,
				port.url,
			)
		}

		return nil
	},
}

// generateURL creates a clickable URL from port and protocol
func generateURL(port int, protocol string) string {
	if protocol == "tcp" {
		// Assume HTTP for common web ports
		if port == 80 || port == 8080 || port == 3000 || port == 4200 || port == 5000 || port == 8000 {
			return fmt.Sprintf("http://localhost:%d", port)
		}
		// For HTTPS ports
		if port == 443 || port == 8443 {
			return fmt.Sprintf("https://localhost:%d", port)
		}
		// Generic TCP
		return fmt.Sprintf("tcp://localhost:%d", port)
	}
	// UDP
	return fmt.Sprintf("udp://localhost:%d", port)
}

func init() {
	rootCmd.AddCommand(portsCmd)
	portsCmd.Flags().StringVarP(&portsEnv, "env", "e", "", "Filter by environment name")
}
