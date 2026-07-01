package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testAirSaneCommit            = "0123456789abcdef0123456789abcdef01234567"
	approvedAirSaneDefaultCommit = "129cc3bf7258251a0a694dee7741285b59d88f9f"
)

func Test_AirSaneSetupPlan_usesApprovedPin_whenCommitMissing(t *testing.T) {
	// Given
	options := setupOptions{Component: setupComponentAirSane}

	// When
	plan := buildAirSaneSetupPlan(options)

	// Then
	if plan.State != setupStatePass {
		t.Fatalf("AirSane plan state = %q, want %q", plan.State, setupStatePass)
	}
	if len(plan.Steps) == 0 {
		t.Fatal("AirSane default-pinned plan produced no steps")
	}
	got := commandLinesForSteps(plan.Steps)
	for _, want := range []string{
		"git -C /tmp/c48x-airbridge/airsane/source fetch --tags origin " + approvedAirSaneDefaultCommit,
		"git -C /tmp/c48x-airbridge/airsane/source checkout --detach " + approvedAirSaneDefaultCommit,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("AirSane default-pinned plan missing %q:\n%s", want, got)
		}
	}
}

func Test_AirSaneSetupPlan_installsConfigAndProofCommands_whenCommitPinned(t *testing.T) {
	// Given
	root := t.TempDir()
	plan := buildAirSaneSetupPlan(setupOptions{Component: setupComponentAirSane, AirSaneCommit: testAirSaneCommit})
	runner := NewSafeHostRunner(SafeHostRunnerConfig{
		Root:           root,
		CommandRunner:  NewFakeCommandRunner(airSaneSuccessCommands(testAirSaneCommit)),
		CommandLogPath: "/var/log/c48x-airbridge/commands.log",
	})

	// When
	result, err := runner.Run(context.Background(), plan.Steps)

	// Then
	if err != nil {
		t.Fatalf("AirSane runner failed: %v", err)
	}
	if result.State != runnerStatePass {
		t.Fatalf("AirSane runner state = %q, want PASS", result.State)
	}
	logData, err := os.ReadFile(filepath.Join(root, "var", "log", "c48x-airbridge", "commands.log"))
	if err != nil {
		t.Fatalf("read command log: %v", err)
	}
	log := string(logData)
	for _, want := range airSaneExpectedCommandLines(testAirSaneCommit) {
		if !strings.Contains(log, want) {
			t.Fatalf("AirSane command log missing %q:\n%s", want, log)
		}
	}
	config, err := os.ReadFile(filepath.Join(root, "etc", "airsane", "access.conf"))
	if err != nil {
		t.Fatalf("read AirSane config: %v", err)
	}
	if !strings.Contains(string(config), "192.168.0.0/16") {
		t.Fatalf("AirSane config missing LAN allow-list:\n%s", config)
	}
}

func Test_AirSaneSetupPlan_runsDownloadedSourceWithoutPrivilege_whenCommitPinned(t *testing.T) {
	// Given
	plan := buildAirSaneSetupPlan(setupOptions{Component: setupComponentAirSane, AirSaneCommit: testAirSaneCommit})

	// When / Then
	for _, step := range plan.Steps {
		command, ok := step.(commandStep)
		if !ok {
			continue
		}
		if (command.command.program == "git" || command.command.program == "cmake") && command.command.privileged {
			t.Fatalf("downloaded AirSane source command runs as root: %s", command.command.DisplayLine())
		}
	}
}

