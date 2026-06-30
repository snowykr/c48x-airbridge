package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func Test_PatchBuildDryRun_rejectsWhitespaceRequiredMetadata_whenFieldBlank(t *testing.T) {
	tests := []struct {
		name      string
		fieldJSON string
		wantError string
	}{
		{"upstream version", `"upstream_version": "  "`, "upstream version"},
		{"upstream source", `"upstream_source": "\t"`, "upstream source"},
		{"failed gate", `"failed_gate_id": "\n"`, "failed gate"},
		{"local diff", `"local_diff": "  "`, "local diff"},
		{"build command", `"build_command": "\t"`, "build command"},
		{"build result", `"build_result": "\n"`, "build result"},
		{"artifact path", `"artifact_path": "  "`, "artifact path"},
		{"rollback note", `"rollback_note": "\t"`, "rollback note"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			out := new(bytes.Buffer)
			errOut := new(bytes.Buffer)
			metadata := writeTempJSON(t, patchMetadataJSON(tt.fieldJSON))
			cmd := NewCommand(Streams{Out: out, Err: errOut})
			cmd.SetArgs([]string{"patch", "build", "--metadata", metadata, "--dry-run"})

			// When
			err := cmd.ExecuteContext(context.Background())

			// Then
			if err == nil {
				t.Fatalf("patch build dry-run accepted whitespace-only %s", tt.wantError)
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("patch build error did not name %s: %v", tt.wantError, err)
			}
		})
	}
}

func patchMetadataJSON(fieldJSON string) string {
	return `{
		"upstream_version": "AirSane 0.4.3",
		"upstream_source": "https://github.com/SimulPiscator/AirSane",
		"failed_gate_id": "gate-airscan-client-macos",
		"local_diff": "patches/airsane-c480.diff",
		"build_command": "cmake --build build",
		"build_result": "dry-run",
		"artifact_path": "dist/airsane-c480.deb",
		"rollback_note": "systemctl revert airsane.service",
		` + fieldJSON + `
	}`
}
