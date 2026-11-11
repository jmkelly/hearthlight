#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

catalog="web/themes.json"
guide="docs/themes.md"
images="docs/themes"
viewport="1440,900"
docs_only=0
server_pid=""
work="$(mktemp -d)"

usage() {
  cat <<'EOF'
Usage: scripts/theme-gallery.sh [--docs-only]

Regenerate the theme table and gallery in docs/themes.md from web/themes.json.
With a headless Chromium or Chrome installed, also capture a 1440x900 demo
screenshot for every variant under docs/themes/ and optimise the PNGs.

Options:
  --docs-only   Update the guide without capturing screenshots.
  -h, --help    Show this help.

Set BROWSER to a specific Chromium or Chrome executable when autodetection
picks the wrong one.
EOF
}

cleanup() {
  if [ -n "$server_pid" ] && kill -0 "$server_pid" 2>/dev/null; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  rm -rf "$work"
}
trap cleanup EXIT

while [ "$#" -gt 0 ]; do
  case "$1" in
    --docs-only) docs_only=1 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
  shift
done

find_browser() {
  if [ -n "${BROWSER:-}" ]; then
    if command -v "$BROWSER" >/dev/null 2>&1; then
      printf '%s\n' "$BROWSER"
      return 0
    fi
    echo "BROWSER=$BROWSER is not executable" >&2
    return 1
  fi
  local candidate
  for candidate in chromium chromium-browser google-chrome google-chrome-stable; do
    if command -v "$candidate" >/dev/null 2>&1; then
      printf '%s\n' "$candidate"
      return 0
    fi
  done
  return 1
}

replace_block() {
  local file="$1" begin="$2" end="$3" block="$4"
  awk -v begin="$begin" -v end="$end" -v block="$block" '
    $0 == begin {
      print
      while ((getline line < block) > 0) print line
      close(block)
      skipping = 1
      next
    }
    $0 == end { skipping = 0 }
    !skipping { print }
  ' "$file" > "$file.tmp"
  mv "$file.tmp" "$file"
}

update_guide() {
  python3 - "$catalog" "$images" "web/omarchy.json" "$work/table.md" "$work/gallery.md" "$work/mapping.md" <<'PY'
import json
import os
import sys

catalog_path, image_dir, omarchy_path, table_path, gallery_path, mapping_path = sys.argv[1:7]

with open(catalog_path, encoding="utf-8") as handle:
    catalog = json.load(handle)

mapping = {}
if os.path.exists(omarchy_path):
    with open(omarchy_path, encoding="utf-8") as handle:
        mapping = json.load(handle)

stock = {}
for name, variant in mapping.get("themes", {}).items():
    stock.setdefault(variant, []).append(name)

# Rosé Pine moved its site; keep the catalog credit pointing at the live
# project page without editing the generated catalog from this script.
source_url_aliases = {
    "https://github.com/rose-pine/rose-pine": "https://rosepinetheme.com/",
}


def omarchy_status(family_id, variant_id):
    names = sorted(stock.get(variant_id, []))
    if names:
        return "Yes. Omarchy " + ", ".join("`%s`" % name for name in names) + "."
    if family_id == "hearthlight":
        return "No, Hearthlight original."
    return "No."


def source_cell(family):
    source = family.get("source") or {}
    name = source.get("name", "")
    url = source.get("url", "")
    url = source_url_aliases.get(url, url)
    if name and url:
        return "[%s](%s)" % (name, url)
    if family.get("id") == "hearthlight":
        return "Hearthlight original"
    return "Source not recorded"


def image_path(variant_id):
    return os.path.join(image_dir, variant_id + ".png")


def cell(value):
    return str(value).replace("|", "\\|")


table = [
    "| Variant | ID | Family | Mode | Source | Omarchy stock | Preview |",
    "| --- | --- | --- | --- | --- | --- | --- |",
]
for family in catalog.get("families", []):
    for variant in family.get("variants", []):
        variant_id = variant.get("id", "")
        preview = (
            "[Preview](themes/%s.png)" % variant_id
            if os.path.isfile(image_path(variant_id))
            else "Pending"
        )
        table.append(
            "| %s | `%s` | %s | %s | %s | %s | %s |"
            % (
                cell(variant.get("name", variant_id)),
                cell(variant_id),
                cell(family.get("name", family.get("id", ""))),
                cell(variant.get("mode", "")),
                cell(source_cell(family)),
                cell(omarchy_status(family.get("id", ""), variant_id)),
                preview,
            )
        )

with open(table_path, "w", encoding="utf-8") as handle:
    handle.write("\n".join(table) + "\n")

mapping_table = [
    "| Omarchy theme | Mode | Hearthlight variant |",
    "| --- | --- | --- |",
]
for name in sorted(mapping.get("themes", {})):
    variant_id = mapping["themes"][name]
    mode = mapping.get("modes", {}).get(name, "")
    mapping_table.append("| `%s` | %s | `%s` |" % (cell(name), cell(mode), cell(variant_id)))

with open(mapping_path, "w", encoding="utf-8") as handle:
    handle.write("\n".join(mapping_table) + "\n")

gallery = []
any_image = any(
    os.path.isfile(image_path(variant.get("id", "")))
    for family in catalog.get("families", [])
    for variant in family.get("variants", [])
)
if not any_image:
    gallery.extend(
        [
            "Screenshots are pending. Install Chromium or Chrome and run",
            "`./scripts/theme-gallery.sh` to generate them.",
            "",
        ]
    )

for family in catalog.get("families", []):
    gallery.append("### %s" % family.get("name", family.get("id", "")))
    gallery.append("")
    for variant in family.get("variants", []):
        variant_id = variant.get("id", "")
        variant_name = variant.get("name", variant_id)
        gallery.append("#### %s" % variant_name)
        gallery.append("")
        if os.path.isfile(image_path(variant_id)):
            gallery.append("![%s map](themes/%s.png)" % (variant_name, variant_id))
        else:
            gallery.append(
                "Screenshot pending. Run `./scripts/theme-gallery.sh` with a headless "
                "Chromium or Chrome installed."
            )
        gallery.append("")

with open(gallery_path, "w", encoding="utf-8") as handle:
    handle.write("\n".join(gallery).rstrip() + "\n")
PY

  replace_block "$guide" '<!-- BEGIN THEME TABLE -->' '<!-- END THEME TABLE -->' "$work/table.md"
  replace_block "$guide" '<!-- BEGIN THEME GALLERY -->' '<!-- END THEME GALLERY -->' "$work/gallery.md"
  replace_block "$guide" '<!-- BEGIN OMARCHY MAPPING -->' '<!-- END OMARCHY MAPPING -->' "$work/mapping.md"
}

