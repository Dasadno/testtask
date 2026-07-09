package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Dasadno/testtask/internal/config"
)

func TestLoad_FromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte("env: prod\nhttp:\n  port: 9090\n  read_timeout: 3s\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Env != config.EnvProd {
		t.Errorf("Env = %q, want %q", cfg.Env, config.EnvProd)
	}
	if cfg.HTTP.Port != 9090 {
		t.Errorf("HTTP.Port = %d, want 9090", cfg.HTTP.Port)
	}
	if cfg.HTTP.ReadTimeout != 3*time.Second {
		t.Errorf("HTTP.ReadTimeout = %v, want 3s", cfg.HTTP.ReadTimeout)
	}
}

func TestLoad_FileMissing_UsesDefaultsAndEnv(t *testing.T) {
	t.Setenv("HTTP_PORT", "9999")

	cfg, err := config.Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Env != config.EnvLocal {
		t.Errorf("Env = %q, want default %q", cfg.Env, config.EnvLocal)
	}
	if cfg.HTTP.Port != 9999 {
		t.Errorf("HTTP.Port = %d, want 9999 from env", cfg.HTTP.Port)
	}
}
