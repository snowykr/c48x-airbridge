package hostprobe

import (
	"context"
	"regexp"
	"strings"
)

var samsungUSBPattern = regexp.MustCompile(`(?i)(04e8|samsung|c48x|c480)`)

func (p Prober) usbResults(ctx context.Context) []Result {
	if _, err := p.runner.LookPath("lsusb"); err != nil {
		return []Result{{Check: CheckUSBDevice, Section: "USB", Name: "Samsung C48x USB device", Status: StatusWarn, Detail: "lsusb unavailable; install usbutils"}}
	}
	run := p.runner.Run(ctx, "lsusb")
	if run.Err != nil || run.ExitCode != 0 {
		return []Result{{Check: CheckUSBDevice, Section: "USB", Name: "Samsung C48x USB device", Status: StatusWarn, Detail: "lsusb failed", Evidence: trimEvidence(run.Stderr)}}
	}
	if samsungUSBPattern.MatchString(run.Stdout) {
		return []Result{{Check: CheckUSBDevice, Section: "USB", Name: "Samsung C48x USB device", Status: StatusPass, Detail: "candidate found", Evidence: matchingLine(run.Stdout)}}
	}
	return []Result{{Check: CheckUSBDevice, Section: "USB", Name: "Samsung C48x USB device", Status: StatusBlocked, Detail: "connect or power on the C48x before scanner setup", Evidence: trimEvidence(run.Stdout)}}
}

func matchingLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if samsungUSBPattern.MatchString(line) {
			return strings.TrimSpace(line)
		}
	}
	return trimEvidence(output)
}
