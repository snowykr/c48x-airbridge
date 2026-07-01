# Host Setup Automation Contract

This contract defines the Linux host behavior that `c48x-airbridge setup` must
automate before macOS or Windows clients are asked to register the shared
printer or scanner. Later implementation tasks must treat this document as the
minimum executable contract, not as a passive inventory.

## Supported Host Target

- Fully supported automated target: Ubuntu 24.04 on amd64.
- Best-effort preflight target: other Debian-based Linux hosts that provide apt,
  systemd, CUPS, Avahi, SANE, USB tooling, and a compatible Samsung C48x/C480
  USB multifunction device.
- The setup workflow must stop before privileged mutation when the OS or
  architecture is unsupported and must report a named blocked or fail state with
  the exact missing prerequisite.
- Privileged commands must run only after a review/apply boundary. The CLI must
  not require `sudo go run`.

## Required Packages

The installer must be able to plan and apply the host packages currently proven
by the local setup scripts:

- CUPS and printer discovery: `cups`, `cups-client`, `avahi-daemon`,
  `avahi-utils`, `printer-driver-splix`, and `system-config-printer`.
- Scanner and USB support: `sane-utils`, `libsane1`, `libsane-common`,
  `libusb-0.1-4`, `usbutils`, `acl`, `curl`, `ca-certificates`, and `gnupg`.
- AirSane build and runtime support: `git`, `cmake`, `g++`, `make`,
  `pkg-config`, `libsane-dev`, `libjpeg-dev`, `libpng-dev`,
  `libavahi-client-dev`, `libusb-1.0-0-dev`, `curl`, and `avahi-utils`.

Package installation must be idempotent and recorded in the reviewed apply
plan. Missing package managers or unavailable packages must not be reported as
`PASS`.

## USB And Model Detection

The setup flow must detect the Samsung C48x/C480 device on the Linux host before
host mutation that depends on the device. Detection must include:

- USB enumeration through `lsusb`.
- Samsung vendor or model evidence, including vendor id `04e8` where present.
- Clear remediation when the printer is powered off or the USB cable is
  disconnected.

A missing USB device may continue only for components that do not require the
device. It must not produce a false host-ready result.

## CUPS Queue And Printing

The installer must automate CUPS queue setup, not merely tell the user to open
the web UI. Required behavior:

- Enable and start `cups` and `avahi-daemon`.
- Enable printer sharing with `cupsctl --share-printers` while preserving the
  current safety boundary: no remote CUPS administration and no unrestricted
  remote access.
- Discover the USB printer path with `lpinfo -v`.
- Create or repair the Samsung C48x/C480 CUPS queue with an explicit driver or
  PPD strategy selected by the reviewed plan, using `lpadmin` or an equivalent
  CUPS queue API.
- Verify the CUPS queue with `lpstat -t` and IPP advertisement with
  `avahi-browse -rt _ipp._tcp`.

The CUPS queue step must be idempotent. Existing compatible queues should be
reused or repaired; incompatible queues should produce a reviewed change plan.

## SANE Backend And Scanner Permissions

The installer must automate the SANE backend path needed by the Samsung scanner:

- Install SANE, USB, and compatibility packages.
- Install `configs/udev/99-samsung-c480-scanner.rules` to
  `/etc/udev/rules.d/99-samsung-c480-scanner.rules`, then reload and trigger
  udev rules.
- Ensure the `scanner` group exists.
- Add `saned` to the `scanner` and `lp` groups when the `saned` account exists.
- Detect whether the Samsung `smfp` SANE backend is already installed.
- Back up `/etc/sane.d/dll.conf` before editing it, then append a single
  idempotent `smfp` entry when required.
- Verify scanner visibility with `scanimage -L`, `sudo scanimage -L`, and
  `sudo -u saned scanimage -L` when those account paths are available.

If the Samsung backend is not available, setup must follow the driver resolver
policy below before returning `BLOCKED_DRIVER_REQUIRED`.

## Samsung Driver Resolver Policy

