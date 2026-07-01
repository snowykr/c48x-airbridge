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

type cupsQueueStep struct {
	name        string
	queueName   string
	driverModel string
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

func NewCUPSQueueStep(name string, queueName string, driverModel string) HostStep {
	return cupsQueueStep{name: name, queueName: queueName, driverModel: driverModel}
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
func (cupsQueueStep) sealedHostStep() {}

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
			if err := runner.writeFile(ctx, typed); err != nil {
				return RunResult{}, err
			}
		case cupsQueueStep:
			stop, err := runner.runCUPSQueue(ctx, typed, &result)
			if err != nil {
				return RunResult{}, err
			}
			if stop {
				return result, nil
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

func (runner *SafeHostRunner) runCUPSQueue(ctx context.Context, step cupsQueueStep, result *RunResult) (bool, error) {
	lpinfo, err := runner.runCommand(ctx, commandStep{name: "discover USB printer", command: HostCommand{program: "lpinfo", args: []string{"-v"}}}, result)
	if err != nil {
		return false, err
	}
	if lpinfo.ExitCode != 0 {
		result.State = setupStateBlockedPrinterRequired
		result.Reason = "BLOCKED_PRINTER_REQUIRED: lpinfo could not discover printers after CUPS packages were installed."
		return true, nil
	}
	deviceURI := samsungPrinterURI(lpinfo.Stdout)
	if deviceURI == "" {
		result.State = setupStateBlockedPrinterRequired
		result.Reason = "BLOCKED_PRINTER_REQUIRED: Samsung C48x/C480 USB printer was not found; connect or power on the printer, then rerun setup --yes --component cups."
		return true, nil
	}

	lpstat, err := runner.runCommand(ctx, commandStep{name: "inspect CUPS queue", command: HostCommand{program: "lpstat", args: []string{"-v", step.queueName}}}, result)
	if err != nil {
		return false, err
	}
	if lpstat.ExitCode == 0 && !queueNeedsRepair(lpstat.Stdout, deviceURI) {
		return false, nil
	}
	_, err = runner.runCommand(ctx, commandStep{
		name: "create or repair CUPS queue",
		command: HostCommand{
			program:    "lpadmin",
			args:       []string{"-p", step.queueName, "-E", "-v", deviceURI, "-m", step.driverModel},
			privileged: true,
		},
	}, result)
	return false, err
}
