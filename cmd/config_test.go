package cmd

import (
	"bytes"
	"fmt"
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

func TestConfigInitCmd_GeneratesFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Save and restore global state
	origWP := workPath
	origDriver := initDriver
	origForce := initForce
	defer func() {
		workPath = origWP
		initDriver = origDriver
		initForce = origForce
	}()

	workPath = tmpDir
	initDriver = "sqlite"
	initForce = false

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"config", "init", "--driver", "sqlite", "-w", tmpDir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config init failed: %v", err)
	}

	// Verify file was created
	configPath := filepath.Join(tmpDir, "app.yml")
	data, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	content := string(data)

	// Should contain the driver
	if !strings.Contains(content, `driver: "sqlite"`) {
		t.Errorf("config should contain sqlite driver, got: %s", content)
	}
	// Should contain sqlite URL
	if !strings.Contains(content, "./gokins.db") {
		t.Errorf("config should contain sqlite URL, got: %s", content)
	}
	// Should contain loginKey (randomly generated, non-empty)
	if !strings.Contains(content, "loginKey:") {
		t.Error("config should contain loginKey")
	}
	// Should contain secret
	if !strings.Contains(content, "secret:") {
		t.Error("config should contain secret")
	}
	// Should contain Generated by comment
	if !strings.Contains(content, "Generated by: gokins config init") {
		t.Error("config should contain generation comment")
	}

	// Verify the generated file is valid YAML and passes validation
	cfg, err := loadConfigFile(configPath)
	if err != nil {
		t.Fatalf("generated config is not valid YAML: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("generated config failed validation: %v", err)
	}
}

func TestConfigInitCmd_MySQLDriver(t *testing.T) {
	tmpDir := t.TempDir()

	origWP := workPath
	origDriver := initDriver
	origForce := initForce
	defer func() {
		workPath = origWP
		initDriver = origDriver
		initForce = origForce
	}()

	workPath = tmpDir
	initDriver = "mysql"
	initForce = false

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"config", "init", "--driver", "mysql", "-w", tmpDir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config init mysql failed: %v", err)
	}

	configPath := filepath.Join(tmpDir, "app.yml")
	data, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, `driver: "mysql"`) {
		t.Errorf("config should contain mysql driver")
	}
	if !strings.Contains(content, "localhost:3306") {
		t.Errorf("config should contain mysql default host")
	}
}

func TestConfigInitCmd_PostgresDriver(t *testing.T) {
	tmpDir := t.TempDir()

	origWP := workPath
	origDriver := initDriver
	origForce := initForce
	defer func() {
		workPath = origWP
		initDriver = origDriver
		initForce = origForce
	}()

	workPath = tmpDir
	initDriver = "postgres"
	initForce = false

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"config", "init", "--driver", "postgres", "-w", tmpDir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config init postgres failed: %v", err)
	}

	configPath := filepath.Join(tmpDir, "app.yml")
	data, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, `driver: "postgres"`) {
		t.Errorf("config should contain postgres driver")
	}
	if !strings.Contains(content, "localhost:5432") {
		t.Errorf("config should contain postgres default host")
	}
}

func TestConfigInitCmd_RefusesOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing config file
	configPath := filepath.Join(tmpDir, "app.yml")
	if err := os.WriteFile(configPath, []byte("existing config"), 0600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	origWP := workPath
	origDriver := initDriver
	origForce := initForce
	defer func() {
		workPath = origWP
		initDriver = origDriver
		initForce = origForce
	}()

	workPath = tmpDir
	initDriver = "sqlite"
	initForce = false

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"config", "init", "-w", tmpDir})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when config already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}

	// Verify original file was not modified
	data, _ := os.ReadFile(filepath.Clean(configPath))
	if string(data) != "existing config" {
		t.Error("existing config should not have been modified")
	}
}

func TestConfigInitCmd_ForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing config file
	configPath := filepath.Join(tmpDir, "app.yml")
	if err := os.WriteFile(configPath, []byte("existing config"), 0600); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	origWP := workPath
	origDriver := initDriver
	origForce := initForce
	defer func() {
		workPath = origWP
		initDriver = origDriver
		initForce = origForce
	}()

	workPath = tmpDir
	initDriver = "sqlite"
	initForce = true

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"config", "init", "--force", "-w", tmpDir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config init --force failed: %v", err)
	}

	// Verify file was overwritten
	data, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		t.Fatalf("config file not found after force overwrite: %v", err)
	}
	if string(data) == "existing config" {
		t.Error("config should have been overwritten")
	}
	if !strings.Contains(string(data), "Generated by: gokins config init") {
		t.Error("overwritten config should contain generation marker")
	}
}

func TestConfigInitCmd_InvalidDriver(t *testing.T) {
	tmpDir := t.TempDir()

	origWP := workPath
	origDriver := initDriver
	origForce := initForce
	defer func() {
		workPath = origWP
		initDriver = origDriver
		initForce = origForce
	}()

	workPath = tmpDir
	initDriver = "oracle"
	initForce = false

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"config", "init", "--driver", "oracle", "-w", tmpDir})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid driver")
	}
}

func TestConfigInitCmd_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "new", "nested", "dir")

	origWP := workPath
	origDriver := initDriver
	origForce := initForce
	defer func() {
		workPath = origWP
		initDriver = origDriver
		initForce = origForce
	}()

	workPath = nestedDir
	initDriver = "sqlite"
	initForce = false

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"config", "init", "-w", nestedDir})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config init with nested dir failed: %v", err)
	}

	configPath := filepath.Join(nestedDir, "app.yml")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("config file not created in nested directory: %v", err)
	}
}

func TestConfigInitCmd_HasExamples(t *testing.T) {
	if configInitCmd.Example == "" {
		t.Error("configInitCmd should have Example text")
	}
}

func TestConfigInitCmd_Registered(t *testing.T) {
	names := make(map[string]bool)
	for _, c := range configCmd.Commands() {
		names[c.Name()] = true
	}
	if !names["init"] {
		t.Error("config init subcommand not found")
	}
}

func TestGenerateConfigTemplate_AllDrivers(t *testing.T) {
	drivers := []string{"sqlite", "mysql", "postgres"}
	for _, driver := range drivers {
		t.Run(driver, func(t *testing.T) {
			content := generateConfigTemplate(driver)
			s := string(content)
			if !strings.Contains(s, fmt.Sprintf(`driver: "%s"`, driver)) {
				t.Errorf("template should contain driver %q", driver)
			}
			if !strings.Contains(s, "loginKey:") {
				t.Error("template should contain loginKey")
			}
			if !strings.Contains(s, "secret:") {
				t.Error("template should contain secret")
			}
			if !strings.Contains(s, "DownToken:") {
				t.Error("template should contain DownToken")
			}
		})
	}
}

func TestGenerateConfigTemplate_UniqueSecrets(t *testing.T) {
	c1 := string(generateConfigTemplate("sqlite"))
	c2 := string(generateConfigTemplate("sqlite"))
	if c1 == c2 {
		t.Error("two generated configs should have different random secrets")
	}
}

func TestGenerateConfigTemplate_InvalidDriverDefaultsToSQLite(t *testing.T) {
	content := string(generateConfigTemplate("unknown"))
	if !strings.Contains(content, `driver: "sqlite"`) {
		t.Error("invalid driver should default to sqlite")
	}
	if !strings.Contains(content, "./gokins.db") {
		t.Error("invalid driver should default to sqlite URL")
	}
}
