package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var errUnsafeHostPath = errors.New("unsafe host path")

func (runner *SafeHostRunner) writeFile(step fileWriteStep) error {
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

func (runner *SafeHostRunner) backupFile(hostPath string, content []byte) error {
	relative, err := cleanHostPath(hostPath)
	if err != nil {
		return err
	}
	name := time.Now().UTC().Format("20060102T150405.000000000Z") + ".bak"
	backupPath := filepath.Join(runner.root, "var", "backups", "c48x-airbridge", relative+"."+name)
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return fmt.Errorf("create backup parent for %s: %w", hostPath, err)
	}
	if err := os.WriteFile(backupPath, content, defaultFileMode); err != nil {
		return fmt.Errorf("backup %s: %w", hostPath, err)
	}
	return nil
}

func (runner *SafeHostRunner) appendCommandLog(line string) (err error) {
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
