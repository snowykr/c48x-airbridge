package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func Test_DiagnoseFormatText_reportsStructuredProbeSections_whenRequested(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"diagnose", "--format", "text"})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("diagnose returned error: %v", err)
	}
	got := out.String()
	for _, want := range []string{"Host", "USB", "SANE", "CUPS", "AirSane"} {
		if !strings.Contains(got, want) {
			t.Fatalf("diagnose output missing %q section:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "status:") {
		t.Fatalf("diagnose output missing structured status fields:\n%s", got)
	}
}
