package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type setupRunnerFixture struct {
	Name           string                   `json:"name"`
	CommandLogPath string                   `json:"command_log_path"`
	FilePreloads   []setupFixtureFile       `json:"file_preloads"`
	Steps          []setupFixtureStep       `json:"steps"`
	Commands       map[string]CommandResult `json:"commands"`
	ProbeOutputs   map[string]string        `json:"probe_outputs"`
	ExpectedWrites []setupFixtureWrite      `json:"expected_writes"`
}

type setupFixtureFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    uint32 `json:"mode"`
}

type setupFixtureWrite struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type setupFixtureStep struct {
	Type    string   `json:"type"`
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Sudo    bool     `json:"sudo"`
	Path    string   `json:"path"`
	Content string   `json:"content"`
	Mode    uint32   `json:"mode"`
}

type setupRunnerRequest struct {
	ctx     context.Context
	streams Streams
	options setupOptions
}

func runSetupFakeRunner(ctx context.Context, streams Streams, options setupOptions) error {
	request := setupRunnerRequest{ctx: ctx, streams: streams, options: options}
	fixture, err := loadSetupRunnerFixture(options.FakeRunner)
	if err != nil {
		return err
	}
	if options.Component == setupComponentCUPS && len(fixture.Steps) == 0 {
		return runCUPSSetupFakeRunner(request, fixture)
	}
	if options.Component == setupComponentAirSane && len(fixture.Steps) == 0 {
		return runAirSaneSetupFakeRunner(ctx, streams, options, fixture)
	}
	root := os.Getenv("C48X_AIRBRIDGE_FAKE_ROOT")
	if err := fixture.preload(root); err != nil {
		return err
	}
	if options.Component == setupComponentScanner {
		return runScannerSetupFakeRunner(ctx, streams, options, fixture)
	}
	runner := NewSafeHostRunner(SafeHostRunnerConfig{
		Root:           root,
		CommandRunner:  NewFakeCommandRunner(fixture.Commands),
		CommandLogPath: fixture.CommandLogPath,
	})
	result, err := runner.Run(ctx, fixture.hostSteps())
	if err != nil {
		return err
	}
	if err := fixture.verifyExpectedWrites(root); err != nil {
		result.State = setupStateFail
		result.Reason = err.Error()
	}
	return printSetupRunnerResult(streams, options, fixture, result)
}

func runAirSaneSetupFakeRunner(ctx context.Context, streams Streams, options setupOptions, fixture setupRunnerFixture) error {
	plan := buildAirSaneSetupPlan(options)
	if plan.State != setupStatePass {
		return printSetupRunnerResult(streams, options, fixture, airSaneBlockedResult(plan, fixture.CommandLogPath))
	}
	root := os.Getenv("C48X_AIRBRIDGE_FAKE_ROOT")
	if err := fixture.preload(root); err != nil {
		return err
	}
	runner := NewSafeHostRunner(SafeHostRunnerConfig{
		Root:           root,
		CommandRunner:  NewFakeCommandRunner(fixture.Commands),
		CommandLogPath: fixture.CommandLogPath,
	})
	result, err := runner.Run(ctx, plan.Steps)
	if err != nil {
		return err
	}
	if err := fixture.verifyExpectedWrites(root); err != nil {
		result.State = setupStateFail
		result.Reason = err.Error()
	}
	return printSetupRunnerResult(streams, options, fixture, result)
}

