# c48x-airbridge

Use a Samsung C48x/C480 USB multifunction printer from macOS and Windows
through an Ubuntu home server.

The supported topology is:

- Linux host: Samsung C48x/C480 connected over USB
- Printing: CUPS shared over IPP with Avahi/Bonjour discovery
- Scanning: SANE Samsung backend exposed through AirSane eSCL/AirScan
- macOS: built-in printer setup plus Image Capture or Preview for scanning
- Windows: built-in network printing plus NAPS2 with the `ESCL Driver`

Windows scanning intentionally uses NAPS2 over eSCL. The Windows built-in Scan
app may not discover this AirSane scanner, and the Samsung Universal Scan Driver
is for scanners connected directly to the Windows PC over USB.

Korean documentation is available at [docs/README.ko.md](docs/README.ko.md).

## Verified Behavior

The current working topology was proven during the Codex setup session on a
Linux host named `snowyserver-n100`. That proof session started before this
repository was renamed from `local-printer-scanner` to `c48x-airbridge`, so old
session transcripts may contain both project names:

- Host-side CUPS, Avahi, SANE `smfp`, AirSane, IPP mDNS, and eSCL mDNS checks
  were observed working when the Samsung C48x/C480 was connected and powered.
- Windows printing worked through the network printer.
- Windows scanning worked through NAPS2 with the `ESCL Driver` over host
  AirSane/eSCL. Windows built-in Scan discovery was not the proven path.
- macOS printing and scanning were both manually confirmed through the native
  macOS printer and scanner clients.

If `verify --live` later reports `BLOCKED_PRINTER_REQUIRED`, that only means
the host cannot currently see the USB device. Power on or reconnect the printer
before rerunning setup or verification.

## Requirements

- Ubuntu or Debian-based Linux host with `sudo`
- Samsung C48x/C480 series USB multifunction printer
- macOS and/or Windows clients on the same LAN
- NAPS2 on Windows for scanning
- A trusted, existing local Samsung/SULDR scanner backend `.deb` if the host
  does not already have the Samsung `smfp` SANE backend
- AirSane builds use the project-approved upstream pin by default:
  `SimulPiscator/AirSane` tag `v0.4.12`, commit
  `129cc3bf7258251a0a694dee7741285b59d88f9f`

The Samsung/SULDR backend is the only external scanner driver artifact that may
require user action. It provides the `smfp` SANE backend, usually visible as
`/usr/lib/sane/libsane-smfp.so.1` or `/usr/lib/*/sane/libsane-smfp.so.1`. This
project does not redistribute HP/Samsung proprietary driver binaries and does
not silently download them.

If the printer is powered off or the USB cable is disconnected, services may be
running but print and scan jobs can still fail. Check power and USB first.

## Quick Start

Clone the repository on the Linux host:

```bash
git clone https://github.com/snowykr/c48x-airbridge.git
cd c48x-airbridge
```

Preview the bootstrap/build command path without changing the host:

```bash
./scripts/bootstrap-setup.sh --dry-run --no-input
```

Run the guided host setup:

```bash
./scripts/bootstrap-setup.sh --yes
```

If setup reports `BLOCKED_DRIVER_REQUIRED` for the Samsung scanner backend,
rerun it with a trusted local Samsung/SULDR driver package:

```bash
./scripts/bootstrap-setup.sh --yes \
  --suldr-deb /path/to/suld-driver2.deb
```

### Finding The Samsung Scanner Driver File

You only need this section when the host does not already have the Samsung
`smfp` SANE backend. Check first:

```bash
test -e /usr/lib/sane/libsane-smfp.so.1 || \
  ls /usr/lib/*/sane/libsane-smfp.so.1
```

For automated setup, `--suldr-deb` must point to an existing regular local
`.deb` file, not a `.tar.gz` installer or a download URL. A practical place to
keep the file is:

```bash
mkdir -p ~/Downloads/c48x-drivers
# put suld-driver2-*.deb here, then pass the real path:
./scripts/bootstrap-setup.sh --yes \
  --suldr-deb "$HOME/Downloads/c48x-drivers/suld-driver2-1.00.39.deb"
```

