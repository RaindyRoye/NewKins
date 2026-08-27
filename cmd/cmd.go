package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gokins/core"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/server"
	"github.com/gokins/gokins/util"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	webHost   string
	workPath  string
	notUpPass bool
	debugLog  bool
)

var rootCmd = &cobra.Command{
	Use:   "gokins",
	Short: "A lightweight CI/CD pipeline tool",
	Long: `Gokins (NewKins) is a lightweight, self-hosted CI/CD pipeline tool built with Go.

It supports Git webhooks from GitHub, GitLab, Gitea, and Gitee, and can run
build pipelines defined in YAML configuration files. Pipelines can be triggered
by code pushes, pull requests, or scheduled timers.

Quick start:
  gokins run                          # Start the server in the foreground
  gokins run --web :8030 -w /data     # Listen on port 8030, workdir /data
  gokins daemon                       # Start as a background process
  gokins version                      # Show version information

Configuration:
  Copy config.example.yml to config.yml and edit it for your environment.
  The server reads config.yml from the working directory (--workdir).`,
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the Gokins server in the foreground",
	Long: `Start the Gokins CI/CD server, including the web UI, API, and build engine.

The server will listen on the address specified by --web (default :8030) and
use the working directory for configuration and data storage.

Press Ctrl+C to gracefully shut down the server.`,
	Example: `  gokins run                           # Start with defaults
  gokins run --web :8080               # Listen on port 8080
  gokins run -w /opt/gokins            # Use custom work directory
  gokins run --debug                   # Enable debug logging
  gokins run --nupass                  # Disable password updates`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProcess()
	},
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start the Gokins server as a background process",
	Long: `Start the Gokins server as a detached background process.

This spawns a new child process with the same flags and returns immediately.
Use 'gokins run' for foreground mode with better log visibility.`,
	Example: `  gokins daemon                        # Start in background
  gokins daemon --web :9090            # Custom port in background`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemon()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the Gokins version, build time, and git commit",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("gokins version: %s\n", comm.Version)
		if comm.BuildTime != "unknown" {
			cmd.Printf("  build time:   %s\n", comm.BuildTime)
		}
		if comm.GitCommit != "unknown" {
			cmd.Printf("  git commit:   %s\n", comm.GitCommit)
		}
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long: `Commands for managing Gokins configuration files.

Use 'config validate' to check your app.yml before starting the server.
Use 'config show' to display the parsed configuration (sensitive values are redacted).`,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate a configuration file",
	Long: `Parse and validate the specified configuration file (or the default app.yml
in the working directory) without starting the server.

This is useful for CI/CD pipelines or pre-flight checks before deployment.`,
	Example: `  gokins config validate                    # Validate default app.yml
  gokins config validate /etc/gokins.yml    # Validate specific file
  gokins config validate -w /opt/gokins     # Validate in custom workdir`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return validateConfig(cmd, args)
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show [file]",
	Short: "Show the parsed configuration (sensitive values redacted)",
	Long: `Parse the configuration file and display the effective settings.
Sensitive values like database URLs and secrets are redacted for safety.`,
	Example: `  gokins config show                        # Show default config
  gokins config show -w /opt/gokins         # Show config from custom workdir`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return showConfig(cmd, args)
	},
}

var (
	initDriver string
	initForce  bool
)

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a new configuration file with secure defaults",
	Long: `Create a new app.yml configuration file in the working directory with
randomly generated secrets (loginKey, secret, DownToken).

If a configuration file already exists, the command will refuse to overwrite it
unless --force is specified.

Supported database drivers: sqlite (default), mysql, postgres.`,
	Example: `  gokins config init                        # SQLite config in current dir
  gokins config init --driver mysql         # MySQL config
  gokins config init --driver postgres      # PostgreSQL config
  gokins config init -w /opt/gokins         # Config in custom workdir
  gokins config init --force                # Overwrite existing config`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return initConfig(cmd)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&webHost, "web", ":8030", "HTTP listen address (e.g. :8030, 0.0.0.0:8080)")
	rootCmd.PersistentFlags().StringVarP(&workPath, "workdir", "w", "", "working directory for config and data (default: current dir)")
	rootCmd.PersistentFlags().BoolVar(&notUpPass, "nupass", false, "disable password updates via API")

	runCmd.Flags().BoolVar(&debugLog, "debug", false, "enable debug-level logging")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)

	configInitCmd.Flags().StringVar(&initDriver, "driver", "sqlite", "database driver: sqlite, mysql, or postgres")
	configInitCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing configuration file")

	rootCmd.SetVersionTemplate("gokins version: {{.Version}}\n")
}