func runCUPSSetupFakeRunner(request setupRunnerRequest, fixture setupRunnerFixture) error {
	root := os.Getenv("C48X_AIRBRIDGE_FAKE_ROOT")
	if err := fixture.preload(root); err != nil {
		return err
	}
	plan := planCUPSSetup(fixture)
	if plan.State != setupStatePass {
		result := RunResult{
			State:            plan.State,
			Reason:           plan.Reason,
			CommandLogPath:   fixture.CommandLogPath,
			RollbackGuidance: "no CUPS queue changes were applied",
			RetryGuidance:    "connect or power on the Samsung C48x/C480 USB printer, then rerun setup --yes --component cups",
		}
		return printSetupRunnerResult(request.streams, request.options, fixture, result)
	}
	runner := NewSafeHostRunner(SafeHostRunnerConfig{
		Root:           root,
		CommandRunner:  NewFakeCommandRunner(fixture.Commands),
		CommandLogPath: fixture.CommandLogPath,
	})
	result, err := runner.Run(request.ctx, plan.Steps)
	if err != nil {
		return err
	}
	if err := fixture.verifyExpectedWrites(root); err != nil {
		result.State = setupStateFail
		result.Reason = err.Error()
	}
	return printSetupRunnerResult(request.streams, request.options, fixture, result)
}

func loadSetupRunnerFixture(path string) (setupRunnerFixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return setupRunnerFixture{}, fmt.Errorf("read fake-runner fixture %s: %w", path, err)
	}
	var fixture setupRunnerFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return setupRunnerFixture{}, fmt.Errorf("parse fake-runner fixture %s: %w", path, err)
	}
	if fixture.CommandLogPath == "" {
		fixture.CommandLogPath = "/var/log/c48x-airbridge/commands.log"
	}
	if fixture.Commands == nil {
		fixture.Commands = map[string]CommandResult{}
	}
	return fixture, nil
}

func (fixture setupRunnerFixture) hostSteps() []HostStep {
	steps := make([]HostStep, 0, len(fixture.Steps))
	for _, step := range fixture.Steps {
		switch step.Type {
		case "command":
			if step.Sudo {
				steps = append(steps, NewPrivilegedCommandStep(step.Name, step.Command, step.Args...))
			} else {
				steps = append(steps, NewCommandStep(step.Name, step.Command, step.Args...))
			}
		case "write_file":
			steps = append(steps, NewFileWriteStep(step.Path, []byte(step.Content), os.FileMode(step.Mode)))
		}
	}
	return steps
}

func (fixture setupRunnerFixture) preload(root string) error {
	for _, file := range fixture.FilePreloads {
		target, err := rootedFixturePath(root, file.Path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create preload parent %s: %w", file.Path, err)
		}
		mode := os.FileMode(file.Mode)
		if mode == 0 {
			mode = defaultFileMode
		}
		if err := os.WriteFile(target, []byte(file.Content), mode); err != nil {
			return fmt.Errorf("preload %s: %w", file.Path, err)
		}
	}
	return nil
}

func (fixture setupRunnerFixture) verifyExpectedWrites(root string) error {
	for _, expected := range fixture.ExpectedWrites {
		target, err := rootedFixturePath(root, expected.Path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(target)
		if err != nil {
			return fmt.Errorf("read expected write %s: %w", expected.Path, err)
		}
		if string(data) != expected.Content {
			return fmt.Errorf("expected write %s mismatch", expected.Path)
		}
	}
	return nil
}

func rootedFixturePath(root string, hostPath string) (string, error) {
	relative, err := cleanHostPath(hostPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Clean(root), relative), nil
}

func printSetupRunnerResult(streams Streams, options setupOptions, fixture setupRunnerFixture, result RunResult) error {
	lines := []string{
		"setup apply result:",
		"fixture: " + fixture.Name,
		"component: " + string(options.Component),
		"state: " + result.State,
	}
	if result.Reason != "" {
		lines = append(lines, "reason: "+result.Reason)
	}
	lines = append(lines,
		"rollback: "+result.RollbackGuidance,
		"retry: "+result.RetryGuidance,
		"command log: "+result.CommandLogPath,
	)
	if len(result.CommandLines) > 0 {
		lines = append(lines, "commands:")
		for _, commandLine := range result.CommandLines {
			lines = append(lines, "- "+commandLine)
		}
	}
	_, err := fmt.Fprintln(streams.Out, strings.Join(lines, "\n"))
	return err
}
