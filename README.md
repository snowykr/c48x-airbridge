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

## Requirements

- Ubuntu or Debian-based Linux host with `sudo`
- Samsung C48x/C480 series USB multifunction printer
- macOS and/or Windows clients on the same LAN
- NAPS2 on Windows for scanning
- A trusted Samsung/SULDR scanner backend `.deb` if the host does not already
  have the Samsung `smfp` SANE backend
- A pinned 40-character AirSane git commit when AirSane must be built

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
./scripts/bootstrap-setup.sh --yes \
  --airsane-commit <40-character-AirSane-commit>
```

If setup reports `BLOCKED_DRIVER_REQUIRED` for the Samsung scanner backend,
rerun it with a trusted local Samsung/SULDR driver package:

```bash
./scripts/bootstrap-setup.sh --yes \
  --suldr-deb /path/to/suld-driver2.deb \
  --airsane-commit <40-character-AirSane-commit>
```

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
- `BLOCKED_DRIVER_REQUIRED`: provide a trusted Samsung/SULDR `.deb` or a pinned
  AirSane commit, as requested by the output.
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

The installer did not find a safe scanner backend or pinned AirSane source.
Rerun setup with the exact option named in the output.

For the Samsung scanner backend:

```bash
./scripts/bootstrap-setup.sh --yes \
  --suldr-deb /path/to/suld-driver2.deb \
  --airsane-commit <40-character-AirSane-commit>
```

For AirSane:

```bash
./scripts/bootstrap-setup.sh --yes \
  --airsane-commit <40-character-AirSane-commit>
```

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
