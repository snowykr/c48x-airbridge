package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"c48x-airbridge/internal/hostprobe"
)

func Test_VerifyLive_writesEvidenceAndPrintsClientHandoff_whenHostIsReady(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	root := t.TempDir()
	output := filepath.Join(root, "verify", "host.json")
	cmd := newVerifyCommandWithProbe(Streams{Out: out, Err: errOut}, staticVerifyProbe(readyProbeReport(root)))
	cmd.SetArgs([]string{"--live", "--output", output})
	t.Setenv("C48X_AIRBRIDGE_FAKE_ROOT", root)

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("verify live returned error: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"state: " + setupStateBlockedClientProof,
		"macOS client handoff:",
		"Windows client handoff:",
		"Add the Samsung C48x IPP printer",
		"scan with Image Capture",
		"scan with a Windows eSCL-compatible app",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("verify live output missing %q:\n%s", want, got)
		}
	}
	bundle := readVerifyOutput(t, output)
	for _, want := range []string{
		`"state": "BLOCKED_CLIENT_PROOF"`,
		`"mode": "live"`,
		`"redacted": true`,
		`"client_handoff"`,
		`"check": "cups.queue"`,
		`"check": "airsane.http"`,
	} {
		if !strings.Contains(bundle, want) {
			t.Fatalf("verify evidence missing %q:\n%s", want, bundle)
		}
	}
	if strings.Contains(bundle, root) {
		t.Fatalf("verify evidence leaked local path %q:\n%s", root, bundle)
	}
}

func Test_VerifyLive_blocksPrinterPowerWithoutClientHandoff_whenUSBIsMissing(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	output := filepath.Join(t.TempDir(), "missing-printer.json")
	cmd := newVerifyCommandWithProbe(Streams{Out: out, Err: errOut}, staticVerifyProbe(missingPrinterProbeReport()))
	cmd.SetArgs([]string{"--live", "--output", output})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("verify live missing printer returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "state: "+setupStateBlockedPrinterRequired) || !strings.Contains(got, "connect or power on") {
		t.Fatalf("verify live did not block missing printer power:\n%s", got)
	}
	if strings.Contains(got, "client handoff") {
		t.Fatalf("verify live printed client handoff before host readiness:\n%s", got)
	}
	bundle := readVerifyOutput(t, output)
	if !strings.Contains(bundle, `"state": "BLOCKED_PRINTER_REQUIRED"`) {
		t.Fatalf("verify evidence did not record printer blocker:\n%s", bundle)
	}
}

func Test_VerifyLive_blocksWithoutClientHandoff_whenAirSaneIsMissing(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	output := filepath.Join(t.TempDir(), "missing-airsane.json")
	cmd := newVerifyCommandWithProbe(Streams{Out: out, Err: errOut}, staticVerifyProbe(missingAirSaneProbeReport()))
	cmd.SetArgs([]string{"--live", "--output", output})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("verify live missing AirSane returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "state: "+verifyStateBlockedHost) || !strings.Contains(got, "AirSane HTTP") {
		t.Fatalf("verify live did not fail missing AirSane:\n%s", got)
	}
	if strings.Contains(got, "client handoff") {
		t.Fatalf("verify live printed client handoff before AirSane readiness:\n%s", got)
	}
}

func Test_VerifyLive_allowsMissingSanedUser_whenCurrentAndRootSANEPass(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	output := filepath.Join(t.TempDir(), "missing-saned.json")
	report := readyProbeReport("")
	replaceProbeResult(&report, hostprobe.Result{Check: hostprobe.CheckSANESaned, Section: "SANE", Name: "scanimage saned", Status: hostprobe.StatusWarn, Detail: "saned user not found"})
	cmd := newVerifyCommandWithProbe(Streams{Out: out, Err: errOut}, staticVerifyProbe(report))
	cmd.SetArgs([]string{"--live", "--output", output})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("verify live missing saned user returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "state: "+setupStateBlockedClientProof) {
		t.Fatalf("verify live treated optional saned account as required:\n%s", got)
	}
	bundle := readVerifyOutput(t, output)
	if !strings.Contains(bundle, `"check": "sane.saned"`) || !strings.Contains(bundle, `"status": "WARN"`) {
		t.Fatalf("verify evidence did not retain optional saned warning:\n%s", bundle)
	}
}

func Test_VerifyFixture_preservesPendingManualEvidenceAndWritesOutput_whenLegacyFixtureIsUsed(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	output := filepath.Join(t.TempDir(), "legacy.json")
	fixture := filepath.Join("..", "..", "testdata", "host-ready-missing-clients.json")
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"verify", "--fixture", fixture, "--output", output})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("verify fixture returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "state: BLOCKED_PENDING_MANUAL_EVIDENCE") {
		t.Fatalf("verify fixture did not preserve legacy pending state:\n%s", got)
	}
	if strings.Contains(got, setupStateBlockedClientProof) {
		t.Fatalf("verify fixture rewrote legacy pending state:\n%s", got)
	}
	bundle := readVerifyOutput(t, output)
	if !strings.Contains(bundle, `"state": "BLOCKED_PENDING_MANUAL_EVIDENCE"`) {
		t.Fatalf("verify fixture output did not preserve pending state:\n%s", bundle)
	}
}

