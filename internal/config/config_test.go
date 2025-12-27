package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Autosleep.ContainerTimeout != 15*time.Minute {
		t.Errorf("expected container timeout 15m, got %v", cfg.Autosleep.ContainerTimeout)
	}
	if cfg.Autosleep.VMTimeout != 30*time.Minute {
		t.Errorf("expected VM timeout 30m, got %v", cfg.Autosleep.VMTimeout)
	}
	if cfg.Autosleep.PollInterval != 30*time.Second {
		t.Errorf("expected poll interval 30s, got %v", cfg.Autosleep.PollInterval)
	}
	if cfg.Autosleep.NoStopVM != false {
		t.Errorf("expected no_stop_vm false, got %v", cfg.Autosleep.NoStopVM)
	}
}

func TestLoad_NoFile(t *testing.T) {
	// Set HOME to a temp dir where config doesn't exist
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error when config doesn't exist, got %v", err)
	}

	// Should return default config
	expected := DefaultConfig()
	if cfg.Autosleep.ContainerTimeout != expected.Autosleep.ContainerTimeout {
		t.Errorf("expected default container timeout, got %v", cfg.Autosleep.ContainerTimeout)
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create .sili directory
	siliDir := filepath.Join(tmpDir, ".sili")
	if err := os.MkdirAll(siliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write config file
	configContent := `autosleep:
  container_timeout: 10m
  vm_timeout: 20m
  poll_interval: 15s
  no_stop_vm: true
`
	configPath := filepath.Join(siliDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Autosleep.ContainerTimeout != 10*time.Minute {
		t.Errorf("expected container timeout 10m, got %v", cfg.Autosleep.ContainerTimeout)
	}
	if cfg.Autosleep.VMTimeout != 20*time.Minute {
		t.Errorf("expected VM timeout 20m, got %v", cfg.Autosleep.VMTimeout)
	}
	if cfg.Autosleep.PollInterval != 15*time.Second {
		t.Errorf("expected poll interval 15s, got %v", cfg.Autosleep.PollInterval)
	}
	if cfg.Autosleep.NoStopVM != true {
		t.Errorf("expected no_stop_vm true, got %v", cfg.Autosleep.NoStopVM)
	}
}

func TestLoad_PartialConfig(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	siliDir := filepath.Join(tmpDir, ".sili")
	if err := os.MkdirAll(siliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Only override container_timeout
	configContent := `autosleep:
  container_timeout: 5m
`
	configPath := filepath.Join(siliDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should have custom container timeout
	if cfg.Autosleep.ContainerTimeout != 5*time.Minute {
		t.Errorf("expected container timeout 5m, got %v", cfg.Autosleep.ContainerTimeout)
	}

	// Other values should be defaults
	if cfg.Autosleep.VMTimeout != 30*time.Minute {
		t.Errorf("expected default VM timeout 30m, got %v", cfg.Autosleep.VMTimeout)
	}
	if cfg.Autosleep.PollInterval != 30*time.Second {
		t.Errorf("expected default poll interval 30s, got %v", cfg.Autosleep.PollInterval)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	siliDir := filepath.Join(tmpDir, ".sili")
	if err := os.MkdirAll(siliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write invalid YAML
	configContent := `autosleep:
  container_timeout: [invalid
`
	configPath := filepath.Join(siliDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}
