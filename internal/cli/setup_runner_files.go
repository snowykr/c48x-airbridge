package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var errUnsafeHostPath = errors.New("unsafe host path")

func (runner *SafeHostRunner) writeFile(ctx context.Context, step fileWriteStep) error {
	if runner.root == string(filepath.Separator) {
		return runner.writeHostFile(ctx, step)
	}
	target, err := runner.rootedPath(step.path)
	if err != nil {
		return err
	}
	current, readErr := os.ReadFile(target)
	if readErr == nil && bytes.Equal(current, step.content) {
		return nil
	}
	if readErr == nil {
		if err := runner.backupFile(step.path, current); err != nil {
			return err
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return fmt.Errorf("read %s: %w", step.path, readErr)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create parent for %s: %w", step.path, err)
	}
	if err := os.WriteFile(target, step.content, step.mode); err != nil {
		return fmt.Errorf("write %s: %w", step.path, err)
	}
	return nil
}

func (runner *SafeHostRunner) writeHostFile(ctx context.Context, step fileWriteStep) error {
	target, err := runner.rootedPath(step.path)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "c48x-airbridge-write-*")
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", step.path, err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmp.Write(step.content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary file for %s: %w", step.path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary file for %s: %w", step.path, err)
	}
	backupPath, err := runner.backupPath(step.path)
	if err != nil {
		return err
	}
	mode := strconv.FormatUint(uint64(step.mode.Perm()), 8)
	script := `set -eu
target=$1
source=$2
mode=$3
backup=$4
if [ -e "$target" ] && cmp -s "$target" "$source"; then
  exit 0
fi
if [ -e "$target" ]; then
  install -D -m 0644 "$target" "$backup"
fi
install -D -m "$mode" "$source" "$target"`
	result, err := runner.commandRunner.Run(ctx, HostCommand{
		program:    "sh",
		args:       []string{"-c", script, "sh", target, tmpPath, mode, backupPath},
		privileged: true,
	})
	if err != nil {
		return fmt.Errorf("install %s: %w", step.path, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("install %s failed with exit code %d", step.path, result.ExitCode)
	}
	return nil
}

func (runner *SafeHostRunner) backupFile(hostPath string, content []byte) error {
	backupPath, err := runner.backupPath(hostPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return fmt.Errorf("create backup parent for %s: %w", hostPath, err)
	}
	if err := os.WriteFile(backupPath, content, defaultFileMode); err != nil {
		return fmt.Errorf("backup %s: %w", hostPath, err)
	}
	return nil
}

func (runner *SafeHostRunner) backupPath(hostPath string) (string, error) {
	relative, err := cleanHostPath(hostPath)
	if err != nil {
		return "", err
	}
	name := time.Now().UTC().Format("20060102T150405.000000000Z") + ".bak"
	return filepath.Join(runner.root, "var", "backups", "c48x-airbridge", relative+"."+name), nil
}

func (runner *SafeHostRunner) appendCommandLog(line string) (err error) {
	if runner.root == string(filepath.Separator) {
		return runner.appendHostCommandLog(line)
	}
	logPath, err := runner.rootedPath(runner.commandLogPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return fmt.Errorf("create command log parent: %w", err)
	}
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, defaultFileMode)
	if err != nil {
		return fmt.Errorf("open command log: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close command log: %w", closeErr)
		}
	}()
	if _, err := fmt.Fprintln(file, line); err != nil {
		return fmt.Errorf("write command log: %w", err)
	}
	return nil
}

func (runner *SafeHostRunner) appendHostCommandLog(line string) error {
	logPath, err := runner.rootedPath(runner.commandLogPath)
	if err != nil {
		return err
	}
	script := `set -eu
log_path=$1
line=$2
install -d -m 0755 "$(dirname "$log_path")"
printf '%s\n' "$line" >> "$log_path"`
	result, err := runner.commandRunner.Run(context.Background(), HostCommand{
		program:    "sh",
		args:       []string{"-c", script, "sh", logPath, line},
		privileged: true,
	})
	if err != nil {
		return fmt.Errorf("append command log: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("append command log failed with exit code %d", result.ExitCode)
	}
	return nil
}

func (runner *SafeHostRunner) rootedPath(hostPath string) (string, error) {
	relative, err := cleanHostPath(hostPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(runner.root, relative), nil
}

func cleanHostPath(hostPath string) (string, error) {
	if hostPath == "" || !filepath.IsAbs(hostPath) {
		return "", fmt.Errorf("%w: %q", errUnsafeHostPath, hostPath)
	}
	cleaned := filepath.Clean(hostPath)
	relative := strings.TrimPrefix(cleaned, string(filepath.Separator))
	if relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == ".." {
		return "", fmt.Errorf("%w: %q", errUnsafeHostPath, hostPath)
	}
	return relative, nil
}
