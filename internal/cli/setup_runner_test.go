package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_SafeHostRunner_WriteFileBacksUpAndIsIdempotent_whenRunTwice(t *testing.T) {
	// Given
	root := t.TempDir()
	target := filepath.Join(root, "etc", "cups", "cupsd.conf")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("create fake config dir: %v", err)
	}
	if err := os.WriteFile(target, []byte("old config\n"), 0o644); err != nil {
		t.Fatalf("preload fake config: %v", err)
	}
	runner := NewSafeHostRunner(SafeHostRunnerConfig{
		Root:           root,
		CommandRunner:  NewFakeCommandRunner(nil),
		CommandLogPath: "/var/log/c48x-airbridge/commands.log",
	})
	steps := []HostStep{
		NewFileWriteStep("/etc/cups/cupsd.conf", []byte("new config\n"), 0o644),
	}

	// When
	first, firstErr := runner.Run(context.Background(), steps)
	second, secondErr := runner.Run(context.Background(), steps)

	// Then
	if firstErr != nil {
		t.Fatalf("first runner apply failed: %v", firstErr)
	}
	if secondErr != nil {
		t.Fatalf("second runner apply failed: %v", secondErr)
	}
	if first.State != runnerStatePass || second.State != runnerStatePass {
		t.Fatalf("runner states = %q then %q, want PASS", first.State, second.State)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read fake config: %v", err)
	}
	if string(got) != "new config\n" {
		t.Fatalf("fake config content = %q", got)
	}
	backups, err := filepath.Glob(filepath.Join(root, "var", "backups", "c48x-airbridge", "etc", "cups", "cupsd.conf.*.bak"))
	if err != nil {
		t.Fatalf("glob backups: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("backup count = %d, want 1: %v", len(backups), backups)
	}
	backup, err := os.ReadFile(backups[0])
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(backup) != "old config\n" {
		t.Fatalf("backup content = %q", backup)
	}
}

func Test_SafeHostRunner_UsesPrivilegedFileInstallAndLogAppend_whenRootIsHost(t *testing.T) {
	// Given
	recorder := &recordingCommandRunner{}
	runner := NewSafeHostRunner(SafeHostRunnerConfig{
		Root:           "/",
		CommandRunner:  recorder,
		CommandLogPath: "/var/log/c48x-airbridge/commands.log",
	})
	steps := []HostStep{
		NewFileWriteStep("/etc/c48x-airbridge/test.conf", []byte("managed\n"), 0o644),
		NewPrivilegedCommandStep("restart service", "systemctl", "restart", "airsaned.service"),
	}

	// When
	result, err := runner.Run(context.Background(), steps)

	// Then
	if err != nil {
		t.Fatalf("host-root runner failed before sudo-backed file writes: %v", err)
	}
	if result.State != runnerStatePass {
		t.Fatalf("runner state = %q, want PASS", result.State)
	}
	if len(recorder.commands) < 3 {
		t.Fatalf("recorded commands = %d, want log append, file install, service restart: %#v", len(recorder.commands), recorder.commands)
	}
	for _, command := range recorder.commands[:2] {
		if !command.privileged || command.program != "sh" {
			t.Fatalf("host-root file/log command was not sudo shell-backed: %#v", command)
		}
	}
	if recorder.commands[len(recorder.commands)-1].CommandLine() != "systemctl restart airsaned.service" {
		t.Fatalf("last command = %q, want service restart", recorder.commands[len(recorder.commands)-1].CommandLine())
	}
}

type recordingCommandRunner struct {
	commands []HostCommand
}

func (runner *recordingCommandRunner) Run(ctx context.Context, command HostCommand) (CommandResult, error) {
	select {
	case <-ctx.Done():
		return CommandResult{}, ctx.Err()
	default:
	}
	runner.commands = append(runner.commands, command)
	return CommandResult{}, nil
}

func Test_SafeHostRunner_LogsTypedCommandsUnderFakeRoot_whenCommandsRun(t *testing.T) {
	// Given
	root := t.TempDir()
	runner := NewSafeHostRunner(SafeHostRunnerConfig{
		Root: root,
		CommandRunner: NewFakeCommandRunner(map[string]FakeCommandResult{
			"apt-get install -y cups":     {ExitCode: 0},
			"systemctl enable --now cups": {ExitCode: 0},
			"lpadmin -p C48X -E":          {ExitCode: 0},
			"cmake -S . -B build":         {ExitCode: 0},
		}),
		CommandLogPath: "/var/log/c48x-airbridge/commands.log",
	})
	steps := []HostStep{
		NewPrivilegedCommandStep("install cups", "apt-get", "install", "-y", "cups"),
		NewPrivilegedCommandStep("enable cups", "systemctl", "enable", "--now", "cups"),
		NewPrivilegedCommandStep("create queue", "lpadmin", "-p", "C48X", "-E"),
		NewCommandStep("configure build", "cmake", "-S", ".", "-B", "build"),
	}

	// When
	result, err := runner.Run(context.Background(), steps)

	// Then
	if err != nil {
		t.Fatalf("runner apply failed: %v", err)
	}
	if result.State != runnerStatePass {
		t.Fatalf("runner state = %q, want PASS", result.State)
	}
	logPath := filepath.Join(root, "var", "log", "c48x-airbridge", "commands.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read command log: %v", err)
	}
	got := string(data)
	for _, want := range []string{
		"sudo apt-get install -y cups",
		"sudo systemctl enable --now cups",
		"sudo lpadmin -p C48X -E",
		"cmake -S . -B build",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("command log missing %q:\n%s", want, got)
		}
	}
}
