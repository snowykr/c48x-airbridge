package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func Test_SetupFakeRunnerApply_reportsFailureGuidance_whenCommandFails(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	fixture := filepath.Join("..", "..", "testdata", "setup", "command-failure.json")
	cmd.SetArgs([]string{"setup", "--yes", "--component", "cups", "--fake-runner", fixture})
	t.Setenv("C48X_AIRBRIDGE_FAKE_ROOT", t.TempDir())

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("setup fake-runner apply returned process error: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"setup apply result:",
		"state: FAIL",
		"rollback:",
		"retry:",
		"command log:",
		"sudo apt-get install -y cups",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("setup fake-runner output missing %q:\n%s", want, got)
		}
	}
}
