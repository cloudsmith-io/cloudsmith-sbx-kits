#!/usr/bin/env bash
# Install the sbx CLI from docker/sbx-releases on Linux.
#
# Usage:
#   scripts/install-sbx.sh              # the latest release
#   scripts/install-sbx.sh v0.42.1      # a specific tag
#
# Environment:
#   PREFIX        install root. Default $HOME/.docker/sbx
#   GITHUB_TOKEN  optional. Authenticates the release lookup, which avoids the
#                 GitHub API rate limit on shared CI runners.
#
# The script prints the directory to add to PATH on stdout. Progress goes to
# stderr, so CI can redirect the one without the other:
#
#   ./scripts/install-sbx.sh >> "$GITHUB_PATH"

set -euo pipefail

version=${1:-}
PREFIX=${PREFIX:-$HOME/.docker/sbx}
ASSET=DockerSandboxes-linux.tar.gz
RELEASES=https://github.com/docker/sbx-releases

# Move the PATH line to file descriptor 3 and send everything else to stderr.
# The vendor installer prints its own progress on stdout, which would otherwise
# land in $GITHUB_PATH as bogus entries. That file takes one path per line and
# validates nothing, so the mistake stays silent.
exec 3>&1 1>&2

if [ "$(uname -s)" != Linux ]; then
  echo "error: this script supports Linux only. On macOS run: brew install docker/tap/sbx" >&2
  exit 1
fi

auth=()
if [ -n "${GITHUB_TOKEN:-}" ]; then
  auth=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
fi

if [ -z "$version" ]; then
  echo "==> resolving the latest release"
  version=$(curl -fsSL "${auth[@]}" \
    https://api.github.com/repos/docker/sbx-releases/releases/latest | jq -r .tag_name)
  if [ -z "$version" ] || [ "$version" = null ]; then
    echo "error: could not resolve the latest sbx release" >&2
    exit 1
  fi
fi
echo "    version ${version}"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "==> downloading ${ASSET}"
curl -fsSL "${auth[@]}" "${RELEASES}/releases/download/${version}/${ASSET}" -o "${tmp}/${ASSET}"
tar xzf "${tmp}/${ASSET}" -C "$tmp"

# The vendor installer runs under sudo. Create the parent directory first as the
# normal user: $PREFIX's parent is usually ~/.docker, and a root-owned ~/.docker
# stops a later `docker login` from writing config.json.
mkdir -p "$(dirname "$PREFIX")"

echo "==> installing into ${PREFIX}"
if [ "$(id -u)" -eq 0 ]; then
  PREFIX="$PREFIX" "${tmp}/docker-sbx/install.sh"
else
  sudo PREFIX="$PREFIX" "${tmp}/docker-sbx/install.sh"
fi

"${PREFIX}/bin/sbx" version

echo "${PREFIX}/bin" >&3
