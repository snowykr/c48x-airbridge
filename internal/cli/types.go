package cli

type saneTriplet struct {
	CurrentUser bool   `json:"current_user"`
	Root        bool   `json:"root"`
	Saned       bool   `json:"saned"`
	Proof       string `json:"proof,omitempty"`
}

type hostState struct {
	UbuntuVersion            string      `json:"ubuntu_version"`
	Architecture             string      `json:"architecture"`
	CUPSMajorVersion         int         `json:"cups_major_version"`
	CUPSVersionGate          string      `json:"cups_version_gate,omitempty"`
	USBVIDPID                string      `json:"usb_vid_pid"`
	LocalFlatbedScanChecksum string      `json:"local_flatbed_scan_checksum"`
	SANETriplet              saneTriplet `json:"sane_triplet"`
	AirSaneHTTP              bool        `json:"airsane_http"`
	AirSaneHTTPURL           string      `json:"airsane_http_url,omitempty"`
	AirSaneHTTPProof         string      `json:"airsane_http_proof,omitempty"`
	UscanMDNS                bool        `json:"uscan_mdns"`
	UscanMDNSService         string      `json:"uscan_mdns_service,omitempty"`
	CUPSQueue                bool        `json:"cups_queue"`
	CUPSQueueName            string      `json:"cups_queue_name,omitempty"`
	IPPMDNS                  bool        `json:"ipp_mdns"`
	IPPMDNSService           string      `json:"ipp_mdns_service,omitempty"`
	RebootPersistence        bool        `json:"reboot_persistence"`
	RebootPersistenceNote    string      `json:"reboot_persistence_note,omitempty"`
}

type evidenceItem struct {
	DiscoveryProof string `json:"discovery_proof"`
	Name           string `json:"name"`
	Result         string `json:"result"`
	Timestamp      string `json:"timestamp"`
	Note           string `json:"note"`
	LogBundleID    string `json:"log_bundle_id"`
}

type manualEvidence struct {
	MacOSPrint   *evidenceItem `json:"macos_print"`
	WindowsPrint *evidenceItem `json:"windows_print"`
	MacOSScan    *evidenceItem `json:"macos_scan"`
	WindowsScan  *evidenceItem `json:"windows_scan"`
}

type verifyFixture struct {
	Host                       hostState      `json:"host"`
	LogBundleReferences        []string       `json:"log_bundle_references,omitempty"`
	FeatureLimits              []string       `json:"feature_limits,omitempty"`
	NoVendorClientDriverPolicy string         `json:"no_vendor_client_driver_policy,omitempty"`
	ManualEvidence             manualEvidence `json:"manual_evidence"`
}

type verifyResult struct {
	State  string   `json:"state"`
	Reason string   `json:"reason"`
	Gates  []string `json:"gates"`
}

type logBundleMetadata struct {
	State                      string         `json:"state"`
	Reason                     string         `json:"reason"`
	Gates                      []string       `json:"gates"`
	LogBundleReferences        []string       `json:"log_bundle_references"`
	Host                       hostState      `json:"host"`
	ManualEvidence             manualEvidence `json:"manual_evidence"`
	FeatureLimits              []string       `json:"feature_limits"`
	NoVendorClientDriverPolicy string         `json:"no_vendor_client_driver_policy"`
}

type patchMetadata struct {
	UpstreamVersion string `json:"upstream_version"`
	UpstreamSource  string `json:"upstream_source"`
	FailedGateID    string `json:"failed_gate_id"`
	LocalDiff       string `json:"local_diff"`
	BuildCommand    string `json:"build_command"`
	BuildResult     string `json:"build_result"`
	ArtifactPath    string `json:"artifact_path"`
	RollbackNote    string `json:"rollback_note"`
}
