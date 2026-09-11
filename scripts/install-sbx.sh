#!/usr/bin/env bash
# Install a checksum-verified release. CLI_ONLY=1 installs only the daemon-free
# CLI without sudo (including on macOS). Stdout contains only the PATH entry.
set -euo pipefail

version=${1:-v0.42.1}
PREFIX=${PREFIX:-$HOME/.docker/sbx}
CLI_ONLY=${CLI_ONLY:-0}
exec 3>&1 1>&2

case "$(uname -s):$(uname -m)" in
  Linux:x86_64) asset=DockerSandboxes-linux-amd64.tar.gz ;;
  Linux:aarch64|Linux:arm64) asset=DockerSandboxes-linux-arm64.tar.gz ;;
  Darwin:arm64)
    [ "$CLI_ONLY" = 1 ] || { echo "Use CLI_ONLY=1 on macOS, or install with Homebrew."; exit 1; }
    asset=DockerSandboxes-darwin.tar.gz ;;
  *) echo "Unsupported platform: $(uname -s) $(uname -m)"; exit 1 ;;
esac
[[ "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ || "$version" = latest ]] ||
  { echo "Expected a release version (vX.Y.Z) or latest"; exit 1; }

auth=()
if [ -n "${GITHUB_TOKEN:-}" ]; then
  auth=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
fi
release_url="https://api.github.com/repos/docker/sbx-releases/releases/tags/$version"
[ "$version" != latest ] || release_url=https://api.github.com/repos/docker/sbx-releases/releases/latest
release=$(curl --proto '=https' --proto-redir '=https' --tlsv1.2 -fsSL \
  --connect-timeout 15 --max-time 60 --retry 2 "${auth[@]}" "$release_url")
url=$(jq -er --arg asset "$asset" '.assets[] | select(.name == $asset) | .browser_download_url' <<< "$release")
digest=$(jq -er --arg asset "$asset" '.assets[] | select(.name == $asset) | .digest' <<< "$release")
[[ "$digest" =~ ^sha256:[a-f0-9]{64}$ ]] || { echo "Release asset has no SHA256 digest"; exit 1; }
[[ "$url" == https://github.com/docker/sbx-releases/releases/download/* ]] ||
  { echo "Unexpected release asset URL"; exit 1; }

work="$PWD/.sbx-install-$$"
mkdir -m 700 "$work"
trap 'rm -rf "$work"' EXIT
curl --proto '=https' --proto-redir '=https' --tlsv1.2 -fsSL \
  --connect-timeout 15 --max-time 300 --retry 2 "$url" -o "$work/release.tar.gz"
if command -v sha256sum >/dev/null 2>&1; then
  printf '%s  %s\n' "${digest#sha256:}" "$work/release.tar.gz" | sha256sum -c -
else
  printf '%s  %s\n' "${digest#sha256:}" "$work/release.tar.gz" | shasum -a 256 -c -
fi

if [ "$CLI_ONLY" = 1 ]; then
  case "$asset" in
    *darwin*) binary=bin/sbx ;;
    *) binary=docker-sbx/sbx ;;
  esac
  tar xzf "$work/release.tar.gz" -C "$work" "$binary"
  mkdir -p "$PREFIX/bin"
  install -m 755 "$work/$binary" "$PREFIX/bin/sbx"
else
  tar xzf "$work/release.tar.gz" -C "$work"
  mkdir -p "$(dirname "$PREFIX")"
  if [ "$(id -u)" -eq 0 ]; then
    PREFIX="$PREFIX" "$work/docker-sbx/install.sh"
  else
    sudo PREFIX="$PREFIX" "$work/docker-sbx/install.sh"
  fi
fi
"$PREFIX/bin/sbx" version
echo "$PREFIX/bin" >&3
