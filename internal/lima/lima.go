package lima

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const (
	Instance = "silibox"
)

type Config struct {
	CPUs   int
	Memory string
	Disk   string
}

type LimaInstance struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func Up(cfg Config) error {
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
	return cmd.Run()
}

func Status() (string, error) {
	out, err := exec.Command("limactl", "list", "--json").CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func Stop() error {
	cmd := exec.Command("limactl", "stop", Instance)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
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

	// Allow overriding the template path via environment for tests or customization
	// Otherwise, try common locations relative to the current working directory
	// and the repository root (walking up from the current directory).
	templatePath := os.Getenv("SILI_LIMA_TEMPLATE")
	if templatePath == "" {
		candidates := []string{
			"build/lima/templates/ubuntu-lts.yaml.tmpl",
			filepath.Clean(filepath.Join("..", "..", "build", "lima", "templates", "ubuntu-lts.yaml.tmpl")),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				templatePath = c
				break
			}
		}
		// As a last resort, walk up a few levels to find the repo root containing build/lima/templates
		if templatePath == "" {
			wd, _ := os.Getwd()
			walk := wd
			for i := 0; i < 4 && templatePath == ""; i++ {
				candidate := filepath.Join(walk, "build", "lima", "templates", "ubuntu-lts.yaml.tmpl")
				if _, err := os.Stat(candidate); err == nil {
					templatePath = candidate
					break
				}
				walk = filepath.Dir(walk)
			}
		}
	}

	if templatePath == "" {
		return fmt.Errorf("missing lima template: could not locate template file")
	}

	tmplBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("missing lima template: %w", err)
	}
	var buf bytes.Buffer
	if err := template.Must(template.New("lima").Parse(string(tmplBytes))).Execute(&buf, cfg); err != nil {
		return err
	}
	return os.WriteFile(yamlPath, buf.Bytes(), 0o644)
}
