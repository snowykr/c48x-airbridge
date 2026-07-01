package cli

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"c48x-airbridge/internal/hostprobe"

	"github.com/spf13/cobra"
)

var (
	checksumPattern    = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	ubuntuVersionReady = regexp.MustCompile(`^24\.04$`)
	usbVIDPIDPattern   = regexp.MustCompile(`^[0-9a-fA-F]{4}:[0-9a-fA-F]{4}$`)
)

type verifyOptions struct {
	FixturePath string
	Live        bool
	OutputPath  string
}

type verifyProbe interface {
	Run(ctx context.Context) hostprobe.Report
}

type verifyRunRequest struct {
	ctx     context.Context
	streams Streams
	options verifyOptions
	probe   verifyProbe
}

func newVerifyCommand(streams Streams) *cobra.Command {
	return newVerifyCommandWithProbe(streams, hostprobe.New(hostprobe.Options{}))
}

func newVerifyCommandWithProbe(streams Streams, probe verifyProbe) *cobra.Command {
	options := verifyOptions{}
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Evaluate host and manual client evidence state",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVerify(verifyRunRequest{
				ctx:     cmd.Context(),
				streams: streams,
				options: options,
				probe:   probe,
			})
		},
	}
	cmd.Flags().StringVar(&options.FixturePath, "fixture", "", "path to host/manual evidence fixture")
	cmd.Flags().BoolVar(&options.Live, "live", false, "run non-mutating live host verification")
	cmd.Flags().StringVar(&options.OutputPath, "output", "", "write structured verification evidence JSON")
	return cmd
}

func runVerify(request verifyRunRequest) error {
	if request.options.Live {
		report := request.probe.Run(request.ctx)
		result := evaluateLiveReport(report)
		bundle := newLiveVerifyBundle(result, report)
		if err := writeVerifyOutput(request.options.OutputPath, bundle); err != nil {
			return err
		}
		return printVerifyResult(request.streams, result, bundle.ClientHandoff)
	}
	fixture, err := loadFixture(request.options.FixturePath)
	if err != nil {
		return err
	}
	result := evaluateFixture(fixture)
	bundle := newFixtureVerifyBundle(result, fixture)
	if err := writeVerifyOutput(request.options.OutputPath, bundle); err != nil {
		return err
	}
	return printVerifyResult(request.streams, result, nil)
}

func loadFixture(path string) (verifyFixture, error) {
	if path == "" {
		return verifyFixture{}, errors.New("verify fixture path is required unless --live is used")
	}
	return readJSON[verifyFixture](path)
}

func evaluateFixture(fixture verifyFixture) verifyResult {
	if !hostReady(fixture.Host) {
		return verifyResult{
			State:  "FAIL",
			Reason: "host verification failed",
			Gates:  []string{"host checks must pass before manual client evidence is evaluated"},
		}
	}
	if manualEvidenceMissing(fixture.ManualEvidence) {
		return verifyResult{
			State:  "BLOCKED_PENDING_MANUAL_EVIDENCE",
			Reason: "manual client evidence missing; PENDING_MANUAL_QA until user supplies macOS/Windows print and scan proof",
			Gates:  []string{"macOS native print", "Windows native print", "macOS native scan", "Windows eSCL client scan"},
		}
	}
	if !manualEvidencePassed(fixture.ManualEvidence) {
		return verifyResult{
			State:  "FAIL",
			Reason: "manual client evidence failed",
			Gates:  []string{"all macOS/Windows print and scan evidence items must report PASS"},
		}
	}
	if !manualEvidenceProofComplete(fixture.ManualEvidence) {
		return verifyResult{
			State:  "BLOCKED_PENDING_MANUAL_EVIDENCE",
			Reason: "manual client evidence proof incomplete; PENDING_MANUAL_QA until user supplies macOS/Windows print and scan proof",
			Gates:  []string{"macOS native print proof", "Windows native print proof", "macOS native scan proof", "Windows eSCL client scan proof"},
		}
	}
	return verifyResult{
		State:  "PASS",
		Reason: "host and manual client evidence passed",
		Gates:  []string{"macOS print", "Windows print", "macOS scan", "Windows eSCL scan"},
	}
}

func hostReady(host hostState) bool {
	return ubuntuVersionReady.MatchString(strings.TrimSpace(host.UbuntuVersion)) &&
		strings.TrimSpace(host.Architecture) == "amd64" &&
		host.CUPSMajorVersion >= 2 &&
		usbVIDPIDPattern.MatchString(strings.TrimSpace(host.USBVIDPID)) &&
		checksumPattern.MatchString(strings.TrimSpace(host.LocalFlatbedScanChecksum)) &&
		saneTripletReady(host.SANETriplet) &&
		host.AirSaneHTTP &&
		host.UscanMDNS &&
		host.CUPSQueue &&
		host.IPPMDNS &&
		host.RebootPersistence
}

func saneTripletReady(triplet saneTriplet) bool {
	return triplet.CurrentUser && triplet.Root && triplet.Saned
}

func manualEvidenceMissing(evidence manualEvidence) bool {
	return evidence.MacOSPrint == nil ||
		evidence.WindowsPrint == nil ||
		evidence.MacOSScan == nil ||
		evidence.WindowsScan == nil
}

func manualEvidencePassed(evidence manualEvidence) bool {
	return evidence.MacOSPrint.Result == "PASS" &&
		evidence.WindowsPrint.Result == "PASS" &&
		evidence.MacOSScan.Result == "PASS" &&
		evidence.WindowsScan.Result == "PASS"
}

func manualEvidenceProofComplete(evidence manualEvidence) bool {
	return evidenceItemProofComplete(evidence.MacOSPrint) &&
		evidenceItemProofComplete(evidence.WindowsPrint) &&
		evidenceItemProofComplete(evidence.MacOSScan) &&
		evidenceItemProofComplete(evidence.WindowsScan)
}

func evidenceItemProofComplete(item *evidenceItem) bool {
	if item == nil {
		return false
	}
	if strings.TrimSpace(item.Name) == "" ||
		strings.TrimSpace(item.DiscoveryProof) == "" ||
		strings.TrimSpace(item.LogBundleID) == "" {
		return false
	}
	_, err := time.Parse(time.RFC3339, strings.TrimSpace(item.Timestamp))
	return err == nil
}

func printVerifyResult(streams Streams, result verifyResult, handoff []string) error {
	lines := []string{
		"state: " + result.State,
		"reason: " + result.Reason,
		"gates:",
		"- " + strings.Join(result.Gates, "\n- "),
	}
	if len(handoff) > 0 {
		lines = append(lines, "")
		lines = append(lines, handoff...)
	}
	_, err := fmt.Fprintln(streams.Out, strings.Join(lines, "\n"))
	return err
}
