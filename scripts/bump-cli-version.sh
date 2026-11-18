#!/usr/bin/env bash
# Pins the default cli-version of the cloudsmith-cli kit to a new Cloudsmith
# CLI release, bumps the patch version of the kit and records the release in
# cloudsmith-cli/CHANGELOG.md. Prints the new kit version.
# Prints nothing and changes nothing when the release is not newer than the
# pinned release or has a different major version.
# Usage: scripts/bump-cli-version.sh CLI_VERSION
set -euo pipefail

cli_version="${1:-}"
[[ "$cli_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] \
  || { echo "usage: $0 MAJOR.MINOR.PATCH" >&2; exit 1; }

spec=cloudsmith-cli/spec.yaml
changelog=cloudsmith-cli/CHANGELOG.md
pinned_version="$(awk '
  /^  cli-version:/ { in_cli_version = 1 }
  in_cli_version && /^    default:/ { gsub(/"/, "", $2); print $2; exit }
' "$spec")"
[[ -n "$pinned_version" ]] || { echo "no cli-version default in $spec" >&2; exit 1; }
if [[ "${cli_version%%.*}" != "${pinned_version%%.*}" ]]; then
  echo "$spec pins Cloudsmith CLI $pinned_version; change the major version by hand" >&2
  exit 0
fi
newest_version="$(printf '%s\n' "$pinned_version" "$cli_version" | sort -V | tail -n 1)"
if [[ "$pinned_version" == "$newest_version" ]]; then
  echo "$spec already pins Cloudsmith CLI $pinned_version" >&2
  exit 0
fi

kit_version="$(sed -n 's/^version: "\([0-9]*\.[0-9]*\.[0-9]*\)"$/\1/p' "$spec")"
[[ -n "$kit_version" ]] || { echo "no kit version in $spec" >&2; exit 1; }
IFS=. read -r major minor patch <<< "$kit_version"
kit_version="$major.$minor.$(( patch + 1 ))"

awk -v kit_version="$kit_version" -v cli_version="$cli_version" '
  /^version: / { $0 = "version: \"" kit_version "\"" }
  /^  cli-version:/ { in_cli_version = 1 }
  in_cli_version && /^    default:/ { sub(/".*"/, "\"" cli_version "\""); in_cli_version = 0 }
  { print }
' "$spec" > "$spec.new"

awk -v kit_version="$kit_version" -v cli_version="$cli_version" -v today="$(date -u +%Y-%m-%d)" '
  !inserted && /^## \[/ {
    print "## [" kit_version "] - " today
    print ""
    print "### Changed"
    print ""
    print "- Pin the default `cli-version` to Cloudsmith CLI " cli_version "."
    print ""
    inserted = 1
  }
  { print }
  END { if (!inserted) exit 1 }
' "$changelog" > "$changelog.new" \
  || { rm -f "$spec.new" "$changelog.new"; echo "no release heading in $changelog" >&2; exit 1; }

mv "$spec.new" "$spec"
mv "$changelog.new" "$changelog"
echo "$kit_version"
