package lima

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/coheez/silibox/internal/state"
)

//go:embed templates/ubuntu-lts.yaml.tmpl
var embeddedTemplate string

const (
	Instance = "silibox"
)

type Config struct {
	CPUs   int
	Memory string
	Disk   string
}

type tmplData struct {
	Config
	Arch        string
	ImageURL    string
	ImageDigest string
}

type LimaInstance struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// StatusInfo represents a minimal view of the VM status for display/JSON.
type StatusInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func Up(cfg Config) error {
	return state.WithLockedState(func(s *state.State) error {
		if err := ensureTemplate(cfg); err != nil {
			return err
		}

		// Check if instance already exists
		if exists, err := instanceExists(); err != nil {
			return err
		} else if !exists {
			// Create the instance using the recommended command
			yamlPath := filepath.Join(os.Getenv("HOME"), ".sili", "lima.yaml")
			cmd := exec.Command("limactl", "create", "--name="+Instance, yamlPath)
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			if err := cmd.Run(); err != nil {
				return err
			}
		}

		// Start the instance
		cmd := exec.Command("limactl", "start", Instance)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}

		// Wait for the VM to reach Running state
		if err := waitForRunning(); err != nil {
			return err
		}

		// Update state
		configData, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".sili", "lima.yaml"))
		if err != nil {
			return fmt.Errorf("failed to read config for checksum: %w", err)
		}

		vmInfo := &state.VMInfo{
			Name:         Instance,
			Backend:      "lima-vz",
			Profile:      "balanced",
			CPUs:         cfg.CPUs,
			Memory:       cfg.Memory,
			Disk:         cfg.Disk,
			Status:       "running",
			ConfigSHA256: state.ComputeConfigSHA256(configData),
			LastActive:   time.Now(),
		}
		s.SetVM(vmInfo)

		return nil
	})
}

func Status() (string, error) {
	return StatusFromState(false)
}

func StatusLive() (string, error) {
	return StatusFromState(true)
}

func StatusFromState(forceLive bool) (string, error) {
	if forceLive {
		// Get live status from lima
		inst, found, err := getInstance()
		if err != nil {
			return "", err
		}
		if !found {
			return "VM not found", nil
		}
		return fmt.Sprintf("VM status: %s", inst.Status), nil
	}

	// Read from state (fast)
	s, err := state.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load state: %w", err)
	}

	vm := s.GetVM()
	if vm == nil {
		return "VM not found", nil
	}

	return fmt.Sprintf("VM status: %s", vm.Status), nil
}

// GetStatus returns structured status information for the silibox instance.
func GetStatus() (StatusInfo, error) {
	s, err := state.Load()
	if err != nil {
		return StatusInfo{}, fmt.Errorf("failed to load state: %w", err)
	}

	vm := s.GetVM()
	if vm == nil {
		return StatusInfo{Name: Instance, Status: "NotFound"}, nil
	}

	return StatusInfo{
		Name:   vm.Name,
		Status: vm.Status,
	}, nil
}

// GetInstance returns the current instance if present.
func GetInstance() (LimaInstance, bool, error) {
	return getInstance()
}

func Stop() error {
	return state.WithLockedState(func(s *state.State) error {
		// Check current state
		inst, found, err := getInstance()
		if err != nil {
			return err
		}
		if !found || inst.Status == "Stopped" {
			// Already stopped or not created; treat as success
			s.UpdateVMStatus("stopped")
			return nil
		}

		// Ask Lima to stop the instance
		cmd := exec.Command("limactl", "stop", Instance)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}

		// Wait until the instance reports Stopped to ensure cleanup
		if err := waitForState("Stopped", 2*time.Minute); err != nil {
			return err
		}

		// Update state
		s.UpdateVMStatus("stopped")
		return nil
	})
}

func instanceExists() (bool, error) {
	out, err := exec.Command("limactl", "list", "--json").CombinedOutput()
	if err != nil {
		return false, err
	}

	// If output doesn't start with '{', it means no instances exist
	outStr := string(out)
	if len(outStr) == 0 || outStr[0] != '{' {
		return false, nil
	}

	var instance LimaInstance
	if err := json.Unmarshal(out, &instance); err != nil {
		return false, err
	}

	return instance.Name == Instance, nil
}

