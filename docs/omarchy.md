# Omarchy theme source

Hearthlight can follow the active Omarchy desktop theme. The Omarchy plugin
writes a normalized JSON palette, and Hearthlight watches that file and exposes
it read-only to the browser. Hearthlight never accepts theme writes over HTTP.

## Requirements

- The Omarchy plugin or a theme hook writes `omarchy-theme.json`.
- Hearthlight runs with read access to that file.
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
`#rrggbb` or `#rrggbbaa`). Unknown fields are ignored and invalid colour entries
are dropped. The server parses values only and never evaluates theme content.

`themeId` is optional. When it names a variant in the Hearthlight catalog,
Settings applies that variant. If it is absent, Hearthlight can use a mapping
from `web/omarchy.json` when that file is present. Otherwise it synthesizes a
theme from the raw colours, using the background ramp, foreground, accent and
named colours.

The plugin should write atomically with mode `0600`, for example with a
temporary file and `rename(2)`.

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
browsers over the existing event stream. No reload is needed.

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

A missing, unreadable, oversized or malformed file returns:

```json
{ "available": false }
```

The endpoint is read-only. There is no matching write endpoint.

## Settings

Settings, Appearance shows **Follow Omarchy theme** only when the endpoint
reports the theme source as available. While it is on:

- The family and mode pickers are locked.
- The current Omarchy theme name is shown.
- Theme changes apply live from the event stream.
- The follow preference is stored under `hearthlight.theme.omarchy.v1`.

If the file disappears or becomes malformed, Hearthlight keeps the last applied
theme and shows a small unavailable note. It does not reset the user's saved
theme. Turning the toggle off restores the normal theme selection.

## NixOS

```nix
services.hearthlight = {
  enable = true;
  omarchyTheme = true;
  omarchyThemeFile = "/var/lib/hearthlight/omarchy-theme.json";
};
```

The module passes `-omarchy-theme` and `-omarchy-theme-file` to the service.

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
session owns.

## Troubleshooting

Check the file directly:

```sh
jq . ~/.config/hearthlight/omarchy-theme.json
```

Check the endpoint:

```sh
curl -s http://127.0.0.1:8090/api/theme/omarchy | jq .
```

If the response is `{"available":false}`, check that the file exists, is valid
JSON, contains `name`, `colors.background` and `colors.foreground`, and is
readable by the Hearthlight user. The server logs a warning when the file
becomes malformed and an informational line when it becomes unavailable.
