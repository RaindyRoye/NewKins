package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/gokins/core"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/server"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"github.com/spf13/cobra"
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
	Short: "Print the Gokins version and exit",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("gokins version:" + comm.Version)
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
	cmd := exec.Command(fullpth, args...) //nolint:gosec // G204: intentional subprocess for daemon mode
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
		s := <-csig
		hbtp.Debugf("get signal(%d):%s", s, s.String())
		comm.Cancel()
	}()
	if core.Debug {
		hbtp.Debug = true
	}
	return server.Run()
}
