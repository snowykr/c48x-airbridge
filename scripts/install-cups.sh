#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/common.sh
source "${ROOT_DIR}/scripts/common.sh"

as_root
log "Installing CUPS, Avahi, and printer discovery tools"
apt_install cups cups-client avahi-daemon avahi-utils printer-driver-splix system-config-printer

log "Enabling CUPS and Avahi"
run systemctl enable --now cups
run systemctl enable --now avahi-daemon

log "Enabling local network printer sharing"
run cupsctl --share-printers --no-remote-admin --no-remote-any

log "CUPS is ready. Add the USB C480 through http://localhost:631/ or lpadmin after confirming lpinfo -v output."
log "Remote CUPS administration and unrestricted remote access are not enabled by this script."
log "Useful checks: lpinfo -v, lpstat -t, avahi-browse -rt _ipp._tcp"
