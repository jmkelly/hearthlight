# Omarchy theme source

Hearthlight can follow the active Omarchy desktop theme. The Omarchy plugin
writes a normalized JSON palette, and Hearthlight watches that file and exposes
it read-only to the browser. Hearthlight never accepts theme writes over HTTP.

The source, the stock name mapping and the plugin are shipped. The plugin lives
in [`integrations/omarchy-hearthlight-theme`](../integrations/omarchy-hearthlight-theme/README.md).

## What ships

- `web/omarchy.json` maps all 23 stock Omarchy Quattro names to Hearthlight
  variants and records each stock mode. It is pinned to commit
  `748fa3c0d0418d7a199818d94c1f87d495d7e804`.
- The server merges that mapping into `GET /api/themes` under `omarchy` and
  serves the file as `/omarchy.json`.
- The server watcher rereads the plugin's normalized palette and publishes it at
  `GET /api/theme/omarchy` and as an `omarchy` event on `/api/events`.
- Settings → Appearance shows **Follow Omarchy theme** when the source is
  available.

## Requirements

- The Omarchy plugin or a theme hook writes `omarchy-theme.json`.
- Hearthlight runs with read access to that file.
- Hearthlight is started with `-omarchy-theme` or
  `HEARTHLIGHT_OMARCHY_THEME=1`.
- No extra port or token is needed. The endpoint uses the same optional token as
  the rest of the API when one is configured.

## Mapping file

`web/omarchy.json` has this shape:

```json
{
  "schemaVersion": 1,
  "source": "https://github.com/omacom/omarchy/tree/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes",
  "commit": "748fa3c0d0418d7a199818d94c1f87d495d7e804",
  "themes": {
    "catppuccin": "catppuccin-mocha",
    "catppuccin-latte": "catppuccin-latte",
    "ethereal": "ethereal",
    "everforest": "everforest-dark",
    "flexoki-light": "flexoki-light",
    "gruvbox": "gruvbox-dark",
    "hackerman": "hackerman",
    "jade-horse": "jade-horse",
    "kanagawa": "kanagawa-wave",
    "last-horizon": "last-horizon",
    "lumon": "lumon",
    "lupine": "lupine",
    "matte-black": "matte-black",
    "miasma": "miasma",
    "nord": "nord-dark",
    "retro-82": "retro-82",
    "ristretto": "ristretto",
    "rose-pine": "rose-pine-dawn",
    "solitude": "solitude",
    "starship": "starship",
    "tokyo-night": "tokyo-night",
    "vantablack": "vantablack",
    "white": "white"
  },
  "modes": {
    "catppuccin": "dark",
    "catppuccin-latte": "light",
    "ethereal": "dark",
    "everforest": "dark",
    "flexoki-light": "light",
    "gruvbox": "dark",
    "hackerman": "dark",
    "jade-horse": "dark",
    "kanagawa": "dark",
    "last-horizon": "dark",
    "lumon": "dark",
    "lupine": "light",
    "matte-black": "dark",
    "miasma": "dark",
    "nord": "dark",
    "retro-82": "dark",
    "ristretto": "dark",
    "rose-pine": "light",
    "solitude": "dark",
    "starship": "dark",
    "tokyo-night": "dark",
    "vantablack": "dark",
    "white": "light"
  }
}
```

Do not infer a variant from a family name. Omarchy's `rose-pine` is the light
Dawn palette, so it maps to `rose-pine-dawn`, not the dark Moon variant. Omarchy's
`kanagawa` is the dark Wave palette, so it maps to `kanagawa-wave`, not Lotus.

`web/omarchy_test.go` pins the source commit and the expected target and mode for
every stock name. The test fails when a stock name is missing, a target does not
exist, a mode disagrees, an unknown name appears, or `rose-pine` regresses to
`rose-pine-moon`.

### Refreshing the mapping

When Omarchy adds, removes or renames a stock theme:

1. Read the new `themes/*/colors.toml` tree at the new commit.
2. Add or update the name in `themes` and `modes` in `web/omarchy.json`.
3. Update the pinned commit, source URL and stock table in `web/omarchy_test.go`.
4. Run `make check`.
5. Update the embedded stock table in
   `integrations/omarchy-hearthlight-theme/Service.qml` so the plugin writes the
   same `themeId` values.
6. Rerun `./scripts/theme-gallery.sh` to refresh the generated mapping table in
   [the theme gallery](themes.md#omarchy).

## Palette file

The plugin writes this schema to `~/.config/hearthlight/omarchy-theme.json`:

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
    "selection": "#283457",
    "red": "#f7768e",
    "yellow": "#e0af68",
    "orange": "#ff9e64",
    "green": "#9ece6a",
    "cyan": "#7dcfff",
    "blue": "#7aa2f7",
    "magenta": "#bb9af7",
    "bright_foreground": "#c0caf5",
    "bright_red": "#ff7a93"
  }
}
```

`name` and the `colors` object are required. `colors.background` and
`colors.foreground` must be present. Colour values must be hex (`#rgb`, `#rgba`,
`#rrggbb` or `#rrggbbaa`). Unknown top-level fields are ignored, and colour
entries with an invalid name or value are dropped. The server parses values only
and never evaluates theme content.

