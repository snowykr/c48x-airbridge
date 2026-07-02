package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
After approval it runs the guided Linux host setup workflow.

AirSane builds use the project-approved upstream pin by default. Use
--airsane-commit only for an advanced 40-character commit override; branch,
tag, and latest source names are rejected.

Completion states: PASS, BLOCKED_PRINTER_REQUIRED, BLOCKED_DRIVER_REQUIRED, BLOCKED_CLIENT_PROOF, FAIL.
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
	cmd.Flags().StringVar(&options.AirSaneCommit, "airsane-commit", "", "advanced override: 40-character AirSane git commit; default uses the approved project pin")
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
	if err := validateSULDRDeb(options.SULDRDeb); err != nil {
		return setupOptions{}, err
	}
	if options.FakeRunner != "" {
		if err := validateFakeRunner(options.FakeRunner); err != nil {
			return setupOptions{}, err
		}
	}
	return options, nil
}

func validateSULDRDeb(path string) error {
	if path == "" {
		return nil
	}
	if filepath.Ext(path) != ".deb" {
		return fmt.Errorf("--suldr-deb must point to an existing local .deb file: %s", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("--suldr-deb must point to an existing local .deb file: %s", path)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("--suldr-deb must point to an existing local .deb file: %s", path)
	}
	return nil
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
	if options.FakeRunner != "" {
		return runSetupFakeRunner(cmd.Context(), streams, options)
	}
	return runSetupReal(cmd.Context(), streams, options, defaultSetupRealRuntime())
}

func promptForSetupApproval(cmd *cobra.Command, streams Streams, options setupOptions) error {
	_, err := fmt.Fprintln(streams.Out, strings.Join(setupPlan(options), "\n"))
	if err != nil {
		return err
	}
	if _, err := fmt.Fprint(streams.Out, "\nApply setup plan? [y/N] "); err != nil {
		return err
	}
	answer, readErr := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return fmt.Errorf("read setup approval: %w", readErr)
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		options.Yes = true
		return runSetup(cmd, streams, options)
	default:
		return errSetupPromptRequired
	}
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
	lines = append(lines, setupDependencyPlanLines(options)...)
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

func setupStates() []string {
	return []string{
		setupStatePass,
		setupStateBlockedPrinterRequired,
		setupStateBlockedDriverRequired,
		setupStateBlockedClientProof,
		setupStateFail,
	}
}