Known source pages to open and use yourself:

- HP support for the Samsung Xpress SL-C480 series. This page may require a
  normal browser session even when command-line tools receive HTTP 403:
  <https://support.hp.com/us-en/drivers/samsung-xpress-sl-c480-color-laser-multifunction-printer-series/16462546>
- SULDR repository overview and Samsung installer notes. These are normal
  public source pages for users to review and download from:
  <https://www.bchemnet.com/suldr/> and
  <https://www.bchemnet.com/suldr/suld.html>

HP may provide a Samsung Unified Linux Driver `uld_*.tar.gz` instead of a `.deb`.
The CLI does not install that archive directly. Either install that ULD package
manually so `smfp` is present before rerunning setup, or provide a trusted
SULDR-packaged `suld-driver2-*.deb` to `--suldr-deb`.

In the working host setup, this HP/Samsung driver-set path is represented by the
installed `smfp` SANE backend. The project checks for `libsane-smfp.so.1` and a
single `smfp` entry in `/etc/sane.d/dll.conf`; it does not need a separate
Windows-style Samsung scan application on the Linux host. In other words, this
project consumes either an already-installed `smfp` backend or an explicit local
`.deb` path that you personally obtained. If the HP download is the only
artifact you have, use it as a manual preinstall step:

```bash
mkdir -p ~/Downloads/c48x-drivers
cd ~/Downloads/c48x-drivers
# place the HP/Samsung uld_*.tar.gz here
tar -xzf uld_*.tar.gz
cd uld
sudo ./install.sh

# confirm the backend exists, then return to c48x-airbridge
test -e /usr/lib/sane/libsane-smfp.so.1 || \
  ls /usr/lib/*/sane/libsane-smfp.so.1
cd /path/to/c48x-airbridge
./scripts/bootstrap-setup.sh --yes
```

AirSane source is not fetched from a floating branch or tag during normal
setup. The default source is the approved upstream tag `v0.4.12` pinned to
commit `129cc3bf7258251a0a694dee7741285b59d88f9f`.

The bootstrap script checks for Go/build tooling, builds the CLI without
`sudo go run`, and then runs `c48x-airbridge setup`. If Go is missing, dry-run
and no-input modes print the exact `apt-get` command instead of changing the
host.

You can use Make targets for the same entrypoints:

```bash
make setup-dry-run
make setup
```

`setup` keeps a review/apply boundary before privileged work. It stops with a
named state instead of guessing:

- `PASS`: selected host checks passed.
- `BLOCKED_PRINTER_REQUIRED`: power on or connect the USB printer, then rerun.
- `BLOCKED_DRIVER_REQUIRED`: provide a trusted Samsung/SULDR `.deb` when the
  Samsung scanner backend is missing, or replace an invalid AirSane override
  with a 40-character commit if the output asks for it.
- `BLOCKED_CLIENT_PROOF`: host setup is ready; finish macOS/Windows client
  print and scan checks.
- `FAIL`: read the reason, fix the host issue, and rerun setup or verify.

## Verify The Host

Run a non-mutating diagnostic summary:

```bash
./bin/c48x-airbridge diagnose
```

Write a structured host verification bundle:

```bash
./bin/c48x-airbridge verify --live --output ./host-verify.json
```

When host checks pass, `verify` prints the macOS and Windows client handoff. The
client steps are manual because they happen on those devices.

## macOS Client Setup

### Printing

1. Open System Settings.
2. Go to Printers & Scanners.
3. Add the Bonjour shared `Samsung C48x Series @ <host>` printer.
4. Print a test page or a real document.

### Scanning

1. Open Image Capture or Preview.
2. Select the Samsung C48x scanner advertised by AirSane/AirScan.
3. Preview or scan a page.

macOS uses its built-in print and scan apps in this topology.

## Windows Client Setup

### Printing

1. Open Settings.
2. Go to Bluetooth & devices, then Printers & scanners.
3. Add the network printer shown as `Samsung C48x Series @ <host>` or a similar
   host-named printer.
