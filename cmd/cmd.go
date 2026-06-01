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
	Short: "A golang workflow application.",
	Long:  `Gokins is a lightweight CI/CD pipeline tool built with Go.`,
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run process",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProcess()
	},
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run process background",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemon()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("gokins version:" + comm.Version)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&webHost, "web", ":8030", "gokins web host")
	rootCmd.PersistentFlags().StringVarP(&workPath, "workdir", "w", "", "gokins work path")
	rootCmd.PersistentFlags().BoolVar(&notUpPass, "nupass", false, "can't update password")

	runCmd.Flags().BoolVar(&debugLog, "debug", false, "debug log show")

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
	cmd := exec.Command(fullpth, args...)
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
	signal.Notify(csig, os.Interrupt, syscall.SIGALRM)
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
