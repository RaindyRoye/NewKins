package comm

import (
	"errors"
	"strings"
	"testing"
)

func TestConfig_Validate_ValidMySQL(t *testing.T) {
	cfg := Config{}
	cfg.Datasource.Driver = "mysql"
	cfg.Datasource.Url = "root:pass@tcp(localhost:3306)/gokins?charset=utf8mb4"
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}
}

func TestConfig_Validate_ValidPostgres(t *testing.T) {
	cfg := Config{}
	cfg.Datasource.Driver = "postgres"
	cfg.Datasource.Url = "postgres://user:pass@localhost:5432/gokins?sslmode=disable"
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}
}

func TestConfig_Validate_ValidSQLite(t *testing.T) {
	cfg := Config{}
	cfg.Datasource.Driver = "sqlite"
	cfg.Datasource.Url = "./gokins.db"
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}
}

func TestConfig_Validate_EmptyDriver(t *testing.T) {
	cfg := Config{}
	cfg.Datasource.Url = "some-url"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty driver")
	}
	if !errors.Is(err, ErrDatasourceDriverRequired) {
		t.Errorf("expected ErrDatasourceDriverRequired, got: %v", err)
	}
	if !strings.Contains(err.Error(), "datasource.driver is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestConfig_Validate_UnsupportedDriver(t *testing.T) {
	cfg := Config{}
	cfg.Datasource.Driver = "oracle"
	cfg.Datasource.Url = "some-url"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}
	if !errors.Is(err, ErrUnsupportedDatasourceDriver) {
		t.Errorf("expected ErrUnsupportedDatasourceDriver, got: %v", err)
	}
	if !strings.Contains(err.Error(), "oracle") {
		t.Errorf("error should mention the invalid driver name: %v", err)
	}
}

func TestConfig_Validate_EmptyURL(t *testing.T) {
	cfg := Config{}
	cfg.Datasource.Driver = "mysql"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
	if !errors.Is(err, ErrDatasourceURLRequired) {
		t.Errorf("expected ErrDatasourceURLRequired, got: %v", err)
	}
}

func TestConfig_Validate_NegativeRunLimit(t *testing.T) {
	cfg := Config{}
	cfg.Datasource.Driver = "sqlite"
	cfg.Datasource.Url = "./test.db"
	cfg.Server.RunLimit = -1
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for negative runLimit")
	}
	if !errors.Is(err, ErrInvalidRunLimit) {
		t.Errorf("expected ErrInvalidRunLimit, got: %v", err)
	}
}

func TestConfig_Validate_ZeroRunLimitIsValid(t *testing.T) {
	cfg := Config{}
	cfg.Datasource.Driver = "sqlite"
	cfg.Datasource.Url = "./test.db"
	cfg.Server.RunLimit = 0
	if err := cfg.Validate(); err != nil {
		t.Errorf("zero runLimit should be valid, got error: %v", err)
	}
}

func TestConfig_Validate_WithServerConfig(t *testing.T) {
	cfg := Config{}
	cfg.Datasource.Driver = "mysql"
	cfg.Datasource.Url = "root:pass@tcp(localhost:3306)/gokins"
	cfg.Server.Host = "https://ci.example.com"
	cfg.Server.Secret = "my-secret"
	cfg.Server.LoginKey = "login-key"
	cfg.Server.RunLimit = 10
	cfg.Server.Shells = []string{"shell@ssh"}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config with server settings, got error: %v", err)
	}
}

func TestDatasourceDriverConstants(t *testing.T) {
	// Ensure deprecated constants still match the new ones
	if DATASOURCE_DRIVER_MYSQL != DatasourceDriverMySQL {
		t.Error("DATASOURCE_DRIVER_MYSQL should equal DatasourceDriverMySQL")
	}
	if DATASOURCE_DRIVER_POSTGRES != DatasourceDriverPostgres {
		t.Error("DATASOURCE_DRIVER_POSTGRES should equal DatasourceDriverPostgres")
	}
	if DATASOURCE_DRIVER_SQLITE != DatasourceDriverSQLite {
		t.Error("DATASOURCE_DRIVER_SQLITE should equal DatasourceDriverSQLite")
	}
}
