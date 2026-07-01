package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"c48x-airbridge/internal/hostprobe"
)

const verifyModeLive = "live"
const verifyModeFixture = "fixture"
const verifyStateBlockedHost = "BLOCKED_HOST_VERIFICATION"

var vidPIDFromEvidence = regexp.MustCompile(`(?i)\b([0-9a-f]{4}:[0-9a-f]{4})\b`)

type verifyOutputBundle struct {
	State          string            `json:"state"`
	Reason         string            `json:"reason"`
	Mode           string            `json:"mode"`
	Gates          []string          `json:"gates"`
	Host           hostState         `json:"host"`
	ManualEvidence manualEvidence    `json:"manual_evidence"`
	HostChecks     []verifyHostCheck `json:"host_checks,omitempty"`
	ClientHandoff  []string          `json:"client_handoff,omitempty"`
	Redacted       bool              `json:"redacted"`
}

type verifyHostCheck struct {
	Check    hostprobe.CheckID `json:"check"`
	Section  string            `json:"section"`
	Name     string            `json:"name"`
	Status   hostprobe.Status  `json:"status"`
	Detail   string            `json:"detail,omitempty"`
	Evidence string            `json:"evidence,omitempty"`
}

func evaluateLiveReport(report hostprobe.Report) verifyResult {
	if result, ok := findLiveResult(report, hostprobe.CheckUSBDevice); ok && result.Status == hostprobe.StatusBlocked {
		return verifyResult{
			State:  setupStateBlockedPrinterRequired,
			Reason: "BLOCKED_PRINTER_REQUIRED: Samsung C48x/C480 USB printer was not found; connect or power on the printer, then rerun verify --live.",
			Gates:  []string{result.Name + ": " + result.Detail},
		}
	}
	if failed, ok := firstIncompleteLiveCheck(report); ok {
		return verifyResult{
			State:  verifyStateBlockedHost,
			Reason: "host verification failed: " + failed.Name + " " + strings.ToLower(string(failed.Status)) + " - " + failed.Detail,
			Gates:  []string{failed.Name + " must pass before macOS/Windows client proof"},
		}
	}
	return verifyResult{
		State:  setupStateBlockedClientProof,
		Reason: "host verification passed; macOS/Windows client print and scan proof is still required.",
		Gates: []string{
			"CUPS queue",
			"IPP mDNS",
			"SANE current/root/saned scanimage triplet",
			"AirSane HTTP/eSCL",
			"AirSane _uscan._tcp mDNS",
			"reboot persistence: rerun verify --live after reboot to refresh host evidence",
		},
	}
}

func findLiveResult(report hostprobe.Report, check hostprobe.CheckID) (hostprobe.Result, bool) {
	for _, result := range report.Results {
		if result.Check == check {
			return result, true
		}
	}
	return hostprobe.Result{}, false
}

func newLiveVerifyBundle(result verifyResult, report hostprobe.Report) verifyOutputBundle {
	checks := make([]verifyHostCheck, 0, len(report.Results))
	for _, probeResult := range report.Results {
		checks = append(checks, verifyHostCheck{
			Check:    probeResult.Check,
			Section:  redactVerifyText(probeResult.Section),
			Name:     redactVerifyText(probeResult.Name),
			Status:   probeResult.Status,
			Detail:   redactVerifyText(probeResult.Detail),
			Evidence: redactVerifyText(probeResult.Evidence),
		})
	}
	return verifyOutputBundle{
		State:         result.State,
		Reason:        redactVerifyText(result.Reason),
		Mode:          verifyModeLive,
		Gates:         redactVerifyLines(result.Gates),
		Host:          hostStateFromLiveReport(report),
		HostChecks:    checks,
		ClientHandoff: clientHandoffForResult(result),
		Redacted:      true,
	}
}

func newFixtureVerifyBundle(result verifyResult, fixture verifyFixture) verifyOutputBundle {
	return verifyOutputBundle{
		State:          result.State,
		Reason:         redactVerifyText(result.Reason),
		Mode:           verifyModeFixture,
		Gates:          redactVerifyLines(result.Gates),
		Host:           redactHostState(fixture.Host),
		ManualEvidence: redactManualEvidence(fixture.ManualEvidence),
		Redacted:       true,
	}
}

