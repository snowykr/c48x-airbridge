package hostprobe_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"c48x-airbridge/internal/hostprobe"
)

func Test_Probe_reportsPresentUSB_whenSamsungDeviceIsConnected(t *testing.T) {
	// Given
	runner := newFakeRunner()
	runner.command("lsusb", hostprobe.CommandResult{Stdout: "Bus 001 Device 004: ID 04e8:3469 Samsung Electronics Co., Ltd C48x Series\n"})

	// When
	report := hostprobe.New(hostprobe.Options{Runner: runner}).Run(context.Background())

	// Then
	result := requireResult(t, report, hostprobe.CheckUSBDevice)
	if result.Status != hostprobe.StatusPass {
		t.Fatalf("USB status = %s, want %s", result.Status, hostprobe.StatusPass)
	}
	if !strings.Contains(result.Evidence, "04e8:3469") {
		t.Fatalf("USB evidence did not include VID/PID: %#v", result)
	}
}

func Test_Probe_reportsBlockedUSB_whenPrinterIsMissingOrPoweredOff(t *testing.T) {
	// Given
	runner := newFakeRunner()
	runner.command("lsusb", hostprobe.CommandResult{Stdout: "Bus 001 Device 002: ID 8087:0026 Intel Corp.\n"})

	// When
	report := hostprobe.New(hostprobe.Options{Runner: runner}).Run(context.Background())

	// Then
	result := requireResult(t, report, hostprobe.CheckUSBDevice)
	if result.Status != hostprobe.StatusBlocked {
		t.Fatalf("USB status = %s, want %s", result.Status, hostprobe.StatusBlocked)
	}
	if !strings.Contains(result.Detail, "connect or power on") {
		t.Fatalf("USB detail did not name the likely recovery action: %#v", result)
	}
}

func Test_Probe_reportsWarnForMissingPackages_whenRequiredCommandsAreAbsent(t *testing.T) {
	// Given
	runner := newFakeRunner()
	runner.missing("lsusb")
	runner.missing("scanimage")
	runner.missing("lpstat")
	runner.missing("avahi-browse")
	runner.missing("curl")

	// When
	report := hostprobe.New(hostprobe.Options{Runner: runner}).Run(context.Background())

	// Then
	for _, check := range []hostprobe.CheckID{
		hostprobe.CheckCommandLSUSB,
		hostprobe.CheckCommandScanimage,
		hostprobe.CheckCommandLPStat,
		hostprobe.CheckCommandAvahiBrowse,
		hostprobe.CheckCommandCurl,
	} {
		result := requireResult(t, report, check)
		if result.Status != hostprobe.StatusWarn {
			t.Fatalf("%s status = %s, want %s", check, result.Status, hostprobe.StatusWarn)
		}
	}
}

func Test_Probe_reportsSANETripletAndSMFPBackend_whenScannerStackIsVisible(t *testing.T) {
	// Given
	runner := newFakeRunner()
	runner.command("scanimage", hostprobe.CommandResult{Stdout: "`smfp:usb;04e8;3469;Z9MJBAK` is a Samsung C48x Series multi-function peripheral\n"})
	runner.command("sudo", hostprobe.CommandResult{Stdout: "`smfp:usb;04e8;3469;Z9MJBAK` is a Samsung C48x Series multi-function peripheral\n"})
	runner.command("id", hostprobe.CommandResult{Stdout: "uid=118(saned) gid=125(saned) groups=125(saned)\n"})
	runner.file("/etc/sane.d/dll.conf", "net\nsmfp\n")
	runner.glob("/usr/lib/sane/libsane-smfp.so.1")

	// When
	report := hostprobe.New(hostprobe.Options{Runner: runner}).Run(context.Background())

	// Then
	for _, check := range []hostprobe.CheckID{
		hostprobe.CheckSANECurrentUser,
		hostprobe.CheckSANERoot,
		hostprobe.CheckSANESaned,
		hostprobe.CheckSMFPConfig,
		hostprobe.CheckSMFPBackend,
	} {
		result := requireResult(t, report, check)
		if result.Status != hostprobe.StatusPass {
			t.Fatalf("%s status = %s, want %s (%#v)", check, result.Status, hostprobe.StatusPass, result)
		}
	}
}