optimise_image() {
  local file="$1"
  if [ ! -s "$file" ]; then
    echo "missing screenshot: $file" >&2
    return 1
  fi
  if command -v magick >/dev/null 2>&1; then
    magick "$file" -strip -colors 256 -depth 8 -define png:compression-level=9 "$file.opt"
    mv "$file.opt" "$file"
  elif command -v convert >/dev/null 2>&1; then
    convert "$file" -strip -colors 256 -depth 8 -define png:compression-level=9 "$file.opt"
    mv "$file.opt" "$file"
  fi
  if command -v identify >/dev/null 2>&1; then
    local size
    size="$(identify -format '%wx%h' "$file")"
    if [ "$size" != "1440x900" ]; then
      echo "warning: $file is $size, expected 1440x900" >&2
    fi
  fi
}

capture_screenshots() {
  local browser
  if ! browser="$(find_browser)"; then
    echo "No headless Chromium or Chrome found. Screenshots are pending." >&2
    return 0
  fi
  if ! command -v curl >/dev/null 2>&1; then
    echo "curl is required to wait for the demo server" >&2
    return 1
  fi

  echo "building demo server"
  go build -o "$work/hearthlight" .

  local port
  port="$(python3 - <<'PY'
import socket
sock = socket.socket()
sock.bind(("127.0.0.1", 0))
print(sock.getsockname()[1])
sock.close()
PY
)"
  local url="http://127.0.0.1:$port"
  "$work/hearthlight" -demo -listen "127.0.0.1:$port" -traffic=false >"$work/server.log" 2>&1 &
  server_pid=$!

  local attempt ready=0
  for attempt in $(seq 1 100); do
    if curl -fsS "$url/healthz" >/dev/null 2>&1; then
      ready=1
      break
    fi
    if ! kill -0 "$server_pid" 2>/dev/null; then
      cat "$work/server.log" >&2
      return 1
    fi
    sleep 0.1
  done
  if [ "$ready" -ne 1 ]; then
    cat "$work/server.log" >&2
    echo "demo server did not become ready" >&2
    return 1
  fi

  mkdir -p "$images"
  local id out
  while IFS= read -r id; do
    out="$images/$id.png"
    echo "capturing $id"
    "$browser" \
      --headless=new \
      --disable-gpu \
      --disable-background-networking \
      --disable-sync \
      --disable-default-apps \
      --disable-extensions \
      --disable-component-update \
      --no-first-run \
      --hide-scrollbars \
      --force-device-scale-factor=1 \
      --force-prefers-reduced-motion \
      --window-size="$viewport" \
      --user-data-dir="$work/chrome-$id" \
      --screenshot="$out" \
      "$url/?theme=$id" >"$work/chrome-$id.log" 2>&1
    optimise_image "$out"
  done < <(python3 - "$catalog" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    catalog = json.load(handle)
for family in catalog.get("families", []):
    for variant in family.get("variants", []):
        print(variant.get("id", ""))
PY
)

  kill "$server_pid" 2>/dev/null || true
  wait "$server_pid" 2>/dev/null || true
  server_pid=""
}

make themes
update_guide

if [ "$docs_only" -eq 0 ]; then
  capture_screenshots
  update_guide
fi

echo "theme gallery updated in $guide"
