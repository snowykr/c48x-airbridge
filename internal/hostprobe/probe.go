package hostprobe

import (
	"context"
)

type Options struct {
	Runner Runner
}

type Prober struct {
	runner Runner
}

func New(options Options) Prober {
	runner := options.Runner
	if runner == nil {
		runner = osRunner{}
	}
	return Prober{runner: runner}
}

func (p Prober) Run(ctx context.Context) Report {
	results := make([]Result, 0, 28)
	results = append(results, p.hostResults(ctx)...)
	results = append(results, p.commandResults()...)
	results = append(results, p.packageResults(ctx)...)
	results = append(results, p.usbResults(ctx)...)
	results = append(results, p.saneResults(ctx)...)
	results = append(results, p.cupsResults(ctx)...)
	results = append(results, p.avahiResults(ctx)...)
	results = append(results, p.airSaneResults(ctx)...)
	return Report{Results: results}
}
