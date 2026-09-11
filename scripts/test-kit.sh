#!/usr/bin/env bash
# Run Docker's kit TCK against one kit.
#
# Usage:
#   scripts/test-kit.sh <kit-dir>         # from the repository root
#   ../scripts/test-kit.sh                # from inside the kit's directory
#
# The script resolves <kit-dir> to an absolute path, exports it as KIT, and
# runs `go test ./tck/...`. Extra arguments go to `go test`.
#
# The TCK starts a container through testcontainers. Docker must run.

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)

if [ $# -gt 0 ] && [[ "$1" != -* ]]; then
  kit_arg=$1
  shift
else
  kit_arg=$PWD
fi

if [ -d "$kit_arg" ]; then
  kit_abs=$(cd "$kit_arg" && pwd)
elif [ -d "$REPO_ROOT/$kit_arg" ]; then
  kit_abs=$(cd "$REPO_ROOT/$kit_arg" && pwd)
else
  echo "kit directory not found: $kit_arg" >&2
  exit 1
fi

if [ ! -f "$kit_abs/spec.yaml" ]; then
  echo "no spec.yaml in $kit_abs — is this a kit directory?" >&2
  exit 1
fi

cd "$REPO_ROOT"
KIT="$kit_abs" exec go test -v -count=1 -timeout 10m -run TestKitTCK "$@" ./tck/...