The scanner driver resolver must never silently fetch or install floating,
unpinned third-party packages. It must resolve in this order:

1. Use an already installed and working Samsung `smfp` backend.
2. Use repo-configured pinned or user-approved Samsung/SULDR/ULD package
   metadata when provenance is recorded and safe to display in the review plan.
3. Use an explicit local `--suldr-deb` package path supplied by the user.
4. Return `BLOCKED_DRIVER_REQUIRED` with exact remediation when no safe source
   is available.

The blocked state is a host setup result, not a successful scanner setup.

## AirSane Source Build And Service

The installer must automate AirSane only through a pinned, reviewed source path:

- Refuse floating AirSane source installs.
- Require a 40-character AirSane commit from repo metadata or an explicit
  `AIRSANE_COMMIT` user input that is shown in the review plan.
- Clone or reuse the AirSane source directory without checking out a floating
  branch.
- Fetch and check out the exact pinned commit.
- Build with CMake and install through the reviewed command runner.
- Install `configs/airsane/access.conf.example` to
  `/etc/airsane/access.conf.example` when present.
- Restart or enable `airsaned.service` only when the unit exists, otherwise
  report actionable service guidance.
- Verify AirSane through `systemctl status airsaned --no-pager`,
  `avahi-browse -rt _uscan._tcp`, and `curl -f http://localhost:8090/`.

AirSane completion must be gated by host-visible eSCL or AirScan proof and must
not pass solely because a build command exited successfully.

## Avahi And CUPS Advertisements

Host setup must publish the services that LAN clients need:

- IPP printing through CUPS and Avahi, verified by `_ipp._tcp`.
- eSCL/AirScan scanning through AirSane and Avahi, verified by `_uscan._tcp`.

The setup workflow must report printer and scanner advertisement checks
separately so one working service cannot hide the other failing service.

## Verification Commands

The final host verification surface must run or encode these checks:

- `lsusb`
- `lpinfo -v`
- `lpstat -t`
- `scanimage -L`
- `sudo scanimage -L`
- `sudo -u saned scanimage -L`
- `systemctl status cups --no-pager`
- `systemctl status avahi-daemon --no-pager`
- `systemctl status airsaned --no-pager`
- `avahi-browse -rt _ipp._tcp`
- `avahi-browse -rt _uscan._tcp`
- `curl -f http://localhost:8090/`

Verification output must be redacted for secrets and suitable to store in an
evidence bundle.

## Completion And Blocked States

The setup workflow must use stable completion states:

- `PASS`: Linux host printing and scanning services are configured and verified.
- `BLOCKED_DRIVER_REQUIRED`: the Samsung scanner backend cannot be acquired or
  verified through a pinned, trusted, or user-supplied source.
- `BLOCKED_CLIENT_PROOF`: the Linux host is ready, but macOS or Windows client
  registration and proof remain outside the host automation boundary.
- `FAIL`: a required host setup step failed after review/apply and requires
  retry, rollback, or user intervention.

Usage errors, unsupported flags, invalid components, and unsafe noninteractive
requests must remain distinct from host setup states.

## Client Proof Boundary

The Linux host installer must not automate client UI setup. macOS and Windows
registration remains client-side registration because those actions happen on
separate devices. After host readiness, the CLI may print exact client
instructions for:

- macOS built-in printer setup and Image Capture or Preview scanning.
- Windows built-in network printing and NAPS2 with the `ESCL Driver`.

The host must not claim full end-to-end success until client-side registration
has been performed and proven by the user. This boundary is represented by
`BLOCKED_CLIENT_PROOF`, not by a hidden manual host setup step.

## Idempotency And Recovery

Every mutating setup step must be safe to rerun:

- Existing apt packages, services, groups, CUPS queues, udev rules, SANE backend
  entries, AirSane source directories, and config files must be detected before
  changes are applied.
- Edited host config files must be backed up before mutation.
- Command logs must show what was applied.
- Partial failures must include retry and rollback guidance.

The reviewed apply plan must make host mutations auditable before execution and
must not report `PASS` when any required host component remains unverified.
