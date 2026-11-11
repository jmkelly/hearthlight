# Omarchy Hearthlight Theme

An Omarchy Quattro service plugin that follows the active Omarchy theme and
writes a normalized palette for Hearthlight. Switch from Tokyo Night to
Gruvbox and the Hearthlight map follows without a reload.

The plugin writes a file. Hearthlight reads it. There is no network write path
in either direction, so the unauthenticated Hearthlight HTTP API does not need
a new mutation endpoint.

## Requirements

- Omarchy Quattro with shell plugin support (`omarchy plugin validate` works).
- Hearthlight built with Omarchy theme support, or another reader for the JSON
  file described below.
- The plugin and Hearthlight run as the same user, or Hearthlight can read the
  output file. The output is mode `0600`.

## Install

`omarchy plugin add` clones a git repository and reads `manifest.json` from its
root. It does not read this subdirectory or a tarball directly. Use one of the
paths below.

### From the published repository

Once this directory is published as `jmkelly/omarchy-hearthlight-theme`, install
it with:

```bash
omarchy plugin add https://github.com/jmkelly/omarchy-hearthlight-theme --enable
```

### From a local checkout

Run these from the Hearthlight repository root:

```bash
install_dir="$HOME/.config/omarchy/plugins/jmkelly.hearthlight-theme"
mkdir -p "$HOME/.config/omarchy/plugins"
rm -rf "$install_dir"
cp -r integrations/omarchy-hearthlight-theme "$install_dir"
omarchy-shell shell rescanPlugins
omarchy plugin enable jmkelly.hearthlight-theme
```

The directory name must match the manifest id.

### From a release archive

Build the archive from the Hearthlight repository root:

```bash
make omarchy-plugin
```

That prints the paths to `dist/omarchy-hearthlight-theme-<version>.tar.gz` and
its `.sha256` checksum. `<version>` is the contents of `VERSION` unless you pass
an argument to `scripts/omarchy-plugin-archive.sh`. The archive root contains
`manifest.json`, `Service.qml`, `README.md`, and `LICENSE`; the files are not
wrapped in a directory.

Verify the checksum, then unpack the archive into the plugin directory:

```bash
version=$(cat VERSION)
(cd dist && sha256sum -c "omarchy-hearthlight-theme-$version.tar.gz.sha256")

install_dir="$HOME/.config/omarchy/plugins/jmkelly.hearthlight-theme"
rm -rf "$install_dir"
mkdir -p "$install_dir"
tar -xzf "dist/omarchy-hearthlight-theme-$version.tar.gz" -C "$install_dir"
omarchy-shell shell rescanPlugins
omarchy plugin enable jmkelly.hearthlight-theme
```

`sha256sum -c` prints `OK` when the archive matches. Omarchy already provides
coreutils, so this needs no extra package.

To install the same archive through `omarchy plugin add`, unpack it into a git
repository and add that repository URL:

```bash
version=$(cat VERSION)
repo_dir=$(mktemp -d)
tar -xzf "dist/omarchy-hearthlight-theme-$version.tar.gz" -C "$repo_dir"
git -C "$repo_dir" init -b main
git -C "$repo_dir" add .
git -C "$repo_dir" commit -m "Omarchy Hearthlight theme plugin"
git -C "$repo_dir" remote add origin <git-url>
git -C "$repo_dir" push -u origin main
omarchy plugin add <git-url> --enable
```

### Publish a standalone repository

To split this directory into its own repository and push it:

```bash
git subtree split --prefix=integrations/omarchy-hearthlight-theme -b omarchy-hearthlight-theme
git push <remote> omarchy-hearthlight-theme:main
```

Marketplace submission is a separate publishing step. Submit the published
repository at https://plugins.omarchy.org/ after release. Neither
`make omarchy-plugin` nor the archive script submits anything.

## Uninstall