func hostStateFromLiveReport(report hostprobe.Report) hostState {
	host := hostState{
		CUPSVersionGate:       "not proven by live probe",
		RebootPersistenceNote: "not proven by a single live run; rerun verify --live after reboot",
	}
	if result, ok := findLiveResult(report, hostprobe.CheckArchitecture); ok {
		host.Architecture = strings.TrimSpace(result.Evidence)
	}
	if result, ok := findLiveResult(report, hostprobe.CheckOSRelease); ok {
		host.UbuntuVersion = strings.TrimSpace(result.Evidence)
	}
	if result, ok := findLiveResult(report, hostprobe.CheckUSBDevice); ok && result.Status == hostprobe.StatusPass {
		host.USBVIDPID = liveVIDPID(result.Evidence)
	}
	host.SANETriplet = saneTriplet{
		CurrentUser: liveCheckPassed(report, hostprobe.CheckSANECurrentUser),
		Root:        liveCheckPassed(report, hostprobe.CheckSANERoot),
		Saned:       liveCheckPassed(report, hostprobe.CheckSANESaned),
		Proof:       "scanimage current user/root/saned live checks",
	}
	host.AirSaneHTTP = liveCheckPassed(report, hostprobe.CheckAirSaneHTTP)
	host.UscanMDNS = liveCheckPassed(report, hostprobe.CheckUSCANService)
	host.CUPSQueue = liveCheckPassed(report, hostprobe.CheckCUPSQueue)
	host.IPPMDNS = liveCheckPassed(report, hostprobe.CheckIPPService)
	return redactHostState(host)
}

func liveCheckPassed(report hostprobe.Report, check hostprobe.CheckID) bool {
	result, ok := findLiveResult(report, check)
	return ok && result.Status == hostprobe.StatusPass
}

func liveVIDPID(evidence string) string {
	match := vidPIDFromEvidence.FindStringSubmatch(evidence)
	if len(match) == 2 {
		return strings.ToLower(match[1])
	}
	return ""
}

func clientHandoffForResult(result verifyResult) []string {
	if result.State != setupStateBlockedClientProof {
		return nil
	}
	return []string{
		"macOS client handoff:",
		"- Add the Samsung C48x IPP printer from System Settings > Printers & Scanners; use the advertised _ipp._tcp/AirPrint entry and print a test page.",
		"- Open Image Capture and scan with Image Capture from the advertised AirSane/eSCL scanner; save the scan proof for the evidence bundle.",
		"Windows client handoff:",
		"- Add the Samsung C48x IPP printer from Settings > Bluetooth & devices > Printers & scanners; use the discovered IPP printer and print a test page.",
		"- Install or open a non-Samsung client such as Windows Scan or NAPS2, then scan with a Windows eSCL-compatible app and save a scan proof.",
	}
}

func writeVerifyOutput(path string, bundle verifyOutputBundle) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return fmt.Errorf("encode verify output: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create verify output parent: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), defaultFileMode); err != nil {
		return fmt.Errorf("write verify output: %w", err)
	}
	return nil
}

func redactHostState(host hostState) hostState {
	host.UbuntuVersion = redactVerifyText(host.UbuntuVersion)
	host.Architecture = redactVerifyText(host.Architecture)
	host.CUPSVersionGate = redactVerifyText(host.CUPSVersionGate)
	host.USBVIDPID = redactVerifyText(host.USBVIDPID)
	host.LocalFlatbedScanChecksum = redactVerifyText(host.LocalFlatbedScanChecksum)
	host.SANETriplet.Proof = redactVerifyText(host.SANETriplet.Proof)
	host.AirSaneHTTPURL = redactVerifyText(host.AirSaneHTTPURL)
	host.AirSaneHTTPProof = redactVerifyText(host.AirSaneHTTPProof)
	host.UscanMDNSService = redactVerifyText(host.UscanMDNSService)
	host.CUPSQueueName = redactVerifyText(host.CUPSQueueName)
	host.IPPMDNSService = redactVerifyText(host.IPPMDNSService)
	host.RebootPersistenceNote = redactVerifyText(host.RebootPersistenceNote)
	return host
}

func redactManualEvidence(evidence manualEvidence) manualEvidence {
	return manualEvidence{
		MacOSPrint:   redactEvidenceItem(evidence.MacOSPrint),
		WindowsPrint: redactEvidenceItem(evidence.WindowsPrint),
		MacOSScan:    redactEvidenceItem(evidence.MacOSScan),
		WindowsScan:  redactEvidenceItem(evidence.WindowsScan),
	}
}

func redactEvidenceItem(item *evidenceItem) *evidenceItem {
	if item == nil {
		return nil
	}
	return &evidenceItem{
		DiscoveryProof: redactVerifyText(item.DiscoveryProof),
		Name:           redactVerifyText(item.Name),
		Result:         redactVerifyText(item.Result),
		Timestamp:      redactVerifyText(item.Timestamp),
		Note:           redactVerifyText(item.Note),
		LogBundleID:    redactVerifyText(item.LogBundleID),
	}
}

func redactVerifyLines(lines []string) []string {
	redacted := make([]string, 0, len(lines))
	for _, line := range lines {
		redacted = append(redacted, redactVerifyText(line))
	}
	return redacted
}

func redactVerifyText(value string) string {
	if root := os.Getenv("C48X_AIRBRIDGE_FAKE_ROOT"); root != "" {
		value = strings.ReplaceAll(value, root, "<fake-root>")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		value = strings.ReplaceAll(value, home, "<home>")
	}
	return value
}
