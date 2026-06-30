#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/common.sh
source "${ROOT_DIR}/scripts/common.sh"

AIRSANE_REPO="${AIRSANE_REPO:-https://github.com/SimulPiscator/AirSane.git}"
AIRSANE_SRC="${AIRSANE_SRC:-/usr/local/src/AirSane}"
AIRSANE_COMMIT="${AIRSANE_COMMIT:-}"
AIRSANE_ALLOW_HOST_INSTALL="${AIRSANE_ALLOW_HOST_INSTALL:-0}"

if [[ "${AIRSANE_ALLOW_HOST_INSTALL}" != "1" ]]; then
  warn "AirSane host build/install is disabled by default."
  log "Planning only: set AIRSANE_ALLOW_HOST_INSTALL=1 and AIRSANE_COMMIT=<40-hex commit> to allow a pinned host install."
  log "Before installing local AirSane patches, record upstream version, upstream source, failed gate, artifact, and rollback metadata."
  if [[ "${DRY_RUN}" == "1" ]]; then
    exit 0
  fi
  fail "refusing to clone/build/install AirSane without explicit pinned host-install opt-in"
fi

if [[ ! "${AIRSANE_COMMIT}" =~ ^[0-9a-fA-F]{40}$ ]]; then
  fail "AIRSANE_COMMIT must be a pinned 40-character git commit before host install"
fi

as_root
log "Installing AirSane build dependencies"
apt_install git cmake g++ make pkg-config libsane-dev libjpeg-dev libpng-dev libavahi-client-dev libusb-1.0-0-dev curl avahi-utils

if [[ ! -d "${AIRSANE_SRC}/.git" ]]; then
  log "Cloning AirSane into ${AIRSANE_SRC} without checking out a floating branch"
  run git clone --no-checkout "${AIRSANE_REPO}" "${AIRSANE_SRC}"
else
  log "Using existing AirSane source in ${AIRSANE_SRC}"
  run git -C "${AIRSANE_SRC}" remote set-url origin "${AIRSANE_REPO}"
fi

log "Checking out pinned AirSane commit ${AIRSANE_COMMIT}"
run git -C "${AIRSANE_SRC}" fetch --tags origin "${AIRSANE_COMMIT}"
run git -C "${AIRSANE_SRC}" checkout --detach "${AIRSANE_COMMIT}"
if [[ "${DRY_RUN}" != "1" ]]; then
  actual_commit="$(git -C "${AIRSANE_SRC}" rev-parse HEAD)"
  if [[ "${actual_commit}" != "${AIRSANE_COMMIT,,}" ]]; then
    fail "AirSane checkout did not resolve to requested commit ${AIRSANE_COMMIT}"
  fi
fi

log "Building AirSane from pinned commit ${AIRSANE_COMMIT}"
run cmake -S "${AIRSANE_SRC}" -B "${AIRSANE_SRC}/build" -DCMAKE_BUILD_TYPE=Release
run cmake --build "${AIRSANE_SRC}/build" --parallel
run cmake --install "${AIRSANE_SRC}/build"

if [[ -f "${ROOT_DIR}/configs/airsane/access.conf.example" ]]; then
  run install -D -m 0644 "${ROOT_DIR}/configs/airsane/access.conf.example" /etc/airsane/access.conf.example
fi

if [[ "${DRY_RUN}" == "1" ]]; then
  restart_service airsaned.service
elif [[ -f /usr/lib/systemd/system/airsaned.service || -f /lib/systemd/system/airsaned.service || -f /etc/systemd/system/airsaned.service ]]; then
  restart_service airsaned.service
else
  warn "AirSane unit was not found after install. Check upstream install output and create/enable airsaned.service if needed."
fi

log "Verify with: systemctl status airsaned --no-pager; avahi-browse -rt _uscan._tcp; curl http://localhost:8090/"
