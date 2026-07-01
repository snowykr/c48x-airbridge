package cli

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func (command HostCommand) CommandLine() string {
	parts := append([]string{command.program}, command.args...)
	return strings.Join(parts, " ")
}

func (command HostCommand) DisplayLine() string {
	if command.privileged {
		return "sudo " + command.CommandLine()
	}
	return command.CommandLine()
}

func (runner *FakeCommandRunner) Run(ctx context.Context, command HostCommand) (CommandResult, error) {
	select {
	case <-ctx.Done():
		return CommandResult{}, ctx.Err()
	default:
	}
	if result, ok := runner.results[command.CommandLine()]; ok {
		return result, nil
	}
	return CommandResult{}, nil
}

func (OSCommandRunner) Run(ctx context.Context, command HostCommand) (CommandResult, error) {
	name := command.program
	args := append([]string(nil), command.args...)
	if command.privileged {
		args = append([]string{name}, args...)
		name = "sudo"
	}
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	result := CommandResult{Stdout: string(out)}
	if err == nil {
		return result, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		result.Stderr = string(exitErr.Stderr)
		return result, nil
	}
	return CommandResult{}, fmt.Errorf("execute %s: %w", command.CommandLine(), err)
}