func ensureTemplate(cfg Config) error {
	dir := filepath.Join(os.Getenv("HOME"), ".sili")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	yamlPath := filepath.Join(dir, "lima.yaml")

	// Use embedded template, but allow override via environment for tests
	tmplContent := embeddedTemplate
	if envPath := os.Getenv("SILI_LIMA_TEMPLATE"); envPath != "" {
		tmplBytes, err := os.ReadFile(envPath)
		if err != nil {
			return fmt.Errorf("failed to read custom template: %w", err)
		}
		tmplContent = string(tmplBytes)
	}

	arch, imgURL, imgDigest := resolveUbuntuImage()
	data := tmplData{
		Config:      cfg,
		Arch:        arch,
		ImageURL:    imgURL,
		ImageDigest: imgDigest,
	}

	var buf bytes.Buffer
	t := template.Must(template.New("lima").Option("missingkey=zero").Parse(tmplContent))
	if err := t.Execute(&buf, data); err != nil {
		return err
	}
	return os.WriteFile(yamlPath, buf.Bytes(), 0o644)
}

// resolveUbuntuImage picks the appropriate Ubuntu Noble image URL and best-effort digest for the host arch.
// It returns (archForYAML, imageURL, sha256Digest). If digest cannot be determined, it returns an empty string,
// and the template conditionally omits the digest field.
func resolveUbuntuImage() (string, string, string) {
	var archYAML, fileSuffix string
	switch runtime.GOARCH {
	case "arm64":
		archYAML, fileSuffix = "aarch64", "arm64"
	case "amd64":
		archYAML, fileSuffix = "x86_64", "amd64"
	default:
		// Default to arm64 settings; Lima will still error usefully if unsupported host
		archYAML, fileSuffix = "aarch64", "arm64"
	}
	base := "https://cloud-images.ubuntu.com/noble/current/"
	file := "noble-server-cloudimg-" + fileSuffix + ".img"
	url := base + file

	// Best-effort digest lookup from SHA256SUMS
	digest := fetchSHA256FromSums(base+"SHA256SUMS", file)
	return archYAML, url, digest
}

func fetchSHA256FromSums(sumsURL, fileName string) string {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(sumsURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}
	defer resp.Body.Close()
	s := bufio.NewScanner(resp.Body)
	for s.Scan() {
		line := s.Text()
		// Lines look like: "<sha256> *noble-server-cloudimg-arm64.img"
		if strings.HasSuffix(line, fileName) {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				return "sha256:" + parts[0]
			}
		}
	}
	return ""
}

// waitForRunning waits for the VM to reach Running state with a timeout
func waitForRunning() error {
	timeout := 5 * time.Minute
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	timeoutC := time.After(timeout)

	for {
		select {
		case <-ticker.C:
			inst, found, err := getInstance()
			if err != nil {
				return fmt.Errorf("failed to check VM status: %w", err)
			}
			if !found {
				continue // Keep waiting if instance not found yet
			}
			switch inst.Status {
			case "Running":
				return nil
			case "Error", "Broken":
				return fmt.Errorf("VM failed to start, status: %s", inst.Status)
			}
		case <-timeoutC:
			return fmt.Errorf("timeout waiting for VM to start (waited %v)", timeout)
		}
	}
}

// waitForState waits until the instance reports the target state or times out.
func waitForState(target string, timeout time.Duration) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	timeoutC := time.After(timeout)

	for {
		select {
		case <-ticker.C:
			inst, found, err := getInstance()
			if err != nil {
				return fmt.Errorf("failed to check VM status: %w", err)
			}
			if !found {
				if target == "NotFound" {
					return nil
				}
				// If target is Stopped but instance disappeared, consider it stopped
				if target == "Stopped" {
					return nil
				}
				continue
			}
			if inst.Status == target {
				return nil
			}
			if inst.Status == "Error" || inst.Status == "Broken" {
				return fmt.Errorf("VM entered failure state: %s", inst.Status)
			}
		case <-timeoutC:
			return fmt.Errorf("timeout waiting for VM state %q (waited %v)", target, timeout)
		}
	}
}

// getInstance returns the current instance if present.
func getInstance() (LimaInstance, bool, error) {
	out, err := exec.Command("limactl", "list", "--json").CombinedOutput()
	if err != nil {
		return LimaInstance{}, false, err
	}

	var instances []LimaInstance
	if err := json.Unmarshal(out, &instances); err != nil {
		var instance LimaInstance
		if err := json.Unmarshal(out, &instance); err != nil {
			return LimaInstance{}, false, fmt.Errorf("failed to parse lima output: %w", err)
		}
		instances = []LimaInstance{instance}
	}

	for _, instance := range instances {
		if instance.Name == Instance {
			return instance, true, nil
		}
	}
	return LimaInstance{}, false, nil
}
