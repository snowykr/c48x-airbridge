package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Verify_reportsPass_whenHostAndManualClientProofsAreComplete(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	fixture := filepath.Join("..", "..", "testdata", "host-ready-clients-pass.json")
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"verify", "--fixture", fixture})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("verify returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "state: PASS") {
		t.Fatalf("verify output did not pass complete manual evidence:\n%s", got)
	}
}

func Test_Verify_reportsFail_whenHostFailureFixtureProvided(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	fixture := filepath.Join("..", "..", "testdata", "host-failure.json")
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
		t.Fatalf("verify output did not fail host failure fixture:\n%s", got)
	}
	if strings.Contains(got, "state: PASS") {
		t.Fatalf("verify output inferred PASS from failed host fixture:\n%s", got)
	}
}

func Test_Verify_returnsParseError_whenFixtureIsMalformed(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	fixture := writeTempJSON(t, `{"host":`)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"verify", "--fixture", fixture})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err == nil {
		t.Fatal("verify succeeded with malformed fixture")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Fatalf("verify error did not identify parse failure: %v", err)
	}
}

func Test_Logs_overwritesStaleMetadataWithAuditFields_whenOutputDirectoryExists(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	bundleDir := t.TempDir()
	metadataPath := filepath.Join(bundleDir, "metadata.json")
	if err := os.WriteFile(metadataPath, []byte(`{"state":"PASS","reason":"stale"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write stale metadata: %v", err)
	}
	fixture := filepath.Join("..", "..", "testdata", "host-ready-missing-clients.json")
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	cmd.SetArgs([]string{"logs", "--fixture", fixture, "--output", bundleDir})

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("logs returned error: %v", err)
	}
	data, readErr := os.ReadFile(metadataPath)
	if readErr != nil {
		t.Fatalf("metadata not written: %v", readErr)
	}
	if len(data) == 0 {
		t.Fatal("metadata is empty")
	}
	var metadata logBundleMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("metadata is not JSON: %v\n%s", err, string(data))
	}
	if metadata.State != "BLOCKED_PENDING_MANUAL_EVIDENCE" {
		t.Fatalf("metadata state = %q, want blocker", metadata.State)
	}
	if strings.Contains(string(data), `"state":"PASS"`) || strings.Contains(metadata.Reason, "stale") {
		t.Fatalf("metadata retained stale state:\n%s", string(data))
	}
	for _, want := range []string{
		"sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"cups>=2",
		"04e8:3433",
		"_uscan._tcp",
		"_ipp._tcp",
		"CUPS Samsung_C480",
		"AirSane HTTP reachable at http://127.0.0.1:8090/eSCL",
		"scanimage current-user/root/saned all detect Samsung C480",
		"manual-client-native-drivers-only",
		"no Samsung vendor client driver installed",
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("metadata missing audit field %q:\n%s", want, string(data))
		}
	}
	if len(metadata.LogBundleReferences) == 0 {
		t.Fatalf("metadata missing log bundle references:\n%s", string(data))
	}
	if len(metadata.FeatureLimits) == 0 {
		t.Fatalf("metadata missing feature limits:\n%s", string(data))
	}
}
