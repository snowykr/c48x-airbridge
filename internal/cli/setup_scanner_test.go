package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_SetupScannerFakeRunner_appliesSaneBackend_whenLocalDebProvided(t *testing.T) {
	// Given
	root := t.TempDir()
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	fixture := filepath.Join("..", "..", "testdata", "setup", "scanner-success.json")
	deb := filepath.Join("..", "..", "testdata", "setup", "suld-driver2.fake.deb")
	cmd.SetArgs([]string{"setup", "--yes", "--component", "scanner", "--suldr-deb", deb, "--fake-runner", fixture})
	t.Setenv("C48X_AIRBRIDGE_FAKE_ROOT", root)

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("setup scanner fake-runner returned process error: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"setup apply result:",
		"fixture: scanner-success",
		"component: scanner",
		"state: PASS",
		"sudo apt-get install -y sane-utils libsane1 libsane-common libusb-0.1-4 usbutils acl curl ca-certificates gnupg",
		"sudo apt-get install -y " + deb,
		"sudo udevadm control --reload-rules",
		"sudo udevadm trigger",
		"sudo groupadd -f scanner",
		"sudo usermod -aG scanner,lp saned",
		"scanimage -L",
		"sudo scanimage -L",
		"sudo -u saned scanimage -L",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("setup scanner output missing %q:\n%s", want, got)
		}
	}
	assertFakeFileContains(t, root, "/etc/udev/rules.d/99-samsung-c480-scanner.rules", `ATTR{idVendor}=="04e8"`)
	assertFakeFileEquals(t, root, "/etc/sane.d/dll.conf", "net\n\nsmfp\n")
	backups, err := filepath.Glob(filepath.Join(root, "var", "backups", "c48x-airbridge", "etc", "sane.d", "dll.conf.*.bak"))
	if err != nil {
		t.Fatalf("glob dll.conf backups: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("dll.conf backup count = %d, want 1: %v", len(backups), backups)
	}
}

func Test_SetupScannerFakeRunner_usesInstalledBackend_whenAlreadyPresent(t *testing.T) {
	// Given
	root := t.TempDir()
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	fixture := filepath.Join("..", "..", "testdata", "setup", "scanner-installed-backend.json")
	cmd.SetArgs([]string{"setup", "--yes", "--component", "scanner", "--fake-runner", fixture})
	t.Setenv("C48X_AIRBRIDGE_FAKE_ROOT", root)

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("setup scanner installed-backend returned process error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "state: PASS") {
		t.Fatalf("installed-backend scanner setup did not pass:\n%s", got)
	}
	if strings.Contains(got, "suld-driver2.fake.deb") {
		t.Fatalf("installed-backend scanner setup tried to install local deb:\n%s", got)
	}
	assertFakeFileEquals(t, root, "/etc/sane.d/dll.conf", "smfp\n")
}

func Test_SetupScannerFakeRunner_blocksDriver_whenBackendMissingAndNoSafeSource(t *testing.T) {
	// Given
	root := t.TempDir()
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := NewCommand(Streams{Out: out, Err: errOut})
	fixture := filepath.Join("..", "..", "testdata", "setup", "scanner-missing-backend.json")
	cmd.SetArgs([]string{"setup", "--yes", "--component", "scanner", "--fake-runner", fixture})
	t.Setenv("C48X_AIRBRIDGE_FAKE_ROOT", root)

	// When
	err := cmd.ExecuteContext(context.Background())

	// Then
	if err != nil {
		t.Fatalf("setup scanner missing backend returned process error: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"setup apply result:",
		"fixture: scanner-missing-backend",
		"component: scanner",
		"state: BLOCKED_DRIVER_REQUIRED",
		blockedDriverRequiredReason,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("setup scanner missing backend output missing %q:\n%s", want, got)
		}
	}
	logPath := filepath.Join(root, "var", "log", "c48x-airbridge", "commands.log")
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("blocked scanner setup wrote command log at %s", logPath)
	}
}

func assertFakeFileContains(t *testing.T, root string, hostPath string, want string) {
	t.Helper()
	data := readFakeFile(t, root, hostPath)
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s missing %q:\n%s", hostPath, want, string(data))
	}
}

func assertFakeFileEquals(t *testing.T, root string, hostPath string, want string) {
	t.Helper()
	data := readFakeFile(t, root, hostPath)
	if string(data) != want {
		t.Fatalf("%s content = %q, want %q", hostPath, string(data), want)
	}
}

func readFakeFile(t *testing.T, root string, hostPath string) []byte {
	t.Helper()
	relative := strings.TrimPrefix(filepath.Clean(hostPath), string(filepath.Separator))
	data, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		t.Fatalf("read fake file %s: %v", hostPath, err)
	}
	return data
}
