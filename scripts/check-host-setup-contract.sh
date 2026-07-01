#!/usr/bin/env bash
set -Eeuo pipefail

contract="${1:-docs/host-setup-automation.md}"

fail() {
  printf 'host setup contract check failed: %s\n' "$*" >&2
  exit 1
}

[[ -f "${contract}" ]] || fail "missing contract file: ${contract}"

require_literal() {
  local needle="$1"
  grep -Fq -- "${needle}" "${contract}" || fail "missing required text: ${needle}"
}

require_regex() {
  local pattern="$1"
  grep -Eq -- "${pattern}" "${contract}" || fail "missing required pattern: ${pattern}"
}

required_headings=(
  "# Host Setup Automation Contract"
  "## Supported Host Target"
  "## Required Packages"
  "## USB And Model Detection"
  "## CUPS Queue And Printing"
  "## SANE Backend And Scanner Permissions"
  "## Samsung Driver Resolver Policy"
  "## AirSane Source Build And Service"
  "## Avahi And CUPS Advertisements"
  "## Verification Commands"
  "## Completion And Blocked States"
  "## Client Proof Boundary"
  "## Idempotency And Recovery"
)

for heading in "${required_headings[@]}"; do
  require_literal "${heading}"
done

required_terms=(
  "Ubuntu 24.04"
  "amd64"
  "apt"
  "lsusb"
  "04e8"
  "CUPS queue"
  "lpadmin"
  "lpinfo -v"
  "lpstat -t"
  "SANE backend"
  "smfp"
  "udev"
  "scanner"
  "AirSane"
  "AIRSANE_COMMIT"
  "_ipp._tcp"
  "_uscan._tcp"
  "BLOCKED_DRIVER_REQUIRED"
  "BLOCKED_CLIENT_PROOF"
  "FAIL"
  "client-side registration"
)

for term in "${required_terms[@]}"; do
  require_literal "${term}"
done

require_regex 'Samsung/(SULDR|ULD)|SULDR/ULD|Samsung/SULDR/ULD'
require_regex 'curl -f? http://localhost:8090/|curl .*http://localhost:8090/'
require_regex 'sudo -u saned scanimage -L'

printf 'host setup contract check passed: %s\n' "${contract}"
