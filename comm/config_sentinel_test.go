package comm

import (
	"errors"
	"testing"
)

// TestConfigValidateSentinelErrors verifies that Config.Validate() returns
// sentinel errors that can be matched with errors.Is().
func TestConfigValidateSentinelErrors(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr error
	}{
		{
			name: "empty driver returns ErrDriverRequired",
			config: &Config{
				Datasource: struct {
					Driver string `yaml:"driver"`
					Url    string `yaml:"url"`
				}{
					Driver: "",
					Url:    "test.db",
				},
			},
			wantErr: ErrDriverRequired,
		},
		{
			name: "unsupported driver returns ErrDriverUnsupported",
			config: &Config{
				Datasource: struct {
					Driver string `yaml:"driver"`
					Url    string `yaml:"url"`
				}{
					Driver: "oracle",
					Url:    "test.db",
				},
			},
			wantErr: ErrDriverUnsupported,
		},
		{
			name: "empty URL returns ErrURLRequired",
			config: &Config{
				Datasource: struct {
					Driver string `yaml:"driver"`
					Url    string `yaml:"url"`
				}{
					Driver: "sqlite",
					Url:    "",
				},
			},
			wantErr: ErrURLRequired,
		},
		{
			name: "negative RunLimit returns ErrRunLimitInvalid",
			config: &Config{
				Datasource: struct {
					Driver string `yaml:"driver"`
					Url    string `yaml:"url"`
				}{
					Driver: "sqlite",
					Url:    "test.db",
				},
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
			},
			wantErr: ErrRunLimitInvalid,
		},
		{
			name: "valid sqlite config returns nil",
			config: &Config{
				Datasource: struct {
					Driver string `yaml:"driver"`
					Url    string `yaml:"url"`
				}{
					Driver: "sqlite",
					Url:    "test.db",
				},
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
					RunLimit: 0,
				},
			},
			wantErr: nil,
		},
		{
			name: "valid mysql config returns nil",
			config: &Config{
				Datasource: struct {
					Driver string `yaml:"driver"`
					Url    string `yaml:"url"`
				}{
					Driver: "mysql",
					Url:    "user:pass@tcp(localhost:3306)/db",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid postgres config returns nil",
			config: &Config{
				Datasource: struct {
					Driver string `yaml:"driver"`
					Url    string `yaml:"url"`
				}{
					Driver: "postgres",
					Url:    "postgres://user:pass@localhost/db",
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Errorf("Validate() = nil, want error matching %v", tt.wantErr)
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, want error matching %v (errors.Is returned false)", err, tt.wantErr)
			}
		})
	}
}

// TestConfigValidateErrorMessageContent verifies that wrapped errors contain
// contextual information (e.g., unsupported driver name, invalid RunLimit value).
func TestConfigValidateErrorMessageContent(t *testing.T) {
	t.Run("unsupported driver error contains driver name", func(t *testing.T) {
		config := &Config{
			Datasource: struct {
				Driver string `yaml:"driver"`
				Url    string `yaml:"url"`
			}{
				Driver: "oracle",
				Url:    "test.db",
			},
		}
		err := config.Validate()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrDriverUnsupported) {
			t.Errorf("error does not wrap ErrDriverUnsupported: %v", err)
		}
		errStr := err.Error()
		if !containsStr(errStr, "oracle") {
			t.Errorf("error message %q does not contain driver name 'oracle'", errStr)
		}
	})

	t.Run("invalid RunLimit error contains the value", func(t *testing.T) {
		config := &Config{
			Datasource: struct {
				Driver string `yaml:"driver"`
				Url    string `yaml:"url"`
			}{
				Driver: "sqlite",
				Url:    "test.db",
			},
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
				RunLimit: -42,
			},
		}
		err := config.Validate()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrRunLimitInvalid) {
			t.Errorf("error does not wrap ErrRunLimitInvalid: %v", err)
		}
		errStr := err.Error()
		if !containsStr(errStr, "-42") {
			t.Errorf("error message %q does not contain RunLimit value '-42'", errStr)
		}
	})
}

// containsStr checks if s contains substr.
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
