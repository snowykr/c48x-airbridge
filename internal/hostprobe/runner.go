package hostprobe

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

type osRunner struct{}

func (osRunner) LookPath(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", err
	}
	return path, nil
}

func (osRunner) Run(ctx context.Context, name string, args ...string) CommandResult {
	cmd := exec.CommandContext(ctx, name, args...)
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd.Stdout = out
	cmd.Stderr = errOut
	err := cmd.Run()
	result := CommandResult{Stdout: out.String(), Stderr: errOut.String()}
	if err == nil {
		return result
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	}
	result.Err = err
	return result
}

func (osRunner) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (osRunner) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
