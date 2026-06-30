package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newDiagnoseCommand(streams Streams) *cobra.Command {
	return &cobra.Command{
		Use:   "diagnose",
		Short: "Run non-mutating USB/CUPS/SANE/AirSane checks",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(streams.Out, strings.Join([]string{
				"diagnose plan:",
				"- USB identity: lsusb for Samsung C480 VID/PID",
				"- CUPS and Avahi: lpstat, cupsctl, avahi-browse _ipp._tcp",
				"- SANE/Samsung: scanimage triplet and smfp backend",
				"- AirSane/eSCL: service, HTTP, and _uscan._tcp",
			}, "\n"))
			return err
		},
	}
}
