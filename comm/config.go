package comm

type Config struct {
	Server struct {
		Host      string   `yaml:"host"` // 外网访问地址
		LoginKey  string   `yaml:"loginKey"`
		RunLimit  int      `yaml:"runLimit"`
		HbtpHost  string   `yaml:"hbtpHost"`
		Secret    string   `yaml:"secret"`
		Shells    []string `yaml:"shells"`
		DownToken string   `yaml:"DownToken"`
	} `yaml:"server"`
	Datasource struct {
		Driver string `yaml:"driver"`
		Url    string `yaml:"url"`
	} `yaml:"datasource"`
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
