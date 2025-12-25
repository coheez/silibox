package container

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/coheez/silibox/internal/state"
)

// parsePortSpec parses a port specification string into a PortMapping
// Supported formats:
//   - "3000" -> host port 3000, container port 3000, tcp
//   - "8080:80" -> host port 8080, container port 80, tcp
//   - "8080:80/tcp" -> host port 8080, container port 80, tcp
//   - "8080:80/udp" -> host port 8080, container port 80, udp
func parsePortSpec(spec string) (state.PortMapping, error) {
	// Default protocol is tcp
	protocol := "tcp"
	portPart := spec

	// Check for protocol suffix
	if strings.Contains(spec, "/") {
		parts := strings.Split(spec, "/")
		if len(parts) != 2 {
			return state.PortMapping{}, fmt.Errorf("invalid port spec format: %s", spec)
		}
		portPart = parts[0]
		protocol = strings.ToLower(parts[1])
		if protocol != "tcp" && protocol != "udp" {
			return state.PortMapping{}, fmt.Errorf("invalid protocol %s (must be tcp or udp)", protocol)
		}
	}

	// Parse port mapping
	var hostPort, containerPort int
	var err error

	if strings.Contains(portPart, ":") {
		// Format: host:container
		parts := strings.Split(portPart, ":")
		if len(parts) != 2 {
			return state.PortMapping{}, fmt.Errorf("invalid port mapping format: %s", portPart)
		}
		hostPort, err = strconv.Atoi(parts[0])
		if err != nil {
			return state.PortMapping{}, fmt.Errorf("invalid host port %s: %w", parts[0], err)
		}
		containerPort, err = strconv.Atoi(parts[1])
		if err != nil {
			return state.PortMapping{}, fmt.Errorf("invalid container port %s: %w", parts[1], err)
		}
	} else {
		// Format: same port on both sides
		hostPort, err = strconv.Atoi(portPart)
		if err != nil {
			return state.PortMapping{}, fmt.Errorf("invalid port %s: %w", portPart, err)
		}
		containerPort = hostPort
	}

	// Validate port ranges
	if err := validatePort(hostPort); err != nil {
		return state.PortMapping{}, fmt.Errorf("invalid host port: %w", err)
	}
	if err := validatePort(containerPort); err != nil {
		return state.PortMapping{}, fmt.Errorf("invalid container port: %w", err)
	}

	return state.PortMapping{
		HostPort:      hostPort,
		ContainerPort: containerPort,
		Protocol:      protocol,
	}, nil
}

// validatePort checks if a port number is in valid range (1-65535)
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port %d out of range (must be 1-65535)", port)
	}
	return nil
}

// ParsePortSpecs parses multiple port specifications
func ParsePortSpecs(specs []string) ([]state.PortMapping, error) {
	mappings := make([]state.PortMapping, 0, len(specs))
	for _, spec := range specs {
		pm, err := parsePortSpec(spec)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, pm)
	}
	return mappings, nil
}
