package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func Test_CUPSSetupPlan_plansPackageServiceSharingAndQueueCommands_whenUSBPrinterPresent(t *testing.T) {
	// Given
	fixture := setupRunnerFixture{
		ProbeOutputs: map[string]string{
			"lpinfo -v": "direct usb://Samsung/C48x%20Series?serial=Z9MJBAK\n",
		},
	}

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
		"lpinfo -v",
		"sudo lpadmin -p C48X-Series -E -v usb://Samsung/C48x%20Series?serial=Z9MJBAK -m drv:///splix-samsung.drv/samsung-c48x.ppd",
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
	fixture := setupRunnerFixture{
		ProbeOutputs: map[string]string{
			"lpinfo -v":             "direct usb://Samsung/C48x%20Series?serial=Z9MJBAK\n",
			"lpstat -v C48X-Series": "device for C48X-Series: usb://Samsung/C48x%20Series?serial=Z9MJBAK\n",
		},
	}

	// When
	plan := planCUPSSetup(fixture)

	// Then
	got := displayLines(plan.Steps)
	if strings.Contains(got, "lpadmin") {
		t.Fatalf("CUPS plan repaired an already compatible queue:\n%s", got)
	}
}

func Test_CUPSSetupPlan_repairsExistingQueue_whenQueueDeviceDiffers(t *testing.T) {
	// Given
	fixture := setupRunnerFixture{
		ProbeOutputs: map[string]string{
			"lpinfo -v":             "direct usb://Samsung/C48x%20Series?serial=Z9MJBAK\n",
			"lpstat -v C48X-Series": "device for C48X-Series: usb://Other/Printer?serial=OLD\n",
		},
	}

	// When
	plan := planCUPSSetup(fixture)

	// Then
	got := displayLines(plan.Steps)
	if !strings.Contains(got, "sudo lpadmin -p C48X-Series -E -v usb://Samsung/C48x%20Series?serial=Z9MJBAK") {
		t.Fatalf("CUPS plan did not repair mismatched queue:\n%s", got)
	}
}

func Test_CUPSSetupPlan_blocksWithRemediation_whenUSBPrinterMissing(t *testing.T) {
	// Given
	fixture := setupRunnerFixture{
		ProbeOutputs: map[string]string{
			"lpinfo -v": "network ipp://printer.example.test/ipp/print\n",
		},
	}

	// When
	plan := planCUPSSetup(fixture)

	// Then
	if plan.State != setupStateBlockedPrinterRequired {
		t.Fatalf("CUPS plan state = %q, want %s", plan.State, setupStateBlockedPrinterRequired)
	}
	if !strings.Contains(plan.Reason, "connect or power on") {
		t.Fatalf("CUPS plan reason did not include remediation: %q", plan.Reason)
	}
	if len(plan.Steps) != 0 {
		t.Fatalf("blocked CUPS plan still returned host steps: %v", displayLines(plan.Steps))
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
		if command, ok := step.(commandStep); ok {
			lines = append(lines, command.command.DisplayLine())
		}
	}
	return strings.Join(lines, "\n")
}
