# Hearthlight themes

Hearthlight ships 26 theme families and 37 variants. Hearthlight Dark is the
default. The catalog in `web/themes.json` drives the Settings picker, the CSS
custom properties and the canvas renderer, so a theme change reaches the map as
well as the chrome.

This page is the gallery and the authoring guide. The table, the screenshots and
the Omarchy mapping are generated from `web/themes.json` and `web/omarchy.json`
by `scripts/theme-gallery.sh`. Do not edit those blocks by hand.

The Settings dialog does not link to this page yet. Open it from the README or
the repository docs. A gallery link inside Settings is a separate integration.

## Families and variants

<!-- BEGIN THEME TABLE -->
| Variant | ID | Family | Mode | Source | Omarchy stock | Preview |
| --- | --- | --- | --- | --- | --- | --- |
| Hearthlight Dark | `hearthlight-dark` | Hearthlight | dark | [Hearthlight](https://github.com/jmkelly/hearthlight) | No, Hearthlight original. | [Preview](themes/hearthlight-dark.png) |
| Hearthlight Light | `hearthlight-light` | Hearthlight | light | [Hearthlight](https://github.com/jmkelly/hearthlight) | No, Hearthlight original. | [Preview](themes/hearthlight-light.png) |
| Catppuccin Mocha | `catppuccin-mocha` | Catppuccin | dark | [Catppuccin](https://github.com/catppuccin/catppuccin) | Yes. Omarchy `catppuccin`. | [Preview](themes/catppuccin-mocha.png) |
| Catppuccin Latte | `catppuccin-latte` | Catppuccin | light | [Catppuccin](https://github.com/catppuccin/catppuccin) | Yes. Omarchy `catppuccin-latte`. | [Preview](themes/catppuccin-latte.png) |
| Tokyo Night | `tokyo-night` | Tokyo Night | dark | [Tokyo Night](https://github.com/folke/tokyonight.nvim) | Yes. Omarchy `tokyo-night`. | [Preview](themes/tokyo-night.png) |
| Tokyo Night Day | `tokyo-night-day` | Tokyo Night | light | [Tokyo Night](https://github.com/folke/tokyonight.nvim) | No. | [Preview](themes/tokyo-night-day.png) |
| Rosé Pine Moon | `rose-pine-moon` | Rosé Pine | dark | [Rosé Pine](https://rosepinetheme.com/) | No. | [Preview](themes/rose-pine-moon.png) |
| Rosé Pine Dawn | `rose-pine-dawn` | Rosé Pine | light | [Rosé Pine](https://rosepinetheme.com/) | Yes. Omarchy `rose-pine`. | [Preview](themes/rose-pine-dawn.png) |
| Kanagawa Wave | `kanagawa-wave` | Kanagawa | dark | [Kanagawa](https://github.com/rebelot/kanagawa.nvim) | Yes. Omarchy `kanagawa`. | [Preview](themes/kanagawa-wave.png) |
| Kanagawa Lotus | `kanagawa-lotus` | Kanagawa | light | [Kanagawa](https://github.com/rebelot/kanagawa.nvim) | No. | [Preview](themes/kanagawa-lotus.png) |
| Everforest Dark | `everforest-dark` | Everforest | dark | [Everforest](https://github.com/sainnhe/everforest) | Yes. Omarchy `everforest`. | [Preview](themes/everforest-dark.png) |
| Everforest Light | `everforest-light` | Everforest | light | [Everforest](https://github.com/sainnhe/everforest) | No. | [Preview](themes/everforest-light.png) |
| Dracula | `dracula` | Dracula | dark | [Dracula](https://github.com/dracula/dracula-theme) | No. | [Preview](themes/dracula.png) |
| Dracula Alucard | `dracula-alucard` | Dracula | light | [Dracula](https://github.com/dracula/dracula-theme) | No. | [Preview](themes/dracula-alucard.png) |
| One Dark | `one-dark` | One | dark | [Atom One Dark](https://github.com/atom/one-dark-syntax) | No. | [Preview](themes/one-dark.png) |
| One Light | `one-light` | One | light | [Atom One Dark](https://github.com/atom/one-dark-syntax) | No. | [Preview](themes/one-light.png) |
| Solarized Dark | `solarized-dark` | Solarized | dark | [Solarized](https://github.com/altercation/solarized) | No. | [Preview](themes/solarized-dark.png) |
| Solarized Light | `solarized-light` | Solarized | light | [Solarized](https://github.com/altercation/solarized) | No. | [Preview](themes/solarized-light.png) |
| Gruvbox Dark | `gruvbox-dark` | Gruvbox | dark | [Gruvbox](https://github.com/morhetz/gruvbox) | Yes. Omarchy `gruvbox`. | [Preview](themes/gruvbox-dark.png) |
| Gruvbox Light | `gruvbox-light` | Gruvbox | light | [Gruvbox](https://github.com/morhetz/gruvbox) | No. | [Preview](themes/gruvbox-light.png) |
| Nord Dark | `nord-dark` | Nord | dark | [Nord](https://github.com/nordtheme/nord) | Yes. Omarchy `nord`. | [Preview](themes/nord-dark.png) |
| Nord Light | `nord-light` | Nord | light | [Nord](https://github.com/nordtheme/nord) | No. | [Preview](themes/nord-light.png) |
| Ethereal | `ethereal` | Ethereal | dark | [Omarchy](https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/ethereal/colors.toml) | Yes. Omarchy `ethereal`. | [Preview](themes/ethereal.png) |
| Hackerman | `hackerman` | Hackerman | dark | [Omarchy](https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/hackerman/colors.toml) | Yes. Omarchy `hackerman`. | [Preview](themes/hackerman.png) |
| Jade Horse | `jade-horse` | Jade Horse | dark | [Omarchy](https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/jade-horse/colors.toml) | Yes. Omarchy `jade-horse`. | [Preview](themes/jade-horse.png) |
| Last Horizon | `last-horizon` | Last Horizon | dark | [Omarchy](https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/last-horizon/colors.toml) | Yes. Omarchy `last-horizon`. | [Preview](themes/last-horizon.png) |
| Lumon | `lumon` | Lumon | dark | [Omarchy](https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/lumon/colors.toml) | Yes. Omarchy `lumon`. | [Preview](themes/lumon.png) |
| Matte Black | `matte-black` | Matte Black | dark | [Omarchy](https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/matte-black/colors.toml) | Yes. Omarchy `matte-black`. | [Preview](themes/matte-black.png) |
| Miasma | `miasma` | Miasma | dark | [Omarchy](https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/miasma/colors.toml) | Yes. Omarchy `miasma`. | [Preview](themes/miasma.png) |
| Retro 82 | `retro-82` | Retro 82 | dark | [Omarchy](https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/retro-82/colors.toml) | Yes. Omarchy `retro-82`. | [Preview](themes/retro-82.png) |
| Flexoki Light | `flexoki-light` | Flexoki Light | light | [Omarchy](https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/flexoki-light/colors.toml) | Yes. Omarchy `flexoki-light`. | [Preview](themes/flexoki-light.png) |
| Lupine | `lupine` | Lupine | light | [Omarchy](https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/lupine/colors.toml) | Yes. Omarchy `lupine`. | [Preview](themes/lupine.png) |
| Ristretto | `ristretto` | Ristretto | dark | [Omarchy](https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/ristretto/colors.toml) | Yes. Omarchy `ristretto`. | [Preview](themes/ristretto.png) |
| Solitude | `solitude` | Solitude | dark | [Omarchy](https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/solitude/colors.toml) | Yes. Omarchy `solitude`. | [Preview](themes/solitude.png) |
| Starship | `starship` | Starship | dark | [Omarchy](https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/starship/colors.toml) | Yes. Omarchy `starship`. | [Preview](themes/starship.png) |
| Vantablack | `vantablack` | Vantablack | dark | [Omarchy](https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/vantablack/colors.toml) | Yes. Omarchy `vantablack`. | [Preview](themes/vantablack.png) |
| White | `white` | White | light | [Omarchy](https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/white/colors.toml) | Yes. Omarchy `white`. | [Preview](themes/white.png) |
<!-- END THEME TABLE -->

`Omarchy stock` says Yes only when `web/omarchy.json` maps a stock Omarchy name
to that exact variant. A No is a Hearthlight interpretation or a Hearthlight
original. The mapping is pinned to Omarchy Quattro commit
`748fa3c0d0418d7a199818d94c1f87d495d7e804`. See [Omarchy](#omarchy) below for
the mapping table and the plugin.

## Gallery

Every image is captured from demo mode at a 1440x900 viewport with reduced
motion, no traffic edges, no hover state and no open panel. The demo map layout
is deterministic, so the screenshots are comparable across variants. The script
optimises the PNGs under `docs/themes/`.

<!-- BEGIN THEME GALLERY -->
### Hearthlight

#### Hearthlight Dark

![Hearthlight Dark map](themes/hearthlight-dark.png)

#### Hearthlight Light

![Hearthlight Light map](themes/hearthlight-light.png)

### Catppuccin

#### Catppuccin Mocha

![Catppuccin Mocha map](themes/catppuccin-mocha.png)

#### Catppuccin Latte

![Catppuccin Latte map](themes/catppuccin-latte.png)

### Tokyo Night

#### Tokyo Night

![Tokyo Night map](themes/tokyo-night.png)

#### Tokyo Night Day

![Tokyo Night Day map](themes/tokyo-night-day.png)

### Rosé Pine

#### Rosé Pine Moon

![Rosé Pine Moon map](themes/rose-pine-moon.png)

#### Rosé Pine Dawn

![Rosé Pine Dawn map](themes/rose-pine-dawn.png)

### Kanagawa

#### Kanagawa Wave

![Kanagawa Wave map](themes/kanagawa-wave.png)

#### Kanagawa Lotus

![Kanagawa Lotus map](themes/kanagawa-lotus.png)

### Everforest

#### Everforest Dark

![Everforest Dark map](themes/everforest-dark.png)

#### Everforest Light

![Everforest Light map](themes/everforest-light.png)

### Dracula

#### Dracula

![Dracula map](themes/dracula.png)

#### Dracula Alucard

![Dracula Alucard map](themes/dracula-alucard.png)

### One

#### One Dark

![One Dark map](themes/one-dark.png)

#### One Light

![One Light map](themes/one-light.png)

### Solarized

#### Solarized Dark

![Solarized Dark map](themes/solarized-dark.png)

#### Solarized Light

![Solarized Light map](themes/solarized-light.png)

### Gruvbox

#### Gruvbox Dark

![Gruvbox Dark map](themes/gruvbox-dark.png)

#### Gruvbox Light

![Gruvbox Light map](themes/gruvbox-light.png)

### Nord

#### Nord Dark

![Nord Dark map](themes/nord-dark.png)

#### Nord Light

![Nord Light map](themes/nord-light.png)

### Ethereal

#### Ethereal

![Ethereal map](themes/ethereal.png)

### Hackerman

#### Hackerman

![Hackerman map](themes/hackerman.png)

### Jade Horse

#### Jade Horse

![Jade Horse map](themes/jade-horse.png)

### Last Horizon

#### Last Horizon

![Last Horizon map](themes/last-horizon.png)

### Lumon

#### Lumon

![Lumon map](themes/lumon.png)

### Matte Black

#### Matte Black

![Matte Black map](themes/matte-black.png)

### Miasma

#### Miasma

![Miasma map](themes/miasma.png)

### Retro 82

#### Retro 82

![Retro 82 map](themes/retro-82.png)

### Flexoki Light

#### Flexoki Light

![Flexoki Light map](themes/flexoki-light.png)

### Lupine

#### Lupine

![Lupine map](themes/lupine.png)

### Ristretto

#### Ristretto

![Ristretto map](themes/ristretto.png)

### Solitude

#### Solitude

![Solitude map](themes/solitude.png)

### Starship

#### Starship

![Starship map](themes/starship.png)

### Vantablack

#### Vantablack

![Vantablack map](themes/vantablack.png)

### White

#### White

![White map](themes/white.png)
<!-- END THEME GALLERY -->

## Interpretation notes

Hearthlight Dark and Hearthlight Light are original Hearthlight palettes. The
third-party families are Hearthlight-authored interpretations of published
palettes, not official ports. Each family links to the project Hearthlight
interpreted. The upstream project may ship other variants, different defaults
or later palette changes.

Every palette needs small contrast adjustments to work as a map. The catalog
records the exact shifts in each family's `source.note`. The notes below explain
what changed and why. For the Omarchy families, the colours come from the pinned
`colors.toml`, but the semantic token assignment, kind colours and category
colours are Hearthlight's.

### Hearthlight

Hearthlight Dark is the default deep-navy map with a cyan accent and a cold
canvas ramp. Hearthlight Light is the authored companion, using the same
semantic roles on a cool white surface. Neither is an Omarchy stock theme.

### Catppuccin

Catppuccin Mocha and Latte both ship. Latte keeps the canonical base, text,
subtext1, blue, teal, mauve and red. Green, yellow, peach and pink darken at the
same hue for 3:1 graphics contrast. Mocha is Omarchy stock `catppuccin`; Latte is
stock `catppuccin-latte`.

### Tokyo Night

Tokyo Night is the dark variant and Tokyo Night Day is the light companion. The
dark palette lightens `history` from `#565f89` to `#707aa8` for 3:1 contrast on
the canvas. The day variant keeps canonical text, blue, cyan, green, orange,
purple, violet and teal, darkens red to `#ef245f`, uses `#4e5c8f` for muted text
and `#5f6685` for history. Only the dark variant is Omarchy stock
`tokyo-night`.

### Rosé Pine

Rosé Pine Moon is the dark variant and Rosé Pine Dawn is the light variant. The
dark palette lightens muted from `#908caa` to `#9591b2` and history from
`#6e6a86` to `#908caa`. Dawn keeps canonical base, text, pine, iris, foam and
love, darkens gold and rose for graphics contrast, and derives a green for
`online` because Dawn defines none. Omarchy stock `rose-pine` is the light Dawn
palette, so it maps to `rose-pine-dawn`. Moon is not an Omarchy stock theme.

### Kanagawa

Kanagawa Wave and Lotus both ship. Wave lightens muted, history, device and
other from `#727169` to `#9e9b82` for text and canvas contrast. Lotus darkens
muted to `#686659` (a final 39.0% to 37.8% lightness step at the same 52 degree
hue, which clears 4.60:1 on `#ede7b7`), green to `#677f49`, teal to `#477f94` and
orange to `#b66200` at the same hues. Wave is Omarchy stock `kanagawa`; Lotus is
not.

### Everforest

Everforest Dark and Light both ship. The dark palette lightens muted to
`#9da9a0`. The light palette darkens muted, accent, yellow, green, orange, aqua,
purple and grey for text and graphics contrast, and adds an authored violet for
media so it stays distinct from the phone magenta. Dark is Omarchy stock
`everforest`; Light is not.

### Dracula

Dracula is the dark variant and Dracula Alucard is the light variant. The dark
palette lightens muted, history, device and other from `#6272a4` to `#8b97bc` at
the same blue-violet hue. Alucard uses the official values unchanged because
every text and graphical pair clears its target on the derived light canvas.
Neither variant is Omarchy stock.

### One

One Dark and One Light are Hearthlight interpretations of Atom One. The dark
palette lightens muted to `#8d94a1` and history to `#77808f`. The light palette
darkens green, orange and the history grey for 3:1 contrast. Neither variant is
Omarchy stock.

### Solarized

Solarized Dark and Light both ship. The dark palette lightens muted to `#79929a`
and history to `#657e86`, and lifts several graphical hues. The light palette
darkens muted, history, green, yellow and cyan at their canonical hues. Neither
variant is Omarchy stock.

### Gruvbox

Gruvbox Dark and Light both ship. The dark palette shifts accent from the warm
canonical blue `#83a598` to `#7fb0b5` at hue 186 so tailnet stays inside the cool
band, shifts media from `#d3869b` to `#b986d3` at hue 280, and uses `fg3` for
muted text. The light variant keeps the canonical light ramp and adds darker
media and IoT colours for separation. Dark is Omarchy stock `gruvbox`; Light is
not.

### Nord

Nord Dark uses the canonical Polar Night and Snow Storm ramps, lightens muted,
history and device, shifts media into the cool band and darkens `canvasBottom`
because Nord defines no darker base. Nord Light is a Hearthlight-authored
pairing, not an official Nord variant. It is built from the Snow Storm ramp and
darkens the Aurora and frost marks at the same hue for graphics contrast. Dark
is Omarchy stock `nord`; Light is not.

### Ethereal

Ethereal is an Omarchy stock dark theme. The canvas uses `lighter_background`
`#131a3a` and `darker_background` `#030610`. Muted lightens from `#6d7db6` to
`#919dd6` for text contrast. `online` becomes the more saturated `#8fd6a4` so it
keeps more chroma than `history`, and the shell accent `#7d82d9` is used
unchanged.

### Hackerman

Hackerman is an Omarchy stock dark theme. The canvas uses `#151828` and
`#06060c`. The stock palette is a green terminal set with no cool accent and no
warm hue, so accent uses the named blue `#829dd4` for tailnet and warning uses
Hearthlight amber `#e0b36a` for LAN. Danger uses `#ff7a7a` because the stock red
is green. Muted lifts to `#8f99c4`, `online` keeps the stock `#82fb9c`, and
history is the low-chroma blue-grey `#7c85ad`.

### Jade Horse

Jade Horse is an Omarchy stock dark theme. The canvas uses `#203b31` and
`#0a1713`. The stock accent `#8bcbb1` sits at hue 160, outside Hearthlight's cool
tailnet band, so accent uses the named blue `#96b7d0`. Muted lifts to `#93bfae`,
`online` keeps the stock green `#8bcbb1`, and history is `#91ad9f`.

### Last Horizon

Last Horizon is an Omarchy stock dark theme. The stock background and
`lighter_background` are both `#0c0b0c`, so the canvas keeps `#0c0b0c` at the
top and uses `darker_background` `#060606` at the bottom. The stock accent is
warm and low-chroma, so accent uses the named cyan `#a5a0b6` and warning uses
`#d1a36f`. Muted lifts to `#a89da1`, `online` uses `#87a9b0` because the palette
has no green, and history is `#6f6a70`. Kind colours use lightness variants
because the stock palette has fewer than nine distinct hues.

### Lumon

Lumon is an Omarchy stock dark theme. The canvas uses `#1b2d40` and `#0b1216`.
The stock palette is blue-only with no warm or green hue, so warning uses
Hearthlight amber `#d9b36a`, danger uses `#c96b6b`, and `online` uses mint
`#7fd1b0`. Muted lifts to `#8fb0cc` and history is `#6f86a0`.

### Matte Black

Matte Black is an Omarchy stock dark theme. The canvas uses `#1e1e1e` and
`#090909`. The stock accent `#e68e0d` is warm, so it becomes warning and accent
uses the cool blue `#6da8d8`. The palette has no green, so online uses
`#7fc97f`. Muted lifts to `#9e9e9e` and history is `#8a8a8d`. Phone, media and
IoT hues are Hearthlight additions.

### Miasma

Miasma is an Omarchy stock dark theme. The canvas uses `#2c2c2c` and `#121212`.
The stock accent `#78824b` is a warm olive at hue 68, so accent uses the muted
blue `#7f93b3`. Warning keeps the stock yellow `#b36d43`, danger uses `#b35a4a`
because the stock red is brown, muted lifts to `#a8a894`, online keeps `#5f875f`
and history is `#85857a`.

### Retro 82

Retro 82 is an Omarchy stock dark theme. The canvas uses `#0a2540` and
`#020c17`. The stock accent `#faa968` is warm, so it becomes warning and accent
uses the named green `#028391` at hue 186. Muted lifts to `#7fb8c2`, online uses
`#8cbfb8`, history is `#7f98a0`, and media and IoT use `#a8c4d8` and `#6fd1b8`
to keep the cool bands distinct.

### Flexoki Light

Flexoki Light is an Omarchy stock light theme. The accent is the shell accent
`#205ea6`. Muted and history darken to `#6f6d64` and `#7c7a72`. Green, yellow,
blue, cyan, orange and magenta darken at the same hues for text and graphics
contrast, and media uses the authored violet `#6f4fa0` so it stays distinct from
the phone magenta.

### Lupine

Lupine is an Omarchy stock light theme. The accent is the shell accent
`#3264eb`. Muted darkens to `#656565` and history to `#747474`. The palette ships
no warm hue, so warning and printer use the authored amber `#8f5410` and
`#b86a1e`. Computer and infra use `#013a6f`, IoT uses `#0c67de`, web uses the
accent, online uses `#4a2fd0` and danger uses `#c900c4`.

### Ristretto

Ristretto is an Omarchy stock dark theme. The shell accent is the warm `#f38d70`,
so accent and media use the magenta slot `#a8a9eb` and `#bebffd` to keep tailnet
inside the cool band. Muted lifts to `#a89d9e`, history is `#b0989a`, and online
keeps the green `#adda78`.

### Solitude

Solitude is an Omarchy stock dark theme. The accent is the shell accent
`#798186`. Muted lifts to `#8a9097` and history is `#9a9a9a`. Online uses the
authored muted green `#8fb59f`, warning uses `#d8cdbd`, and media uses the
authored lavender-grey `#a2a7c4`.

### Starship

Starship is an Omarchy stock dark theme. The shell accent is the warm `#ff9f0a`,
so accent and media use the blue and cyan slots `#8fb8de` and `#a6dcef`.
Warning uses `#ffc27a`, online uses the green slot `#ff9f0a` because this
palette treats it as a macOS-style orange, and history uses `#98989d`.

### Vantablack

Vantablack is an Omarchy stock dark theme on pure black. The accent is a
cool-tinted `#8d9ba3` derived from the shell accent `#8d8d8d`. The greys are
deliberately tinted while staying near-monochrome. Muted is `#8a8a8a`, warning
is `#c9bda3`, online is `#86b386`, history is `#8f8f8f`, and every kind and
category keeps a distinct lightness so marks stay separable.

### White

White is an Omarchy stock light theme on pure white. The accent is a cool-tinted
`#626b73` derived from the shell accent `#6e6e6e`. Muted darkens to `#666666`,
warning uses `#7a694d`, online uses `#2f6b3a`, history uses `#6f6f6f`, and every
kind and category keeps a distinct dark shade for separation on `#e8e8e8`.

## How themes work

`web/themes.json` is the canonical catalog. It has `schemaVersion: 1` and a
`families` array. Each family has an `id`, a `name`, an optional `source` object
with `name`, `url` and an optional `note`, and a `variants` array. Each variant
has an `id`, a `name`, a `mode` of `dark` or `light`, and the palette, kind,
category and effect data.

```json
{
  "schemaVersion": 1,
  "families": [
    {
      "id": "example",
      "name": "Example",
      "source": { "name": "Example project", "url": "https://example.com" },
      "variants": [
        {
          "id": "example-dark",
          "name": "Example Dark",
          "mode": "dark",
          "palette": {},
          "kinds": {},
          "categories": {},
          "effects": {}
        }
      ]
    }
  ]
}
```

`web/themes.js` is generated from the catalog:

```bash
make themes
```

That command writes `window.HearthlightThemes = <catalog>;`. Never edit
`web/themes.js` by hand. A test fails when it is stale or when the embedded copy
in the Go binary differs from the checked-in file.

`web/index.html` loads `themes.js` and `theme-boot.js` in the document head,
before the stylesheet. `theme-boot.js` reads the stored preference, applies any
URL override, resolves the mode and sets CSS custom properties on the root
element. It also sets `data-theme` to the family ID and `data-mode` to the
resolved mode. The renderer reads the same values from
`window.HearthlightTheme.resolve()` and applies the palette to the canvas.
`theme-import.js` loads with `defer` before `app.js` so stored custom themes are
registered before the map starts.

When the requested mode is unavailable, the boot script falls back to the
family's dark variant or its first variant and marks the result as a fallback.
Settings shows a note when that happens.

The read-only catalog is available at `GET /api/themes`. The response is the
catalog plus an `omarchy` object containing the same mapping as
`web/omarchy.json`. The mapping is also served as `/omarchy.json`.

### URL overrides

`?theme=<variant-id>` selects an exact variant for that page load. `?mode=dark`,
`?mode=light` or `?mode=auto` changes only the mode and keeps the stored family.
An unknown variant, unknown mode or invalid value is ignored and the stored
preference applies. URL overrides are not persisted.

```text
http://127.0.0.1:8090/?theme=hearthlight-light
http://127.0.0.1:8090/?theme=kanagawa-wave&mode=auto
http://127.0.0.1:8090/?mode=light
```

### High contrast

Settings → Appearance has a **High contrast** toggle. It stores the choice under
`hearthlight.contrast.v1` as `high` or `normal`. Until the user picks a value,
Hearthlight follows `prefers-contrast: more`. High contrast keeps the palette but
removes blur, glow and shadow, promotes muted chrome text to the palette's ink,
strengthens labels and outlines, and makes focus rings heavier. It is separate
from the theme mode and works with every family.

**Reset to default** restores Hearthlight Dark and removes the stored contrast
override, so the operating system preference applies again.

### Palette tokens

Every variant must define these palette tokens:

| Token | Used for |
| --- | --- |
| `bg` | Page background and base surface. |
| `ink` | Primary text and high-contrast marks. |
| `muted` | Secondary text and quiet labels. |
| `accent` | Focus, selection, links and tailnet highlights. |
| `line` | Borders and dividers. |
| `glass` | Glass panel fill. |
| `canvasTop` | Map canvas gradient at the top. |
| `canvasMiddle` | Map canvas gradient in the middle. |
| `canvasBottom` | Map canvas gradient at the bottom. |
| `grid` | Canvas grid lines. |
| `ambientGlow` | Ambient colour behind the territories. |
| `online` | Online state and live indicators. |
| `warning` | LAN traffic, warnings and warm state. |
| `danger` | Errors and destructive state. |
| `history` | Historic devices and services. |
| `vignette` | Edge shading over the canvas. |
| `labelHalo` | Text halo behind map labels. |
| `bubble` | Device bubble fill. |
| `panel` | Detail panel and settings surface. |

Camel case names become kebab-case CSS custom properties. For example,
`canvasTop` becomes `--theme-canvas-top`.

Each built-in variant also needs all nine `kinds` entries, each with a `label`
and a `color`: `router`, `server`, `computer`, `phone`, `nas`, `media`, `iot`,
`printer`, `device`.

The seven built-in `categories` entries are `web`, `media`, `data`, `remote`,
`infra`, `iot` and `other`, each with a colour.

Effects tune the renderer rather than the palette:

| Effect | Range | Controls |
| --- | --- | --- |
| `vignette` | 0 to 1 | Vignette strength. |
| `deviceGlow` | 0 to 1 | Glow around device bubbles. |
| `devicePulse` | 0 to 1 | Pulse strength for online devices. |
| `serviceGlow` | 0 to 1 | Glow around service bubbles. |
| `territoryBlur` | 0 to 64 | Blur applied to territory fills. |
| `ribbonBlur` | 0 to 64 | Blur applied to the LAN and tailnet ribbons. |
| `lensBlur` | 0 to 64 | Blur applied to the overlap lens. |
| `glassBlur` | 0 to 64 | Blur behind glass panels. |
| `panelBlur` | 0 to 64 | Blur behind the detail panel. |
| `territoryBlend` | `screen` or `multiply` | Canvas composite operation. |
| `territoryAlpha` | 0 to 1 | Opacity multiplier for territory fills. |
| `glowStrength` | 0 to 1 | Global glow multiplier. |
| `labelHalo` | 0 to 1 | Halo opacity for map labels. |
| `shadowStrength` | 0 to 1 | Shadow multiplier for device and service marks. |

All effect values except `territoryBlend` are finite, non-negative numbers.

### Storage keys

Theme preferences are per browser. They are stored in `localStorage` and are
never sent to the server.

| Key | Contents |
| --- | --- |
| `hearthlight.theme.v1` | Selected family and mode: `{"family":"hearthlight","mode":"dark"}`. `mode` is `dark`, `light` or `auto`. |
| `hearthlight.contrast.v1` | `high` or `normal`. Absent means follow `prefers-contrast: more`. |
| `hearthlight.theme.omarchy.v1` | Follow preference: `{"enabled":true}`. |
| `hearthlight.themes.custom.v1` | Custom theme catalog document with a `families` array. |

## Import and export

Settings → Appearance → **Import theme** accepts a pasted theme or a `.json` /
`.toml` file. A live preview shows the resolved mode and palette swatches before
saving. Invalid input reports field-level errors and does not change the active
theme.

Custom themes appear in a **Custom** section below the built-in families. They
can be renamed, duplicated, exported or deleted. Built-in themes are never
modified.

### Hearthlight JSON

The importer accepts a full catalog, a single family, a single variant, or a
bare variant object. A family can carry `variants` as an array or as an object
keyed by `dark` and `light`.

```json
{
  "schemaVersion": 1,
  "families": [
    {
      "id": "my-theme",
      "name": "My Theme",
      "source": { "name": "My Theme project", "url": "https://example.com" },
      "variants": [
        {
          "id": "my-theme-dark",
          "name": "My Theme Dark",
          "mode": "dark",
          "palette": {
            "bg": "#101014",
            "ink": "#eceff4"
          },
          "effects": {}
        }
      ]
    }
  ]
}
```

A variant needs `id`, `name` and either `palette` or `colors`. It needs `mode`
unless it is keyed under `dark` or `light`, and it must include every palette
token listed above. `effects` is optional and defaults to the dark or light
effect set. `kinds` and `categories` are optional for custom themes; when they
are missing, Hearthlight uses the built-in labels and colours. When they are
present, every entry is validated.

`schemaVersion` is optional for a single theme and must be `1` when present.
Unknown top-level keys are ignored. Exported files include `schemaVersion` and a
`source` block so a future version can migrate them.

### Omarchy `colors.toml`

The importer also reads the simple Omarchy `colors.toml` format. It uses a small
line parser, not a TOML dependency. Comments, blank lines and nested table
headers are ignored. Values must be quoted strings, as in the Omarchy files.

```toml
mode = "dark"
background = "#1a1b26"
foreground = "#c0caf5"
accent = "#7aa2f7"
green = "#9ece6a"
yellow = "#e0af68"
red = "#f7768e"
blue = "#7aa2f7"
cyan = "#7dcfff"
magenta = "#bb9af7"
```

The importer maps the source colours onto Hearthlight's semantic tokens:

| Omarchy source | Hearthlight tokens |
| --- | --- |
| `background`, `dark_background`, `darker_background`, `lighter_background` | `bg`, `canvasTop`, `canvasBottom` |
| `foreground`, `bright_foreground`, `light_foreground`, `dark_foreground` | `ink`, `muted`, `history` |
| `accent`, falling back to `blue` or `cyan` | `accent` |
| `green` | `online`, NAS, remote |
| `yellow` or `orange` | `warning`, router, printer, data |
| `red` | `danger` |
| `blue` | computer, infra |
| `cyan` | IoT, web |
| `magenta` | phone, media |

Unknown top-level keys are ignored and listed in the preview. When a required
source group is missing, the importer derives it from the current Hearthlight
theme or the built-in fallback and tells you which keys were derived. When
`mode` is missing, it is inferred from the background luminance and reported as
derived. The imported family gets a content-derived ID such as
`omarchy-1a2b3c4d5e6f`, because the import path does not consult the stock name
mapping. Importing a stock `colors.toml` creates a custom theme; use the plugin
when you want the live Omarchy theme.

### Validation and limits

| Rule | Limit |
| --- | --- |
| Import size | 256 KiB per paste or file. |
| Custom families | 50 stored families. |
| Name | 64 characters, no control characters. |
| ID | 64 bytes, starts with a letter or digit, then letters, digits, dot, underscore, hyphen or colon. |
| Built-in IDs | Cannot be reused by a custom family or variant. |
| Duplicate modes | A family cannot contain two variants with the same mode. |
| Schema | `schemaVersion` must be `1` when present. |
| Colours | Hex (`#rgb`, `#rgba`, `#rrggbb`, `#rrggbbaa`) or `rgb()` / `rgba()`. |
| Effects | Finite values inside the documented range; `territoryBlend` must be `screen` or `multiply`. |
| Source URL | `http` or `https`, at most 512 bytes. |
| Forbidden keys | `__proto__`, `constructor` and `prototype` are rejected anywhere in a JSON document. |

The import validator allows `glowStrength` and `shadowStrength` up to 4. The
built-in catalog test keeps shipped variants at 0 to 1, so use the narrower range
when you add a family to `web/themes.json`.

The importer never fetches a remote URL, never evaluates theme content, and
escapes names and labels before inserting them into HTML. An import that fails
validation leaves the active theme unchanged. If an imported family reuses a
custom family ID, the interface asks before replacing it. Built-in themes are
never overwritten.

### Storage and private browsing

Custom themes are stored under `hearthlight.themes.custom.v1`. If the browser
storage quota is full, Hearthlight keeps the previous theme and shows a toast
instead of partially saving. If `localStorage` is disabled, the import still
applies for the current session but is not remembered after a reload.

On boot, each stored custom family is validated again. A malformed entry is
skipped with a console warning and does not stop the built-in theme catalog from
loading. If a custom theme is deleted while active, Hearthlight returns to
Hearthlight Dark.

### Custom theme management

- **Rename** changes the display name and keeps the IDs.
- **Duplicate** copies the family with a ` copy` name suffix and collision-free
  IDs.
- **Export** downloads the selected family as `<family-id>.json`.
- **Export all** downloads every custom family as
  `hearthlight-custom-themes.json`.
- **Delete** removes the family after confirmation.

An export is a normal catalog document with `schemaVersion`, a Hearthlight
`source` block and the selected `families`, so exporting and re-importing
round-trips without loss.

## Omarchy

Hearthlight ships the full stock Omarchy name mapping in `web/omarchy.json`. It
maps 23 Omarchy Quattro names to shipped Hearthlight variants and records each
stock mode. The file is pinned to commit
`748fa3c0d0418d7a199818d94c1f87d495d7e804`; the `source` field carries the full
tree URL and the `commit` field carries the SHA.

The server merges the mapping into `GET /api/themes` under `omarchy` and serves
it as `/omarchy.json`. The Settings picker uses it for the Omarchy source badge.
When the live theme arrives with a `themeId`, that variant is used directly.
Otherwise the browser looks up the Omarchy theme name in the mapping. If neither
matches, Hearthlight synthesizes a palette from the raw colours.

<!-- BEGIN OMARCHY MAPPING -->
| Omarchy theme | Mode | Hearthlight variant |
| --- | --- | --- |
| `catppuccin` | dark | `catppuccin-mocha` |
| `catppuccin-latte` | light | `catppuccin-latte` |
| `ethereal` | dark | `ethereal` |
| `everforest` | dark | `everforest-dark` |
| `flexoki-light` | light | `flexoki-light` |
| `gruvbox` | dark | `gruvbox-dark` |
| `hackerman` | dark | `hackerman` |
| `jade-horse` | dark | `jade-horse` |
| `kanagawa` | dark | `kanagawa-wave` |
| `last-horizon` | dark | `last-horizon` |
| `lumon` | dark | `lumon` |
| `lupine` | light | `lupine` |
| `matte-black` | dark | `matte-black` |
| `miasma` | dark | `miasma` |
| `nord` | dark | `nord-dark` |
| `retro-82` | dark | `retro-82` |
| `ristretto` | dark | `ristretto` |
| `rose-pine` | light | `rose-pine-dawn` |
| `solitude` | dark | `solitude` |
| `starship` | dark | `starship` |
| `tokyo-night` | dark | `tokyo-night` |
| `vantablack` | dark | `vantablack` |
| `white` | light | `white` |
<!-- END OMARCHY MAPPING -->

To refresh the mapping after an Omarchy update, read the new stock
`themes/*/colors.toml` tree, update `web/omarchy.json` and the pinned constants
in `web/omarchy_test.go`, then run `make check`. The test fails when a stock name
is missing, a target does not exist, a mode disagrees, or `rose-pine` maps to the
dark Moon variant instead of Dawn.

The plugin lives at `integrations/omarchy-hearthlight-theme`. Build the release
archive with:

```bash
make omarchy-plugin
```

That writes `dist/omarchy-hearthlight-theme-<version>.tar.gz` and its `.sha256`
checksum. The archive root contains `manifest.json`, `Service.qml`, `README.md`
and `LICENSE`. Unpack it into
`$HOME/.config/omarchy/plugins/jmkelly.hearthlight-theme`, then enable it with
`omarchy plugin enable jmkelly.hearthlight-theme`. The plugin README has the
local-checkout, published-repository, update and uninstall commands.

The plugin watches the active Omarchy theme and writes
`~/.config/hearthlight/omarchy-theme.json` with mode `0600`. Hearthlight watches
that file, rereads it about once per second and pushes changes over the existing
event stream. Omarchy plugins run unsandboxed inside the long-lived
`omarchy-shell` process, so review the plugin before enabling it. See
[Omarchy theme source](omarchy.md) for the server flags, file schema, API, NixOS
options and security model.

## Adding a theme

1. Add a family and variant to `web/themes.json`. Family IDs and variant IDs use
   kebab-case and must be unique. A variant ID is global across families because
   `?theme=` and `omarchy.json` use the variant ID. Variant IDs also cannot
   repeat a word, so `tokyo-night` is valid and `tokyo-tokyo` is not.
2. Set `mode` to `dark` or `light`. Do not invent a light variant for a
   dark-only palette just to fill the slot. Auto falls back and Settings explains
   it.
3. Add a `source` object for a third-party palette:

   ```json
   "source": {
     "name": "Example Palette",
     "url": "https://example.com/palette",
     "note": "Describe any contrast adjustment here."
   }
   ```

   Credit the project you interpreted. If you adjust a colour for contrast,
   record that in `source.note` or in the pull request description. Do not call
   an interpretation an official port.
4. Define every palette token, every kind, every category and every effect
   listed above. Copying an existing variant and replacing the palette is the
   normal path.
5. Run `make themes` to regenerate `web/themes.js`.
6. Run `make check`. It checks formatting, vet and the full test suite. The theme
   tests validate the schema, required tokens, effect ranges, kebab-case IDs,
   distinct kind and category colours, WCAG contrast, the dark-map hue bands,
   freshness of `web/themes.js` and that the embedded files match the checkout.
   If a graphical pair needs an exception, the variant must name the pair and
   give a reason in `contrastExceptions`. Text pairs cannot be exempt.
7. Add or update the family's note in this page when the palette needs an
   explanation.
8. Capture the new variant and refresh this page:

   ```bash
   ./scripts/theme-gallery.sh
   ```

   The script updates the table, gallery and Omarchy mapping from the catalog,
   captures a 1440x900 demo screenshot for every variant when a headless
   Chromium or Chrome is available, and optimises the images under
   `docs/themes/`. It needs Go and Python 3. ImageMagick's `magick` or `convert`
   optimises the PNGs; without it the screenshots stay uncompressed. Set
   `BROWSER` to choose a specific Chromium or Chrome executable. Use
   `./scripts/theme-gallery.sh --docs-only` to refresh the generated blocks
   without capturing screenshots.

### Contrast targets

Use WCAG 2.2 relative luminance and contrast ratio. `make check` enforces the
targets below for every shipped variant.

| Pair | Minimum |
| --- | --- |
| Body text (`ink` and `muted`) on `bg`, `canvasMiddle`, `panel`, bubble surfaces, glass surfaces and label halos | 4.5:1 |
| Default light palette (`Hearthlight Light`) `ink` on `bg` and `labelHalo` | 7:1 |
| Focus ring (`accent`) against the adjacent background | 3:1 |
| Device and service marks, territory boundaries, traffic edges, reach badges | 3:1 |
| Status colours (`online`, `history`) against the canvas | 3:1 |
| Category and kind dots against `bubble` or the canvas | 3:1 |

The tests also enforce map semantics. Dark variants need a warm warning hue
between 20 and 70 degrees, a cool accent and media hue between 180 and 290
degrees, and an `online` colour with more saturation than `history`. Kind and
category colours must be distinct, and every built-in variant must define all
nine kinds and all seven categories.

Canonical palettes sometimes miss these values. Adjust lightness or chroma while
keeping the palette's character, then note the adjustment in `source.note`. Do
not lower the target. A graphical exception is a last resort and needs a reason.

## Troubleshooting

### The theme does not stick

Theme selection is per browser. It is stored in `localStorage` under
`hearthlight.theme.v1` and is never sent to the server. A different browser,
device or profile keeps its own choice.

Private browsing and hardened browser settings can disable `localStorage`.
Hearthlight still applies the theme for the current page, but it cannot remember
it after a reload. Try a normal window or allow site data for the Hearthlight
origin. Clearing site data also resets the choice.

To inspect the stored value, open the browser console on the Hearthlight page:

```js
localStorage.getItem("hearthlight.theme.v1")
```

### Flash of the default theme on load

`themes.js` and `theme-boot.js` must load in the document head before the
stylesheet. Both are small, blocking scripts served by Hearthlight itself. If the
page flashes Hearthlight Dark before applying a stored or URL theme, check that
`/themes.js` and `/theme-boot.js` return 200, that a reverse proxy or CSP is not
blocking them, and that a stale cache is not serving an old `index.html`. A hard
reload usually clears it.

`theme-boot.js` catches storage and query parsing failures and falls back to the
default. A disabled `localStorage` produces the default on every load, not a
flash after first paint.

### Light or Auto shows a dark palette

Some families ship only a dark variant. Light and Auto fall back to the
available palette, and Settings shows a note. That is intentional. Use a family
with a light variant when you need a light map. The light variants are
Hearthlight Light, Catppuccin Latte, Tokyo Night Day, Rosé Pine Dawn, Kanagawa
Lotus, Everforest Light, Dracula Alucard, One Light, Solarized Light, Gruvbox
Light, Nord Light, Flexoki Light, Lupine and White.

### `?theme=` is ignored

`theme` takes a variant ID, not a family ID. Use an ID from the table, such as
`hearthlight-light` or `kanagawa-wave`. An unknown ID, an unknown mode or a
malformed query is ignored, and the stored preference applies. The override is
not saved.

### An import is rejected

The preview lists the failing path and the reason. Common causes are a missing
palette token, an invalid colour, a `schemaVersion` other than `1`, a duplicate
ID, a repeated mode in one family, an effect outside its range, a forbidden key
or a file larger than 256 KiB. Failed validation does not change the active
theme. Correct the field and paste again.

### A custom theme disappeared

Custom themes are stored under `hearthlight.themes.custom.v1`. Private browsing,
disabled storage, cleared site data and a full quota can all prevent persistence.
The import dialog shows the storage state. Export a custom theme before clearing
site data if you want to keep it. On boot, a malformed stored family is skipped
with a console warning. The 50-family cap is a separate limit.

### Screenshots are pending

The gallery script needs Go, Python 3 and curl, plus a headless Chromium or
Chrome for capture. Install one, or set `BROWSER` to its executable, then run
`./scripts/theme-gallery.sh`. `--docs-only` refreshes the generated blocks
without images and leaves any existing screenshots in place.

### Omarchy theme not detected

See the ordered checks in [Omarchy theme source](omarchy.md#troubleshooting).
Start with `-omarchy-theme`, the plugin's output file and
`GET /api/theme/omarchy`.
