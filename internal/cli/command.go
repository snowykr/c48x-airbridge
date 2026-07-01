package cli

import (
	"errors"
	"io"

	"github.com/spf13/cobra"
)

var errMissingPatchMetadata = errors.New("patch metadata missing failed gate")

type Streams struct {
	Out io.Writer
	Err io.Writer
}

func NewCommand(streams Streams) *cobra.Command {
	if streams.Out == nil {
		streams.Out = io.Discard
	}
	if streams.Err == nil {
		streams.Err = io.Discard
	}

	cmd := &cobra.Command{
		Use:           "c48x-airbridge",
		Short:         "Operate a host-native Samsung C480 printer/scanner bridge",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.SetOut(streams.Out)
	cmd.SetErr(streams.Err)
	cmd.AddCommand(newDiagnoseCommand(streams), newInstallCommand(streams), newVerifyCommand(streams), newLogsCommand(streams), newPatchCommand(streams))
	return cmd
}
