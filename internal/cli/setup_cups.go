package cli

import "strings"

const (
	setupStateBlockedPrinterRequired = "BLOCKED_PRINTER_REQUIRED"
	cupsQueueName                    = "C48X-Series"
	cupsDriverModel                  = "drv:///splix-samsung.drv/samsung-c48x.ppd"
)

type cupsSetupPlan struct {
	State  string
	Reason string
	Steps  []HostStep
}

func planCUPSSetup(fixture setupRunnerFixture) cupsSetupPlan {
	steps := []HostStep{
		NewPrivilegedCommandStep("install CUPS and Avahi packages", "apt-get", "install", "-y", "cups", "cups-client", "avahi-daemon", "avahi-utils", "printer-driver-splix", "system-config-printer"),
		NewPrivilegedCommandStep("enable CUPS", "systemctl", "enable", "--now", "cups"),
		NewPrivilegedCommandStep("enable Avahi", "systemctl", "enable", "--now", "avahi-daemon"),
		NewPrivilegedCommandStep("enable printer sharing", "cupsctl", "--share-printers", "--no-remote-admin", "--no-remote-any"),
		NewCUPSQueueStep("create or repair CUPS queue", cupsQueueName, cupsDriverModel),
	}
	steps = append(steps,
		NewCommandStep("verify CUPS queue", "lpstat", "-t"),
		NewCommandStep("verify IPP mDNS", "avahi-browse", "-rt", "_ipp._tcp"),
	)
	return cupsSetupPlan{State: setupStatePass, Steps: steps}
}

func samsungPrinterURI(lpinfoOutput string) string {
	for _, line := range strings.Split(lpinfoOutput, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		uri := fields[len(fields)-1]
		if strings.HasPrefix(uri, "usb://") && samsungPrinterLine(line) {
			return uri
		}
	}
	return ""
}

func samsungPrinterLine(line string) bool {
	normalized := strings.ToLower(line)
	return strings.Contains(normalized, "samsung") &&
		(strings.Contains(normalized, "c48") || strings.Contains(normalized, "c480"))
}

func queueNeedsRepair(lpstatOutput string, deviceURI string) bool {
	return !strings.Contains(lpstatOutput, deviceURI)
}
