package cli

import (
	"context"
	"os"
	"strings"
)

type setupRealPreflightResult struct {
	pass   bool
	reason string
	retry  string
}

func setupRealPreflight(ctx context.Context, runtime setupRealRuntime) setupRealPreflightResult {
	const retry = "run setup on Ubuntu/Debian Linux amd64 or arm64 with apt-get available"
	if !setupRealPlatformSupported(runtime) {
		return setupRealPreflightResult{
			reason: "unsupported host platform " + runtime.goos + "/" + runtime.goarch,
			retry:  retry,
		}
	}
	if !setupRealDebianLike(runtime.root) {
		return setupRealPreflightResult{
			reason: "supported setup requires Ubuntu/Debian; /etc/os-release is not Debian-like",
			retry:  retry,
		}
	}
	result, err := runtime.commandRunner.Run(ctx, HostCommand{program: "sh", args: []string{"-c", "command -v apt-get"}})
	if err != nil || result.ExitCode != 0 {
		return setupRealPreflightResult{
			reason: "apt-get is required before host setup can install CUPS/SANE/AirSane dependencies",
			retry:  retry,
		}
	}
	return setupRealPreflightResult{pass: true}
}

func setupRealDebianLike(root string) bool {
	data, err := os.ReadFile(setupRealRootedGlob(root, "/etc/os-release"))
	if err != nil {
		return false
	}
	values := parseOSRelease(data)
	if osReleaseMatches(values["ID"], "debian", "ubuntu") {
		return true
	}
	for _, field := range strings.Fields(values["ID_LIKE"]) {
		if osReleaseMatches(field, "debian", "ubuntu") {
			return true
		}
	}
	return false
}

func parseOSRelease(data []byte) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || key == "" {
			continue
		}
		values[key] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return values
}

func osReleaseMatches(value string, supported ...string) bool {
	for _, item := range supported {
		if value == item {
			return true
		}
	}
	return false
}