`themeId` is optional. When it names a variant in the Hearthlight catalog,
Settings applies that variant. Otherwise the browser looks up the theme name in
`web/omarchy.json`. If neither matches, Hearthlight synthesizes a theme from the
raw colours, using the background ramp, foreground, accent and named colours.
This is the path for third-party and custom Omarchy themes.

The plugin writes atomically with mode `0600`, using a temporary file and a
rename. It keeps the last good file when a theme write fails.

## Server flags

| Flag | Environment variable | Default | Meaning |
| --- | --- | --- | --- |
| `-omarchy-theme` | `HEARTHLIGHT_OMARCHY_THEME` | `false` | Enable the read-only Omarchy theme source. |
| `-omarchy-theme-file` | `HEARTHLIGHT_OMARCHY_THEME_FILE` | `~/.config/hearthlight/omarchy-theme.json` | Normalized palette file to watch. |

Boolean environment values accept `1`, `true`, `yes` and `on` for true, and
`0`, `false`, `no` and `off` for false.

The default path expands the home directory of the user that starts Hearthlight.
A system service usually runs under a different user, so set
`-omarchy-theme-file` explicitly there.

Example:

```sh
hearthlight \
  -omarchy-theme \
  -omarchy-theme-file /home/alice/.config/hearthlight/omarchy-theme.json
```

The watcher rereads the file about once per second and pushes changes to open
browsers over the existing event stream. No reload is needed. The file is
limited to 1 MiB; a larger file is treated as unavailable.

## API

`GET /api/theme/omarchy` returns the current palette:

```json
{
  "available": true,
  "schemaVersion": 1,
  "name": "tokyo-night",
  "displayName": "Tokyo Night",
  "mode": "dark",
  "generation": 1762700000,
  "themeId": "tokyo-night",
  "colors": {
    "background": "#1a1b26",
    "foreground": "#a9b1d6",
    "accent": "#7aa2f7"
  }
}
```

`available`, `name` and the `colors` object are always present when a theme is
readable. `displayName`, `mode`, `generation`, `themeId` and individual colour
entries are omitted when the file does not provide them. `mode` must be `dark`
or `light`; any other value is returned empty and the browser derives the mode
from the background.

A missing, unreadable, oversized or malformed file returns:

```json
{ "available": false }
```

The endpoint is read-only. There is no matching write endpoint.

`GET /api/themes` returns the theme catalog plus the mapping under `omarchy`. The
same mapping is also available as `/omarchy.json`.

## Settings

Settings → Appearance shows **Follow Omarchy theme** when the endpoint reports
the theme source as available. It stays visible while enabled so the user can
turn it off if the file later becomes unavailable. While it is on:

- The family and mode pickers are locked.
- The current Omarchy theme name is shown.
- Theme changes apply live from the event stream.
- The follow preference is stored under `hearthlight.theme.omarchy.v1`.

When a stock theme is active, Settings applies the mapped Hearthlight variant,
including its mode. Third-party and custom themes are synthesized from their raw
colours. If the file disappears or becomes malformed, the browser keeps the last
applied theme and shows a small unavailable note. It does not reset the user's
stored manual theme. Turning the toggle off restores the normal theme selection.

## Plugin install

The plugin is an Omarchy Quattro service plugin. It writes a file; Hearthlight
reads it. There is no network write path in either direction, so the
unauthenticated Hearthlight HTTP API does not need a mutation endpoint.

### Release archive

Build the archive from the Hearthlight repository root:

```bash
make omarchy-plugin
```

That runs `scripts/omarchy-plugin-archive.sh` and prints the paths to
`dist/omarchy-hearthlight-theme-<version>.tar.gz` and its `.sha256` checksum.
`<version>` is the contents of `VERSION`. The archive root contains
`manifest.json`, `Service.qml`, `README.md` and `LICENSE`; the files are not
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

The directory name must match the manifest id. `omarchy plugin add` clones a git
repository and reads `manifest.json` from its root, so it does not read this
subdirectory or a tarball directly.

### Local checkout

From the Hearthlight repository root:

```bash
install_dir="$HOME/.config/omarchy/plugins/jmkelly.hearthlight-theme"
mkdir -p "$HOME/.config/omarchy/plugins"
rm -rf "$install_dir"
cp -r integrations/omarchy-hearthlight-theme "$install_dir"
omarchy-shell shell rescanPlugins
omarchy plugin enable jmkelly.hearthlight-theme
```

### Published repository

Once the plugin directory is published as `jmkelly/omarchy-hearthlight-theme`,
the normal command is:

```bash
omarchy plugin add https://github.com/jmkelly/omarchy-hearthlight-theme --enable
```

## Plugin behavior

The plugin watches:

