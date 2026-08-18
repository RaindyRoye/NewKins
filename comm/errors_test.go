package comm

import (
	"errors"
	"testing"
)

func TestErrConfigInvalid_Wrapping(t *testing.T) {
	cfg := &Config{}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty config, got nil")
	}
	if !errors.Is(err, ErrConfigInvalid) {
		t.Errorf("expected errors.Is(err, ErrConfigInvalid) to be true, got false; err=%v", err)
	}
}

func TestErrConfigInvalid_UnsupportedDriver(t *testing.T) {
	cfg := &Config{}
	cfg.Datasource.Driver = "oracle"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}
	if !errors.Is(err, ErrConfigInvalid) {
		t.Errorf("expected errors.Is(err, ErrConfigInvalid) to be true; err=%v", err)
	}
}

func TestErrConfigInvalid_MissingURL(t *testing.T) {
	cfg := &Config{}
	cfg.Datasource.Driver = DatasourceDriverSQLite
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
	if !errors.Is(err, ErrConfigInvalid) {
		t.Errorf("expected errors.Is(err, ErrConfigInvalid) to be true; err=%v", err)
	}
}

func TestErrConfigInvalid_NegativeRunLimit(t *testing.T) {
	cfg := &Config{}
	cfg.Datasource.Driver = DatasourceDriverSQLite
	cfg.Datasource.Url = "./test.db"
	cfg.Server.RunLimit = -1
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for negative run limit")
	}
	if !errors.Is(err, ErrConfigInvalid) {
		t.Errorf("expected errors.Is(err, ErrConfigInvalid) to be true; err=%v", err)
	}
}

func TestErrConfigInvalid_ValidConfig(t *testing.T) {
	cfg := &Config{}
	cfg.Datasource.Driver = DatasourceDriverSQLite
	cfg.Datasource.Url = "./test.db"
	cfg.Server.RunLimit = 5
	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected no error for valid config, got: %v", err)
	}
}

func TestErrAssetNotFound_Asset(t *testing.T) {
	_, err := Asset("nonexistent/file.sql")
	if err == nil {
		t.Fatal("expected error for nonexistent asset")
	}
	if !errors.Is(err, ErrAssetNotFound) {
		t.Errorf("expected errors.Is(err, ErrAssetNotFound) to be true; err=%v", err)
	}
}

func TestErrAssetNotFound_AssetDir(t *testing.T) {
	_, err := AssetDir("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
	if !errors.Is(err, ErrAssetNotFound) {
		t.Errorf("expected errors.Is(err, ErrAssetNotFound) to be true; err=%v", err)
	}
}
