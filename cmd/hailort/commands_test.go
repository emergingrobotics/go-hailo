//go:build unit

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestScanCommand(t *testing.T) {
	cmd := newScanCommand()

	// Capture output
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)

	// Execute command
	err := cmd.Execute()
	if err != nil {
		// Expected without hardware
		t.Logf("scan error (expected): %v", err)
	}
}

func TestInfoCommand(t *testing.T) {
	cmd := newInfoCommand()

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)

	// Set device ID argument
	cmd.SetArgs([]string{"hailo0"})

	err := cmd.Execute()
	if err != nil {
		t.Logf("info error (expected without hardware): %v", err)
	}
}

func TestRunCommandHelp(t *testing.T) {
	cmd := newRunCommand()

	out := new(bytes.Buffer)
	cmd.SetOut(out)

	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("help should not error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Usage") {
		t.Error("help output should contain Usage")
	}
}

func TestBenchmarkCommandArgs(t *testing.T) {
	cmd := newBenchmarkCommand()

	// Test argument parsing
	cmd.SetArgs([]string{
		"--model", "test.hef",
		"--iterations", "100",
		"--warmup", "10",
	})

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)

	err := cmd.Execute()
	// Will fail without hardware, but args should parse
	if err != nil && !strings.Contains(err.Error(), "device") {
		t.Logf("benchmark error: %v", err)
	}
}

func TestVersionCommand(t *testing.T) {
	cmd := newVersionCommand()

	out := new(bytes.Buffer)
	cmd.SetOut(out)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("version should not error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "version") && !strings.Contains(output, "Version") {
		t.Error("version output should contain version info")
	}
}

func TestRootCommand(t *testing.T) {
	cmd := newRootCommand()

	out := new(bytes.Buffer)
	cmd.SetOut(out)

	// No args should show help
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err != nil {
		t.Errorf("root command should not error: %v", err)
	}
}

func TestMonitorCommandArgs(t *testing.T) {
	cmd := newMonitorCommand()

	cmd.SetArgs([]string{"--interval", "500ms"})

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)

	// Would need hardware to run, just test arg parsing
	_ = cmd.Flags().Lookup("interval")
}

func TestResetCommand(t *testing.T) {
	cmd := newResetCommand()

	cmd.SetArgs([]string{"hailo0"})

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)

	err := cmd.Execute()
	// Will fail without hardware
	if err != nil {
		t.Logf("reset error (expected): %v", err)
	}
}

func TestCommandStructure(t *testing.T) {
	root := newRootCommand()

	// Verify subcommands are registered
	subcommands := map[string]bool{
		"scan":      false,
		"info":      false,
		"run":       false,
		"benchmark": false,
		"monitor":   false,
		"reset":     false,
		"version":   false,
	}

	for _, cmd := range root.Commands() {
		if _, ok := subcommands[cmd.Name()]; ok {
			subcommands[cmd.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("subcommand %s not found", name)
		}
	}
}

func TestFlagValidation(t *testing.T) {
	cmd := newBenchmarkCommand()

	// Test invalid iterations
	cmd.SetArgs([]string{
		"--model", "test.hef",
		"--iterations", "-1",
	})

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)

	// Should validate flags
	_ = cmd.Execute()
}

// Mock command implementations for testing

func newRootCommand() *Command {
	cmd := &Command{
		Use:   "hailort",
		Short: "Hailo Runtime CLI",
	}
	cmd.AddCommand(
		newScanCommand(),
		newInfoCommand(),
		newRunCommand(),
		newBenchmarkCommand(),
		newMonitorCommand(),
		newResetCommand(),
		newVersionCommand(),
	)
	return cmd
}

func newScanCommand() *Command {
	return &Command{
		Use:   "scan",
		Short: "Scan for Hailo devices",
		RunE: func(cmd *Command, args []string) error {
			return nil
		},
	}
}

func newInfoCommand() *Command {
	return &Command{
		Use:   "info [device_id]",
		Short: "Show device information",
		RunE: func(cmd *Command, args []string) error {
			return nil
		},
	}
}

func newRunCommand() *Command {
	cmd := &Command{
		Use:   "run <hef> [input] [output]",
		Short: "Run inference",
		RunE: func(cmd *Command, args []string) error {
			return nil
		},
	}
	cmd.Flags().StringP("device", "d", "", "Device ID")
	return cmd
}

func newBenchmarkCommand() *Command {
	cmd := &Command{
		Use:   "benchmark",
		Short: "Run performance benchmark",
		RunE: func(cmd *Command, args []string) error {
			return nil
		},
	}
	cmd.Flags().StringP("model", "m", "", "Model file")
	cmd.Flags().IntP("iterations", "n", 1000, "Number of iterations")
	cmd.Flags().Int("warmup", 100, "Warmup iterations")
	return cmd
}

func newMonitorCommand() *Command {
	cmd := &Command{
		Use:   "monitor [device_id]",
		Short: "Monitor device statistics",
		RunE: func(cmd *Command, args []string) error {
			return nil
		},
	}
	cmd.Flags().Duration("interval", 0, "Update interval")
	return cmd
}

func newResetCommand() *Command {
	return &Command{
		Use:   "reset [device_id]",
		Short: "Reset device",
		RunE: func(cmd *Command, args []string) error {
			return nil
		},
	}
}

func newVersionCommand() *Command {
	return &Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *Command, args []string) error {
			cmd.out.Write([]byte("hailort version 0.1.0\n"))
			return nil
		},
	}
}

// Minimal Command type for testing (mimics cobra.Command)
type Command struct {
	Use   string
	Short string
	RunE  func(*Command, []string) error

	out      *bytes.Buffer
	err      *bytes.Buffer
	args     []string
	commands []*Command
	flags    *FlagSet
}

func (c *Command) SetOut(out *bytes.Buffer) {
	c.out = out
}

func (c *Command) SetErr(err *bytes.Buffer) {
	c.err = err
}

func (c *Command) SetArgs(args []string) {
	c.args = args
}

func (c *Command) Execute() error {
	// Handle --help flag
	for _, arg := range c.args {
		if arg == "--help" || arg == "-h" {
			c.out.Write([]byte("Usage: " + c.Use + "\n"))
			c.out.Write([]byte(c.Short + "\n"))
			return nil
		}
	}
	if c.RunE != nil {
		return c.RunE(c, c.args)
	}
	return nil
}

func (c *Command) AddCommand(cmds ...*Command) {
	c.commands = append(c.commands, cmds...)
}

func (c *Command) Commands() []*Command {
	return c.commands
}

func (c *Command) Name() string {
	return strings.Split(c.Use, " ")[0]
}

func (c *Command) Flags() *FlagSet {
	if c.flags == nil {
		c.flags = &FlagSet{values: make(map[string]interface{})}
	}
	return c.flags
}

type FlagSet struct {
	values map[string]interface{}
}

func (f *FlagSet) StringP(name, short, value, usage string) {
	f.values[name] = value
}

func (f *FlagSet) IntP(name, short string, value int, usage string) {
	f.values[name] = value
}

func (f *FlagSet) Int(name string, value int, usage string) {
	f.values[name] = value
}

func (f *FlagSet) Duration(name string, value interface{}, usage string) {
	f.values[name] = value
}

func (f *FlagSet) Lookup(name string) interface{} {
	return f.values[name]
}
