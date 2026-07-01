package hostprobe

import "context"

func (p Prober) airSaneResults(ctx context.Context) []Result {
	return []Result{
		serviceResult(ctx, p.runner, serviceSpec{commandProbe: commandProbe{check: CheckAirSaneService, section: "AirSane", name: "AirSane service", passDetail: "service active", warnDetail: "service inactive or missing"}, service: "airsane"}),
		p.airSaneHTTPResult(ctx),
	}
}

func (p Prober) airSaneHTTPResult(ctx context.Context) Result {
	if _, err := p.runner.LookPath("curl"); err != nil {
		return Result{Check: CheckAirSaneHTTP, Section: "AirSane", Name: "AirSane HTTP", Status: StatusWarn, Detail: "curl unavailable"}
	}
	run := p.runner.Run(ctx, "curl", "-fsS", "--max-time", "2", "http://localhost:8090/")
	probe := commandProbe{check: CheckAirSaneHTTP, section: "AirSane", name: "AirSane HTTP", passDetail: "HTTP endpoint reachable", warnDetail: "HTTP endpoint unavailable"}
	return resultFromCommand(probe, run)
}
