package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_RootHelp_listsOperationalCommands_whenRequested(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"help"})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	got := out.String()
	for _, want := range []string{"diagnose", "install", "verify", "logs", "patch"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "install-scanservjs") || strings.Contains(got, "Paperless") || strings.Contains(got, "Docker") {
		t.Fatalf("help output contains deferred scope:\n%s", got)
	}
}

func Test_InstallDryRun_printsHostNativePlan_whenRequested(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"install", "--dry-run"})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("install dry-run returned error: %v", err)
	}
	got := out.String()
	for _, want := range []string{"CUPS", "Avahi", "SANE", "Samsung", "udev", "AirSane"} {
		if !strings.Contains(got, want) {
			t.Fatalf("dry-run output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "scanservjs") || strings.Contains(got, "Paperless") || strings.Contains(got, "OCR") || strings.Contains(got, "Docker") {
		t.Fatalf("dry-run output contains deferred scope:\n%s", got)
	}
}

func Test_Verify_reportsPendingManualEvidence_whenHostReadyFixtureLacksClients(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	fixture := filepath.Join("..", "..", "testdata", "host-ready-missing-clients.json")
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"verify", "--fixture", fixture})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("verify returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "BLOCKED_PENDING_MANUAL_EVIDENCE") {
		t.Fatalf("verify output missing blocker state:\n%s", got)
	}
	if strings.Contains(got, "PASS") {
		t.Fatalf("verify output inferred PASS without manual evidence:\n%s", got)
	}
}

func Test_Verify_reportsFail_whenManualEvidenceFails(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	fixture := writeTempJSON(t, `{
		"host": {
			"ubuntu_version": "24.04",
			"architecture": "amd64",
			"cups_major_version": 2,
			"usb_vid_pid": "04e8:3433",
			"local_flatbed_scan_checksum": "sha256:1111111111111111111111111111111111111111111111111111111111111111",
			"sane_triplet": {
				"current_user": true,
				"root": true,
				"saned": true
			},
			"airsane_http": true,
			"uscan_mdns": true,
			"cups_queue": true,
			"ipp_mdns": true,
			"reboot_persistence": true
		},
		"manual_evidence": {
			"macos_print": {"name": "macOS print", "result": "PASS"},
			"windows_print": {"name": "Windows print", "result": "PASS"},
			"macos_scan": {"name": "macOS scan", "result": "FAIL"},
			"windows_scan": {"name": "Windows scan", "result": "PASS"}
		}
	}`)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"verify", "--fixture", fixture})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("verify returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "state: FAIL") {
		t.Fatalf("verify output did not fail failed manual evidence:\n%s", got)
	}
}

func Test_Verify_reportsPendingManualEvidence_whenManualEvidenceHasNoProof(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	fixture := writeTempJSON(t, `{
		"host": {
			"ubuntu_version": "24.04",
			"architecture": "amd64",
			"cups_major_version": 2,
			"usb_vid_pid": "04e8:3433",
			"local_flatbed_scan_checksum": "sha256:1111111111111111111111111111111111111111111111111111111111111111",
			"sane_triplet": {
				"current_user": true,
				"root": true,
				"saned": true
			},
			"airsane_http": true,
			"uscan_mdns": true,
			"cups_queue": true,
			"ipp_mdns": true,
			"reboot_persistence": true
		},
		"manual_evidence": {
			"macos_print": {"name": "macOS print", "result": "PASS"},
			"windows_print": {"name": "Windows print", "result": "PASS"},
			"macos_scan": {"name": "macOS scan", "result": "PASS"},
			"windows_scan": {"name": "Windows scan", "result": "PASS"}
		}
	}`)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"verify", "--fixture", fixture})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("verify returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "state: BLOCKED_PENDING_MANUAL_EVIDENCE") {
		t.Fatalf("verify output did not block proofless manual evidence:\n%s", got)
	}
	if strings.Contains(got, "state: PASS") {
		t.Fatalf("verify output inferred PASS from proofless manual evidence:\n%s", got)
	}
}

