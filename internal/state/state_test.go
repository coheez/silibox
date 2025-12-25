package state

import (
	"testing"
)

func TestIsPortInUse(t *testing.T) {
	// Create a state with some environments
	s := NewState()

	// Add environment with ports
	env1 := &EnvInfo{
		Name:  "web",
		Image: "nginx",
		Ports: []PortMapping{
			{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
			{HostPort: 8443, ContainerPort: 443, Protocol: "tcp"},
		},
	}
	s.UpsertEnv(env1)

	env2 := &EnvInfo{
		Name:  "api",
		Image: "node",
		Ports: []PortMapping{
			{HostPort: 3000, ContainerPort: 3000, Protocol: "tcp"},
		},
	}
	s.UpsertEnv(env2)

	tests := []struct {
		name        string
		port        int
		wantInUse   bool
		wantEnvName string
	}{
		{
			name:        "port in use by web",
			port:        8080,
			wantInUse:   true,
			wantEnvName: "web",
		},
		{
			name:        "port in use by api",
			port:        3000,
			wantInUse:   true,
			wantEnvName: "api",
		},
		{
			name:        "port not in use",
			port:        5000,
			wantInUse:   false,
			wantEnvName: "",
		},
		{
			name:        "another port in use by web",
			port:        8443,
			wantInUse:   true,
			wantEnvName: "web",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotInUse, gotEnvName := s.IsPortInUse(tt.port)
			if gotInUse != tt.wantInUse {
				t.Errorf("IsPortInUse() inUse = %v, want %v", gotInUse, tt.wantInUse)
			}
			if gotEnvName != tt.wantEnvName {
				t.Errorf("IsPortInUse() envName = %v, want %v", gotEnvName, tt.wantEnvName)
			}
		})
	}
}

func TestRemoveEnvReleasesPort(t *testing.T) {
	s := NewState()

	// Add environment with port
	env := &EnvInfo{
		Name:  "test",
		Image: "nginx",
		Ports: []PortMapping{
			{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
		},
	}
	s.UpsertEnv(env)

	// Verify port is in use
	if inUse, _ := s.IsPortInUse(8080); !inUse {
		t.Error("Port 8080 should be in use")
	}

	// Remove environment
	s.RemoveEnv("test")

	// Verify port is no longer in use
	if inUse, _ := s.IsPortInUse(8080); inUse {
		t.Error("Port 8080 should not be in use after environment removal")
	}
}
