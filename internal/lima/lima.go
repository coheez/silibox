package lima

import (
	"bytes"
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

func Up(cfg Config) error {
	if err := ensureTemplate(cfg); err != nil {
		return err
	}
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

func ensureTemplate(cfg Config) error {
	dir := filepath.Join(os.Getenv("HOME"), ".sili")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	yamlPath := filepath.Join(dir, "lima.yaml")

	tmplBytes, err := os.ReadFile("build/lima/templates/ubuntu-lts.yaml.tmpl")
	if err != nil {
		return fmt.Errorf("missing lima template: %w", err)
	}
	var buf bytes.Buffer
	if err := template.Must(template.New("lima").Parse(string(tmplBytes))).Execute(&buf, cfg); err != nil {
		return err
	}
	return os.WriteFile(yamlPath, buf.Bytes(), 0o644)
}