#!/usr/bin/env bash
set -Eeuo pipefail

cat >&2 <<'EOF'
ARCHIVE ONLY: scanservjs is deferred for v1.

The active v1 path is host-native CUPS/Avahi printing, SANE/Samsung local
scanner support, AirSane eSCL/AirScan exposure, and user-supplied macOS/Windows
manual evidence. This archived helper intentionally performs no installation.
EOF
exit 64
