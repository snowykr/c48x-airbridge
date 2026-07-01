package hostprobe

import (
	"context"
	"strings"
)

type commandProbe struct {
	check      CheckID
	section    string
	name       string
	passDetail string
	warnDetail string
}

type commandSpec struct {
	check   CheckID
	name    string
	command string
}

type serviceSpec struct {
	commandProbe
	service string
}

func resultFromCommand(spec commandProbe, run CommandResult) Result {
	if run.Err == nil && run.ExitCode == 0 {
		return Result{Check: spec.check, Section: spec.section, Name: spec.name, Status: StatusPass, Detail: spec.passDetail, Evidence: trimEvidence(run.Stdout)}
	}
	return Result{Check: spec.check, Section: spec.section, Name: spec.name, Status: StatusWarn, Detail: spec.warnDetail, Evidence: trimEvidence(run.Stderr + "\n" + run.Stdout)}
}

func serviceResult(ctx context.Context, runner Runner, spec serviceSpec) Result {
	if _, err := runner.LookPath("systemctl"); err != nil {
		return Result{Check: spec.check, Section: spec.section, Name: spec.name, Status: StatusWarn, Detail: "systemctl unavailable"}
	}
	run := runner.Run(ctx, "systemctl", "is-active", spec.service)
	return resultFromCommand(spec.commandProbe, run)
}

func globAll(runner Runner, patterns []string) []string {
	var paths []string
	for _, pattern := range patterns {
		matches, err := runner.Glob(pattern)
		if err == nil {
			paths = append(paths, matches...)
		}
	}
	return paths
}

func firstLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) != "" {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func trimEvidence(output string) string {
	return strings.TrimSpace(output)
}
