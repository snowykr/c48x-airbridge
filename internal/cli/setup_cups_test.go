package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_CUPSSetupPlan_plansPackageServiceSharingAndQueueCommands_whenUSBPrinterPresent(t *testing.T) {
	// Given
	fixture := setupRunnerFixture{}

	// When
	plan := planCUPSSetup(fixture)

	// Then
	if plan.State != setupStatePass {
		t.Fatalf("CUPS plan state = %q, want PASS: %s", plan.State, plan.Reason)
	}
	got := displayLines(plan.Steps)
	for _, want := range []string{
		"sudo apt-get install -y cups cups-client avahi-daemon avahi-utils printer-driver-splix system-config-printer",
		"sudo systemctl enable --now cups",
		"sudo systemctl enable --now avahi-daemon",
		"sudo cupsctl --share-printers --no-remote-admin --no-remote-any",
		"dynamic CUPS queue: lpinfo -v",
		"lpstat -t",
		"avahi-browse -rt _ipp._tcp",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("CUPS plan missing %q:\n%s", want, got)
		}
	}
	for _, forbidden := range []string{"--remote-admin", "--remote-any"} {
		if strings.Contains(strings.ReplaceAll(got, "--no-"+strings.TrimPrefix(forbidden, "--"), ""), forbidden) {
			t.Fatalf("CUPS plan enabled unsafe remote access flag %q:\n%s", forbidden, got)
		}
	}
}

func Test_CUPSSetupPlan_reusesExistingCompatibleQueue_whenQueueMatchesUSBPrinter(t *testing.T) {
	// Given
	root := t.TempDir()
	fixture := setupRunnerFixture{}
	runner := NewSafeHostRunner(SafeHostRunnerConfig{
		Root: root,
		CommandRunner: NewFakeCommandRunner(map[string]FakeCommandResult{
			"apt-get install -y cups cups-client avahi-daemon avahi-utils printer-driver-splix system-config-printer": {ExitCode: 0},
			"systemctl enable --now cups":                                {ExitCode: 0},
			"systemctl enable --now avahi-daemon":                        {ExitCode: 0},
			"cupsctl --share-printers --no-remote-admin --no-remote-any": {ExitCode: 0},
			"lpinfo -v":                  {Stdout: "direct usb://Samsung/C48x%20Series?serial=Z9MJBAK\n"},
			"lpstat -v C48X-Series":      {Stdout: "device for C48X-Series: usb://Samsung/C48x%20Series?serial=Z9MJBAK\n"},
			"lpstat -t":                  {ExitCode: 0},
			"avahi-browse -rt _ipp._tcp": {ExitCode: 0},
		}),
		CommandLogPath: "/var/log/c48x-airbridge/commands.log",
	})

	// When
	plan := planCUPSSetup(fixture)
	result, err := runner.Run(context.Background(), plan.Steps)

	// Then
	if err != nil {
		t.Fatalf("CUPS runner failed: %v", err)
	}
	if result.State != runnerStatePass {
		t.Fatalf("CUPS runner state = %q, want PASS", result.State)
	}
	logData := readCommandLog(t, root)
	if strings.Contains(logData, "lpadmin") {
		t.Fatalf("CUPS runner repaired an already compatible queue:\n%s", logData)
	}
}

func Test_CUPSSetupPlan_repairsExistingQueue_whenQueueDeviceDiffers(t *testing.T) {
	// Given
	root := t.TempDir()
	fixture := setupRunnerFixture{}
	runner := NewSafeHostRunner(SafeHostRunnerConfig{
		Root: root,
		CommandRunner: NewFakeCommandRunner(map[string]FakeCommandResult{
			"apt-get install -y cups cups-client avahi-daemon avahi-utils printer-driver-splix system-config-printer": {ExitCode: 0},
			"systemctl enable --now cups":                                {ExitCode: 0},
			"systemctl enable --now avahi-daemon":                        {ExitCode: 0},
			"cupsctl --share-printers --no-remote-admin --no-remote-any": {ExitCode: 0},
			"lpinfo -v":             {Stdout: "direct usb://Samsung/C48x%20Series?serial=Z9MJBAK\n"},
			"lpstat -v C48X-Series": {Stdout: "device for C48X-Series: usb://Other/Printer?serial=OLD\n"},
			"lpadmin -p C48X-Series -E -v usb://Samsung/C48x%20Series?serial=Z9MJBAK -m drv:///splix-samsung.drv/samsung-c48x.ppd": {ExitCode: 0},
			"lpstat -t":                  {ExitCode: 0},
			"avahi-browse -rt _ipp._tcp": {ExitCode: 0},
		}),
		CommandLogPath: "/var/log/c48x-airbridge/commands.log",
	})

	// When
	plan := planCUPSSetup(fixture)
	result, err := runner.Run(context.Background(), plan.Steps)

	// Then
	if err != nil {
		t.Fatalf("CUPS runner failed: %v", err)
	}
	if result.State != runnerStatePass {
		t.Fatalf("CUPS runner state = %q, want PASS", result.State)
	}
	logData := readCommandLog(t, root)
	if !strings.Contains(logData, "sudo lpadmin -p C48X-Series -E -v usb://Samsung/C48x%20Series?serial=Z9MJBAK") {
		t.Fatalf("CUPS runner did not repair mismatched queue:\n%s", logData)
	}
}