func Test_AirSaneSetupPlan_returnsFailureGuidance_whenServiceMissing(t *testing.T) {
	// Given
	commands := airSaneSuccessCommands(testAirSaneCommit)
	commands["systemctl enable --now airsaned.service"] = FakeCommandResult{ExitCode: 5, Stderr: "unit not found"}
	runner := NewSafeHostRunner(SafeHostRunnerConfig{
		Root:           t.TempDir(),
		CommandRunner:  NewFakeCommandRunner(commands),
		CommandLogPath: "/var/log/c48x-airbridge/commands.log",
	})
	plan := buildAirSaneSetupPlan(setupOptions{Component: setupComponentAirSane, AirSaneCommit: testAirSaneCommit})

	// When
	result, err := runner.Run(context.Background(), plan.Steps)

	// Then
	if err != nil {
		t.Fatalf("AirSane runner returned process error: %v", err)
	}
	if result.State != runnerStateFail {
		t.Fatalf("AirSane service-missing state = %q, want FAIL", result.State)
	}
	if !strings.Contains(result.Reason, "enable AirSane service") {
		t.Fatalf("AirSane service-missing reason = %q", result.Reason)
	}
	if result.RollbackGuidance == "" || result.RetryGuidance == "" {
		t.Fatalf("AirSane service-missing result lacked guidance: %+v", result)
	}
}

func Test_SetupAirSaneFakeRunner_appliesApprovedPin_whenCommitMissing(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	fixture := filepath.Join("..", "..", "testdata", "setup", "airsane-success.json")
	cmd.SetArgs([]string{"setup", "--yes", "--component", "airsane", "--fake-runner", fixture})
	t.Setenv("C48X_AIRBRIDGE_FAKE_ROOT", t.TempDir())

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("AirSane default-pinned fake-runner returned process error: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"state: PASS",
		"git -C /tmp/c48x-airbridge/airsane/source checkout --detach " + approvedAirSaneDefaultCommit,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("AirSane default-pinned output missing %q:\n%s", want, got)
		}
	}
}

func Test_SetupAirSaneFakeRunner_rejectsFloatingSource_whenCommitIsBranch(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	fixture := filepath.Join("..", "..", "testdata", "setup", "airsane-success.json")
	cmd.SetArgs([]string{"setup", "--yes", "--component", "airsane", "--airsane-commit", "main", "--fake-runner", fixture})
	t.Setenv("C48X_AIRBRIDGE_FAKE_ROOT", t.TempDir())

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("AirSane floating-source fake-runner returned process error: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"state: BLOCKED_DRIVER_REQUIRED",
		"refusing floating AirSane source",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("AirSane floating-source output missing %q:\n%s", want, got)
		}
	}
}

func Test_SetupAirSaneFakeRunner_appliesPinnedBuildAndConfig_whenCommitProvided(t *testing.T) {
	// Given
	root := t.TempDir()
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	fixture := filepath.Join("..", "..", "testdata", "setup", "airsane-success.json")
	cmd.SetArgs([]string{"setup", "--yes", "--component", "airsane", "--airsane-commit", testAirSaneCommit, "--fake-runner", fixture})
	t.Setenv("C48X_AIRBRIDGE_FAKE_ROOT", root)

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("AirSane pinned fake-runner returned process error: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"state: PASS",
		"git clone --no-checkout",
		"git -C /tmp/c48x-airbridge/airsane/source checkout --detach " + testAirSaneCommit,
		"cmake --install /tmp/c48x-airbridge/airsane/source/build --prefix /tmp/c48x-airbridge/airsane/stage/usr/local",
		"sudo rsync -a /tmp/c48x-airbridge/airsane/stage/usr/local/ /usr/local/",
		"avahi-browse -rt _uscan._tcp",
		"curl -fsS --max-time 2 http://localhost:8090/eSCL/ScannerStatus",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("AirSane pinned output missing %q:\n%s", want, got)
		}
	}
	config, err := os.ReadFile(filepath.Join(root, "etc", "airsane", "access.conf"))
	if err != nil {
		t.Fatalf("read fake-root AirSane config: %v", err)
	}
	if !strings.Contains(string(config), "10.0.0.0/8") {
		t.Fatalf("fake-root AirSane config missing private LAN:\n%s", config)
	}
}

