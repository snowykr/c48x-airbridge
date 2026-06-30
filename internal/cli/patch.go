package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newPatchCommand(streams Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patch",
		Short: "Manage evidence-gated local patch workflows",
	}
	cmd.AddCommand(newPatchBuildCommand(streams))
	return cmd
}

func newPatchBuildCommand(streams Streams) *cobra.Command {
	var metadataPath string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Validate and plan an evidence-gated local patch build",
		RunE: func(cmd *cobra.Command, args []string) error {
			if metadataPath == "" {
				return errors.New("patch build requires --metadata")
			}
			metadata, err := loadPatchMetadata(metadataPath)
			if err != nil {
				return err
			}
			if err := metadata.validate(); err != nil {
				return err
			}
			if !dryRun {
				return errors.New("patch build requires --dry-run in repository QA")
			}
			_, err = fmt.Fprintln(streams.Out, formatPatchBuildDryRun(metadata))
			return err
		},
	}
	cmd.Flags().StringVar(&metadataPath, "metadata", "", "path to patch metadata JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate and print the build plan without running it")
	return cmd
}

func loadPatchMetadata(path string) (patchMetadata, error) {
	return readJSON[patchMetadata](path)
}

func formatPatchBuildDryRun(metadata patchMetadata) string {
	return fmt.Sprintf(strings.Join([]string{
		"patch build dry-run:",
		"upstream: %s (%s)",
		"failed gate: %s",
		"local diff: %s",
		"build command: %s",
		"build result: %s",
		"artifact: %s",
		"rollback: %s",
	}, "\n"), metadata.UpstreamVersion, metadata.UpstreamSource, metadata.FailedGateID, metadata.LocalDiff, metadata.BuildCommand, metadata.BuildResult, metadata.ArtifactPath, metadata.RollbackNote)
}

func (metadata patchMetadata) validate() error {
	if missingPatchMetadata(metadata.FailedGateID) {
		return errMissingPatchMetadata
	}
	required := map[string]string{
		"upstream version": metadata.UpstreamVersion,
		"upstream source":  metadata.UpstreamSource,
		"local diff":       metadata.LocalDiff,
		"build command":    metadata.BuildCommand,
		"build result":     metadata.BuildResult,
		"artifact path":    metadata.ArtifactPath,
		"rollback note":    metadata.RollbackNote,
	}
	for label, value := range required {
		if missingPatchMetadata(value) {
			return fmt.Errorf("patch metadata missing %s", label)
		}
	}
	return nil
}

func missingPatchMetadata(value string) bool {
	return strings.TrimSpace(value) == ""
}