func Test_Probe_reportsAirSaneHTTPStatus_whenHTTPProbeSucceedsOrFails(t *testing.T) {
	// Given
	successRunner := newFakeRunner()
	successRunner.command("curl", hostprobe.CommandResult{Stdout: "AirSane\n"})
	failureRunner := newFakeRunner()
	failureRunner.command("curl", hostprobe.CommandResult{ExitCode: 7, Err: errors.New("connection refused")})

	// When
	success := hostprobe.New(hostprobe.Options{Runner: successRunner}).Run(context.Background())
	failure := hostprobe.New(hostprobe.Options{Runner: failureRunner}).Run(context.Background())

	// Then
	successResult := requireResult(t, success, hostprobe.CheckAirSaneHTTP)
	if successResult.Status != hostprobe.StatusPass {
		t.Fatalf("AirSane success status = %s, want %s", successResult.Status, hostprobe.StatusPass)
	}
	failureResult := requireResult(t, failure, hostprobe.CheckAirSaneHTTP)
	if failureResult.Status != hostprobe.StatusWarn {
		t.Fatalf("AirSane failure status = %s, want %s", failureResult.Status, hostprobe.StatusWarn)
	}
}

func Test_Probe_usesOnlyNonMutatingCommands_whenRun(t *testing.T) {
	// Given
	runner := newFakeRunner()
	runner.command("uname", hostprobe.CommandResult{Stdout: "x86_64\n"})

	// When
	_ = hostprobe.New(hostprobe.Options{Runner: runner}).Run(context.Background())

	// Then
	for _, call := range runner.calls {
		if slices.Contains([]string{"apt", "apt-get", "dpkg", "systemctl enable", "systemctl start", "service"}, call) {
			t.Fatalf("probe ran mutating command %q; calls: %v", call, runner.calls)
		}
	}
}

func requireResult(t *testing.T, report hostprobe.Report, check hostprobe.CheckID) hostprobe.Result {
	t.Helper()
	for _, result := range report.Results {
		if result.Check == check {
			return result
		}
	}
	t.Fatalf("report missing check %s: %#v", check, report.Results)
	return hostprobe.Result{}
}

type fakeRunner struct {
	calls           []string
	commands        map[string]hostprobe.CommandResult
	files           map[string]string
	globs           []string
	missingCommands map[string]bool
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{
		commands:        map[string]hostprobe.CommandResult{},
		files:           map[string]string{},
		missingCommands: map[string]bool{},
	}
}

func (r *fakeRunner) command(name string, result hostprobe.CommandResult) {
	r.commands[name] = result
}

func (r *fakeRunner) missing(name string) {
	r.missingCommands[name] = true
}

func (r *fakeRunner) file(path string, content string) {
	r.files[path] = content
}

func (r *fakeRunner) glob(path string) {
	r.globs = append(r.globs, path)
}

func (r *fakeRunner) LookPath(name string) (string, error) {
	if r.missingCommands[name] {
		return "", errors.New("missing")
	}
	return "/usr/bin/" + name, nil
}

func (r *fakeRunner) Run(ctx context.Context, name string, args ...string) hostprobe.CommandResult {
	r.calls = append(r.calls, strings.Join(append([]string{name}, args...), " "))
	if result, ok := r.commands[name]; ok {
		return result
	}
	return hostprobe.CommandResult{}
}

func (r *fakeRunner) ReadFile(path string) (string, error) {
	if content, ok := r.files[path]; ok {
		return content, nil
	}
	return "", errors.New("missing file")
}

func (r *fakeRunner) Glob(pattern string) ([]string, error) {
	return r.globs, nil
}
