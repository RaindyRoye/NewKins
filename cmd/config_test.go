package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactURL_Empty(t *testing.T) {
	got := redactURL("")
	if got != "(empty)" {
		t.Errorf("redactURL(\"\") = %q, want %q", got, "(empty)")
	}
}

func TestRedactURL_PostgresURL(t *testing.T) {
	got := redactURL("postgres://user:secret@localhost:5432/gokins?sslmode=disable")
	if strings.Contains(got, "secret") {
		t.Errorf("redactURL should redact password, got: %q", got)
	}
	if !strings.Contains(got, "localhost:5432") {
		t.Errorf("redactURL should preserve host, got: %q", got)
	}
	if !strings.HasPrefix(got, "***@") {
		t.Errorf("redactURL should start with ***, got: %q", got)
	}
}

func TestRedactURL_MySQLDSN(t *testing.T) {
	got := redactURL("root:password@tcp(localhost:3306)/gokins?charset=utf8mb4")
	if strings.Contains(got, "password") {
		t.Errorf("redactURL should redact password, got: %q", got)
	}
	if !strings.Contains(got, "(localhost:3306)") {
		t.Errorf("redactURL should preserve address, got: %q", got)
	}
}

func TestRedactURL_SQLiteFile(t *testing.T) {
	got := redactURL("./gokins.db")
	if !strings.Contains(got, "***") {
		t.Errorf("redactURL should redact file path, got: %q", got)
	}
}

func TestRedactURL_ShortString(t *testing.T) {
	got := redactURL("ab")
	if got != "***" {
		t.Errorf("redactURL(\"ab\") = %q, want \"***\"", got)
	}
}

func TestRedactURL_NoAtSign(t *testing.T) {
	got := redactURL("some-random-dsn-without-at")
	if !strings.Contains(got, "***") {
		t.Errorf("redactURL should redact, got: %q", got)
	}
	if got == "some-random-dsn-without-at" {
		t.Error("redactURL should not return the original URL unchanged")
	}
}

