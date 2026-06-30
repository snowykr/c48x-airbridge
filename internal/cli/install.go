package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newInstallCommand(streams Streams) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "install --dry-run",
		Short: "Print the host-native CUPS/SANE/AirSane install plan",
		Long: strings.TrimSpace(`
Print the planning-only host-native CUPS/SANE/AirSane install plan.

Repository QA does not run privileged or mutating install workflows. Use
--dry-run to review the commands and verification gates before applying any
manual host changes outside this repository.
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !dryRun {
				return errors.New("install is planning-only in repository QA; rerun with --dry-run to print the plan")
			}
			_, err := fmt.Fprintln(streams.Out, strings.Join(installPlan(), "\n"))
			return err
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the planning-only host-native install plan without mutating the host")
	return cmd
}

func installPlan() []string {
	return []string{
		"install dry-run plan:",
		"- CUPS and Avahi: install cups, cups-client, avahi-daemon, avahi-utils; enable printer sharing",
		"- SANE/Samsung: install sane-utils, libsane, USB support, Samsung smfp driver source if selected",
		"- udev: install product-id-pinned Samsung C480 scanner permissions rule and reload udev",
		"- AirSane: build/install AirSane, install access config, enable airsane.service",
		"- verify: run scanimage triplet, AirSane HTTP, _uscan._tcp, CUPS queue, _ipp._tcp, and manual client evidence gates",
	}
}