```bash
omarchy plugin disable jmkelly.hearthlight-theme
omarchy plugin remove jmkelly.hearthlight-theme --yes
rm -f "$HOME/.config/hearthlight/omarchy-theme.json"
rm -f "$HOME/.config/omarchy/hearthlight-theme.json"
```

Disabling the plugin stops updates but leaves the last palette on disk, so
Hearthlight keeps showing that theme. Delete the output file to make
Hearthlight report the Omarchy source as unavailable.

## Config

The optional config file is:

```
~/.config/omarchy/hearthlight-theme.json
```

It only overrides the output path:

```json
{
  "outputPath": "~/.config/hearthlight/omarchy-theme.json"
}
```

`outputPath` must be absolute or start with `~/`. A missing, malformed, or
invalid config falls back to the default path. The file is watched, so a change
is applied without restarting the shell.

`hearthlightUrl` is accepted for forward compatibility but ignored with one
warning. This version has no HTTP client and no network write path. Same-host
file sync is the supported design; a token-protected remote sync is a separate
follow-up.

## Files

Read:

- `~/.local/state/omarchy/current/theme.name`
- `~/.local/state/omarchy/current/theme/colors.toml`
- `~/.local/state/omarchy/current/theme.generation` when present
- `~/.config/omarchy/hearthlight-theme.json` when present

Written:

- `~/.config/hearthlight/omarchy-theme.json` by default
- `<outputPath>.tmp.<n>` during a write, then renamed over the target

The plugin watches the theme files and the `current` directory with Quickshell
`FileView { watchChanges: true }`. It does not poll. File events are coalesced
for 250 ms, then the files reload and a 150 ms write timer starts. A theme
switch writes once, about 400 ms after the last file event.

Writes go to a temporary file in the target directory with `umask 077`, then
`chmod 600` runs, then `mv` replaces the target. The target is never truncated
in place. If a write fails, the previous file stays intact and the error is
logged once.

The JSON is passed as an argv value to a fixed shell script. Palette content is
never interpolated into shell source.

## Output schema

```json
{
  "schemaVersion": 1,
  "name": "tokyo-night",
  "displayName": "Tokyo Night",
  "mode": "dark",
  "generation": 1762700000,
  "themeId": "tokyo-night",
  "colors": {
    "background": "#1a1b26",
    "foreground": "#a9b1d6",
    "accent": "#7aa2f7",
    "muted": "#565f89",
    "selection": "#33467c"
  }
}
```

`colors` includes `background`, `foreground`, `accent`, `muted`, `selection`,
the named colours (`red`, `yellow`, `orange`, `green`, `cyan`, `blue`,
`magenta`, `brown`), the background and foreground ramps, and `bright_*`
values when the theme defines them. Values are validated hex colours and are
lowercased. Missing background, foreground, accent, muted, or selection values
are filled from the nearest palette entry.

`themeId` is included for stock Omarchy names with a known Hearthlight variant.
The table is embedded in `Service.qml` and mirrors the Omarchy mapping in the
Hearthlight repository. Unknown or custom theme names omit `themeId`, and
Hearthlight synthesizes a palette from the raw colours. `rose-pine` maps to
`rose-pine-dawn` because Omarchy's Rose Pine is the light Dawn palette.

`generation` is the integer in `theme.generation`, or `0` on Omarchy builds
that do not publish that file. `mode` comes from `colors.toml` when present,
otherwise from background luminance.

## Hearthlight server options

When the server-side Omarchy support lands, the matching NixOS options are:

```nix
services.hearthlight.omarchyTheme = true;
services.hearthlight.omarchyThemeFile = "/home/james/.config/hearthlight/omarchy-theme.json";
```

The file is mode `0600`, so Hearthlight must run as the same user that owns the
Omarchy state, or the file must be made readable to its service user. Running
Hearthlight as a system service with a different home directory will not see
the default path.

## Troubleshooting

Check that the shell sees the plugin:

