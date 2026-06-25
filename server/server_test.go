package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/gokins/comm"
	"gopkg.in/yaml.v3"
)

func TestParseConfig_MissingFile(t *testing.T) {
	origWorkPath := comm.WorkPath
	defer func() { comm.WorkPath = origWorkPath }()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	err := parseConfig()
	if err == nil {
		t.Fatal("expected error when config file is missing, got nil")
	}
}

func TestParseConfig_ValidConfig(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCfg := comm.Cfg
	defer func() {
		comm.WorkPath = origWorkPath
		comm.Cfg = origCfg
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	cfg := comm.Config{}
	cfg.Datasource.Driver = "sqlite"
	cfg.Datasource.Url = filepath.Join(tmpDir, "test.db")
	cfg.Server.Host = "http://localhost:8080"
	cfg.Server.RunLimit = 10

	bts, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "app.yml"), bts, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	err = parseConfig()
	if err != nil {
		t.Fatalf("expected no error for valid config, got: %v", err)
	}
	if comm.Cfg.Datasource.Driver != "sqlite" {
		t.Errorf("expected driver 'sqlite', got %q", comm.Cfg.Datasource.Driver)
	}
	if comm.Cfg.Server.RunLimit != 10 {
		t.Errorf("expected runLimit 10, got %d", comm.Cfg.Server.RunLimit)
	}
}

func TestParseConfig_InvalidYAML(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCfg := comm.Cfg
	defer func() {
		comm.WorkPath = origWorkPath
		comm.Cfg = origCfg
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Write invalid YAML content
	invalidYAML := []byte(":\n  invalid:\n    - [broken yaml")
	if err := os.WriteFile(filepath.Join(tmpDir, "app.yml"), invalidYAML, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	err := parseConfig()
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestParseConfig_InvalidDriver(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCfg := comm.Cfg
	defer func() {
		comm.WorkPath = origWorkPath
		comm.Cfg = origCfg
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	cfg := comm.Config{}
	cfg.Datasource.Driver = "oracle" // unsupported driver
	cfg.Datasource.Url = "some-url"

	bts, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "app.yml"), bts, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	err = parseConfig()
	if err == nil {
		t.Fatal("expected validation error for unsupported driver, got nil")
	}
}

func TestParseConfig_YamlFallback(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCfg := comm.Cfg
	defer func() {
		comm.WorkPath = origWorkPath
		comm.Cfg = origCfg
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	cfg := comm.Config{}
	cfg.Datasource.Driver = "mysql"
	cfg.Datasource.Url = "user:pass@tcp(localhost)/db"

	bts, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	// Write as app.yaml (not app.yml) — should be picked up as fallback
	if err := os.WriteFile(filepath.Join(tmpDir, "app.yaml"), bts, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	err = parseConfig()
	if err != nil {
		t.Fatalf("expected no error when using app.yaml fallback, got: %v", err)
	}
	if comm.Cfg.Datasource.Driver != "mysql" {
		t.Errorf("expected driver 'mysql', got %q", comm.Cfg.Datasource.Driver)
	}
}

func TestParseConfig_NegativeRunLimit(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCfg := comm.Cfg
	defer func() {
		comm.WorkPath = origWorkPath
		comm.Cfg = origCfg
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	cfg := comm.Config{}
	cfg.Datasource.Driver = "sqlite"
	cfg.Datasource.Url = filepath.Join(tmpDir, "test.db")
	cfg.Server.RunLimit = -1

	bts, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "app.yml"), bts, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	err = parseConfig()
	if err == nil {
		t.Fatal("expected validation error for negative runLimit, got nil")
	}
}
