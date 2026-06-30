#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DRY_RUN="${DRY_RUN:-0}"

log() {
  printf '[local-printer-scanner] %s\n' "$*"
}

warn() {
  printf '[local-printer-scanner] WARN: %s\n' "$*" >&2
}

fail() {
  printf '[local-printer-scanner] ERROR: %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

as_root() {
  if [[ "${DRY_RUN}" == "1" ]]; then
    return 0
  fi
  if [[ "${EUID}" -ne 0 ]]; then
    fail "run as root or through sudo"
  fi
}

run() {
  if [[ "${DRY_RUN}" == "1" ]]; then
    printf '+ '
    printf '%q ' "$@"
    printf '\n'
  else
    "$@"
  fi
}

run_shell() {
  if [[ "${DRY_RUN}" == "1" ]]; then
    printf '+ %s\n' "$*"
  else
    bash -c "$*"
  fi
}

apt_install() {
  as_root
  run apt-get update
  run apt-get install -y "$@"
}

copy_file() {
  local src="$1"
  local dest="$2"
  as_root
  run install -D -m 0644 "$src" "$dest"
}

restart_service() {
  as_root
  run systemctl daemon-reload
  run systemctl enable --now "$1"
}
