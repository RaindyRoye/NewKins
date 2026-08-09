package comm

import (
	"errors"
	"fmt"
)

// Sentinel errors for configuration validation.
var (
	// ErrDatasourceDriverRequired is returned when the datasource driver is missing.
	ErrDatasourceDriverRequired = errors.New("datasource.driver is required")

	// ErrUnsupportedDatasourceDriver is returned when an unsupported driver is specified.
	ErrUnsupportedDatasourceDriver = errors.New("unsupported datasource driver")

	// ErrDatasourceURLRequired is returned when the datasource URL is missing.
	ErrDatasourceURLRequired = errors.New("datasource.url is required")

	// ErrInvalidRunLimit is returned when the run limit is negative.
	ErrInvalidRunLimit = errors.New("server.runLimit must be non-negative")
)

// Config holds the application configuration loaded from app.yml / app.yaml.
type Config struct {
	Server struct {
		Host        string   `yaml:"host"` // 外网访问地址
		LoginKey    string   `yaml:"loginKey"`
		RunLimit    int      `yaml:"runLimit"`
		HbtpHost    string   `yaml:"hbtpHost"`
		Secret      string   `yaml:"secret"`
		Shells      []string `yaml:"shells"`
		DownToken   string   `yaml:"DownToken"`
		EnablePprof bool     `yaml:"enablePprof"` // 启用性能分析端点（生产环境可通过此配置开启）
	} `yaml:"server"`
	Datasource struct {
		Driver string `yaml:"driver"`
		Url    string `yaml:"url"`
	} `yaml:"datasource"`
}

// Validate checks the configuration for common misconfiguration issues.
// It returns an error describing the first problem found, or nil if valid.
func (c *Config) Validate() error {
	if c.Datasource.Driver == "" {
		return fmt.Errorf("config validation: %w", ErrDatasourceDriverRequired)
	}
	switch c.Datasource.Driver {
	case DatasourceDriverMySQL, DatasourceDriverPostgres, DatasourceDriverSQLite:
		// valid
	default:
		return fmt.Errorf("config validation: %w: %q (must be one of: mysql, postgres, sqlite)", ErrUnsupportedDatasourceDriver, c.Datasource.Driver)
	}
	if c.Datasource.Url == "" {
		return fmt.Errorf("config validation: %w", ErrDatasourceURLRequired)
	}
	if c.Server.RunLimit < 0 {
		return fmt.Errorf("config validation: %w: got %d", ErrInvalidRunLimit, c.Server.RunLimit)
	}
	return nil
}

// Idiomatic Go constant names for datasource drivers.
const (
	DatasourceDriverMySQL    = "mysql"
	DatasourceDriverPostgres = "postgres"
	DatasourceDriverSQLite   = "sqlite"
)

// Deprecated: Use DatasourceDriverMySQL instead.
const DATASOURCE_DRIVER_MYSQL = DatasourceDriverMySQL

// Deprecated: Use DatasourceDriverPostgres instead.
const DATASOURCE_DRIVER_POSTGRES = DatasourceDriverPostgres

// Deprecated: Use DatasourceDriverSQLite instead.
const DATASOURCE_DRIVER_SQLITE = DatasourceDriverSQLite
