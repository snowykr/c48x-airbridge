package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	setupStatePass                  = "PASS"
	setupStateBlockedDriverRequired = "BLOCKED_DRIVER_REQUIRED"
	setupStateBlockedClientProof    = "BLOCKED_CLIENT_PROOF"
	setupStateFail                  = "FAIL"
)

var (
	errSetupPromptRequired  = errors.New("prompt required: rerun with --dry-run to review the setup plan or --yes to approve it")
	errSetupFakeRootMissing = errors.New("C48X_AIRBRIDGE_FAKE_ROOT must point at an existing directory when --fake-runner is used")
)

type setupOptions struct {
	DryRun         bool
	Yes            bool
	NonInteractive bool
	Force          bool
	SULDRDeb       string
	AirSaneCommit  string
	Component      setupComponent
	FakeRunner     string
}

type setupComponent string

const (
	setupComponentAll     setupComponent = "all"
	setupComponentCUPS    setupComponent = "cups"
	setupComponentScanner setupComponent = "scanner"
	setupComponentAirSane setupComponent = "airsane"
	setupComponentVerify  setupComponent = "verify"
)

func newSetupCommand(streams Streams) *cobra.Command {
	options := setupOptions{Component: setupComponentAll}
	component := string(options.Component)
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Guide Linux host setup for the Samsung C48x bridge",
		Long: strings.TrimSpace(`
Guide Linux host setup for the Samsung C48x bridge.

The setup command uses line-based prompts, supports non-interactive review with
--dry-run, and keeps a review/apply boundary before any privileged host action.
This contract scaffold does not mutate the host yet.

Completion states: PASS, BLOCKED_DRIVER_REQUIRED, BLOCKED_CLIENT_PROOF, FAIL.
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			parsed, err := parseSetupOptions(options, component)
			if err != nil {
				return err
			}
			return runSetup(cmd, streams, parsed)
		},
	}
	cmd.Flags().BoolVar(&options.DryRun, "dry-run", false, "print the reviewed setup plan without mutating the host")
	cmd.Flags().BoolVar(&options.Yes, "yes", false, "approve the reviewed setup plan without prompting")
	cmd.Flags().BoolVar(&options.NonInteractive, "no-input", false, "fail instead of prompting for setup decisions")
	cmd.Flags().BoolVar(&options.Force, "force", false, "allow rebuilding or repairing setup steps that already appear present")
	cmd.Flags().StringVar(&options.SULDRDeb, "suldr-deb", "", "path to a locally provided Samsung/SULDR driver .deb")
	cmd.Flags().StringVar(&options.AirSaneCommit, "airsane-commit", "", "pinned AirSane git commit to use when AirSane installation is selected")
	cmd.Flags().StringVar(&component, "component", string(setupComponentAll), "setup component: all, cups, scanner, airsane, or verify")
	cmd.Flags().StringVar(&options.FakeRunner, "fake-runner", "", "test-only fake runner fixture")
	if flag := cmd.Flags().Lookup("fake-runner"); flag != nil {
		flag.Hidden = true
	}
	return cmd
}

func parseSetupOptions(options setupOptions, component string) (setupOptions, error) {
	parsedComponent, err := newSetupComponent(component)
	if err != nil {
		return setupOptions{}, err
	}
	options.Component = parsedComponent
	if options.FakeRunner != "" {
		if err := validateFakeRunner(options.FakeRunner); err != nil {
			return setupOptions{}, err
		}
	}
	return options, nil
}

func newSetupComponent(value string) (setupComponent, error) {
	component := setupComponent(value)
	switch component {
	case setupComponentAll, setupComponentCUPS, setupComponentScanner, setupComponentAirSane, setupComponentVerify:
		return component, nil
	default:
		return "", fmt.Errorf("invalid component %q: expected all, cups, scanner, airsane, or verify", value)
	}
}

func validateFakeRunner(path string) error {
	if !strings.HasSuffix(filepath.ToSlash(path), ".json") || !strings.Contains(filepath.ToSlash(path), "testdata/setup/") {
		return fmt.Errorf("fake-runner %q must be a JSON fixture under testdata/setup: %s", path, setupStateFail)
	}
	root := os.Getenv("C48X_AIRBRIDGE_FAKE_ROOT")
	if root == "" {
		return errSetupFakeRootMissing
	}
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("stat C48X_AIRBRIDGE_FAKE_ROOT: %w", errSetupFakeRootMissing)
	}
	if !info.IsDir() {
		return errSetupFakeRootMissing
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read fake-runner fixture %s: %w", path, err)
	}
	var fixture map[string]json.RawMessage
	if err := json.Unmarshal(data, &fixture); err != nil {
		return fmt.Errorf("parse fake-runner fixture %s: %w", path, err)
	}
	return nil
}

func runSetup(cmd *cobra.Command, streams Streams, options setupOptions) error {
	if options.NonInteractive && !options.DryRun && !options.Yes {
		return errSetupPromptRequired
	}
	if options.DryRun {
		_, err := fmt.Fprintln(streams.Out, strings.Join(setupPlan(options), "\n"))
		return err
	}
	if !options.Yes {
		return promptForSetupApproval(cmd, streams, options)
	}
	_, err := fmt.Fprintln(streams.Out, strings.Join(setupApplyScaffold(options), "\n"))
	return err
}

func promptForSetupApproval(cmd *cobra.Command, streams Streams, options setupOptions) error {
	if !isTerminalInput(cmd) {
		return errSetupPromptRequired
	}
	_, err := fmt.Fprintln(streams.Out, strings.Join(setupPlan(options), "\n"))
	if err != nil {
		return err
	}
	return errSetupPromptRequired
}

func isTerminalInput(cmd *cobra.Command) bool {
	return cmd.InOrStdin() == os.Stdin
}

func setupPlan(options setupOptions) []string {
	lines := []string{
		"setup dry-run plan:",
		"component: " + string(options.Component),
		"review/apply boundary: no privileged host mutation runs before approval",
		"- CUPS: plan package install, service enablement, sharing, and Samsung C48x queue repair",
		"- SANE: plan Samsung scanner backend, udev permissions, group membership, and scanimage triplet checks",
		"- AirSane: plan pinned source build, config install, service enablement, and eSCL proof",
		"- verify: plan CUPS, SANE, AirSane, mDNS, reboot persistence, and client-proof handoff checks",
		"states: " + strings.Join(setupStates(), ", "),
	}
	if options.Force {
		lines = append(lines, "force: enabled for repair/rebuild candidates")
	}
	if options.SULDRDeb != "" {
		lines = append(lines, "suldr deb: "+options.SULDRDeb)
	}
	if options.AirSaneCommit != "" {
		lines = append(lines, "airsane commit: "+options.AirSaneCommit)
	}
	if options.FakeRunner != "" {
		lines = append(lines, "fake runner fixture: "+options.FakeRunner)
	}
	return lines
}

func setupApplyScaffold(options setupOptions) []string {
	lines := setupPlan(options)
	lines[0] = "setup apply scaffold:"
	lines = append(lines, "state: "+setupStateBlockedDriverRequired)
	lines = append(lines, "reason: setup host mutation is reserved for the later guided installer workflow")
	return lines
}

func setupStates() []string {
	return []string{
		setupStatePass,
		setupStateBlockedDriverRequired,
		setupStateBlockedClientProof,
		setupStateFail,
	}
}