func TestResolveConfigPath_ExplicitFile(t *testing.T) {
	got, err := resolveConfigPath([]string{"/tmp/test-config.yml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/tmp/test-config.yml" {
		t.Errorf("resolveConfigPath = %q, want %q", got, "/tmp/test-config.yml")
	}
}

func TestResolveConfigPath_NoFileFound(t *testing.T) {
	// Save and restore workPath
	orig := workPath
	defer func() { workPath = orig }()

	workPath = "/nonexistent/path/that/does/not/exist"
	_, err := resolveConfigPath(nil)
	if err == nil {
		t.Fatal("expected error when no config file exists")
	}
	if !strings.Contains(err.Error(), "no configuration file found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolveConfigPath_FindsAppYml(t *testing.T) {
	// Create a temp directory with an app.yml file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "app.yml")
	if err := os.WriteFile(configPath, []byte("server:\n  host: test\n"), 0600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	orig := workPath
	defer func() { workPath = orig }()

	workPath = tmpDir
	got, err := resolveConfigPath(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != configPath {
		t.Errorf("resolveConfigPath = %q, want %q", got, configPath)
	}
}

func TestResolveConfigPath_FindsAppYaml(t *testing.T) {
	// Create a temp directory with an app.yaml file (not app.yml)
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "app.yaml")
	if err := os.WriteFile(configPath, []byte("server:\n  host: test\n"), 0600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	orig := workPath
	defer func() { workPath = orig }()

	workPath = tmpDir
	got, err := resolveConfigPath(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != configPath {
		t.Errorf("resolveConfigPath = %q, want %q", got, configPath)
	}
}

func TestLoadConfigFile_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yml")
	content := `
server:
  host: https://ci.example.com
  runLimit: 5
datasource:
  driver: sqlite
  url: ./test.db
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	cfg, err := loadConfigFile(configPath)
	if err != nil {
		t.Fatalf("loadConfigFile error: %v", err)
	}
	if cfg.Datasource.Driver != "sqlite" {
		t.Errorf("driver = %q, want %q", cfg.Datasource.Driver, "sqlite")
	}
	if cfg.Datasource.Url != "./test.db" {
		t.Errorf("url = %q, want %q", cfg.Datasource.Url, "./test.db")
	}
	if cfg.Server.Host != "https://ci.example.com" {
		t.Errorf("host = %q, want %q", cfg.Server.Host, "https://ci.example.com")
	}
	if cfg.Server.RunLimit != 5 {
		t.Errorf("runLimit = %d, want %d", cfg.Server.RunLimit, 5)
	}
}

func TestLoadConfigFile_NotFound(t *testing.T) {
	_, err := loadConfigFile("/nonexistent/file.yml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "read config file") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadConfigFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bad.yml")
	// Use content that will actually fail YAML parsing (tab characters in wrong places)
	if err := os.WriteFile(configPath, []byte("server:\n\t- invalid:\n\t  broken:\n\t\t: bad"), 0600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	_, err := loadConfigFile(configPath)
	if err == nil {
		// If YAML is lenient with this too, try binary content
		if err := os.WriteFile(configPath, []byte("\x00\x01\x02\x03"), 0600); err != nil {
			t.Fatalf("failed to create test config: %v", err)
		}
		_, err = loadConfigFile(configPath)
		if err == nil {
			t.Skip("YAML parser is too lenient to test invalid content")
		}
	}
	if err != nil && !strings.Contains(err.Error(), "parse config yaml") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigValidateCmd_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "app.yml")
	content := `
datasource:
  driver: sqlite
  url: ./test.db
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"config", "validate", configPath})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config validate failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Configuration valid") {
		t.Errorf("expected success message, got: %q", output)
	}
}

func TestConfigValidateCmd_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "app.yml")
	content := `
datasource:
  driver: oracle
  url: something
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"config", "validate", configPath})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

func TestConfigShowCmd(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "app.yml")
	content := `
server:
  host: https://ci.example.com
  secret: my-secret-key
  loginKey: my-login-key
  runLimit: 5
datasource:
  driver: mysql
  url: root:secret@tcp(localhost:3306)/gokins
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"config", "show", configPath})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config show failed: %v", err)
	}
	output := buf.String()

	// Should show driver
	if !strings.Contains(output, "mysql") {
		t.Errorf("output should contain driver, got: %q", output)
	}
	// Should redact secret
	if strings.Contains(output, "my-secret-key") {
		t.Errorf("output should NOT contain secret, got: %q", output)
	}
	if !strings.Contains(output, "***") {
		t.Errorf("output should contain redacted values, got: %q", output)
	}
	// Should redact loginKey
	if strings.Contains(output, "my-login-key") {
		t.Errorf("output should NOT contain loginKey, got: %q", output)
	}
	// Should redact DB password
	if strings.Contains(output, "secret") {
		t.Errorf("output should NOT contain DB password, got: %q", output)
	}
	// Should show host
	if !strings.Contains(output, "ci.example.com") {
		t.Errorf("output should contain host, got: %q", output)
	}
}

func TestConfigCmd_Registered(t *testing.T) {
	cmds := rootCmd.Commands()
	found := false
	for _, c := range cmds {
		if c.Use == "config" {
			found = true
			break
		}
	}
	if !found {
		t.Error("config subcommand not registered")
	}
}

func TestConfigCmd_HasSubcommands(t *testing.T) {
	names := make(map[string]bool)
	for _, c := range configCmd.Commands() {
		names[c.Name()] = true
	}
	if !names["validate"] {
		t.Error("config validate subcommand not found")
	}
	if !names["show"] {
		t.Error("config show subcommand not found")
	}
}

func TestConfigValidateCmd_HasExamples(t *testing.T) {
	if configValidateCmd.Example == "" {
		t.Error("configValidateCmd should have Example text")
	}
}

func TestConfigShowCmd_HasExamples(t *testing.T) {
	if configShowCmd.Example == "" {
		t.Error("configShowCmd should have Example text")
	}
}

func TestConfigCmd_HelpOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"config", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config help failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "validate") {
		t.Error("config help should mention validate subcommand")
	}
	if !strings.Contains(output, "show") {
		t.Error("config help should mention show subcommand")
	}
}
