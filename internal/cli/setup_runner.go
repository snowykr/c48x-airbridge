package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

const defaultFileMode os.FileMode = 0o644

const (
	runnerStatePass = "PASS"
	runnerStateFail = "FAIL"
)

type SafeHostRunnerConfig struct {
	Root           string
	CommandRunner  CommandRunner
	CommandLogPath string
}

type SafeHostRunner struct {
	root           string
	commandRunner  CommandRunner
	commandLogPath string
}

type RunResult struct {
	State            string
	Reason           string
	CommandLogPath   string
	RollbackGuidance string
	RetryGuidance    string
	CommandLines     []string
}

type HostStep interface {
	sealedHostStep()
}

type commandStep struct {
	name    string
	command HostCommand
}

type fileWriteStep struct {
	path    string
	content []byte
	mode    os.FileMode
}

type HostCommand struct {
	program    string
	args       []string
	privileged bool
}

type CommandRunner interface {
	Run(context.Context, HostCommand) (CommandResult, error)
}

type CommandResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

type FakeCommandResult = CommandResult

type FakeCommandRunner struct {
	results map[string]CommandResult
}

type OSCommandRunner struct{}

func NewSafeHostRunner(config SafeHostRunnerConfig) *SafeHostRunner {
	commandRunner := config.CommandRunner
	if commandRunner == nil {
		commandRunner = OSCommandRunner{}
	}
	commandLogPath := config.CommandLogPath
	if commandLogPath == "" {
		commandLogPath = "/var/log/c48x-airbridge/commands.log"
	}
	root := config.Root
	if root == "" {
		root = "/"
	}
	return &SafeHostRunner{
		root:           filepath.Clean(root),
		commandRunner:  commandRunner,
		commandLogPath: commandLogPath,
	}
}

func NewCommandStep(name string, program string, args ...string) HostStep {
	return commandStep{name: name, command: HostCommand{program: program, args: append([]string(nil), args...)}}
}

func NewPrivilegedCommandStep(name string, program string, args ...string) HostStep {
	return commandStep{name: name, command: HostCommand{program: program, args: append([]string(nil), args...), privileged: true}}
}

func NewFileWriteStep(path string, content []byte, mode os.FileMode) HostStep {
	if mode == 0 {
		mode = defaultFileMode
	}
	return fileWriteStep{path: path, content: append([]byte(nil), content...), mode: mode}
}

func NewFakeCommandRunner(results map[string]FakeCommandResult) *FakeCommandRunner {
	copied := make(map[string]CommandResult, len(results))
	for commandLine, result := range results {
		copied[commandLine] = result
	}
	return &FakeCommandRunner{results: copied}
}

func (commandStep) sealedHostStep()   {}
func (fileWriteStep) sealedHostStep() {}

func (runner *SafeHostRunner) Run(ctx context.Context, steps []HostStep) (RunResult, error) {
	result := RunResult{
		State:            runnerStatePass,
		CommandLogPath:   runner.commandLogPath,
		RollbackGuidance: "restore files from /var/backups/c48x-airbridge before retrying",
		RetryGuidance:    "rerun setup --yes after correcting the failed step",
	}
	for _, step := range steps {
		switch typed := step.(type) {
		case commandStep:
			commandResult, err := runner.runCommand(ctx, typed, &result)
			if err != nil {
				return RunResult{}, err
			}
			if commandResult.ExitCode != 0 {
				result.State = runnerStateFail
				result.Reason = fmt.Sprintf("%s failed with exit code %d", typed.name, commandResult.ExitCode)
				return result, nil
			}
		case fileWriteStep:
			if err := runner.writeFile(typed); err != nil {
				return RunResult{}, err
			}
		default:
			return RunResult{}, fmt.Errorf("unsupported host step %T", step)
		}
	}
	return result, nil
}

func (runner *SafeHostRunner) runCommand(ctx context.Context, step commandStep, result *RunResult) (CommandResult, error) {
	display := step.command.DisplayLine()
	result.CommandLines = append(result.CommandLines, display)
	if err := runner.appendCommandLog(display); err != nil {
		return CommandResult{}, err
	}
	commandResult, err := runner.commandRunner.Run(ctx, step.command)
	if err != nil {
		return CommandResult{}, fmt.Errorf("run %s: %w", step.name, err)
	}
	return commandResult, nil
}
