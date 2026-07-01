package cli

import (
	"os"
)

const guidedSetupEvidencePath = "/var/log/c48x-airbridge/setup-evidence.json"

type guidedSetupSection struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}

type guidedSetupPlan struct {
	State    string
	Reason   string
	Sections []guidedSetupSection
	Steps    []HostStep
}

type guidedRunRequest struct {
	setup   setupRunnerRequest
	fixture setupRunnerFixture
	root    string
	plan    guidedSetupPlan
}

func runGuidedSetupFakeRunner(request setupRunnerRequest, fixture setupRunnerFixture) error {
	root := os.Getenv("C48X_AIRBRIDGE_FAKE_ROOT")
	if err := fixture.preload(root); err != nil {
		return err
	}
	options := request.options
	plan, err := buildGuidedSetupPlan(options, fixture, root)
	if err != nil {
		return err
	}
	result := guidedBlockedResult(fixture, plan)
	if plan.State == setupStatePass {
		result, err = runGuidedSetupPlan(guidedRunRequest{
			setup:   request,
			fixture: fixture,
			root:    root,
			plan:    plan,
		})
		if err != nil {
			return err
		}
	}
	evidencePath, err := writeGuidedSetupEvidence(guidedEvidenceRequest{
		root:     root,
		options:  options,
		fixture:  fixture,
		sections: plan.Sections,
		result:   result,
	})
	if err != nil {
		return err
	}
	return printGuidedSetupResult(guidedSetupReport{
		streams:      request.streams,
		options:      options,
		fixture:      fixture,
		sections:     plan.Sections,
		result:       result,
		evidencePath: evidencePath,
	})
}

func buildGuidedSetupPlan(options setupOptions, fixture setupRunnerFixture, root string) (guidedSetupPlan, error) {
	sections := []guidedSetupSection{{Name: "preflight", State: setupStatePass, Detail: "reviewed setup inputs"}}
	cupsPlan := planCUPSSetup(fixture)
	if cupsPlan.State != setupStatePass {
		sections = append(sections, guidedSetupSection{Name: "CUPS", State: cupsPlan.State, Detail: cupsPlan.Reason})
		return guidedSetupPlan{State: cupsPlan.State, Reason: cupsPlan.Reason, Sections: sections}, nil
	}
	sections = append(sections, guidedSetupSection{Name: "CUPS", State: setupStatePass, Detail: "queue and IPP sharing planned"})

	resolution := resolveSetupDependencies(setupDependencyRequest{
		InstalledSamsungBackend: fixture.scannerBackendInstalled(),
		SULDRDeb:                options.SULDRDeb,
		AirSaneCommit:           options.AirSaneCommit,
		Metadata:                defaultSetupDependencyMetadata,
	})
	if resolution.State == setupStateBlockedDriverRequired {
		sections = append(sections, guidedSetupSection{Name: "SANE", State: setupStateBlockedDriverRequired, Detail: resolution.Reason})
		return guidedSetupPlan{State: resolution.State, Reason: resolution.Reason, Sections: sections}, nil
	}
	scannerSteps, err := fixture.scannerHostSteps(root, resolution)
	if err != nil {
		return guidedSetupPlan{}, err
	}
	sections = append(sections, guidedSetupSection{Name: "SANE", State: setupStatePass, Detail: "scanner backend and triplet checks planned"})

	airSanePlan := buildAirSaneSetupPlan(options)
	if airSanePlan.State != setupStatePass {
		sections = append(sections, guidedSetupSection{Name: "AirSane", State: airSanePlan.State, Detail: airSanePlan.Reason})
		return guidedSetupPlan{State: airSanePlan.State, Reason: airSanePlan.Reason, Sections: sections}, nil
	}
	sections = append(sections, guidedSetupSection{Name: "AirSane", State: setupStatePass, Detail: "pinned build, service, mDNS, and eSCL proof planned"})

	steps := make([]HostStep, 0, len(cupsPlan.Steps)+len(scannerSteps)+len(airSanePlan.Steps))
	steps = append(steps, cupsPlan.Steps...)
	steps = append(steps, scannerSteps...)
	steps = append(steps, airSanePlan.Steps...)
	return guidedSetupPlan{State: setupStatePass, Sections: sections, Steps: steps}, nil
}

func runGuidedSetupPlan(request guidedRunRequest) (RunResult, error) {
	runner := NewSafeHostRunner(SafeHostRunnerConfig{
		Root:           request.root,
		CommandRunner:  NewFakeCommandRunner(request.fixture.Commands),
		CommandLogPath: request.fixture.CommandLogPath,
	})
	result, err := runner.Run(request.setup.ctx, request.plan.Steps)
	if err != nil {
		return RunResult{}, err
	}
	if err := request.fixture.verifyExpectedWrites(request.root); err != nil {
		result.State = setupStateFail
		result.Reason = err.Error()
		return result, nil
	}
	if result.State == setupStatePass {
		result.State = setupStateBlockedClientProof
		result.Reason = "Linux host setup completed; macOS/Windows client print and scan proof is still required."
		result.RetryGuidance = "run verify after collecting macOS and Windows client proof"
	}
	return result, nil
}

func guidedBlockedResult(fixture setupRunnerFixture, plan guidedSetupPlan) RunResult {
	return RunResult{
		State:            plan.State,
		Reason:           plan.Reason,
		CommandLogPath:   fixture.CommandLogPath,
		RollbackGuidance: "no host changes were applied",
		RetryGuidance:    guidedRetryGuidance(plan.State),
	}
}

func guidedRetryGuidance(state string) string {
	switch state {
	case setupStateBlockedDriverRequired:
		return "provide a trusted --suldr-deb or pinned --airsane-commit, then rerun setup --yes --component all"
	case setupStateBlockedPrinterRequired:
		return "connect or power on the Samsung C48x/C480 USB printer, then rerun setup --yes --component all"
	default:
		return "correct the blocked setup input, then rerun setup --yes --component all"
	}
}
