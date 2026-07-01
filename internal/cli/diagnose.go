package cli

import (
	"fmt"
	"strings"

	"c48x-airbridge/internal/hostprobe"

	"github.com/spf13/cobra"
)

func newDiagnoseCommand(streams Streams) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "diagnose",
		Short: "Run non-mutating USB/CUPS/SANE/AirSane checks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "text" {
				return fmt.Errorf("invalid diagnose format %q: expected text", format)
			}
			report := hostprobe.New(hostprobe.Options{}).Run(cmd.Context())
			_, err := fmt.Fprintln(streams.Out, formatDiagnoseText(report))
			return err
		},
	}
	cmd.Flags().StringVar(&format, "format", "text", "output format: text")
	return cmd
}

func formatDiagnoseText(report hostprobe.Report) string {
	sections := []string{"Host", "USB", "SANE", "CUPS", "AirSane"}
	lines := []string{"diagnose report:"}
	for _, section := range sections {
		lines = append(lines, "", section)
		for _, result := range report.Results {
			if result.Section == section {
				lines = append(lines, fmt.Sprintf("- %s status: %s", result.Name, result.Status))
				if result.Detail != "" {
					lines = append(lines, "  detail: "+result.Detail)
				}
				if result.Evidence != "" {
					lines = append(lines, "  evidence: "+result.Evidence)
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}
