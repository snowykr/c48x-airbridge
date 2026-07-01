package hostprobe

import (
	"context"
	"strings"
)

func (p Prober) usbResults(ctx context.Context) []Result {
	if _, err := p.runner.LookPath("lsusb"); err != nil {
		return []Result{{Check: CheckUSBDevice, Section: "USB", Name: "Samsung C48x USB device", Status: StatusWarn, Detail: "lsusb unavailable; install usbutils"}}
	}
	run := p.runner.Run(ctx, "lsusb")
	if run.Err != nil || run.ExitCode != 0 {
		return []Result{{Check: CheckUSBDevice, Section: "USB", Name: "Samsung C48x USB device", Status: StatusWarn, Detail: "lsusb failed", Evidence: trimEvidence(run.Stderr)}}
	}
	if samsungC48xUSBOutput(run.Stdout) {
		return []Result{{Check: CheckUSBDevice, Section: "USB", Name: "Samsung C48x USB device", Status: StatusPass, Detail: "candidate found", Evidence: matchingLine(run.Stdout)}}
	}
	return []Result{{Check: CheckUSBDevice, Section: "USB", Name: "Samsung C48x USB device", Status: StatusBlocked, Detail: "connect or power on the C48x before scanner setup", Evidence: trimEvidence(run.Stdout)}}
}

func matchingLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if samsungC48xUSBLine(line) {
			return strings.TrimSpace(line)
		}
	}
	return trimEvidence(output)
}

func samsungC48xUSBOutput(output string) bool {
	for _, line := range strings.Split(output, "\n") {
		if samsungC48xUSBLine(line) {
			return true
		}
	}
	return false
}

func samsungC48xUSBLine(line string) bool {
	normalized := strings.ToLower(line)
	hasSamsungVendor := strings.Contains(normalized, "04e8") || strings.Contains(normalized, "samsung")
	hasC48xModel := strings.Contains(normalized, "c48") || strings.Contains(normalized, "c480")
	return hasSamsungVendor && hasC48xModel
}