func Run() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getArgs() []string {
	args := []string{"run"}
	if webHost != "" {
		args = append(args, "--web", webHost)
	}
	if workPath != "" {
		args = append(args, "--workdir", workPath)
	}
	if notUpPass {
		args = append(args, "--nupass")
	}
	return args
}

func runDaemon() error {
	args := getArgs()
	fullpth, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	fmt.Println("starting background process...")
	cmd := exec.Command(fullpth, args...) //nolint:gosec,noctx // G204: CLI tool needs to execute user commands; noctx: daemon doesn't inherit parent context
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start daemon process: %w", err)
	}
	fmt.Printf("daemon started (pid: %d)\n", cmd.Process.Pid)
	return nil
}

func runProcess() error {
	// Apply global flags to comm package
	comm.WebHost = webHost
	comm.WorkPath = workPath
	comm.NotUpPass = notUpPass
	core.Debug = debugLog

	csig := make(chan os.Signal, 1)
	signal.Notify(csig, os.Interrupt, syscall.SIGTERM, syscall.SIGALRM)
	go func() {
		defer util.RecoverLog("signal handler")
		s := <-csig
		hbtp.Debugf("get signal(%d):%s", s, s.String())
		comm.Cancel()
	}()
	if core.Debug {
		hbtp.Debug = true
	}
	return server.Run()
}

// resolveConfigPath determines the config file path from args or defaults.
func resolveConfigPath(args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	wp := workPath
	if wp == "" {
		wp = "."
	}
	// Try app.yml first, then app.yaml
	for _, name := range []string{"app.yml", "app.yaml"} {
		pth := filepath.Join(wp, name)
		if _, err := os.Stat(pth); err == nil {
			return pth, nil
		}
	}
	return "", fmt.Errorf("no configuration file found in %s (tried app.yml, app.yaml)", wp)
}

// loadConfigFile reads and parses a config file, returning the parsed Config.
func loadConfigFile(pth string) (*comm.Config, error) {
	bts, err := os.ReadFile(filepath.Clean(pth))
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", pth, err)
	}
	cfg := &comm.Config{}
	if err := yaml.Unmarshal(bts, cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml %s: %w", pth, err)
	}
	return cfg, nil
}

// validateConfig implements the 'config validate' subcommand.
func validateConfig(cmd *cobra.Command, args []string) error {
	pth, err := resolveConfigPath(args)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}
	cfg, err := loadConfigFile(pth)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		cmd.Printf("❌ Configuration invalid: %v\n", err)
		cmd.Printf("   File: %s\n", pth)
		return fmt.Errorf("validate config: %w", err)
	}
	cmd.Printf("✅ Configuration valid: %s\n", pth)
	cmd.Printf("   Driver: %s\n", cfg.Datasource.Driver)
	if cfg.Server.Host != "" {
		cmd.Printf("   Server: %s\n", cfg.Server.Host)
	}
	if cfg.Server.RunLimit > 0 {
		cmd.Printf("   Run limit: %d\n", cfg.Server.RunLimit)
	}
	return nil
}

// redactURL masks the password portion of a database URL for safe display.
func redactURL(url string) string {
	if url == "" {
		return "(empty)"
	}
	// For URLs with @, redact the credentials part
	if idx := strings.LastIndex(url, "@"); idx > 0 {
		return "***@" + url[idx+1:]
	}
	// For DSN-style strings (user:pass@protocol(addr)/db), redact between : and @
	if strings.Contains(url, ":") && strings.Contains(url, "(") {
		return "***" + url[strings.Index(url, "("):]
	}
	// For simple file paths (like sqlite), show as-is
	if len(url) < 3 {
		return "***"
	}
	return url[:2] + "***"
}