4. Print a test page.

### Scanning

Use NAPS2 with its eSCL driver.

1. Install NAPS2.
2. Create or edit a profile.
3. Set Driver to `ESCL Driver`.
4. Select the Samsung C48x/AirSane scanner.
5. Choose source, page size, resolution, and color settings.
6. Run Scan.

It is normal if the Windows built-in Scan app does not show the scanner. The
scanner is attached to the Linux host, not directly to the Windows PC over USB.

## Useful Commands

```bash
./bin/c48x-airbridge help
./bin/c48x-airbridge setup --help
./bin/c48x-airbridge setup --dry-run
./bin/c48x-airbridge verify --live --output ./host-verify.json
make check
```

## Troubleshooting

### Clients Cannot See The Printer Or Scanner

- Confirm that the printer is powered on and connected over USB.
- Confirm that the Linux host and clients are on the same LAN.
- Check Avahi:

```bash
systemctl status avahi-daemon --no-pager
avahi-browse -rt _ipp._tcp
avahi-browse -rt _uscan._tcp
```

### The Host Sees The Scanner But Clients Do Not

- Check firewall rules for CUPS/IPP and AirSane/eSCL.
- From a client, test:

```bash
curl http://<host>:8090/eSCL/ScannerStatus
```

- On Windows, use a NAPS2 profile with `ESCL Driver`.

### `BLOCKED_DRIVER_REQUIRED`

The installer did not find a safe Samsung scanner backend, or an explicit
AirSane override was not a 40-character commit. Rerun setup with the exact
option named in the output.

For the Samsung scanner backend:

```bash
./scripts/bootstrap-setup.sh --yes \
  --suldr-deb /path/to/suld-driver2.deb
```

If you only found an HP/Samsung `uld_*.tar.gz`, do not pass it to
`--suldr-deb`. Install it manually first, confirm `libsane-smfp.so.1` exists,
then rerun setup without `--suldr-deb`; or use an existing local trusted SULDR
`.deb` package that you personally obtained. That is the same driver-set path
used by the verified host setup: once ULD has installed `smfp`,
`c48x-airbridge setup` treats it as an already-installed Samsung scanner
backend.

For AirSane advanced override:

```bash
./scripts/bootstrap-setup.sh --yes \
  --airsane-commit <40-character-AirSane-commit>
```

Normal setup does not require `--airsane-commit`; it uses the approved upstream
pin `129cc3bf7258251a0a694dee7741285b59d88f9f`.

### Printing Works But Scanning Does Not

Printing and scanning use different host paths. Printing success does not prove
that SANE or AirSane is working.

```bash
scanimage -L
sudo -u saned scanimage -L
curl -f http://localhost:8090/eSCL/ScannerStatus
avahi-browse -rt _uscan._tcp
```

### Manual Fallback

Use the scripts below only for troubleshooting or targeted repair after the
guided setup output tells you which component is blocked:

```bash
sudo ./scripts/install-cups.sh
sudo ./scripts/install-sane-samsung.sh
sudo AIRSANE_ALLOW_HOST_INSTALL=1 \
  AIRSANE_COMMIT=<40-character-AirSane-commit> \
  ./scripts/install-airsane.sh
```

## Repository Layout

- `bin/c48x-airbridge`: CLI entrypoint
- `cmd/c48x-airbridge/`: Go CLI main package
- `internal/cli/`: CLI command implementation
- `scripts/bootstrap-setup.sh`: one-command host setup entrypoint
- `scripts/install-cups.sh`: CUPS/Avahi repair helper
- `scripts/install-sane-samsung.sh`: SANE/Samsung scanner backend repair helper
- `scripts/install-airsane.sh`: pinned AirSane build/install repair helper
- `scripts/diagnose.sh`: legacy non-mutating host diagnostics
- `configs/udev/99-samsung-c480-scanner.rules`: Samsung USB scanner permission rule
- `configs/airsane/access.conf.example`: AirSane access control example
- `docs/README.ko.md`: Korean README
- `testdata/`: CLI test fixtures
