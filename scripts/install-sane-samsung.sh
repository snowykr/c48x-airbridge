#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/common.sh
source "${ROOT_DIR}/scripts/common.sh"

as_root
log "Installing SANE, USB, and compatibility packages"
apt_install sane-utils libsane1 libsane-common libusb-0.1-4 usbutils acl curl ca-certificates gnupg

log "Installing Samsung USB scanner udev rule"
copy_file "${ROOT_DIR}/configs/udev/99-samsung-c480-scanner.rules" "/etc/udev/rules.d/99-samsung-c480-scanner.rules"
run udevadm control --reload-rules
run udevadm trigger

log "Ensuring scanner group exists"
run groupadd -f scanner

if id saned >/dev/null 2>&1; then
  log "Adding saned to scanner and lp groups"
  run usermod -aG scanner,lp saned
fi

if [[ -n "${SULDR_DEB:-}" ]]; then
  log "Installing SULDR/Samsung driver package from SULDR_DEB=${SULDR_DEB}"
  run apt-get install -y "${SULDR_DEB}"
else
  warn "Samsung ULD/SULDR package is not bundled. Install a trusted suld-driver2-* package if scanimage still cannot see the C480."
fi

if [[ -r /etc/sane.d/dll.conf ]] && ! grep -Eq '^[[:space:]]*smfp[[:space:]]*$' /etc/sane.d/dll.conf; then
  log "Adding smfp backend to /etc/sane.d/dll.conf"
  if [[ "${DRY_RUN}" == "1" ]]; then
    printf '+ append smfp to /etc/sane.d/dll.conf\n'
  else
    printf '\nsmfp\n' >> /etc/sane.d/dll.conf
  fi
fi

log "Reconnect the USB cable, then run: scanimage -L; sudo scanimage -L; sudo -u saned scanimage -L"
