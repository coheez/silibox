package container

import (
	"testing"

	"github.com/coheez/silibox/internal/state"
)

func TestParsePortSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		want    state.PortMapping
		wantErr bool
	}{
		{
			name: "single port",
			spec: "3000",
			want: state.PortMapping{
				HostPort:      3000,
				ContainerPort: 3000,
				Protocol:      "tcp",
			},
			wantErr: false,
		},
		{
			name: "port mapping",
			spec: "8080:80",
			want: state.PortMapping{
				HostPort:      8080,
				ContainerPort: 80,
				Protocol:      "tcp",
			},
			wantErr: false,
		},
		{
			name: "port mapping with tcp",
			spec: "8080:80/tcp",
			want: state.PortMapping{
				HostPort:      8080,
				ContainerPort: 80,
				Protocol:      "tcp",
			},
			wantErr: false,
		},
		{
			name: "port mapping with udp",
			spec: "5353:53/udp",
			want: state.PortMapping{
				HostPort:      5353,
				ContainerPort: 53,
				Protocol:      "udp",
			},
			wantErr: false,
		},
		{
			name: "single port with tcp",
			spec: "3000/tcp",
			want: state.PortMapping{
				HostPort:      3000,
				ContainerPort: 3000,
				Protocol:      "tcp",
			},
			wantErr: false,
		},
		{
			name:    "invalid protocol",
			spec:    "3000/http",
			wantErr: true,
		},
		{
			name:    "invalid port number",
			spec:    "abc",
			wantErr: true,
		},
		{
			name:    "invalid host port",
			spec:    "abc:80",
			wantErr: true,
		},
		{
			name:    "invalid container port",
			spec:    "80:abc",
			wantErr: true,
		},
		{
			name:    "port out of range low",
			spec:    "0",
			wantErr: true,
		},
		{
			name:    "port out of range high",
			spec:    "70000",
			wantErr: true,
		},
		{
			name:    "host port out of range",
			spec:    "70000:80",
			wantErr: true,
		},
		{
			name:    "container port out of range",
			spec:    "80:70000",
			wantErr: true,
		},
		{
			name:    "too many colons",
			spec:    "8080:80:80",
			wantErr: true,
		},
		{
			name:    "too many slashes",
			spec:    "8080/tcp/udp",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePortSpec(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePortSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.HostPort != tt.want.HostPort {
					t.Errorf("parsePortSpec() HostPort = %v, want %v", got.HostPort, tt.want.HostPort)
				}
				if got.ContainerPort != tt.want.ContainerPort {
					t.Errorf("parsePortSpec() ContainerPort = %v, want %v", got.ContainerPort, tt.want.ContainerPort)
				}
				if got.Protocol != tt.want.Protocol {
					t.Errorf("parsePortSpec() Protocol = %v, want %v", got.Protocol, tt.want.Protocol)
				}
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"valid port 1", 1, false},
		{"valid port 80", 80, false},
		{"valid port 3000", 3000, false},
		{"valid port 65535", 65535, false},
		{"invalid port 0", 0, true},
		{"invalid port -1", -1, true},
		{"invalid port 65536", 65536, true},
		{"invalid port 70000", 70000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePort(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePort() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParsePortSpecs(t *testing.T) {
	tests := []struct {
		name    string
		specs   []string
		want    int // number of mappings expected
		wantErr bool
	}{
		{
			name:    "empty",
			specs:   []string{},
			want:    0,
			wantErr: false,
		},
		{
			name:    "single spec",
			specs:   []string{"3000"},
			want:    1,
			wantErr: false,
		},
		{
			name:    "multiple specs",
			specs:   []string{"3000", "8080:80", "5353:53/udp"},
			want:    3,
			wantErr: false,
		},
		{
			name:    "invalid spec in list",
			specs:   []string{"3000", "invalid", "8080:80"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePortSpecs(tt.specs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePortSpecs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.want {
				t.Errorf("ParsePortSpecs() returned %d mappings, want %d", len(got), tt.want)
			}
		})
	}
}
