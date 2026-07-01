package cli

import "os"

const (
	airSaneSourceDir = "/usr/local/src/AirSane"
	airSaneConfig    = "# Managed by c48x-airbridge\n127.0.0.1/32\n::1/128\n192.168.0.0/16\n10.0.0.0/8\n172.16.0.0/12\n"
)

type airSaneSetupPlan struct {
	State  string
	Reason string
	Steps  []HostStep
}

func buildAirSaneSetupPlan(options setupOptions) airSaneSetupPlan {
	source := resolveAirSaneSource(defaultSetupDependencyMetadata, options.AirSaneCommit)
	if source.Source != setupSourcePinned {
		return airSaneSetupPlan{
			State:  setupStateBlockedDriverRequired,
			Reason: "BLOCKED_DRIVER_REQUIRED: refusing floating AirSane source; provide a 40-character --airsane-commit before setup can fetch source.",
		}
	}
	return airSaneSetupPlan{
		State: setupStatePass,
		Steps: []HostStep{
			NewPrivilegedCommandStep("install AirSane build dependencies", "apt-get", "install", "-y", "git", "cmake", "g++", "make", "pkg-config", "libsane-dev", "libjpeg-dev", "libpng-dev", "libavahi-client-dev", "libusb-1.0-0-dev", "curl", "avahi-utils"),
			NewPrivilegedCommandStep("clone AirSane source", "git", "clone", "--no-checkout", source.Repo, airSaneSourceDir),
			NewPrivilegedCommandStep("fetch pinned AirSane commit", "git", "-C", airSaneSourceDir, "fetch", "--tags", "origin", source.Commit),
			NewPrivilegedCommandStep("checkout pinned AirSane commit", "git", "-C", airSaneSourceDir, "checkout", "--detach", source.Commit),
			NewPrivilegedCommandStep("configure AirSane build", "cmake", "-S", airSaneSourceDir, "-B", airSaneSourceDir+"/build", "-DCMAKE_BUILD_TYPE=Release"),
			NewPrivilegedCommandStep("build AirSane", "cmake", "--build", airSaneSourceDir+"/build", "--parallel"),
			NewPrivilegedCommandStep("install AirSane", "cmake", "--install", airSaneSourceDir+"/build"),
			NewFileWriteStep("/etc/airsane/access.conf", []byte(airSaneConfig), os.FileMode(0o644)),
			NewPrivilegedCommandStep("enable AirSane service", "systemctl", "enable", "--now", "airsaned.service"),
			NewPrivilegedCommandStep("restart AirSane service", "systemctl", "restart", "airsaned.service"),
			NewCommandStep("check AirSane service", "systemctl", "status", "airsaned.service", "--no-pager"),
			NewCommandStep("verify AirSane _uscan mDNS", "avahi-browse", "-rt", "_uscan._tcp"),
			NewCommandStep("verify AirSane eSCL HTTP", "curl", "-fsS", "--max-time", "2", "http://localhost:8090/eSCL/ScannerStatus"),
		},
	}
}

func airSaneBlockedResult(plan airSaneSetupPlan, commandLogPath string) RunResult {
	return RunResult{
		State:            plan.State,
		Reason:           plan.Reason,
		CommandLogPath:   commandLogPath,
		RollbackGuidance: "no host changes were made before AirSane source validation",
		RetryGuidance:    "rerun setup --yes --component airsane --airsane-commit <40-character commit>",
	}
}
