package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	os.WriteFile(cfgPath, []byte(`{"server":"http://192.168.1.50:20080"}`), 0644)
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Server != "http://192.168.1.50:20080" {
		t.Errorf("Server = %q", cfg.Server)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/config.json")
	if err == nil {
		t.Error("expected error")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	os.WriteFile(cfgPath, []byte("not json"), 0644)
	_, err := Load(cfgPath)
	if err == nil {
		t.Error("expected error")
	}
}

func TestDefaultPath(t *testing.T) {
	if DefaultPath() == "" {
		t.Error("DefaultPath should not be empty")
	}
}
