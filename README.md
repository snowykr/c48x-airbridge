# c48x-airbridge

Use a Samsung C48x/C480 USB multifunction printer from macOS and Windows
through an Ubuntu home server.

The verified setup is:

- Linux host: Samsung C48x/C480 connected over USB
- Printing: CUPS with Avahi/Bonjour discovery
- Scanning: SANE Samsung backend exposed through AirSane eSCL/AirScan
- macOS: built-in printer setup and Image Capture or Preview for scanning
- Windows: built-in network printing and NAPS2 with the `ESCL Driver` for
  scanning

Windows scanning intentionally uses NAPS2 over eSCL. The Windows built-in Scan
app may not discover this AirSane scanner, and the Samsung Universal Scan Driver
is for a scanner connected directly to the Windows PC over USB. That is not this
topology.

Korean documentation is available at [docs/README.ko.md](docs/README.ko.md).

## Requirements

- Ubuntu or Debian-based Linux host
- Samsung C48x/C480 series USB multifunction printer
- macOS and/or Windows clients on the same LAN
- A Linux account with `sudo`
- NAPS2 on Windows for scanning

If the printer is powered off or the USB cable is disconnected, CUPS and AirSane
may still be running, but printing and scanning will fail. Check power and USB
before debugging the network path.

## Clone The Repository

```bash
git clone https://github.com/snowykr/c48x-airbridge.git
cd c48x-airbridge
```

The repository includes a small Go CLI for non-mutating checks.

```bash
./bin/c48x-airbridge help
./bin/c48x-airbridge diagnose
./bin/c48x-airbridge install --dry-run
```

`install --dry-run` does not change the host. Use the scripts below when you are
ready to apply each host setup step.

## Linux Host Setup

### 1. Confirm USB Detection

Turn on the printer, connect it to the Linux host over USB, then run:

```bash
lsusb | grep -i samsung
```

Continue only after the Samsung device appears.

### 2. Install CUPS And Avahi Printing

```bash
sudo ./scripts/install-cups.sh
```

This installs CUPS, Avahi, and related printer tools. It enables printer sharing
only. It does not enable remote CUPS administration or unrestricted remote
access.

After installation, add the USB printer queue in CUPS.

```bash
lpinfo -v
lpstat -t
```

If needed, open CUPS locally on the host:

```text
http://localhost:631/
```

Add the Samsung C48x/C480 USB printer, then print a test page.

### 3. Prepare SANE And The Samsung Scanner Backend

```bash
sudo ./scripts/install-sane-samsung.sh
```

This installs SANE and USB support packages, installs the Samsung USB scanner
udev rule, and gives `saned` scanner access through the `scanner` and `lp`
groups.

Some C48x/C480 scanners need the Samsung `smfp` backend from a trusted SULDR or
ULD package. If the scanner does not appear after this script, install the
trusted Samsung/SULDR package for your host and check again.

Verify local scanner visibility through each account path:

```bash
scanimage -L
sudo scanimage -L
sudo -u saned scanimage -L
```

For AirSane, the `saned` check should see the same scanner.

### 4. Install AirSane

The AirSane script refuses to clone, build, or install by default. Pin the
upstream AirSane commit and explicitly opt in before installing on the host.

```bash
sudo AIRSANE_ALLOW_HOST_INSTALL=1 \
  AIRSANE_COMMIT=<40-character-git-commit> \
  ./scripts/install-airsane.sh
```

Verify AirSane after installation:

```bash
systemctl status airsaned --no-pager
avahi-browse -rt _uscan._tcp
curl -f http://localhost:8090/
```

When `_uscan._tcp` is advertised and the HTTP endpoint responds, LAN clients can
discover the scanner through eSCL/AirScan.

## macOS Usage

### Printing

1. Open System Settings.
2. Go to Printers & Scanners.
3. Add the Bonjour shared `Samsung C48x Series @ <host>` printer.
4. Print a test page or a real document.

### Scanning

1. Open Image Capture or Preview.
2. Select the Samsung C48x scanner shown through AirSane/AirScan.
3. Preview or scan a page.

The verified macOS path uses the built-in printer setup and built-in scanning
apps.

## Windows Usage

### Printing

1. Open Settings.
2. Go to Bluetooth & devices, then Printers & scanners.
3. Add the network printer shown as `Samsung C48x Series @ <host>` or a similar
   host-named printer.
4. Print a test page.

Windows printing uses the built-in network printer path.

### Scanning

Use NAPS2 with its eSCL driver.

1. Install NAPS2.
2. Create or edit a profile.
3. Set Driver to `ESCL Driver`.
4. Select the Samsung C48x/AirSane scanner.
5. Choose source, page size, resolution, and color settings.
6. Run Scan.

It is normal if the Windows built-in Scan app does not show the scanner in this
setup. The scanner is attached to the Linux host, not directly to the Windows PC
over USB.

## Routine Checks

Check printing on the Linux host:

```bash
lpstat -t
lpinfo -v
avahi-browse -rt _ipp._tcp
```

Check scanning on the Linux host:

```bash
scanimage -L
sudo -u saned scanimage -L
systemctl status airsaned --no-pager
avahi-browse -rt _uscan._tcp
curl -f http://localhost:8090/
```

Run the repository's non-mutating CLI check:

```bash
./bin/c48x-airbridge diagnose
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

- On Windows, use a NAPS2 profile with `ESCL Driver` instead of the built-in
  Scan app.

### Scanning Does Not Start Or Hangs

- Check the printer panel for sleep mode, cover, paper, or device errors.
- Confirm that local scanning works on the Linux host:

```bash
scanimage -L
scanimage --format=png --output-file=/tmp/c48x-test.png
```

- Restart AirSane:

```bash
sudo systemctl restart airsaned
systemctl status airsaned --no-pager
```

### Printing Works But Scanning Does Not

Printing and scanning use different host paths. Printing success does not prove
that SANE or AirSane is working.

```bash
sudo -u saned scanimage -L
curl -f http://localhost:8090/
avahi-browse -rt _uscan._tcp
```

Check these before debugging the client app.

## Repository Layout

- `bin/c48x-airbridge`: CLI entrypoint
- `cmd/c48x-airbridge/`: Go CLI main package
- `internal/cli/`: CLI command implementation
- `scripts/install-cups.sh`: CUPS and Avahi print sharing setup
- `scripts/install-sane-samsung.sh`: SANE and Samsung scanner backend setup
- `scripts/install-airsane.sh`: pinned AirSane build/install helper
- `scripts/diagnose.sh`: non-mutating host diagnostics
- `configs/udev/99-samsung-c480-scanner.rules`: Samsung USB scanner permission rule
- `configs/airsane/access.conf.example`: AirSane access control example
- `docs/README.ko.md`: Korean README
- `testdata/`: CLI test fixtures
