package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"c48x-airbridge/internal/hostprobe"
)

const setupRealFixtureName = "live-host"

type setupRealRuntime struct {
	root           string
	commandRunner  CommandRunner
	commandLogPath string
	goos           string
	goarch         string
}

func defaultSetupRealRuntime() setupRealRuntime {
	return setupRealRuntime{
		root:           "/",
		commandRunner:  OSCommandRunner{},
		commandLogPath: "/var/log/c48x-airbridge/commands.log",
		goos:           goruntime.GOOS,
		goarch:         goruntime.GOARCH,
	}
}

func runSetupReal(ctx context.Context, streams Streams, options setupOptions, runtime setupRealRuntime) error {
	runtime = normalizeSetupRealRuntime(runtime)
	preflight := setupRealPreflight(ctx, runtime)
	if !preflight.pass {
		fixture := setupRunnerFixture{Name: setupRealFixtureName, CommandLogPath: runtime.commandLogPath}
		return printGuidedSetupResult(guidedSetupReport{
			streams:  streams,
			options:  options,
			fixture:  fixture,
			sections: []guidedSetupSection{{Name: "preflight", State: setupStateFail, Detail: preflight.reason}},
			result: RunResult{
				State:            setupStateFail,
				Reason:           preflight.reason,
				CommandLogPath:   runtime.commandLogPath,
				RollbackGuidance: "no host changes were applied",
				RetryGuidance:    preflight.retry,
			},
			evidencePath: "not written: blocked before host mutation",
		})
	}
	fixture := setupRealFixture(ctx, runtime)
	switch options.Component {
	case setupComponentAll:
		return runGuidedSetupReal(ctx, streams, options, runtime, fixture)
	case setupComponentCUPS:
		return runCUPSSetupReal(ctx, streams, options, runtime, fixture)
	case setupComponentScanner:
		return runScannerSetupReal(ctx, streams, options, runtime, fixture)
	case setupComponentAirSane:
		return runAirSaneSetupReal(ctx, streams, options, runtime, fixture)
	case setupComponentVerify:
		return runVerify(verifyRunRequest{
			ctx:     ctx,
			streams: streams,
			options: verifyOptions{Live: true},
			probe:   hostprobe.New(hostprobe.Options{}),
		})
	default:
		return fmt.Errorf("unsupported setup component %q", options.Component)
	}
}

func normalizeSetupRealRuntime(runtime setupRealRuntime) setupRealRuntime {
	if runtime.root == "" {
		runtime.root = "/"
	}
	if runtime.commandRunner == nil {
		runtime.commandRunner = OSCommandRunner{}
	}
	if runtime.commandLogPath == "" {
		runtime.commandLogPath = "/var/log/c48x-airbridge/commands.log"
	}
	if runtime.goos == "" {
		runtime.goos = goruntime.GOOS
	}
	if runtime.goarch == "" {
		runtime.goarch = goruntime.GOARCH
	}
	return runtime
}

func setupRealFixture(ctx context.Context, runtime setupRealRuntime) setupRunnerFixture {
	backendState := "missing"
	if setupRealSMFPBackendInstalled(runtime.root) {
		backendState = "installed"
	}
	return setupRunnerFixture{
		Name:           setupRealFixtureName,
		CommandLogPath: runtime.commandLogPath,
		Commands:       map[string]CommandResult{},
		ProbeOutputs: map[string]string{
			"lpinfo -v":                  setupRealCommandOutput(ctx, runtime.commandRunner, "lpinfo", "-v"),
			"lpstat -v " + cupsQueueName: setupRealCommandOutput(ctx, runtime.commandRunner, "lpstat", "-v", cupsQueueName),
			"sane.smfp_backend":          backendState,
			"saned.user":                 setupRealSanedUser(ctx, runtime.commandRunner),
		},
	}
}

func setupRealSanedUser(ctx context.Context, runner CommandRunner) string {
	result, err := runner.Run(ctx, HostCommand{program: "id", args: []string{"saned"}})
	if err != nil || result.ExitCode != 0 {
		return "missing"
	}
	return "present"
}

func setupRealPlatformSupported(runtime setupRealRuntime) bool {
	if runtime.goos != "linux" {
		return false
	}
	return runtime.goarch == "amd64" || runtime.goarch == "arm64"
}

func setupRealCommandOutput(ctx context.Context, runner CommandRunner, program string, args ...string) string {
	result, err := runner.Run(ctx, HostCommand{program: program, args: append([]string(nil), args...)})
	if err != nil {
		return err.Error()
	}
	return strings.TrimSpace(strings.Join([]string{result.Stdout, result.Stderr}, "\n"))
}

