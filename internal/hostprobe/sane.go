package hostprobe

import (
	"context"
	"strings"
)

type scanimageSpec struct {
	check         CheckID
	name          string
	command       string
	args          []string
	missingDetail string
}

func (p Prober) saneResults(ctx context.Context) []Result {
	return []Result{
		p.scanimageResult(ctx, scanimageSpec{check: CheckSANECurrentUser, name: "scanimage current user", command: "scanimage", args: []string{"-L"}, missingDetail: "scanimage unavailable; install sane-utils"}),
		p.scanimageResult(ctx, scanimageSpec{check: CheckSANERoot, name: "scanimage root", command: "sudo", args: []string{"scanimage", "-L"}, missingDetail: "sudo unavailable for root scanimage check"}),
		p.sanedResult(ctx),
		p.smfpConfigResult(),
		p.smfpBackendResult(),
	}
}

func (p Prober) scanimageResult(ctx context.Context, spec scanimageSpec) Result {
	if _, err := p.runner.LookPath(spec.command); err != nil {
		return Result{Check: spec.check, Section: "SANE", Name: spec.name, Status: StatusWarn, Detail: spec.missingDetail}
	}
	run := p.runner.Run(ctx, spec.command, spec.args...)
	return resultFromCommand(scanimageProbe(spec), run)
}

func (p Prober) sanedResult(ctx context.Context) Result {
	idResult := p.runner.Run(ctx, "id", "saned")
	if idResult.Err != nil || idResult.ExitCode != 0 {
		return Result{Check: CheckSANESaned, Section: "SANE", Name: "scanimage saned", Status: StatusWarn, Detail: "saned user not found"}
	}
	spec := scanimageSpec{check: CheckSANESaned, name: "scanimage saned", command: "sudo", args: []string{"-u", "saned", "scanimage", "-L"}, missingDetail: "sudo unavailable for saned scanimage check"}
	return p.scanimageResult(ctx, spec)
}

func (p Prober) smfpConfigResult() Result {
	content, err := p.runner.ReadFile("/etc/sane.d/dll.conf")
	if err != nil {
		return Result{Check: CheckSMFPConfig, Section: "SANE", Name: "smfp dll.conf", Status: StatusWarn, Detail: "unable to read /etc/sane.d/dll.conf"}
	}
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == "smfp" {
			return Result{Check: CheckSMFPConfig, Section: "SANE", Name: "smfp dll.conf", Status: StatusPass, Detail: "smfp backend enabled", Evidence: "smfp"}
		}
	}
	return Result{Check: CheckSMFPConfig, Section: "SANE", Name: "smfp dll.conf", Status: StatusWarn, Detail: "smfp backend entry missing"}
}

func (p Prober) smfpBackendResult() Result {
	paths := globAll(p.runner, []string{"/usr/lib/sane/libsane-smfp.so.1", "/usr/lib/*/sane/libsane-smfp.so.1"})
	if len(paths) == 0 {
		return Result{Check: CheckSMFPBackend, Section: "SANE", Name: "smfp backend library", Status: StatusWarn, Detail: "libsane-smfp.so.1 missing"}
	}
	return Result{Check: CheckSMFPBackend, Section: "SANE", Name: "smfp backend library", Status: StatusPass, Detail: "backend library found", Evidence: strings.Join(paths, ", ")}
}

func scanimageProbe(spec scanimageSpec) commandProbe {
	return commandProbe{check: spec.check, section: "SANE", name: spec.name, passDetail: "scanner visible", warnDetail: "scanner not visible"}
}
