#!/bin/bash
set -Eeuo pipefail

ROOT_DIR="$(cd -- "${BASH_SOURCE[0]%/*}/.." && printf '%s\n' "$PWD")"
BUILD_DIR="${ROOT_DIR}/.cache/c48x-airbridge/bootstrap"
CLI_BIN="${BUILD_DIR}/c48x-airbridge"
APT_INSTALL_GO_CMD="sudo apt-get update && sudo apt-get install -y golang-go build-essential"

DRY_RUN=0
YES=0
NO_INPUT=0
SETUP_ARGS=()

log() {
  printf '[c48x-airbridge] %s\n' "$*"
}

fail() {
  printf '[c48x-airbridge] ERROR: %s\n' "$*" >&2
  exit 1
}

has_cmd() {
  command -v "$1" >/dev/null 2>&1
}

shell_quote() {
  local arg
  for arg in "$@"; do
    printf '%q ' "$arg"
  done
}

sudo_cmd() {
  if [[ "${EUID}" -eq 0 ]]; then
    "$@"
  elif has_cmd sudo; then
    sudo "$@"
  else
    fail "sudo is required for package installation. Run manually: ${APT_INSTALL_GO_CMD}"
  fi
}

install_go() {
  log "Go/build tooling is missing."
  log "Reviewed install command: ${APT_INSTALL_GO_CMD}"

  if [[ "${DRY_RUN}" -eq 1 ]]; then
    log "Dry run: would run ${APT_INSTALL_GO_CMD}"
    return 0
  fi

  if [[ "${YES}" -eq 1 ]]; then
    has_cmd apt-get || fail "apt-get is required. Run manually: ${APT_INSTALL_GO_CMD}"
    sudo_cmd apt-get update
    sudo_cmd apt-get install -y golang-go build-essential
    return 0
  fi

  if [[ "${NO_INPUT}" -eq 1 ]]; then
    fail "Go is required. Run manually: ${APT_INSTALL_GO_CMD}"
  fi

  printf 'Install Go/build tooling now? [y/N] '
  local reply
  read -r reply
  case "${reply}" in
    y|Y|yes|YES)
      has_cmd apt-get || fail "apt-get is required. Run manually: ${APT_INSTALL_GO_CMD}"
      sudo_cmd apt-get update
      sudo_cmd apt-get install -y golang-go build-essential
      ;;
    *)
      fail "Go is required. Run manually: ${APT_INSTALL_GO_CMD}"
      ;;
  esac
}

parse_args() {
  while [[ "$#" -gt 0 ]]; do
    case "$1" in
      --dry-run)
        DRY_RUN=1
        SETUP_ARGS+=("$1")
        ;;
      --yes|-y)
        YES=1
        SETUP_ARGS+=("$1")
        ;;
      --no-input)
        NO_INPUT=1
        SETUP_ARGS+=("$1")
        ;;
      --help|-h)
        SETUP_ARGS+=("$1")
        ;;
      --)
        shift
        SETUP_ARGS+=("$@")
        break
        ;;
      *)
        SETUP_ARGS+=("$1")
        ;;
    esac
    shift
  done
}

build_cli() {
  has_cmd go || {
    install_go
    if [[ "${DRY_RUN}" -eq 1 ]]; then
      return 2
    fi
    has_cmd go || return 1
  }

  if [[ "${DRY_RUN}" -eq 1 ]]; then
    log "Dry run: would build ${CLI_BIN}"
    return 0
  fi

  mkdir -p "${BUILD_DIR}"
  go build -o "${CLI_BIN}" "${ROOT_DIR}/cmd/c48x-airbridge"
}

run_setup() {
  local display_bin="${CLI_BIN}"

  if [[ "${DRY_RUN}" -eq 1 ]]; then
    log "Dry run: would run c48x-airbridge setup"
    printf '+ '
    shell_quote "${display_bin}" setup "${SETUP_ARGS[@]}"
    printf '\n'
    return 0
  fi

  [[ -x "${CLI_BIN}" ]] || fail "built CLI not found: ${CLI_BIN}"
  exec "${CLI_BIN}" setup "${SETUP_ARGS[@]}"
}

main() {
  [[ -n "${BASH_VERSION:-}" ]] || fail "bash is required"
  parse_args "$@"

  if ! has_cmd go && [[ "${DRY_RUN}" -eq 0 ]]; then
    install_go
  fi

  local build_status=0
  if build_cli; then
    run_setup
  else
    build_status=$?
    [[ "${build_status}" -eq 2 ]] && return 0
    return "${build_status}"
  fi
}

main "$@"
