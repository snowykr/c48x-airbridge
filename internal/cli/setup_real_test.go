package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_SetupReal_runsGuidedPlanWithSafeRunner_whenApproved(t *testing.T) {
	// Given
	root := t.TempDir()
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	runtime := setupRealRuntime{
		root:           root,
		commandRunner:  NewFakeCommandRunner(realSetupSuccessCommands()),
		commandLogPath: "/var/log/c48x-airbridge/commands.log",
	}
	writeRealSetupOSRelease(t, root, "ID=ubuntu\nID_LIKE=debian\n")
	writeRealSetupBackend(t, root)
	options := setupOptions{
		Yes:           true,
		Component:     setupComponentAll,
		AirSaneCommit: guidedAirSaneCommit,
	}

	// When
	err := runSetupReal(context.Background(), Streams{Out: out, Err: errOut}, options, runtime)

	// Then
	if err != nil {
		t.Fatalf("real setup returned error: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"setup review:",
		"state: " + setupStateBlockedClientProof,
		"sudo lpadmin -p C48X-Series",
		"git -C /tmp/c48x-airbridge/airsane/source checkout --detach " + guidedAirSaneCommit,
		"sudo rsync -a /tmp/c48x-airbridge/airsane/stage/usr/local/ /usr/local/",
		"evidence bundle: /var/log/c48x-airbridge/setup-evidence.json",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("real setup output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "setup apply scaffold") || strings.Contains(got, "later guided installer workflow") {
		t.Fatalf("real setup still used scaffold:\n%s", got)
	}
	commandLog, err := os.ReadFile(filepath.Join(root, "var", "log", "c48x-airbridge", "commands.log"))
	if err != nil {
		t.Fatalf("read command log: %v", err)
	}
	if !strings.Contains(string(commandLog), "sudo lpadmin -p C48X-Series") {
		t.Fatalf("command log did not include CUPS apply:\n%s", string(commandLog))
	}
	if _, err := os.Stat(filepath.Join(root, "var", "log", "c48x-airbridge", "setup-evidence.json")); err != nil {
		t.Fatalf("guided setup evidence was not written: %v", err)
	}
}

func Test_SetupReal_blocksBeforeMutation_whenAirSaneCommitIsMissing(t *testing.T) {
	// Given
	root := t.TempDir()
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	runtime := setupRealRuntime{
		root:           root,
		commandRunner:  NewFakeCommandRunner(realSetupSuccessCommands()),
		commandLogPath: "/var/log/c48x-airbridge/commands.log",
	}
	writeRealSetupOSRelease(t, root, "ID=ubuntu\nID_LIKE=debian\n")
	writeRealSetupBackend(t, root)
	options := setupOptions{Yes: true, Component: setupComponentAll}

	// When
	err := runSetupReal(context.Background(), Streams{Out: out, Err: errOut}, options, runtime)

	// Then
	if err != nil {
		t.Fatalf("real setup blocked path returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "state: "+setupStateBlockedDriverRequired) {
		t.Fatalf("real setup did not report blocked dependency:\n%s", got)
	}
	if strings.Contains(got, "setup apply scaffold") || strings.Contains(got, "later guided installer workflow") {
		t.Fatalf("real setup blocked path still used scaffold:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(root, "var", "log", "c48x-airbridge", "commands.log")); !os.IsNotExist(err) {
		t.Fatalf("blocked real setup wrote command log before mutation: %v", err)
	}
}

func Test_SetupReal_blocksBeforeMutation_whenPlatformUnsupported(t *testing.T) {
	// Given
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	recorder := &recordingCommandRunner{}
	runtime := setupRealRuntime{
		root:           t.TempDir(),
		commandRunner:  recorder,
		commandLogPath: "/var/log/c48x-airbridge/commands.log",
		goos:           "darwin",
		goarch:         "arm64",
	}
	options := setupOptions{Yes: true, Component: setupComponentAll, AirSaneCommit: guidedAirSaneCommit}

	// When
	err := runSetupReal(context.Background(), Streams{Out: out, Err: errOut}, options, runtime)

	// Then
	if err != nil {
		t.Fatalf("unsupported platform returned process error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "state: "+setupStateFail) || !strings.Contains(got, "unsupported host platform darwin/arm64") {
		t.Fatalf("unsupported platform did not block with clear state:\n%s", got)
	}
	if len(recorder.commands) != 0 {
		t.Fatalf("unsupported platform ran host commands before blocking: %#v", recorder.commands)
	}
}

func Test_SetupReal_blocksBeforeMutation_whenLinuxDistroUnsupported(t *testing.T) {
	// Given
	root := t.TempDir()
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	recorder := &recordingCommandRunner{}
	runtime := setupRealRuntime{
		root:           root,
		commandRunner:  recorder,
		commandLogPath: "/var/log/c48x-airbridge/commands.log",
		goos:           "linux",
		goarch:         "amd64",
	}
	writeRealSetupOSRelease(t, root, "ID=fedora\n")
	options := setupOptions{Yes: true, Component: setupComponentAll, AirSaneCommit: guidedAirSaneCommit}

	// When
	err := runSetupReal(context.Background(), Streams{Out: out, Err: errOut}, options, runtime)

	// Then
	if err != nil {
		t.Fatalf("unsupported distro returned process error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "state: "+setupStateFail) || !strings.Contains(got, "supported setup requires Ubuntu/Debian") {
		t.Fatalf("unsupported distro did not block with clear state:\n%s", got)
	}
	if len(recorder.commands) != 0 {
		t.Fatalf("unsupported distro ran host commands before blocking: %#v", recorder.commands)
	}
}

func Test_SetupReal_blocksBeforeMutation_whenAptGetIsMissing(t *testing.T) {
	// Given
	root := t.TempDir()
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	runtime := setupRealRuntime{
		root:           root,
		commandRunner:  NewFakeCommandRunner(map[string]FakeCommandResult{"sh -c command -v apt-get": {ExitCode: 1}}),
		commandLogPath: "/var/log/c48x-airbridge/commands.log",
		goos:           "linux",
		goarch:         "amd64",
	}
	writeRealSetupOSRelease(t, root, "ID=ubuntu\nID_LIKE=debian\n")
	options := setupOptions{Yes: true, Component: setupComponentAll, AirSaneCommit: guidedAirSaneCommit}

	// When
	err := runSetupReal(context.Background(), Streams{Out: out, Err: errOut}, options, runtime)

	// Then
	if err != nil {
		t.Fatalf("missing apt-get returned process error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "state: "+setupStateFail) || !strings.Contains(got, "apt-get is required") {
		t.Fatalf("missing apt-get did not block with clear state:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(root, "var", "log", "c48x-airbridge", "commands.log")); !os.IsNotExist(err) {
		t.Fatalf("missing apt-get wrote command log before mutation: %v", err)
	}
}

func writeRealSetupOSRelease(t *testing.T, root string, content string) {
	t.Helper()
	path := filepath.Join(root, "etc", "os-release")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create os-release parent: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), defaultFileMode); err != nil {
		t.Fatalf("write os-release: %v", err)
	}
}

func writeRealSetupBackend(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "usr", "lib", "sane", "libsane-smfp.so.1")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create backend parent: %v", err)
	}
	if err := os.WriteFile(path, []byte("fake smfp backend"), defaultFileMode); err != nil {
		t.Fatalf("write backend: %v", err)
	}
}

func realSetupSuccessCommands() map[string]FakeCommandResult {
	return map[string]FakeCommandResult{
		"lpinfo -v":             {Stdout: "direct usb://Samsung/C48x%20Series?serial=TEST\n"},
		"lpstat -v C48X-Series": {Stdout: "device for C48X-Series: usb://old/device\n"},
	}
}
