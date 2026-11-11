#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

plugin_dir="integrations/omarchy-hearthlight-theme"
version="${1:-$(cat VERSION)}"
out_dir="dist"
archive_name="omarchy-hearthlight-theme-$version.tar.gz"

fail() {
  echo "omarchy-plugin-archive: $*" >&2
  exit 1
}

if [[ -z $version || ! $version =~ ^[0-9A-Za-z._+-]+$ ]]; then
  fail "invalid version: $version"
fi

files=(LICENSE README.md Service.qml manifest.json)
if [[ -f "$plugin_dir/preview.png" ]]; then
  files+=(preview.png)
fi

for file in "${files[@]}"; do
  [[ -f "$plugin_dir/$file" ]] || fail "missing $plugin_dir/$file"
done

mkdir -p "$out_dir"
tmp=$(mktemp "$out_dir/.omarchy-plugin-archive.XXXXXX")
trap 'rm -f "$tmp"' EXIT

LC_ALL=C tar \
  --sort=name \
  --format=ustar \
  --owner=0 \
  --group=0 \
  --numeric-owner \
  --mtime='@0' \
  -C "$plugin_dir" \
  -cf - \
  "${files[@]}" | gzip -9 -n >"$tmp"

mv "$tmp" "$out_dir/$archive_name"
trap - EXIT

(cd "$out_dir" && sha256sum "$archive_name" >"$archive_name.sha256")

printf '%s\n' "$out_dir/$archive_name" "$out_dir/$archive_name.sha256"
