package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const scannerUdevRule = `# Samsung USB scanner access for C480/C48x class devices.
# Prefer pinning the observed C480 product id after ` + "`lsusb`" + ` confirms it.
# The product wildcard below is the documented v1 exception because C480/C480W/C48x
# variants can report different product ids under Samsung vendor 04e8.
ACTION=="add", SUBSYSTEM=="usb", ATTR{idVendor}=="04e8", ENV{libsane_matched}="yes", GROUP="scanner", MODE="0664"
`

func runScannerSetupFakeRunner(ctx context.Context, streams Streams, options setupOptions, fixture setupRunnerFixture) error {
	root := os.Getenv("C48X_AIRBRIDGE_FAKE_ROOT")
	resolution := resolveSetupDependencies(setupDependencyRequest{
		InstalledSamsungBackend: fixture.scannerBackendInstalled(),
		SULDRDeb:                options.SULDRDeb,
		AirSaneCommit:           options.AirSaneCommit,
		Metadata:                defaultSetupDependencyMetadata,
	})
	if resolution.State == setupStateBlockedDriverRequired {
		return printSetupRunnerResult(streams, options, fixture, scannerBlockedResult(fixture, resolution))
	}
	steps, err := fixture.scannerHostSteps(root, resolution)
	if err != nil {
		return err
	}
	runner := NewSafeHostRunner(SafeHostRunnerConfig{
		Root:           root,
		CommandRunner:  NewFakeCommandRunner(fixture.Commands),
		CommandLogPath: fixture.CommandLogPath,
	})
	result, err := runner.Run(ctx, steps)
	if err != nil {
		return err
	}
	if err := fixture.verifyExpectedWrites(root); err != nil {
		result.State = setupStateFail
		result.Reason = err.Error()
	}
	return printSetupRunnerResult(streams, options, fixture, result)
}

func scannerBlockedResult(fixture setupRunnerFixture, resolution setupDependencyResolution) RunResult {
	return RunResult{
		State:            setupStateBlockedDriverRequired,
		Reason:           resolution.Reason,
		CommandLogPath:   fixture.CommandLogPath,
		RollbackGuidance: "no scanner setup changes were applied",
		RetryGuidance:    "rerun setup --yes --component scanner with a trusted --suldr-deb or approved pinned driver metadata",
	}
}

func (fixture setupRunnerFixture) scannerHostSteps(root string, resolution setupDependencyResolution) ([]HostStep, error) {
	dllContent, err := scannerDLLConfContent(root)
	if err != nil {
		return nil, err
	}
	steps := []HostStep{
		NewPrivilegedCommandStep("install sane packages", "apt-get", "install", "-y", "sane-utils", "libsane1", "libsane-common", "libusb-0.1-4", "usbutils", "acl", "curl", "ca-certificates", "gnupg"),
	}
	if resolution.Driver.Source == setupDriverSourceLocalDeb {
		steps = append(steps, NewPrivilegedCommandStep("install samsung scanner driver", "apt-get", "install", "-y", resolution.Driver.Path))
	}
	steps = append(steps,
		NewFileWriteStep("/etc/udev/rules.d/99-samsung-c480-scanner.rules", []byte(scannerUdevRule), 0o644),
		NewPrivilegedCommandStep("reload udev scanner rules", "udevadm", "control", "--reload-rules"),
		NewPrivilegedCommandStep("trigger udev scanner rules", "udevadm", "trigger"),
		NewPrivilegedCommandStep("ensure scanner group", "groupadd", "-f", "scanner"),
		NewFileWriteStep("/etc/sane.d/dll.conf", dllContent, 0o644),
		NewCommandStep("verify scanner current user", "scanimage", "-L"),
		NewCommandStep("verify scanner root", "sudo", "scanimage", "-L"),
	)
	if fixture.sanedAvailable() {
		steps = append(steps,
			NewPrivilegedCommandStep("add saned to scanner groups", "usermod", "-aG", "scanner,lp", "saned"),
			NewCommandStep("verify scanner saned", "sudo", "-u", "saned", "scanimage", "-L"),
		)
	}
	return steps, nil
}

func (fixture setupRunnerFixture) scannerBackendInstalled() bool {
	value := strings.TrimSpace(strings.ToLower(fixture.ProbeOutputs["sane.smfp_backend"]))
	switch value {
	case "", "missing", "absent", "not_found":
		return false
	default:
		return true
	}
}

func (fixture setupRunnerFixture) sanedAvailable() bool {
	value := strings.TrimSpace(strings.ToLower(fixture.ProbeOutputs["saned.user"]))
	return value != "missing" && value != "absent" && value != "not_found"
}

func scannerDLLConfContent(root string) ([]byte, error) {
	path := filepath.Join(filepath.Clean(root), "etc", "sane.d", "dll.conf")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []byte("smfp\n"), nil
		}
		return nil, fmt.Errorf("read /etc/sane.d/dll.conf: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "smfp" {
			return data, nil
		}
	}
	base := strings.TrimRight(string(data), "\n")
	if base == "" {
		return []byte("smfp\n"), nil
	}
	return []byte(base + "\n\nsmfp\n"), nil
}