func Test_VerifyFixture_suppressesClientHandoff_whenHostIsMissingAirSane(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	output := filepath.Join(t.TempDir(), "missing-airsane.json")
	fixture := filepath.Join("testdata", "verify_host_missing_airsane.json")
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"verify", "--fixture", fixture, "--output", output})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("verify missing AirSane fixture returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "state: FAIL") {
		t.Fatalf("verify missing AirSane fixture did not fail:\n%s", got)
	}
	if strings.Contains(got, "client handoff") || strings.Contains(got, "Add the Samsung C48x IPP printer") {
		t.Fatalf("verify missing AirSane fixture printed client-ready instructions:\n%s", got)
	}
	bundle := readVerifyOutput(t, output)
	if !strings.Contains(bundle, `"state": "FAIL"`) {
		t.Fatalf("verify missing AirSane output did not record failure:\n%s", bundle)
	}
}

func readyProbeReport(path string) hostprobe.Report {
	return hostprobe.Report{Results: []hostprobe.Result{
		{Check: hostprobe.CheckOSRelease, Section: "Host", Name: "OS release", Status: hostprobe.StatusPass, Evidence: "Ubuntu 24.04 LTS"},
		{Check: hostprobe.CheckArchitecture, Section: "Host", Name: "Architecture", Status: hostprobe.StatusPass, Evidence: "x86_64"},
		{Check: hostprobe.CheckUSBDevice, Section: "USB", Name: "Samsung C48x USB device", Status: hostprobe.StatusPass, Detail: "candidate found", Evidence: "Bus 001 Device 004: ID 04e8:3469 Samsung C48x " + path},
		{Check: hostprobe.CheckCUPSService, Section: "CUPS", Name: "CUPS service", Status: hostprobe.StatusPass, Detail: "service active"},
		{Check: hostprobe.CheckCUPSQueue, Section: "CUPS", Name: "CUPS queue", Status: hostprobe.StatusPass, Detail: "printer queue visible", Evidence: "printer C48X-Series is idle"},
		{Check: hostprobe.CheckAvahiService, Section: "CUPS", Name: "Avahi service", Status: hostprobe.StatusPass, Detail: "service active"},
		{Check: hostprobe.CheckIPPService, Section: "CUPS", Name: "IPP printer mDNS", Status: hostprobe.StatusPass, Detail: "service advertised", Evidence: "_ipp._tcp Samsung C48x"},
		{Check: hostprobe.CheckSANECurrentUser, Section: "SANE", Name: "scanimage current user", Status: hostprobe.StatusPass, Detail: "scanner visible"},
		{Check: hostprobe.CheckSANERoot, Section: "SANE", Name: "scanimage root", Status: hostprobe.StatusPass, Detail: "scanner visible"},
		{Check: hostprobe.CheckSANESaned, Section: "SANE", Name: "scanimage saned", Status: hostprobe.StatusPass, Detail: "scanner visible"},
		{Check: hostprobe.CheckSMFPConfig, Section: "SANE", Name: "smfp dll.conf", Status: hostprobe.StatusPass, Detail: "smfp backend enabled"},
		{Check: hostprobe.CheckSMFPBackend, Section: "SANE", Name: "smfp backend library", Status: hostprobe.StatusPass, Detail: "backend library found"},
		{Check: hostprobe.CheckAirSaneService, Section: "AirSane", Name: "AirSane service", Status: hostprobe.StatusPass, Detail: "service active"},
		{Check: hostprobe.CheckAirSaneHTTP, Section: "AirSane", Name: "AirSane HTTP", Status: hostprobe.StatusPass, Detail: "HTTP endpoint reachable", Evidence: "http://localhost:8090/eSCL/ScannerStatus"},
		{Check: hostprobe.CheckUSCANService, Section: "AirSane", Name: "AirScan/eSCL mDNS", Status: hostprobe.StatusPass, Detail: "service advertised", Evidence: "_uscan._tcp Samsung C48x"},
	}}
}

func missingPrinterProbeReport() hostprobe.Report {
	report := readyProbeReport("")
	replaceProbeResult(&report, hostprobe.Result{Check: hostprobe.CheckUSBDevice, Section: "USB", Name: "Samsung C48x USB device", Status: hostprobe.StatusBlocked, Detail: "connect or power on the C48x before scanner setup"})
	return report
}

func missingAirSaneProbeReport() hostprobe.Report {
	report := readyProbeReport("")
	replaceProbeResult(&report, hostprobe.Result{Check: hostprobe.CheckAirSaneHTTP, Section: "AirSane", Name: "AirSane HTTP", Status: hostprobe.StatusWarn, Detail: "HTTP endpoint unavailable"})
	replaceProbeResult(&report, hostprobe.Result{Check: hostprobe.CheckUSCANService, Section: "AirSane", Name: "AirScan/eSCL mDNS", Status: hostprobe.StatusWarn, Detail: "service missing"})
	return report
}

func replaceProbeResult(report *hostprobe.Report, replacement hostprobe.Result) {
	for index, result := range report.Results {
		if result.Check == replacement.Check {
			report.Results[index] = replacement
			return
		}
	}
	report.Results = append(report.Results, replacement)
}

type staticVerifyProbe hostprobe.Report

func (probe staticVerifyProbe) Run(ctx context.Context) hostprobe.Report {
	return hostprobe.Report(probe)
}

func readVerifyOutput(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read verify output: %v", err)
	}
	return string(data)
}
