#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/common.sh
source "${ROOT_DIR}/scripts/common.sh"

section() {
  printf '\n== %s ==\n' "$1"
}

try() {
  local label="$1"
  shift
  printf '\n$ %s\n' "$*"
  if "$@"; then
    printf '[ok] %s\n' "${label}"
  else
    printf '[warn] %s failed\n' "${label}"
  fi
}

section "Host"
try "kernel" uname -a
try "os release" bash -c 'test -r /etc/os-release && . /etc/os-release && printf "%s %s\n" "${PRETTY_NAME:-unknown}" "${VERSION_CODENAME:-}"'
try "architecture" uname -m

section "USB devices"
if command -v lsusb >/dev/null 2>&1; then
  try "all usb devices" lsusb
  if lsusb | grep -Ei '04e8|samsung'; then
    printf '[ok] Samsung USB device candidate found\n'
  else
    warn "Samsung USB device was not found; connect/power on the C480 before scanner setup"
  fi
else
  warn "lsusb not installed; install usbutils"
fi

section "IPP-over-USB candidate"
if command -v ipp-usb >/dev/null 2>&1; then
  try "ipp-usb check" ipp-usb check
  try "ipp-usb status" systemctl status ipp-usb --no-pager
else
  warn "ipp-usb not installed; this is optional for C480"
fi

section "SANE scanner visibility"
if command -v scanimage >/dev/null 2>&1; then
  try "scanimage as current user" scanimage -L
  if command -v sudo >/dev/null 2>&1; then
    try "scanimage as root" sudo scanimage -L
    if id saned >/dev/null 2>&1; then
      try "scanimage as saned" sudo -u saned scanimage -L
    else
      warn "saned user not found"
    fi
  fi
else
  warn "scanimage not installed; install sane-utils"
fi

section "SANE backends"
try "dll.conf smfp entry" bash -c 'test -r /etc/sane.d/dll.conf && grep -E "^[[:space:]]*smfp[[:space:]]*$" /etc/sane.d/dll.conf'
try "smfp backend library" bash -c 'for d in /usr/lib/sane /usr/lib/*/sane; do test -e "$d/libsane-smfp.so.1" && printf "%s\n" "$d/libsane-smfp.so.1"; done | sort -u'

section "CUPS printer sharing"
if command -v lpstat >/dev/null 2>&1; then
  try "lpstat" lpstat -t
else
  warn "lpstat not installed; install cups-client"
fi
if command -v cupsctl >/dev/null 2>&1; then
  try "cupsctl" cupsctl
fi

section "mDNS discovery"
if command -v avahi-browse >/dev/null 2>&1; then
  try "AirScan/eSCL services" avahi-browse -rt _uscan._tcp
  try "IPP printer services" avahi-browse -rt _ipp._tcp
else
  warn "avahi-browse not installed; install avahi-utils"
fi

section "Local web services"
if command -v curl >/dev/null 2>&1; then
  try "AirSane web" curl -fsS --max-time 2 http://localhost:8090/
else
  warn "curl not installed"
fi
