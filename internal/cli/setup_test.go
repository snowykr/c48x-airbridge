package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_RootHelp_listsSetupCommand_whenRequested(t *testing.T) {
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
	if !strings.Contains(got, "setup") {
		t.Fatalf("help output missing setup command:\n%s", got)
	}
}

func Test_SetupHelp_documentsPublicContract_whenRequested(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"setup", "--help"})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("setup help returned error: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"setup",
		"--dry-run",
		"--yes",
		"--no-input",
		"--force",
		"--suldr-deb",
		"--airsane-commit",
		"--component",
		"BLOCKED_DRIVER_REQUIRED",
		"BLOCKED_CLIENT_PROOF",
		"FAIL",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("setup help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "fake-runner") {
		t.Fatalf("setup help exposed hidden fake-runner flag:\n%s", got)
	}
}

func Test_SetupDryRun_printsReviewedPlanWithoutMutation_whenRequested(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"setup", "--dry-run", "--component", "all"})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("setup dry-run returned error: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"setup dry-run plan",
		"component: all",
		"review/apply boundary",
		"CUPS",
		"SANE",
		"AirSane",
		"BLOCKED_DRIVER_REQUIRED",
		"BLOCKED_CLIENT_PROOF",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("setup dry-run output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "state: PASS") {
		t.Fatalf("setup dry-run claimed completion:\n%s", got)
	}
}

func Test_SetupNoInput_rejectsPromptRequiredFlow_whenReviewRequired(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"setup", "--no-input"})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err == nil {
		t.Fatal("setup --no-input succeeded without --dry-run or --yes")
	}
	if !strings.Contains(err.Error(), "prompt required") {
		t.Fatalf("setup --no-input error did not name prompt requirement: %v", err)
	}
}

func Test_SetupComponent_rejectsInvalidValue_whenProvided(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"setup", "--no-input", "--component", "invalid"})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err == nil {
		t.Fatal("setup accepted invalid component")
	}
	if !strings.Contains(err.Error(), "invalid component") {
		t.Fatalf("setup component error did not name invalid component: %v", err)
	}
}

func Test_SetupFakeRunner_requiresFakeRoot_whenProvided(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	fixture := filepath.Join("testdata", "setup", "contract.json")
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"setup", "--dry-run", "--fake-runner", fixture})
	t.Setenv("C48X_AIRBRIDGE_FAKE_ROOT", "")

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err == nil {
		t.Fatal("setup accepted fake-runner without fake root")
	}
	if !strings.Contains(err.Error(), "C48X_AIRBRIDGE_FAKE_ROOT") {
		t.Fatalf("setup fake-runner error did not name fake root: %v", err)
	}
}

func Test_SetupFakeRunner_acceptsFixtureWithExistingFakeRoot_whenDryRun(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	fixtureDir := filepath.Join(t.TempDir(), "testdata", "setup")
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatalf("create fixture dir: %v", err)
	}
	fixture := filepath.Join(fixtureDir, "contract.json")
	if err := os.WriteFile(fixture, []byte(`{"name":"contract"}`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"setup", "--dry-run", "--fake-runner", fixture})
	t.Setenv("C48X_AIRBRIDGE_FAKE_ROOT", t.TempDir())

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("setup rejected fake-runner with fake root: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "fake runner fixture") {
		t.Fatalf("setup dry-run output missing fake runner fixture:\n%s", got)
	}
}

func Test_InstallDryRun_remainsCompatible_whenSetupCommandExists(t *testing.T) {
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
	if !strings.Contains(got, "install dry-run plan") {
		t.Fatalf("install dry-run output changed unexpectedly:\n%s", got)
	}
}
