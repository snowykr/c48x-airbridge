package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func Test_InstallHelp_documentsPlanningOnlyDryRun_whenRequested(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"install", "--help"})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("install help returned error: %v", err)
	}
	got := out.String()
	for _, want := range []string{"planning-only", "Repository QA does not run privileged or mutating install workflows", "--dry-run"} {
		if !strings.Contains(got, want) {
			t.Fatalf("install help output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Plan or run") {
		t.Fatalf("install help still advertises running install:\n%s", got)
	}
}

func Test_Install_rejectsNonDryRunAsPlanningOnly_whenRequested(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"install"})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err == nil {
		t.Fatal("install without --dry-run succeeded")
	}
	if !strings.Contains(err.Error(), "planning-only in repository QA") {
		t.Fatalf("install error did not explain planning-only behavior: %v", err)
	}
}
