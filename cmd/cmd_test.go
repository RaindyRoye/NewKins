package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gokins/gokins/comm"
)

func TestRootCommand_HasSubcommands(t *testing.T) {
	cmds := rootCmd.Commands()
	names := make(map[string]bool)
	for _, c := range cmds {
		names[c.Use] = true
	}

	expected := []string{"run", "daemon", "version"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected subcommand %q not found; got: %v", name, names)
		}
	}
}

func TestRootCommand_UseAndDescription(t *testing.T) {
	if rootCmd.Use != "gokins" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "gokins")
	}
	if rootCmd.Short == "" {
		t.Error("rootCmd.Short should not be empty")
	}
	if rootCmd.Long == "" {
		t.Error("rootCmd.Long should not be empty")
	}
}

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, comm.Version) {
		t.Errorf("version output %q should contain %q", output, comm.Version)
	}
}

func TestGetArgs_Defaults(t *testing.T) {
	// Save and restore global flag state
	origWebHost := webHost
	origWorkPath := workPath
	origNotUpPass := notUpPass
	defer func() {
		webHost = origWebHost
		workPath = origWorkPath
		notUpPass = origNotUpPass
	}()

	webHost = ":8030"
	workPath = ""
	notUpPass = false

	args := getArgs()
	if args[0] != "run" {
		t.Errorf("first arg should be 'run', got %q", args[0])
	}
	// Should contain --web flag
	found := false
	for _, a := range args {
		if a == "--web" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected --web flag in args: %v", args)
	}
}

func TestGetArgs_WithWorkPath(t *testing.T) {
	origWebHost := webHost
	origWorkPath := workPath
	origNotUpPass := notUpPass
	defer func() {
		webHost = origWebHost
		workPath = origWorkPath
		notUpPass = origNotUpPass
	}()

	webHost = ":9090"
	workPath = "/tmp/test"
	notUpPass = true

	args := getArgs()

	// Check workdir flag is present
	foundWorkdir := false
	foundNupass := false
	for i, a := range args {
		if a == "--workdir" && i+1 < len(args) && args[i+1] == "/tmp/test" {
			foundWorkdir = true
		}
		if a == "--nupass" {
			foundNupass = true
		}
	}
	if !foundWorkdir {
		t.Errorf("expected --workdir /tmp/test in args: %v", args)
	}
	if !foundNupass {
		t.Errorf("expected --nupass in args: %v", args)
	}
}

func TestGetArgs_EmptyWebHost(t *testing.T) {
	origWebHost := webHost
	origWorkPath := workPath
	origNotUpPass := notUpPass
	defer func() {
		webHost = origWebHost
		workPath = origWorkPath
		notUpPass = origNotUpPass
	}()

	webHost = ""
	workPath = ""
	notUpPass = false

	args := getArgs()
	// When webHost is empty, --web should not be added
	for _, a := range args {
		if a == "--web" {
			t.Errorf("--web should not be present when webHost is empty: %v", args)
		}
	}
}

func TestPersistentFlags_Registered(t *testing.T) {
	flags := rootCmd.PersistentFlags()

	tests := []struct {
		name         string
		defaultValue string
	}{
		{"web", ":8030"},
		{"workdir", ""},
		{"nupass", "false"},
	}

	for _, tt := range tests {
		f := flags.Lookup(tt.name)
		if f == nil {
			t.Errorf("persistent flag %q not found", tt.name)
			continue
		}
		if f.DefValue != tt.defaultValue {
			t.Errorf("flag %q default = %q, want %q", tt.name, f.DefValue, tt.defaultValue)
		}
	}
}

func TestRunCommand_DebugFlag(t *testing.T) {
	f := runCmd.Flags().Lookup("debug")
	if f == nil {
		t.Fatal("run command should have --debug flag")
	}
	if f.DefValue != "false" {
		t.Errorf("debug flag default = %q, want %q", f.DefValue, "false")
	}
}

func TestHelpOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("help command failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "run") {
		t.Error("help output should mention 'run' command")
	}
	if !strings.Contains(output, "daemon") {
		t.Error("help output should mention 'daemon' command")
	}
	if !strings.Contains(output, "version") {
		t.Error("help output should mention 'version' command")
	}
}

func TestRootHelp_ContainsQuickStart(t *testing.T) {
	if !strings.Contains(rootCmd.Long, "Quick start") {
		t.Error("rootCmd.Long should contain Quick start section")
	}
	if !strings.Contains(rootCmd.Long, "Configuration") {
		t.Error("rootCmd.Long should contain Configuration section")
	}
}

func TestRunCommand_HasExamples(t *testing.T) {
	if runCmd.Example == "" {
		t.Error("runCmd should have Example text")
	}
	if !strings.Contains(runCmd.Example, "--web") {
		t.Error("runCmd examples should include --web flag usage")
	}
	if !strings.Contains(runCmd.Example, "--debug") {
		t.Error("runCmd examples should include --debug flag usage")
	}
}

func TestDaemonCommand_HasExamples(t *testing.T) {
	if daemonCmd.Example == "" {
		t.Error("daemonCmd should have Example text")
	}
}

func TestFlagDescriptions_NotEmpty(t *testing.T) {
	tests := []struct {
		flag string
	}{
		{"web"},
		{"workdir"},
		{"nupass"},
	}
	flags := rootCmd.PersistentFlags()
	for _, tt := range tests {
		f := flags.Lookup(tt.flag)
		if f == nil {
			t.Errorf("flag %q not found", tt.flag)
			continue
		}
		if f.Usage == "" {
			t.Errorf("flag %q should have a description", tt.flag)
		}
	}
}
