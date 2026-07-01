package hostprobe

import "context"

type Status string

const (
	StatusPass    Status = "PASS"
	StatusWarn    Status = "WARN"
	StatusBlocked Status = "BLOCKED"
)

type CheckID string

const (
	CheckOSRelease          CheckID = "host.os_release"
	CheckArchitecture       CheckID = "host.architecture"
	CheckCommandLSUSB       CheckID = "command.lsusb"
	CheckCommandScanimage   CheckID = "command.scanimage"
	CheckCommandLPStat      CheckID = "command.lpstat"
	CheckCommandAvahiBrowse CheckID = "command.avahi_browse"
	CheckCommandCurl        CheckID = "command.curl"
	CheckUSBDevice          CheckID = "usb.samsung_c48x"
	CheckSANECurrentUser    CheckID = "sane.current_user"
	CheckSANERoot           CheckID = "sane.root"
	CheckSANESaned          CheckID = "sane.saned"
	CheckSMFPConfig         CheckID = "sane.smfp_config"
	CheckSMFPBackend        CheckID = "sane.smfp_backend"
	CheckCUPSService        CheckID = "cups.service"
	CheckCUPSQueue          CheckID = "cups.queue"
	CheckAvahiService       CheckID = "avahi.service"
	CheckIPPService         CheckID = "avahi.ipp"
	CheckUSCANService       CheckID = "avahi.uscan"
	CheckAirSaneService     CheckID = "airsane.service"
	CheckAirSaneHTTP        CheckID = "airsane.http"
	CheckPackageUSBUtils    CheckID = "package.usbutils"
	CheckPackageSANEUtils   CheckID = "package.sane_utils"
	CheckPackageCUPSClient  CheckID = "package.cups_client"
	CheckPackageAvahiUtils  CheckID = "package.avahi_utils"
	CheckPackageCurl        CheckID = "package.curl"
)

type Result struct {
	Check    CheckID
	Section  string
	Name     string
	Status   Status
	Detail   string
	Evidence string
}

type Report struct {
	Results []Result
}

type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

type Runner interface {
	LookPath(name string) (string, error)
	Run(ctx context.Context, name string, args ...string) CommandResult
	ReadFile(path string) (string, error)
	Glob(pattern string) ([]string, error)
}