- `~/.local/state/omarchy/current/theme.name`
- `~/.local/state/omarchy/current/theme/colors.toml`
- `~/.local/state/omarchy/current/theme.generation` when present
- `~/.config/omarchy/hearthlight-theme.json` when present

It writes a normalized palette atomically to:

```text
~/.config/hearthlight/omarchy-theme.json
```

The default output path can be overridden in
`~/.config/omarchy/hearthlight-theme.json`:

```json
{
  "outputPath": "~/.config/hearthlight/omarchy-theme.json"
}
```

`outputPath` must be absolute or start with `~/`. The file is watched, so a
change applies without restarting the shell. `hearthlightUrl` is accepted for
forward compatibility but ignored with one warning; this version has no HTTP
client and no network write path.

The plugin uses Quickshell `FileView { watchChanges: true }` and does not poll.
File events are coalesced for 250 ms, then a 150 ms write timer starts. A theme
switch writes once, about 400 ms after the last file event. Writes go to a
temporary file in the target directory, then `chmod 600` runs, then `mv` replaces
the target. The target is never truncated in place. If a write fails, the
previous file stays intact and the error is logged once.

For a non-plugin setup, a `theme-set` hook under
`~/.config/omarchy/hooks/theme-set.d/` can write the same schema. It does not
carry the stock `themeId` mapping; Hearthlight can map the name itself. The
plugin remains preferred because it reacts to any theme file change, writes the
mapping and needs no hook. The
[plugin README](../integrations/omarchy-hearthlight-theme/README.md) has a
complete hook example.

## NixOS

```nix
services.hearthlight = {
  enable = true;
  omarchyTheme = true;
  omarchyThemeFile = "/var/lib/hearthlight/omarchy-theme.json";
};
```

The module passes `-omarchy-theme` and `-omarchy-theme-file` to the service. The
option defaults are `omarchyTheme = false` and
`omarchyThemeFile = "/var/lib/hearthlight/omarchy-theme.json"`.

The default service uses `DynamicUser = true` and `ProtectHome = true`, so it
cannot read a palette under a user's home. Either point the plugin at
`/var/lib/hearthlight/omarchy-theme.json` and grant the Hearthlight service read
access, or run the service as the desktop user with an override:

```nix
systemd.services.hearthlight.serviceConfig = {
  DynamicUser = lib.mkForce false;
  User = "alice";
  Group = "users";
  ProtectHome = lib.mkForce false;
};

services.hearthlight.omarchyThemeFile =
  "/home/alice/.config/hearthlight/omarchy-theme.json";
```

Use the second form when Hearthlight must read the same files the Omarchy
session owns. The service user needs read access to the file and every parent
directory. Do not weaken the mode to `0644` on a shared machine.

## Troubleshooting

Check the file directly:

```sh
jq . ~/.config/hearthlight/omarchy-theme.json
stat -c '%a %U %n' ~/.config/hearthlight/omarchy-theme.json
```

Check the endpoint:

```sh
curl -s http://127.0.0.1:8090/api/theme/omarchy | jq .
```

Check the plugin:

```sh
omarchy plugin list --json | jq '.[] | select(.id == "jmkelly.hearthlight-theme")'
omarchy theme current
cat ~/.local/state/omarchy/current/theme.name
```

If the response is `{"available":false}`, check that Hearthlight was started with
`-omarchy-theme` or `HEARTHLIGHT_OMARCHY_THEME=1`, and that the file exists, is
valid JSON, contains `name`, `colors.background` and `colors.foreground`, is no
larger than 1 MiB, and is readable by the Hearthlight user.

The server logs a warning when the file becomes malformed, an informational line
when it becomes unavailable, and one line at startup with the path it follows.
The plugin logs once per failure episode and keeps the last good file.

If the plugin output is stale, confirm the theme switch changed `theme.name` or
`theme/colors.toml`. A malformed palette is ignored. If `themeId` is absent, the
theme name is not in the stock mapping; that is expected for third-party and
custom themes, and Hearthlight synthesizes the palette.

If Hearthlight cannot read the file, check ownership and mode. Running
Hearthlight as a system service with a different home directory will not see the
default path. Do not weaken the mode to `0644` on a shared machine.

## Security

Omarchy plugins run unsandboxed inside the long-lived `omarchy-shell` process.
Review any plugin before enabling it.

This plugin:

- reads only the Omarchy theme files and its own config file;
- writes only the configured output file and its temporary sibling;
- parses TOML values as strings and never executes or evaluates theme content;
- accepts only hex colours and ignores other theme keys;
- has no HTTP client, no shell interpolation of palette content, and no network
  write path;
- writes the palette atomically with mode `0600`.

The theme name and palette are untrusted input. They are parsed, validated and
copied into JSON. Palette content is never interpolated into shell source, and
it is never passed to `eval` or a network request. Hearthlight only reads the
normalized file. It does not add an unauthenticated write endpoint to the HTTP
API. Remote live sync is a follow-up.
