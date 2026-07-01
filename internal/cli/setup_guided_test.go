package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const guidedAirSaneCommit = "129cc3bf7258251a0a694dee7741285b59d88f9f"

func Test_SetupGuidedWorkflow_appliesAllComponentsAndWritesEvidence_whenYesApproved(t *testing.T) {
	// Given
	root := t.TempDir()
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	fixture := filepath.Join("..", "..", "testdata", "setup", "success.json")
	cmd.SetArgs([]string{"setup", "--yes", "--component", "all", "--fake-runner", fixture})
	t.Setenv("C48X_AIRBRIDGE_FAKE_ROOT", root)

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("guided setup returned process error: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"setup review:",
		"section: CUPS PASS",
		"section: SANE PASS",
		"section: AirSane PASS",
		"state: " + setupStateBlockedClientProof,
		"evidence bundle: /var/log/c48x-airbridge/setup-evidence.json",
		"sudo lpadmin -p C48X-Series",
		"scanimage -L",
		"git -C /tmp/c48x-airbridge/airsane/source checkout --detach " + guidedAirSaneCommit,
		"sudo rsync -a /tmp/c48x-airbridge/airsane/stage/usr/local/ /usr/local/",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("guided setup output missing %q:\n%s", want, got)
		}
	}
	evidence := readGuidedEvidence(t, root)
	for _, want := range []string{
		`"state": "BLOCKED_CLIENT_PROOF"`,
		`"component": "all"`,
		`"redacted": true`,
		`"command_log_path": "/var/log/c48x-airbridge/commands.log"`,
	} {
		if !strings.Contains(evidence, want) {
			t.Fatalf("guided evidence missing %q:\n%s", want, evidence)
		}
	}
	if strings.Contains(evidence, root) {
		t.Fatalf("guided evidence leaked fake root %q:\n%s", root, evidence)
	}
}

func Test_SetupGuidedWorkflow_appliesAfterInteractiveApproval_whenUserAnswersYes(t *testing.T) {
	// Given
	root := t.TempDir()
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	fixture := filepath.Join("..", "..", "testdata", "setup", "success.json")
	cmd.SetArgs([]string{"setup", "--component", "all", "--fake-runner", fixture})
	cmd.SetIn(strings.NewReader("yes\n"))
	t.Setenv("C48X_AIRBRIDGE_FAKE_ROOT", root)

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("interactive guided setup returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Apply setup plan?") || !strings.Contains(got, "state: "+setupStateBlockedClientProof) {
		t.Fatalf("interactive guided setup did not review and apply:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(root, "var", "log", "c48x-airbridge", "setup-evidence.json")); err != nil {
		t.Fatalf("interactive guided setup did not write evidence bundle: %v", err)
	}
}

func Test_SetupGuidedWorkflow_rejectsNoInput_whenReviewApprovalIsRequired(t *testing.T) {
	// Given
	root := t.TempDir()
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	fixture := filepath.Join("..", "..", "testdata", "setup", "success.json")
	cmd.SetArgs([]string{"setup", "--no-input", "--component", "all", "--fake-runner", fixture})
	t.Setenv("C48X_AIRBRIDGE_FAKE_ROOT", root)

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err == nil {
		t.Fatal("guided setup --no-input succeeded without --yes")
	}
	if !strings.Contains(err.Error(), "prompt required") {
		t.Fatalf("guided setup --no-input error did not name prompt requirement: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "var", "log", "c48x-airbridge", "commands.log")); !os.IsNotExist(statErr) {
		t.Fatalf("guided setup --no-input wrote command log before approval: %v", statErr)
	}
}

func Test_SetupGuidedWorkflow_blocksDriverBeforeMutation_whenScannerBackendHasNoSafeSource(t *testing.T) {
	// Given
	root := t.TempDir()
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	fixture := filepath.Join("..", "..", "testdata", "setup", "blocked-driver.json")
	cmd.SetArgs([]string{"setup", "--yes", "--component", "all", "--fake-runner", fixture})
	t.Setenv("C48X_AIRBRIDGE_FAKE_ROOT", root)

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("blocked-driver guided setup returned process error: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"state: " + setupStateBlockedDriverRequired,
		blockedDriverRequiredReason,
		"section: SANE BLOCKED_DRIVER_REQUIRED",
		"evidence bundle: /var/log/c48x-airbridge/setup-evidence.json",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("blocked-driver output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "sudo lpadmin") || strings.Contains(got, "scanimage -L") {
		t.Fatalf("blocked-driver guided setup ran mutating or verification commands:\n%s", got)
	}
}

func Test_SetupGuidedWorkflow_respectsComponentSelection_whenCUPSOnlyRequested(t *testing.T) {
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
		t.Fatalf("CUPS-only setup returned process error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "component: cups") || !strings.Contains(got, "state: PASS") {
		t.Fatalf("CUPS-only setup did not pass:\n%s", got)
	}
	for _, forbidden := range []string{"scanimage -L", "AirSane", "cmake --install"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("CUPS-only setup leaked other component %q:\n%s", forbidden, got)
		}
	}
}

func readGuidedEvidence(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "var", "log", "c48x-airbridge", "setup-evidence.json"))
	if err != nil {
		t.Fatalf("read guided evidence: %v", err)
	}
	return string(data)
}