```bash
omarchy plugin list --json | jq '.[] | select(.id == "jmkelly.hearthlight-theme")'
```

Check the output after a theme switch:

```bash
stat -c '%a %y %n' "$HOME/.config/hearthlight/omarchy-theme.json"
jq . "$HOME/.config/hearthlight/omarchy-theme.json"
```

Common causes:

- The output file is missing. Confirm the theme files exist, the plugin is
  enabled, and `outputPath` is valid.
- The output does not change. Confirm the theme switch changed
  `theme.name` or `theme/colors.toml`. A malformed palette is ignored, and the
  last good file is kept.
- Hearthlight cannot read the file. Check ownership and mode. Do not weaken
  the mode to `0644` on a shared machine.
- `themeId` is absent. The theme name is not in the stock mapping. This is
  expected for third-party and custom themes.

The plugin logs once per failure episode. A repaired tree or a later theme
switch retries the write.

## Security

Omarchy plugins run unsandboxed inside the long-lived `omarchy-shell` process.
Review any plugin before enabling it.

This plugin:

- reads only the Omarchy theme files and its own config file;
- writes only the configured output file and its temporary sibling;
- parses TOML values as strings and never executes or evaluates theme content;
- accepts only hex colours and ignores other theme keys;
- has no HTTP client, no shell interpolation of palette content, and no
  network write path;
- writes the palette atomically with mode `0600`.

The theme name and palette are untrusted input. They are parsed, validated,
and copied into JSON. They are never passed to a shell, `eval`, or a network
request.

## Non-plugin hook alternative

If you do not want an Omarchy shell plugin, use a `theme-set` hook. It must
write the same schema so Hearthlight sees the same contract:

```
~/.config/omarchy/hooks/theme-set.d/hearthlight
```

Make it executable with `chmod 700`. Omarchy runs it after each theme set.
This version uses `jq`, which Omarchy already requires for plugin management.

```bash
#!/bin/bash
set -euo pipefail

state="$HOME/.local/state/omarchy/current"
out="$HOME/.config/hearthlight/omarchy-theme.json"

name=$(head -n1 "$state/theme.name" | tr -d '[:space:]')
[[ -n $name ]] || exit 0

mode=$(awk -F'"' '/^[[:space:]]*mode[[:space:]]*=/{print $2; exit}' "$state/theme/colors.toml")
[[ $mode == light ]] || mode=dark

generation=$(cat "$state/theme.generation" 2>/dev/null || printf 0)
[[ $generation =~ ^[0-9]+$ ]] || generation=0

display=$(printf '%s\n' "$name" | sed 's/[_-]/ /g' |
  awk '{for (i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) substr($i,2); print}')

mkdir -p "$(dirname "$out")"
tmp=$(mktemp "$out.XXXXXX")
trap 'rm -f "$tmp"' EXIT

omarchy-theme-color --raw |
  jq -Rn \
    --arg name "$name" \
    --arg display "$display" \
    --arg mode "$mode" \
    --argjson generation "$generation" '
      ([inputs
        | split("\t")
        | select(length == 2)
        | select(.[0] | test("^(background|dark_background|darker_background|lighter_background|foreground|dark_foreground|light_foreground|bright_foreground|accent|selection|muted|red|yellow|orange|green|cyan|blue|magenta|brown|bright_(red|yellow|orange|green|cyan|blue|magenta))$"))
        | {(.[0]): (.[1] | ascii_downcase)}]
       | add // {}) as $colors
      | {
          schemaVersion: 1,
          name: $name,
          displayName: $display,
          mode: $mode,
          generation: $generation,
          colors: $colors
        }' >"$tmp"

chmod 600 "$tmp"
mv -f "$tmp" "$out"
trap - EXIT
```

The hook does not carry the stock `themeId` mapping. Hearthlight can map the
name itself. The plugin remains preferred because it reacts to any theme file
change, writes the mapping, and needs no hook.

## License

MIT. See `LICENSE`.
