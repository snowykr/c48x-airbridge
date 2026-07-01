package cli

import (
	"strings"

	"c48x-airbridge/internal/hostprobe"
)

func firstIncompleteLiveCheck(report hostprobe.Report) (hostprobe.Result, bool) {
	for _, check := range requiredLiveChecks(report) {
		result, ok := findLiveResult(report, check)
		if !ok {
			return hostprobe.Result{Check: check, Name: string(check), Status: hostprobe.StatusWarn, Detail: "check missing from live probe report"}, true
		}
		if result.Status != hostprobe.StatusPass {
			return result, true
		}
	}
	return hostprobe.Result{}, false
}

func requiredLiveChecks(report hostprobe.Report) []hostprobe.CheckID {
	checks := []hostprobe.CheckID{
		hostprobe.CheckUSBDevice,
		hostprobe.CheckCUPSService,
		hostprobe.CheckCUPSQueue,
		hostprobe.CheckAvahiService,
		hostprobe.CheckIPPService,
		hostprobe.CheckSANECurrentUser,
		hostprobe.CheckSANERoot,
		hostprobe.CheckSMFPConfig,
		hostprobe.CheckSMFPBackend,
		hostprobe.CheckAirSaneService,
		hostprobe.CheckAirSaneHTTP,
		hostprobe.CheckUSCANService,
	}
	if sanedCheckRequired(report) {
		return append(checks[:7], append([]hostprobe.CheckID{hostprobe.CheckSANESaned}, checks[7:]...)...)
	}
	return checks
}

func sanedCheckRequired(report hostprobe.Report) bool {
	result, ok := findLiveResult(report, hostprobe.CheckSANESaned)
	if !ok {
		return false
	}
	return result.Status == hostprobe.StatusPass || !strings.Contains(strings.ToLower(result.Detail), "saned user not found")
}
