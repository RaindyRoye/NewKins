package comm

import (
	"errors"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr error
	}{
		{
			name: "valid config",
			config: &Config{
				Server: struct {
					Host        string   `yaml:"host"`
					LoginKey    string   `yaml:"loginKey"`
					RunLimit    int      `yaml:"runLimit"`
					HbtpHost    string   `yaml:"hbtpHost"`
					Secret      string   `yaml:"secret"`
					Shells      []string `yaml:"shells"`
					DownToken   string   `yaml:"DownToken"`
					EnablePprof bool     `yaml:"enablePprof"`
				}{
					RunLimit: 100,
				},
				Datasource: struct {
					Driver string `yaml:"driver"`
					Url    string `yaml:"url"`
				}{
					Driver: DatasourceDriverSQLite,
					Url:    "./data.db",
				},
			},
			wantErr: nil,
		},
		{
			name: "missing driver",
			config: &Config{
				Datasource: struct {
					Driver string `yaml:"driver"`
					Url    string `yaml:"url"`
				}{
					Url: "./data.db",
				},
			},
			wantErr: ErrConfigDriverRequired,
		},
		{
			name: "unsupported driver",
			config: &Config{
				Datasource: struct {
					Driver string `yaml:"driver"`
					Url    string `yaml:"url"`
				}{
					Driver: "oracle",
					Url:    "oracle://localhost/db",
				},
			},
			wantErr: ErrConfigDriverUnsupported,
		},
		{
			name: "missing url",
			config: &Config{
				Datasource: struct {
					Driver string `yaml:"driver"`
					Url    string `yaml:"url"`
				}{
					Driver: DatasourceDriverMySQL,
				},
			},
			wantErr: ErrConfigURLRequired,
		},
		{
			name: "negative run limit",
			config: &Config{
				Server: struct {
					Host        string   `yaml:"host"`
					LoginKey    string   `yaml:"loginKey"`
					RunLimit    int      `yaml:"runLimit"`
					HbtpHost    string   `yaml:"hbtpHost"`
					Secret      string   `yaml:"secret"`
					Shells      []string `yaml:"shells"`
					DownToken   string   `yaml:"DownToken"`
					EnablePprof bool     `yaml:"enablePprof"`
				}{
					RunLimit: -1,
				},
				Datasource: struct {
					Driver string `yaml:"driver"`
					Url    string `yaml:"url"`
				}{
					Driver: DatasourceDriverPostgres,
					Url:    "postgres://localhost/db",
				},
			},
			wantErr: ErrConfigRunLimitInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() error = %v, wantErr nil", err)
				}
				return
			}
			if err == nil {
				t.Errorf("Validate() error = nil, wantErr %v", tt.wantErr)
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, want errors.Is(%v)", err, tt.wantErr)
			}
		})
	}
}