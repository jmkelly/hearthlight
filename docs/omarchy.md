# Omarchy theme source

Hearthlight can follow the active Omarchy desktop theme. The Omarchy plugin
writes a normalized JSON palette, and Hearthlight watches that file and exposes
it read-only to the browser. Hearthlight never accepts theme writes over HTTP.

The plugin lives in
[`integrations/omarchy-hearthlight-theme`](../integrations/omarchy-hearthlight-theme/README.md).
The stock Omarchy name mapping is still tracked in
[#22](https://github.com/jmkelly/hearthlight/issues/22).

## Requirements

- The Omarchy plugin or a theme hook writes `omarchy-theme.json`.
- Hearthlight runs with read access to that file.
- Hearthlight is started with `-omarchy-theme` (or
  `HEARTHLIGHT_OMARCHY_THEME=1`).
- No extra port or token is needed. The endpoint uses the same optional token as
  the rest of the API when one is configured.

## Palette file

The file schema is:

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
entries with an invalid name or value are dropped. The server parses values
only and never evaluates theme content.

`themeId` is optional. When it names a variant in the Hearthlight catalog,
Settings applies that variant. Otherwise the browser looks for an optional
`/omarchy.json` name mapping. This build does not ship that file, so the stock
mapping is pending in #22. Without a match, Hearthlight synthesizes a theme from
the raw colours, using the background ramp, foreground, accent and named
colours.

The plugin writes atomically with mode `0600`, using a temporary file and a
rename. It keeps the last good file when a theme write fails.

## Server flags

| Flag | Environment variable | Default | Meaning |
| --- | --- | --- | --- |
| `-omarchy-theme` | `HEARTHLIGHT_OMARCHY_THEME` | `false` | Enable the read-only Omarchy theme source. |
| `-omarchy-theme-file` | `HEARTHLIGHT_OMARCHY_THEME_FILE` | `~/.config/hearthlight/omarchy-theme.json` | Normalized palette file to watch. |

Boolean environment values accept `1`, `true`, `yes` and `on` for true, and
`0`, `false`, `no` and `off` for false.

The default path expands the home directory of the user that starts
Hearthlight. A system service usually runs under a different user, so set
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

## Settings

Settings → Appearance shows **Follow Omarchy theme** when the endpoint reports
the theme source as available. It stays visible while enabled so the user can turn
it off if the file later becomes unavailable. While it is on:

- The family and mode pickers are locked.
- The current Omarchy theme name is shown.
- Theme changes apply live from the event stream.
- The follow preference is stored under `hearthlight.theme.omarchy.v1`.

If the file disappears or becomes malformed, the browser keeps the last applied
theme and shows a small unavailable note. It does not reset the user's stored
manual theme. Turning the toggle off restores the normal theme selection.

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

If the response is `{"available":false}`, check that the file exists, is valid
JSON, contains `name`, `colors.background` and `colors.foreground`, is no larger
than 1 MiB, and is readable by the Hearthlight user. Also check that Hearthlight
was started with `-omarchy-theme` or `HEARTHLIGHT_OMARCHY_THEME=1`.

The server logs a warning when the file becomes malformed, an informational line
when it becomes unavailable, and one line at startup with the path it follows.