func Test_CUPSSetupPlan_blocksWithRemediation_whenUSBPrinterMissing(t *testing.T) {
	// Given
	fixture := setupRunnerFixture{}
	runner := NewSafeHostRunner(SafeHostRunnerConfig{
		Root: t.TempDir(),
		CommandRunner: NewFakeCommandRunner(map[string]FakeCommandResult{
			"apt-get install -y cups cups-client avahi-daemon avahi-utils printer-driver-splix system-config-printer": {ExitCode: 0},
			"systemctl enable --now cups":                                {ExitCode: 0},
			"systemctl enable --now avahi-daemon":                        {ExitCode: 0},
			"cupsctl --share-printers --no-remote-admin --no-remote-any": {ExitCode: 0},
			"lpinfo -v": {Stdout: "network ipp://printer.example.test/ipp/print\n"},
		}),
		CommandLogPath: "/var/log/c48x-airbridge/commands.log",
	})

	// When
	plan := planCUPSSetup(fixture)
	result, err := runner.Run(context.Background(), plan.Steps)

	// Then
	if err != nil {
		t.Fatalf("CUPS runner returned process error: %v", err)
	}
	if result.State != setupStateBlockedPrinterRequired {
		t.Fatalf("CUPS runner state = %q, want %s", result.State, setupStateBlockedPrinterRequired)
	}
	if !strings.Contains(result.Reason, "connect or power on") {
		t.Fatalf("CUPS runner reason did not include remediation: %q", result.Reason)
	}
}

func Test_SetupCUPSFakeRunnerApply_reportsPassAndCommandLog_whenFixtureSucceeds(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	fixture := filepath.Join("..", "..", "testdata", "setup", "cups-success.json")
	cmd.SetArgs([]string{"setup", "--yes", "--component", "cups", "--fake-runner", fixture})
	t.Setenv("C48X_AIRBRIDGE_FAKE_ROOT", t.TempDir())

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("setup cups fake-runner returned process error: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"fixture: cups-success",
		"state: PASS",
		"sudo lpadmin -p C48X-Series",
		"sudo cupsctl --share-printers --no-remote-admin --no-remote-any",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("setup cups output missing %q:\n%s", want, got)
		}
	}
}

func Test_SetupCUPSFakeRunnerApply_reportsBlocked_whenPrinterFixtureMissing(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	fixture := filepath.Join("..", "..", "testdata", "setup", "no-printer.json")
	cmd.SetArgs([]string{"setup", "--yes", "--component", "cups", "--fake-runner", fixture})
	t.Setenv("C48X_AIRBRIDGE_FAKE_ROOT", t.TempDir())

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("setup no-printer fake-runner returned process error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "state: "+setupStateBlockedPrinterRequired) {
		t.Fatalf("setup no-printer output missing blocked state:\n%s", got)
	}
	if strings.Contains(got, "state: PASS") {
		t.Fatalf("setup no-printer reported false PASS:\n%s", got)
	}
	if !strings.Contains(got, "connect or power on") {
		t.Fatalf("setup no-printer output missing remediation:\n%s", got)
	}
}

func displayLines(steps []HostStep) string {
	lines := make([]string, 0, len(steps))
	for _, step := range steps {
		switch typed := step.(type) {
		case commandStep:
			lines = append(lines, typed.command.DisplayLine())
		case cupsQueueStep:
			lines = append(lines, "dynamic CUPS queue: lpinfo -v -> lpstat -v "+typed.queueName+" -> sudo lpadmin -p "+typed.queueName)
		}
	}
	return strings.Join(lines, "\n")
}

func readCommandLog(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "var", "log", "c48x-airbridge", "commands.log"))
	if err != nil {
		t.Fatalf("read command log: %v", err)
	}
	return string(data)
}