func Test_Verify_reportsFail_whenHostEvidenceFieldsAreMalformed(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	fixture := writeTempJSON(t, `{
		"host": {
			"ubuntu_version": "noble",
			"architecture": "x86_64",
			"cups_major_version": 1,
			"usb_vid_pid": "Samsung C480",
			"local_flatbed_scan_checksum": "1111111111111111111111111111111111111111111111111111111111111111",
			"sane_triplet": {
				"current_user": true,
				"root": true,
				"saned": true
			},
			"airsane_http": true,
			"uscan_mdns": true,
			"cups_queue": true,
			"ipp_mdns": true,
			"reboot_persistence": true
		},
		"manual_evidence": {
			"macos_print": {"name": "macOS print", "result": "PASS", "discovery_proof": "printer C480 visible in macOS Add Printer", "timestamp": "2026-06-30T00:00:00Z", "log_bundle_id": "macos-print-20260630"},
			"windows_print": {"name": "Windows print", "result": "PASS", "discovery_proof": "printer C480 visible in Windows Printers & scanners", "timestamp": "2026-06-30T00:00:00Z", "log_bundle_id": "windows-print-20260630"},
			"macos_scan": {"name": "macOS scan", "result": "PASS", "discovery_proof": "scanner C480 visible in Image Capture", "timestamp": "2026-06-30T00:00:00Z", "log_bundle_id": "macos-scan-20260630"},
			"windows_scan": {"name": "Windows eSCL scan", "result": "PASS", "discovery_proof": "scanner C480 visible in a non-Samsung eSCL client", "timestamp": "2026-06-30T00:00:00Z", "log_bundle_id": "windows-scan-20260630"}
		}
	}`)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"verify", "--fixture", fixture})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("verify returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "state: FAIL") {
		t.Fatalf("verify output did not fail malformed host evidence:\n%s", got)
	}
	if strings.Contains(got, "state: PASS") {
		t.Fatalf("verify output inferred PASS from malformed host evidence:\n%s", got)
	}
}

func Test_Logs_writesBundleMetadata_whenFixtureProvided(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	bundleDir := t.TempDir()
	fixture := filepath.Join("..", "..", "testdata", "host-ready-missing-clients.json")
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"logs", "--fixture", fixture, "--output", bundleDir})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("logs returned error: %v", err)
	}
	metadataPath := filepath.Join(bundleDir, "metadata.json")
	data, readErr := os.ReadFile(metadataPath)
	if readErr != nil {
		t.Fatalf("metadata not written: %v", readErr)
	}
	got := string(data)
	if !strings.Contains(got, "BLOCKED_PENDING_MANUAL_EVIDENCE") || !strings.Contains(got, "manual client evidence missing") {
		t.Fatalf("metadata missing blocker fields:\n%s", got)
	}
}

func Test_PatchBuild_rejectsMissingGateMetadata_whenMetadataIncomplete(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	metadata := filepath.Join("..", "..", "testdata", "patch-missing-gate.json")
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"patch", "build", "--metadata", metadata})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err == nil {
		t.Fatal("patch build succeeded without failed gate metadata")
	}
	if !strings.Contains(err.Error(), "failed gate") {
		t.Fatalf("patch build error did not name missing failed gate: %v", err)
	}
}

func Test_PatchBuildDryRun_printsCompleteMetadata_whenMetadataComplete(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	metadata := filepath.Join("..", "..", "testdata", "patch-complete.json")
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"patch", "build", "--metadata", metadata, "--dry-run"})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("patch build dry-run returned error: %v", err)
	}
	got := out.String()
	for _, want := range []string{"AirSane 0.4.3", "gate-airscan-client-macos", "dist/airsane-c480.deb", "systemctl revert airsane.service"} {
		if !strings.Contains(got, want) {
			t.Fatalf("patch dry-run output missing %q:\n%s", want, got)
		}
	}
}

func writeTempJSON(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}
