package hostprobe

import (
	"context"
	"strings"
)

type avahiSpec struct {
	check   CheckID
	name    string
	service string
}

func (p Prober) cupsResults(ctx context.Context) []Result {
	return []Result{
		serviceResult(ctx, p.runner, serviceSpec{commandProbe: commandProbe{check: CheckCUPSService, section: "CUPS", name: "CUPS service", passDetail: "service active", warnDetail: "service inactive or missing"}, service: "cups"}),
		p.cupsQueueResult(ctx),
	}
}

func (p Prober) cupsQueueResult(ctx context.Context) Result {
	if _, err := p.runner.LookPath("lpstat"); err != nil {
		return Result{Check: CheckCUPSQueue, Section: "CUPS", Name: "CUPS queue", Status: StatusWarn, Detail: "lpstat unavailable; install cups-client"}
	}
	run := p.runner.Run(ctx, "lpstat", "-t")
	if run.Err != nil || run.ExitCode != 0 {
		return Result{Check: CheckCUPSQueue, Section: "CUPS", Name: "CUPS queue", Status: StatusWarn, Detail: "lpstat failed", Evidence: trimEvidence(run.Stderr)}
	}
	if strings.Contains(strings.ToLower(run.Stdout), "printer") {
		return Result{Check: CheckCUPSQueue, Section: "CUPS", Name: "CUPS queue", Status: StatusPass, Detail: "printer queue visible", Evidence: firstLine(run.Stdout)}
	}
	return Result{Check: CheckCUPSQueue, Section: "CUPS", Name: "CUPS queue", Status: StatusWarn, Detail: "printer queue missing", Evidence: trimEvidence(run.Stdout)}
}

func (p Prober) avahiResults(ctx context.Context) []Result {
	return []Result{
		p.avahiResult(ctx, avahiSpec{check: CheckIPPService, name: "IPP printer mDNS", service: "_ipp._tcp"}),
		p.avahiResult(ctx, avahiSpec{check: CheckUSCANService, name: "AirScan/eSCL mDNS", service: "_uscan._tcp"}),
	}
}

func (p Prober) avahiResult(ctx context.Context, spec avahiSpec) Result {
	if _, err := p.runner.LookPath("avahi-browse"); err != nil {
		return Result{Check: spec.check, Section: "CUPS", Name: spec.name, Status: StatusWarn, Detail: "avahi-browse unavailable; install avahi-utils"}
	}
	run := p.runner.Run(ctx, "avahi-browse", "-rt", spec.service)
	probe := commandProbe{check: spec.check, section: "CUPS", name: spec.name, passDetail: "service advertised", warnDetail: "service missing"}
	return resultFromCommand(probe, run)
}