func airSaneSuccessCommands(commit string) map[string]FakeCommandResult {
	return map[string]FakeCommandResult{
		"apt-get install -y git cmake g++ make pkg-config libsane-dev libjpeg-dev libpng-dev libavahi-client-dev libusb-1.0-0-dev curl avahi-utils": {ExitCode: 0},
		"rm -rf /tmp/c48x-airbridge/airsane":   {ExitCode: 0},
		"mkdir -p /tmp/c48x-airbridge/airsane": {ExitCode: 0},
		"git clone --no-checkout https://github.com/SimulPiscator/AirSane.git /tmp/c48x-airbridge/airsane/source":            {ExitCode: 0},
		"git -C /tmp/c48x-airbridge/airsane/source fetch --tags origin " + commit:                                            {ExitCode: 0},
		"git -C /tmp/c48x-airbridge/airsane/source checkout --detach " + commit:                                              {ExitCode: 0},
		"cmake -S /tmp/c48x-airbridge/airsane/source -B /tmp/c48x-airbridge/airsane/source/build -DCMAKE_BUILD_TYPE=Release": {ExitCode: 0},
		"cmake --build /tmp/c48x-airbridge/airsane/source/build --parallel":                                                  {ExitCode: 0},
		"cmake --install /tmp/c48x-airbridge/airsane/source/build --prefix /tmp/c48x-airbridge/airsane/stage/usr/local":      {ExitCode: 0},
		"rsync -a /tmp/c48x-airbridge/airsane/stage/usr/local/ /usr/local/":                                                  {ExitCode: 0},
		"systemctl enable --now airsaned.service":                                                                            {ExitCode: 0},
		"systemctl restart airsaned.service":                                                                                 {ExitCode: 0},
		"systemctl status airsaned.service --no-pager":                                                                       {ExitCode: 0},
		"avahi-browse -rt _uscan._tcp":                                                                                       {ExitCode: 0, Stdout: "_uscan._tcp Samsung C480"},
		"curl -fsS --max-time 2 http://localhost:8090/eSCL/ScannerStatus":                                                    {ExitCode: 0, Stdout: "<ScannerStatus>Idle</ScannerStatus>"},
	}
}

func airSaneExpectedCommandLines(commit string) []string {
	return []string{
		"sudo apt-get install -y git cmake g++ make pkg-config libsane-dev libjpeg-dev libpng-dev libavahi-client-dev libusb-1.0-0-dev curl avahi-utils",
		"rm -rf /tmp/c48x-airbridge/airsane",
		"mkdir -p /tmp/c48x-airbridge/airsane",
		"git clone --no-checkout https://github.com/SimulPiscator/AirSane.git /tmp/c48x-airbridge/airsane/source",
		"git -C /tmp/c48x-airbridge/airsane/source fetch --tags origin " + commit,
		"git -C /tmp/c48x-airbridge/airsane/source checkout --detach " + commit,
		"cmake -S /tmp/c48x-airbridge/airsane/source -B /tmp/c48x-airbridge/airsane/source/build -DCMAKE_BUILD_TYPE=Release",
		"cmake --build /tmp/c48x-airbridge/airsane/source/build --parallel",
		"cmake --install /tmp/c48x-airbridge/airsane/source/build --prefix /tmp/c48x-airbridge/airsane/stage/usr/local",
		"sudo rsync -a /tmp/c48x-airbridge/airsane/stage/usr/local/ /usr/local/",
		"sudo systemctl enable --now airsaned.service",
		"sudo systemctl restart airsaned.service",
		"systemctl status airsaned.service --no-pager",
		"avahi-browse -rt _uscan._tcp",
		"curl -fsS --max-time 2 http://localhost:8090/eSCL/ScannerStatus",
	}
}

func commandLinesForSteps(steps []HostStep) string {
	var lines []string
	for _, step := range steps {
		command, ok := step.(commandStep)
		if !ok {
			continue
		}
		lines = append(lines, command.command.DisplayLine())
	}
	return strings.Join(lines, "\n")
}
