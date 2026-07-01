package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type guidedEvidenceRequest struct {
	root     string
	options  setupOptions
	fixture  setupRunnerFixture
	sections []guidedSetupSection
	result   RunResult
}

type guidedSetupEvidence struct {
	State          string               `json:"state"`
	Reason         string               `json:"reason,omitempty"`
	Component      string               `json:"component"`
	Fixture        string               `json:"fixture"`
	Sections       []guidedSetupSection `json:"sections"`
	CommandLogPath string               `json:"command_log_path"`
	Commands       []string             `json:"commands,omitempty"`
	ClientProof    string               `json:"client_proof,omitempty"`
	Redacted       bool                 `json:"redacted"`
}

type guidedSetupReport struct {
	streams      Streams
	options      setupOptions
	fixture      setupRunnerFixture
	sections     []guidedSetupSection
	result       RunResult
	evidencePath string
}

func writeGuidedSetupEvidence(request guidedEvidenceRequest) (string, error) {
	bundle := guidedSetupEvidence{
		State:          request.result.State,
		Reason:         redactSetupEvidence(request.result.Reason, request.root),
		Component:      string(request.options.Component),
		Fixture:        request.fixture.Name,
		Sections:       redactGuidedSections(request.sections, request.root),
		CommandLogPath: request.result.CommandLogPath,
		Commands:       redactCommandLines(request.result.CommandLines, request.root),
		Redacted:       true,
	}
	if request.result.State == setupStateBlockedClientProof {
		bundle.ClientProof = "manual macOS/Windows print and scan proof required"
	}
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode guided setup evidence: %w", err)
	}
	target, err := rootedFixturePath(request.root, guidedSetupEvidencePath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", fmt.Errorf("create guided setup evidence parent: %w", err)
	}
	if err := os.WriteFile(target, append(data, '\n'), defaultFileMode); err != nil {
		return "", fmt.Errorf("write guided setup evidence: %w", err)
	}
	return guidedSetupEvidencePath, nil
}

func redactGuidedSections(sections []guidedSetupSection, root string) []guidedSetupSection {
	redacted := make([]guidedSetupSection, 0, len(sections))
	for _, section := range sections {
		section.Detail = redactSetupEvidence(section.Detail, root)
		redacted = append(redacted, section)
	}
	return redacted
}

func redactCommandLines(commandLines []string, root string) []string {
	redacted := make([]string, 0, len(commandLines))
	for _, line := range commandLines {
		redacted = append(redacted, redactSetupEvidence(line, root))
	}
	return redacted
}

func redactSetupEvidence(value string, root string) string {
	if root != "" {
		value = strings.ReplaceAll(value, root, "<fake-root>")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		value = strings.ReplaceAll(value, home, "<home>")
	}
	return value
}

func printGuidedSetupResult(report guidedSetupReport) error {
	lines := []string{
		"setup review:",
		"component: " + string(report.options.Component),
		"review/apply boundary: approved before host mutation",
	}
	for _, section := range report.sections {
		lines = append(lines, "section: "+section.Name+" "+section.State)
	}
	lines = append(lines,
		"",
		"setup apply result:",
		"fixture: "+report.fixture.Name,
		"component: "+string(report.options.Component),
		"state: "+report.result.State,
	)
	if report.result.Reason != "" {
		lines = append(lines, "reason: "+report.result.Reason)
	}
	lines = append(lines,
		"rollback: "+report.result.RollbackGuidance,
		"retry: "+report.result.RetryGuidance,
		"command log: "+report.result.CommandLogPath,
		"evidence bundle: "+report.evidencePath,
	)
	if len(report.result.CommandLines) > 0 {
		lines = append(lines, "commands:")
		for _, commandLine := range report.result.CommandLines {
			lines = append(lines, "- "+commandLine)
		}
	}
	_, err := fmt.Fprintln(report.streams.Out, strings.Join(lines, "\n"))
	return err
}