// showConfig implements the 'config show' subcommand.
func showConfig(cmd *cobra.Command, args []string) error {
	pth, err := resolveConfigPath(args)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}
	cfg, err := loadConfigFile(pth)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	cmd.Printf("Configuration: %s\n", pth)
	cmd.Printf("───────────────────────────────────\n")
	cmd.Printf("Datasource:\n")
	cmd.Printf("  Driver:  %s\n", cfg.Datasource.Driver)
	cmd.Printf("  URL:     %s\n", redactURL(cfg.Datasource.Url))
	cmd.Printf("Server:\n")
	if cfg.Server.Host != "" {
		cmd.Printf("  Host:      %s\n", cfg.Server.Host)
	}
	cmd.Printf("  RunLimit:  %d\n", cfg.Server.RunLimit)
	if cfg.Server.HbtpHost != "" {
		cmd.Printf("  HbtpHost:  %s\n", cfg.Server.HbtpHost)
	}
	if cfg.Server.LoginKey != "" {
		cmd.Printf("  LoginKey:  ***\n")
	}
	if cfg.Server.Secret != "" {
		cmd.Printf("  Secret:    ***\n")
	}
	if len(cfg.Server.Shells) > 0 {
		cmd.Printf("  Shells:    %v\n", cfg.Server.Shells)
	}
	if cfg.Server.DownToken != "" {
		cmd.Printf("  DownToken: ***\n")
	}
	validationErr := "valid"
	if err := cfg.Validate(); err != nil {
		validationErr = fmt.Sprintf("INVALID: %v", err)
	}
	cmd.Printf("───────────────────────────────────\n")
	cmd.Printf("Validation: %s\n", validationErr)
	return nil
}

// generateConfigTemplate creates a config file with secure random defaults.
func generateConfigTemplate(driver string) []byte {
	loginKey := utils.RandomString(32)
	secret := utils.RandomString(32)
	downToken := utils.RandomString(32)

	var dbURL string
	switch driver {
	case comm.DatasourceDriverMySQL:
		dbURL = "root:changeme@tcp(localhost:3306)/gokins?charset=utf8mb4&parseTime=true" //nolint:gosec // G101: example config template, not real credentials
	case comm.DatasourceDriverPostgres:
		dbURL = "postgres://gokins:***@localhost:5432/gokins?sslmode=disable" //nolint:gosec // G101: example config template, not real credentials
	default:
		dbURL = "./gokins.db"
		driver = comm.DatasourceDriverSQLite
	}

	content := fmt.Sprintf(`# Gokins Configuration
# Generated by: gokins config init
# Generated at: %s

server:
  # External access URL for the web UI (used in webhook callbacks, emails, etc.)
  host: ""

  # JWT signing key for user authentication tokens (auto-generated).
  loginKey: "%s"

  # Maximum number of concurrent builds. Default: 5
  runLimit: 5

  # HyperByte Transfer Protocol listen address (for runner communication).
  # Leave empty to auto-detect.
  hbtpHost: ""

  # Secret used for runner authentication (auto-generated).
  secret: "%s"

  # Additional shell runner plugins to load.
  # Built-in: "shell@ssh", "gokins@git"
  shells: []

  # Token for downloading artifacts (auto-generated).
  DownToken: "%s"

datasource:
  # Database driver: "sqlite", "mysql", or "postgres"
  driver: "%s"

  # Database connection string.
  url: "%s"
`, time.Now().UTC().Format(time.RFC3339), loginKey, secret, downToken, driver, dbURL)

	return []byte(content)
}

// initConfig implements the 'config init' subcommand.
func initConfig(cmd *cobra.Command) error {
	wp := workPath
	if wp == "" {
		wp = "."
	}

	// Validate driver early
	switch initDriver {
	case comm.DatasourceDriverMySQL, comm.DatasourceDriverPostgres, comm.DatasourceDriverSQLite:
		// valid
	default:
		return fmt.Errorf("unsupported driver %q (must be one of: sqlite, mysql, postgres)", initDriver)
	}

	// Ensure working directory exists
	if err := os.MkdirAll(wp, 0750); err != nil {
		return fmt.Errorf("create working directory %s: %w", wp, err)
	}

	configPath := filepath.Join(wp, "app.yml")

	// Check if file already exists
	if !initForce {
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("configuration file already exists: %s (use --force to overwrite)", configPath)
		}
	}

	content := generateConfigTemplate(initDriver)

	if err := os.WriteFile(configPath, content, 0600); err != nil {
		return fmt.Errorf("write config file %s: %w", configPath, err)
	}

	cmd.Printf("✅ Configuration generated: %s\n", configPath)
	cmd.Printf("   Driver: %s\n", initDriver)
	cmd.Printf("\n")
	cmd.Printf("Next steps:\n")
	cmd.Printf("  1. Edit %s to set your server host and database credentials\n", configPath)
	cmd.Printf("  2. Run: gokins run -w %s\n", wp)
	return nil
}
