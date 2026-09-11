#!/usr/bin/env bash
# Run the end-to-end TCK against one kit with a real, installed sbx CLI.
#
# Usage:
#   scripts/test-kit-e2e.sh <kit-dir>     # from the repository root
#   ../scripts/test-kit-e2e.sh            # from inside the kit's directory
#
# Every sbx call is scoped to APP_NAME, which gives the run its own daemon,
# sandboxes, policy and credential store. The day-to-day sbx state is not
# touched. The scoped daemon starts on the `deny-all` network policy, so the
# run is a real test of each kit's permissions.network.allow list.
#
# Prerequisites:
#   - sbx on PATH. Install it from https://github.com/docker/sbx-releases.
#   - The scoped daemon must be signed in to Docker Hub:
#         sbx --app-name cs-kits login
#
# Environment overrides:
#   APP_NAME  scoped app name. Keep
#             it very short: the name goes into the daemon's containerd socket
#             path, which cannot exceed 104 bytes. On a GitHub runner the rest
#             of that path costs 88 bytes, leaving 16 for the name.
#   POLICY    network policy for the scoped daemon. Default `deny-all`.
#             Set POLICY= to leave the current policy alone.

set -euo pipefail

APP_NAME=${APP_NAME:-cs-kits}
POLICY=${POLICY-deny-all}
export APP_NAME

if [[ ! "$APP_NAME" =~ ^cs-kits(-[a-z0-9]{1,8})?$ ]]; then
  echo "APP_NAME must be cs-kits or cs-kits-<1-8 lowercase letters/digits>; use a dedicated test daemon." >&2
  exit 1
fi

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

if ! command -v sbx >/dev/null 2>&1; then
  echo "sbx not on PATH — install it from https://github.com/docker/sbx-releases" >&2
  exit 1
fi

# Fail fast when the scoped daemon cannot reach the runtime. The usual cause
# is a missing `sbx login`, which otherwise surfaces minutes later as a failed
# `sbx create`. stdin comes from /dev/null so an interactive prompt reports its
# message instead of looking like a hang.
if ! probe_err=$(sbx --app-name "$APP_NAME" ls 2>&1 >/dev/null </dev/null); then
  cat >&2 <<EOF
ERROR: sbx --app-name $APP_NAME is not usable.

$probe_err

The scoped daemon has its own credential store. Sign it in once:

  sbx --app-name $APP_NAME login

EOF
  exit 1
fi

# `policy init` works once per daemon. Fall back to a reset so a reused local
# daemon still lands on the requested baseline.
if [ -n "$POLICY" ]; then
  echo "Setting the --app-name=$APP_NAME network policy to $POLICY"
  if ! sbx --app-name "$APP_NAME" policy init "$POLICY" >/dev/null 2>&1; then
    sbx --app-name "$APP_NAME" policy reset --force </dev/null >/dev/null 2>&1 || true
    if ! init_out=$(sbx --app-name "$APP_NAME" policy init "$POLICY" </dev/null 2>&1); then
      printf '%s\n' "$init_out" >&2
      echo "ERROR: could not set the --app-name=$APP_NAME network policy to $POLICY" >&2
      exit 1
    fi
  fi
fi

# The most common e2e failure is a host missing from permissions.network.allow.
# Print the policy log on failure: in CI the runner and its daemon disappear
# when the job ends, so instructions to run it later are of no use.
sbx_name_log="$REPO_ROOT/.sbx-e2e-names-$$"
(set -o noclobber; : > "$sbx_name_log")
export SBX_E2E_NAME_LOG="$sbx_name_log"

on_exit() {
  rc=$?
  if [ "$rc" -ne 0 ] && [ -s "$sbx_name_log" ]; then
    while IFS= read -r sandbox_name; do
      [ -n "$sandbox_name" ] || continue
      echo "" >&2
      echo "Policy log for $sandbox_name (policy: ${POLICY:-current default}):" >&2
      sbx --app-name "$APP_NAME" policy log "$sandbox_name" >&2 2>&1 || true
    done < "$sbx_name_log"
    cat >&2 <<EOF

Review blocked requests before changing permissions.network.allow.
Expected firewall denials are not missing allowlist entries.
EOF
  fi
  rm -f "$sbx_name_log"
  exit "$rc"
}
trap on_exit EXIT

cd "$REPO_ROOT"
KIT_UNDER_TEST="$kit_abs" go test -tags=e2e -v -count=1 -timeout 25m -run TestE2EKit "$@" ./tck/...
