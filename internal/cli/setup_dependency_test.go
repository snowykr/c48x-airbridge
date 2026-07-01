package cli

import (
	"strings"
	"testing"
)

func Test_SetupDependencyResolver_prefersInstalledBackend_whenAvailable(t *testing.T) {
	// Given
	request := setupDependencyRequest{
		InstalledSamsungBackend: true,
		SULDRDeb:                "/tmp/suld-driver2.deb",
	}

	// When
	got := resolveSetupDependencies(request)

	// Then
	if got.Driver.Source != setupDriverSourceInstalled {
		t.Fatalf("resolver did not prefer installed backend: %+v", got.Driver)
	}
	if got.State != setupStatePass {
		t.Fatalf("resolver returned state %q, want %q", got.State, setupStatePass)
	}
}

func Test_SetupDependencyResolver_acceptsPinnedAirSaneCommit_whenConfigured(t *testing.T) {
	// Given
	request := setupDependencyRequest{
		AirSaneCommit: "0123456789abcdef0123456789abcdef01234567",
	}

	// When
	got := resolveSetupDependencies(request)

	// Then
	if got.AirSane.Source != setupSourcePinned {
		t.Fatalf("resolver did not mark AirSane as pinned source: %+v", got.AirSane)
	}
	if got.AirSane.Commit != request.AirSaneCommit {
		t.Fatalf("resolver commit = %q, want %q", got.AirSane.Commit, request.AirSaneCommit)
	}
}

func Test_SetupDependencyResolver_acceptsExplicitLocalDeb_whenProvided(t *testing.T) {
	// Given
	request := setupDependencyRequest{
		SULDRDeb: "/home/user/Downloads/suld-driver2-1.00.39.deb",
	}

	// When
	got := resolveSetupDependencies(request)

	// Then
	if got.Driver.Source != setupDriverSourceLocalDeb {
		t.Fatalf("resolver did not accept explicit local .deb: %+v", got.Driver)
	}
	if got.Driver.Path != request.SULDRDeb {
		t.Fatalf("resolver local .deb path = %q, want %q", got.Driver.Path, request.SULDRDeb)
	}
	if got.State != setupStatePass {
		t.Fatalf("resolver returned state %q, want %q", got.State, setupStatePass)
	}
}

func Test_SetupDependencyResolver_prefersPinnedDriverMetadata_beforeExplicitLocalDeb(t *testing.T) {
	// Given
	request := setupDependencyRequest{
		SULDRDeb: "/home/user/Downloads/suld-driver2-1.00.39.deb",
		Metadata: setupDependencyMetadata{
			AirSaneRepo:              defaultSetupDependencyMetadata.AirSaneRepo,
			SamsungDriverPackage:     "suld-driver2-1.00.39.deb",
			SamsungDriverSHA256:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			SamsungDriverProvenance:  "user-approved local inventory",
			SamsungDriverUserAllowed: true,
		},
	}

	// When
	got := resolveSetupDependencies(request)

	// Then
	if got.Driver.Source != setupDriverSourcePinnedMeta {
		t.Fatalf("resolver did not prefer pinned driver metadata: %+v", got.Driver)
	}
	if got.Driver.Package != request.Metadata.SamsungDriverPackage {
		t.Fatalf("resolver package = %q, want %q", got.Driver.Package, request.Metadata.SamsungDriverPackage)
	}
}

func Test_SetupDependencyResolver_rejectsFloatingSource_whenConfigured(t *testing.T) {
	// Given
	request := setupDependencyRequest{
		AirSaneCommit: "main",
	}

	// When
	got := resolveSetupDependencies(request)

	// Then
	if got.AirSane.Source != setupSourceRejected {
		t.Fatalf("resolver did not reject floating AirSane source: %+v", got.AirSane)
	}
	if !strings.Contains(got.Reason, "floating") {
		t.Fatalf("resolver reason did not explain floating source rejection: %q", got.Reason)
	}
}

func Test_SetupDependencyResolver_blocksDriver_whenNoSafeSourceExists(t *testing.T) {
	// Given
	request := setupDependencyRequest{}

	// When
	got := resolveSetupDependencies(request)

	// Then
	if got.State != setupStateBlockedDriverRequired {
		t.Fatalf("resolver returned state %q, want %q", got.State, setupStateBlockedDriverRequired)
	}
	want := "BLOCKED_DRIVER_REQUIRED: Samsung scanner backend is not installed and no safe pinned or user-approved driver package metadata is configured; provide a trusted local .deb with --suldr-deb."
	if got.Reason != want {
		t.Fatalf("resolver reason = %q, want %q", got.Reason, want)
	}
}
