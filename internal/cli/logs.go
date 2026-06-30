package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newLogsCommand(streams Streams) *cobra.Command {
	var fixturePath string
	var outputDir string
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Create a log bundle metadata directory from evidence state",
		RunE: func(cmd *cobra.Command, args []string) error {
			if outputDir == "" {
				return errors.New("logs requires --output")
			}
			fixture, err := loadFixture(fixturePath)
			if err != nil {
				return err
			}
			result := evaluateFixture(fixture)
			if err := writeBundleMetadata(outputDir, fixture, result); err != nil {
				return err
			}
			_, err = fmt.Fprintf(streams.Out, "log bundle: %s\nstate: %s\n", outputDir, result.State)
			return err
		},
	}
	cmd.Flags().StringVar(&fixturePath, "fixture", "", "path to host/manual evidence fixture")
	cmd.Flags().StringVar(&outputDir, "output", "", "directory for log bundle metadata")
	return cmd
}

func writeBundleMetadata(outputDir string, fixture verifyFixture, result verifyResult) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create log bundle directory: %w", err)
	}
	metadata := logBundleMetadata{
		State:                      result.State,
		Reason:                     result.Reason,
		Gates:                      result.Gates,
		LogBundleReferences:        fixture.LogBundleReferences,
		Host:                       fixture.Host,
		ManualEvidence:             fixture.ManualEvidence,
		FeatureLimits:              fixture.FeatureLimits,
		NoVendorClientDriverPolicy: fixture.NoVendorClientDriverPolicy,
	}
	data := new(bytes.Buffer)
	encoder := json.NewEncoder(data)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metadata); err != nil {
		return fmt.Errorf("encode log bundle metadata: %w", err)
	}
	metadataPath := filepath.Join(outputDir, "metadata.json")
	if err := os.WriteFile(metadataPath, data.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write log bundle metadata: %w", err)
	}
	return nil
}
