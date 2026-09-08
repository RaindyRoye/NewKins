package comm

import (
	"errors"
	"testing"
)

func TestSentinelErrors_Config_InvalidDriver(t *testing.T) {
	cfg := &Config{}
	cfg.Datasource.Driver = "invalid"
	cfg.Datasource.Url = "test"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid driver")
	}
	if !errors.Is(err, ErrUnsupportedDriver) {
		t.Errorf("expected error to wrap ErrUnsupportedDriver, got: %v", err)
	}
}

func TestSentinelErrors_Config_MissingDriver(t *testing.T) {
	cfg := &Config{}
	cfg.Datasource.Driver = ""
	cfg.Datasource.Url = "test"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing driver")
	}
	if !errors.Is(err, ErrDriverRequired) {
		t.Errorf("expected error to wrap ErrDriverRequired, got: %v", err)
	}
}

func TestSentinelErrors_Config_MissingURL(t *testing.T) {
	cfg := &Config{}
	cfg.Datasource.Driver = DatasourceDriverSQLite
	cfg.Datasource.Url = ""

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
	if !errors.Is(err, ErrURLRequired) {
		t.Errorf("expected error to wrap ErrURLRequired, got: %v", err)
	}
}

func TestSentinelErrors_Config_InvalidRunLimit(t *testing.T) {
	cfg := &Config{}
	cfg.Datasource.Driver = DatasourceDriverSQLite
	cfg.Datasource.Url = "test.db"
	cfg.Server.RunLimit = -1

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid run limit")
	}
	if !errors.Is(err, ErrInvalidRunLimit) {
		t.Errorf("expected error to wrap ErrInvalidRunLimit, got: %v", err)
	}
}
