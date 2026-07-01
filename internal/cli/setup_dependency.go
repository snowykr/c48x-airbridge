package cli

import (
	"path/filepath"
	"regexp"
)

const blockedDriverRequiredReason = "BLOCKED_DRIVER_REQUIRED: Samsung scanner backend is not installed and no safe pinned or user-approved driver package metadata is configured; provide a trusted local .deb with --suldr-deb."

var fortyHexCommitPattern = regexp.MustCompile(`\A[0-9a-fA-F]{40}\z`)

type setupSource string

const (
	setupSourceUnavailable setupSource = "unavailable"
	setupSourcePinned      setupSource = "pinned"
	setupSourceRejected    setupSource = "rejected"
)

type setupDriverSource string

const (
	setupDriverSourceBlocked    setupDriverSource = "blocked"
	setupDriverSourceInstalled  setupDriverSource = "installed"
	setupDriverSourceLocalDeb   setupDriverSource = "local-deb"
	setupDriverSourcePinnedMeta setupDriverSource = "pinned-metadata"
)

type setupDependencyMetadata struct {
	AirSaneRepo              string
	AirSaneApprovedTag       string
	AirSaneApprovedCommit    string
	SamsungDriverPackage     string
	SamsungDriverSHA256      string
	SamsungDriverProvenance  string
	SamsungDriverUserAllowed bool
}

type setupDependencyRequest struct {
	InstalledSamsungBackend bool
	SULDRDeb                string
	AirSaneCommit           string
	Metadata                setupDependencyMetadata
}

type setupAirSaneResolution struct {
	Source setupSource
	Repo   string
	Commit string
}

type setupDriverResolution struct {
	Source     setupDriverSource
	Path       string
	Package    string
	SHA256     string
	Provenance string
}

type setupDependencyResolution struct {
	State   string
	Reason  string
	AirSane setupAirSaneResolution
	Driver  setupDriverResolution
}

var defaultSetupDependencyMetadata = setupDependencyMetadata{
	AirSaneRepo:           "https://github.com/SimulPiscator/AirSane.git",
	AirSaneApprovedTag:    "v0.4.12",
	AirSaneApprovedCommit: "129cc3bf7258251a0a694dee7741285b59d88f9f",

	SamsungDriverProvenance: "Samsung/SULDR driver package is not bundled; explicit local .deb or pinned approved metadata is required.",
}

func newSetupDependencyRequest(options setupOptions) setupDependencyRequest {
	return setupDependencyRequest{
		SULDRDeb:      options.SULDRDeb,
		AirSaneCommit: options.AirSaneCommit,
		Metadata:      defaultSetupDependencyMetadata,
	}
}

func resolveSetupDependencies(request setupDependencyRequest) setupDependencyResolution {
	metadata := normalizeSetupDependencyMetadata(request.Metadata)
	airSane := resolveAirSaneSource(metadata, request.AirSaneCommit)
	if airSane.Source == setupSourceRejected {
		return setupDependencyResolution{
			State:   setupStateBlockedDriverRequired,
			Reason:  "BLOCKED_DRIVER_REQUIRED: refusing floating AirSane source; provide a 40-character --airsane-commit before setup can fetch source.",
			AirSane: airSane,
			Driver:  resolveSamsungDriverSource(request, metadata),
		}
	}
	driver := resolveSamsungDriverSource(request, metadata)
	if driver.Source == setupDriverSourceBlocked {
		return setupDependencyResolution{
			State:   setupStateBlockedDriverRequired,
			Reason:  blockedDriverRequiredReason,
			AirSane: airSane,
			Driver:  driver,
		}
	}
	return setupDependencyResolution{
		State:   setupStatePass,
		Reason:  "dependency resolver accepted safe source metadata",
		AirSane: airSane,
		Driver:  driver,
	}
}

func resolveAirSaneSource(metadata setupDependencyMetadata, commit string) setupAirSaneResolution {
	metadata = normalizeSetupDependencyMetadata(metadata)
	resolution := setupAirSaneResolution{
		Source: setupSourceUnavailable,
		Repo:   metadata.AirSaneRepo,
	}
	if commit == "" {
		if metadata.AirSaneApprovedCommit == "" {
			return resolution
		}
		resolution.Source = setupSourcePinned
		resolution.Commit = metadata.AirSaneApprovedCommit
		return resolution
	}
	resolution.Commit = commit
	if !fortyHexCommitPattern.MatchString(commit) {
		resolution.Source = setupSourceRejected
		return resolution
	}
	resolution.Source = setupSourcePinned
	return resolution
}

func normalizeSetupDependencyMetadata(metadata setupDependencyMetadata) setupDependencyMetadata {
	if metadata.AirSaneRepo == "" {
		metadata.AirSaneRepo = defaultSetupDependencyMetadata.AirSaneRepo
	}
	if metadata.AirSaneApprovedTag == "" {
		metadata.AirSaneApprovedTag = defaultSetupDependencyMetadata.AirSaneApprovedTag
	}
	if metadata.AirSaneApprovedCommit == "" {
		metadata.AirSaneApprovedCommit = defaultSetupDependencyMetadata.AirSaneApprovedCommit
	}
	if metadata.SamsungDriverProvenance == "" {
		metadata.SamsungDriverProvenance = defaultSetupDependencyMetadata.SamsungDriverProvenance
	}
	return metadata
}

func resolveSamsungDriverSource(request setupDependencyRequest, metadata setupDependencyMetadata) setupDriverResolution {
	if request.InstalledSamsungBackend {
		return setupDriverResolution{Source: setupDriverSourceInstalled}
	}
	if metadata.SamsungDriverPackage != "" && metadata.SamsungDriverSHA256 != "" && metadata.SamsungDriverUserAllowed {
		return setupDriverResolution{
			Source:     setupDriverSourcePinnedMeta,
			Package:    metadata.SamsungDriverPackage,
			SHA256:     metadata.SamsungDriverSHA256,
			Provenance: metadata.SamsungDriverProvenance,
		}
	}
	if request.SULDRDeb != "" && filepath.Ext(request.SULDRDeb) == ".deb" {
		return setupDriverResolution{
			Source: setupDriverSourceLocalDeb,
			Path:   request.SULDRDeb,
		}
	}
	return setupDriverResolution{
		Source:     setupDriverSourceBlocked,
		Provenance: metadata.SamsungDriverProvenance,
	}
}

func setupDependencyPlanLines(options setupOptions) []string {
	resolution := resolveSetupDependencies(newSetupDependencyRequest(options))
	lines := []string{
		"dependency resolver state: " + resolution.State,
		"Samsung backend source: " + string(resolution.Driver.Source),
		"AirSane source: " + string(resolution.AirSane.Source),
		"AirSane repo: " + resolution.AirSane.Repo,
	}
	if resolution.Driver.Path != "" {
		lines = append(lines, "Samsung driver .deb: "+resolution.Driver.Path)
	}
	if resolution.AirSane.Commit != "" {
		lines = append(lines, "AirSane commit: "+resolution.AirSane.Commit)
	}
	if resolution.State == setupStateBlockedDriverRequired {
		lines = append(lines, "dependency resolver reason: "+resolution.Reason)
	}
	return lines
}
