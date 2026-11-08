#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

version="${VERSION:-$(cat VERSION)}"
out="${OUT:-dist}"
targets="${TARGETS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64}"

if [ -z "$version" ]; then
  echo "version is empty" >&2
  exit 1
fi
if [ -z "$out" ] || [ "$out" = "/" ]; then
  echo "invalid output directory: $out" >&2
  exit 1
fi

rm -rf "$out"
mkdir -p "$out"

for target in $targets; do
  os="${target%/*}"
  arch="${target#*/}"
  if [ "$os" = "$target" ] || [ -z "$arch" ]; then
    echo "invalid target: $target (want os/arch)" >&2
    exit 1
  fi

  name="hearthlight_${version}_${os}_${arch}"
  stage="$out/$name"
  mkdir -p "$stage"

  ext=""
  if [ "$os" = "windows" ]; then
    ext=".exe"
  fi

  echo "building $name"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build \
    -trimpath \
    -ldflags "-s -w -X main.version=$version" \
    -o "$stage/hearthlight$ext" .

  cp README.md LICENSE "$stage/"

  if [ "$os" = "windows" ]; then
    if command -v zip >/dev/null 2>&1; then
      (cd "$out" && zip -qr "$name.zip" "$name")
    else
      python3 - "$stage" "$out/$name.zip" <<'PY'
import os
import sys
import zipfile

root, dest = sys.argv[1], sys.argv[2]
with zipfile.ZipFile(dest, "w", zipfile.ZIP_DEFLATED) as archive:
    for base, _, files in os.walk(root):
        for name in files:
            path = os.path.join(base, name)
            archive.write(path, os.path.relpath(path, os.path.dirname(root)))
PY
    fi
  else
    tar -C "$out" -czf "$out/$name.tar.gz" "$name"
  fi

  rm -rf "$stage"
done

if [ "${CHECKSUMS:-1}" = "1" ]; then
  (cd "$out" && sha256sum *.tar.gz *.zip > checksums.txt)
fi

echo "artifacts in $out"
ls -lh "$out"
