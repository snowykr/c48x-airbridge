package hostprobe

import "context"

func (p Prober) hostResults(ctx context.Context) []Result {
	osRelease, err := p.runner.ReadFile("/etc/os-release")
	osResult := Result{Check: CheckOSRelease, Section: "Host", Name: "OS release", Status: StatusPass, Evidence: firstLine(osRelease)}
	if err != nil {
		osResult = Result{Check: CheckOSRelease, Section: "Host", Name: "OS release", Status: StatusWarn, Detail: "unable to read /etc/os-release"}
	}
	arch := p.runner.Run(ctx, "uname", "-m")
	return []Result{
		osResult,
		resultFromCommand(commandProbe{check: CheckArchitecture, section: "Host", name: "Architecture", passDetail: "architecture detected", warnDetail: "unable to detect architecture"}, arch),
	}
}

func (p Prober) commandResults() []Result {
	commands := []commandSpec{
		{CheckCommandLSUSB, "usbutils / lsusb", "lsusb"},
		{CheckCommandScanimage, "sane-utils / scanimage", "scanimage"},
		{CheckCommandLPStat, "cups-client / lpstat", "lpstat"},
		{CheckCommandAvahiBrowse, "avahi-utils / avahi-browse", "avahi-browse"},
		{CheckCommandCurl, "curl", "curl"},
	}
	results := make([]Result, 0, len(commands))
	for _, command := range commands {
		results = append(results, p.commandAvailable(command))
	}
	return results
}

func (p Prober) packageResults(ctx context.Context) []Result {
	packages := []struct {
		check CheckID
		name  string
		pkg   string
	}{
		{CheckPackageUSBUtils, "usbutils package", "usbutils"},
		{CheckPackageSANEUtils, "sane-utils package", "sane-utils"},
		{CheckPackageCUPSClient, "cups-client package", "cups-client"},
		{CheckPackageAvahiUtils, "avahi-utils package", "avahi-utils"},
		{CheckPackageCurl, "curl package", "curl"},
	}
	results := make([]Result, 0, len(packages))
	if _, err := p.runner.LookPath("dpkg-query"); err != nil {
		for _, pkg := range packages {
			results = append(results, Result{Check: pkg.check, Section: "Host", Name: pkg.name, Status: StatusWarn, Detail: "dpkg-query unavailable"})
		}
		return results
	}
	for _, pkg := range packages {
		run := p.runner.Run(ctx, "dpkg-query", "-W", "-f=${Status}", pkg.pkg)
		spec := commandProbe{check: pkg.check, section: "Host", name: pkg.name, passDetail: "package installed", warnDetail: "package missing"}
		results = append(results, resultFromCommand(spec, run))
	}
	return results
}

func (p Prober) commandAvailable(command commandSpec) Result {
	if path, err := p.runner.LookPath(command.command); err == nil {
		return Result{Check: command.check, Section: "Host", Name: command.name, Status: StatusPass, Detail: "command available", Evidence: path}
	}
	return Result{Check: command.check, Section: "Host", Name: command.name, Status: StatusWarn, Detail: "command unavailable"}
}