func setupRealSMFPBackendInstalled(root string) bool {
	for _, pattern := range []string{"/usr/lib/sane/libsane-smfp.so.1", "/usr/lib/*/sane/libsane-smfp.so.1"} {
		matches, err := filepath.Glob(setupRealRootedGlob(root, pattern))
		if err == nil && len(matches) > 0 {
			return true
		}
	}
	return false
}

func setupRealRootedGlob(root string, hostPattern string) string {
	relative := strings.TrimPrefix(filepath.Clean(hostPattern), string(os.PathSeparator))
	return filepath.Join(filepath.Clean(root), relative)
}

func runGuidedSetupReal(ctx context.Context, streams Streams, options setupOptions, runtime setupRealRuntime, fixture setupRunnerFixture) error {
	plan, err := buildGuidedSetupPlan(options, fixture, runtime.root)
	if err != nil {
		return err
	}
	if plan.State != setupStatePass {
		return printGuidedSetupResult(guidedSetupReport{
			streams:      streams,
			options:      options,
			fixture:      fixture,
			sections:     plan.Sections,
			result:       guidedBlockedResult(fixture, plan),
			evidencePath: "not written: blocked before host mutation",
		})
	}
	result, err := runSetupRealSteps(ctx, runtime, plan.Steps)
	if err != nil {
		return err
	}
	if result.State == setupStatePass {
		result.State = setupStateBlockedClientProof
		result.Reason = "Linux host setup completed; macOS/Windows client print and scan proof is still required."
		result.RetryGuidance = "run verify after collecting macOS and Windows client proof"
	}
	evidencePath, err := writeGuidedSetupEvidence(guidedEvidenceRequest{
		root:     runtime.root,
		options:  options,
		fixture:  fixture,
		sections: plan.Sections,
		result:   result,
	})
	if err != nil {
		return err
	}
	return printGuidedSetupResult(guidedSetupReport{
		streams:      streams,
		options:      options,
		fixture:      fixture,
		sections:     plan.Sections,
		result:       result,
		evidencePath: evidencePath,
	})
}

func runCUPSSetupReal(ctx context.Context, streams Streams, options setupOptions, runtime setupRealRuntime, fixture setupRunnerFixture) error {
	plan := planCUPSSetup(fixture)
	if plan.State != setupStatePass {
		return printSetupRunnerResult(streams, options, fixture, RunResult{
			State:            plan.State,
			Reason:           plan.Reason,
			CommandLogPath:   runtime.commandLogPath,
			RollbackGuidance: "no CUPS queue changes were applied",
			RetryGuidance:    "connect or power on the Samsung C48x/C480 USB printer, then rerun setup --yes --component cups",
		})
	}
	result, err := runSetupRealSteps(ctx, runtime, plan.Steps)
	if err != nil {
		return err
	}
	return printSetupRunnerResult(streams, options, fixture, result)
}

func runScannerSetupReal(ctx context.Context, streams Streams, options setupOptions, runtime setupRealRuntime, fixture setupRunnerFixture) error {
	resolution := resolveSetupDependencies(setupDependencyRequest{
		InstalledSamsungBackend: fixture.scannerBackendInstalled(),
		SULDRDeb:                options.SULDRDeb,
		AirSaneCommit:           options.AirSaneCommit,
		Metadata:                defaultSetupDependencyMetadata,
	})
	if resolution.State == setupStateBlockedDriverRequired {
		return printSetupRunnerResult(streams, options, fixture, scannerBlockedResult(fixture, resolution))
	}
	steps, err := fixture.scannerHostSteps(runtime.root, resolution)
	if err != nil {
		return err
	}
	result, err := runSetupRealSteps(ctx, runtime, steps)
	if err != nil {
		return err
	}
	return printSetupRunnerResult(streams, options, fixture, result)
}

func runAirSaneSetupReal(ctx context.Context, streams Streams, options setupOptions, runtime setupRealRuntime, fixture setupRunnerFixture) error {
	plan := buildAirSaneSetupPlan(options)
	if plan.State != setupStatePass {
		return printSetupRunnerResult(streams, options, fixture, airSaneBlockedResult(plan, runtime.commandLogPath))
	}
	result, err := runSetupRealSteps(ctx, runtime, plan.Steps)
	if err != nil {
		return err
	}
	return printSetupRunnerResult(streams, options, fixture, result)
}

func runSetupRealSteps(ctx context.Context, runtime setupRealRuntime, steps []HostStep) (RunResult, error) {
	runner := NewSafeHostRunner(SafeHostRunnerConfig{
		Root:           runtime.root,
		CommandRunner:  runtime.commandRunner,
		CommandLogPath: runtime.commandLogPath,
	})
	return runner.Run(ctx, steps)
}
