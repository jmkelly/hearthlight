package web

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

type themeCatalog struct {
	SchemaVersion int           `json:"schemaVersion"`
	Families      []themeFamily `json:"families"`
}

type themeFamily struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Source   themeSource    `json:"source"`
	Variants []themeVariant `json:"variants"`
}

type themeSource struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type themeVariant struct {
	ID         string               `json:"id"`
	Name       string               `json:"name"`
	Mode       string               `json:"mode"`
	Palette    map[string]string    `json:"palette"`
	Kinds      map[string]themeKind `json:"kinds"`
	Categories map[string]string    `json:"categories"`
	Effects    map[string]any       `json:"effects"`
}

type themeKind struct {
	Label string `json:"label"`
	Color string `json:"color"`
}

const (
	themesPrefix = "window.HearthlightThemes = "
	themesSuffix = ";\n"
)

var wantDarkPalette = map[string]string{
	"bg":           "#050a12",
	"ink":          "#e9f2fa",
	"muted":        "#7f8fa6",
	"accent":       "#5ee7ff",
	"line":         "rgba(151, 196, 225, 0.12)",
	"glass":        "rgba(9, 17, 29, 0.74)",
	"canvasTop":    "#0a1526",
	"canvasMiddle": "#060b14",
	"canvasBottom": "#03060c",
	"grid":         "rgba(120, 185, 230, 0.055)",
	"ambientGlow":  "rgba(70, 150, 190, 0.1)",
	"online":       "#5ee7a5",
	"warning":      "#ffd166",
	"danger":       "#ff7b72",
	"history":      "#8494a8",
	"vignette":     "rgba(0, 0, 0, 0.52)",
	"labelHalo":    "rgba(3, 6, 12, 0.86)",
	"bubble":       "rgba(6, 11, 20, 0.94)",
	"panel":        "rgba(12, 21, 35, 0.95)",
}

var wantKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#ffd166"},
	"server":   {Label: "Server", Color: "#5ee7ff"},
	"computer": {Label: "Computer", Color: "#8fb3ff"},
	"phone":    {Label: "Phone", Color: "#ff9ec7"},
	"nas":      {Label: "NAS", Color: "#5ee0a5"},
	"media":    {Label: "Media", Color: "#b18cff"},
	"iot":      {Label: "IoT", Color: "#f08fc8"},
	"printer":  {Label: "Printer", Color: "#ffb86b"},
	"device":   {Label: "Device", Color: "#94a3b8"},
}

var wantCategories = map[string]string{
	"web":    "#5ee7ff",
	"media":  "#b18cff",
	"data":   "#ffb454",
	"remote": "#5ee0a5",
	"infra":  "#6aa9ff",
	"iot":    "#f08fc8",
	"other":  "#93a4b8",
}

var wantDarkEffects = map[string]any{
	"vignette":       0.52,
	"deviceGlow":     0.22,
	"devicePulse":    0.3,
	"serviceGlow":    0.16,
	"territoryBlur":  26.0,
	"ribbonBlur":     16.0,
	"lensBlur":       14.0,
	"glassBlur":      16.0,
	"panelBlur":      22.0,
	"territoryBlend": "screen",
	"territoryAlpha": 1.0,
	"glowStrength":   1.0,
	"labelHalo":      1.0,
	"shadowStrength": 1.0,
}

var wantCatppuccinMochaPalette = map[string]string{
	"bg":           "#1e1e2e",
	"ink":          "#cdd6f4",
	"muted":        "#a6adc8",
	"accent":       "#89dceb",
	"line":         "rgba(205, 214, 244, 0.14)",
	"glass":        "rgba(30, 30, 46, 0.74)",
	"canvasTop":    "#313244",
	"canvasMiddle": "#181825",
	"canvasBottom": "#11111b",
	"grid":         "rgba(137, 180, 250, 0.06)",
	"ambientGlow":  "rgba(116, 199, 236, 0.12)",
	"online":       "#a6e3a1",
	"warning":      "#f9e2af",
	"danger":       "#f38ba8",
	"history":      "#7f849c",
	"vignette":     "rgba(17, 17, 27, 0.55)",
	"labelHalo":    "rgba(17, 17, 27, 0.88)",
	"bubble":       "rgba(24, 24, 37, 0.94)",
	"panel":        "rgba(30, 30, 46, 0.95)",
}

var wantCatppuccinMochaKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#f9e2af"},
	"server":   {Label: "Server", Color: "#89dceb"},
	"computer": {Label: "Computer", Color: "#89b4fa"},
	"phone":    {Label: "Phone", Color: "#f5c2e7"},
	"nas":      {Label: "NAS", Color: "#a6e3a1"},
	"media":    {Label: "Media", Color: "#cba6f7"},
	"iot":      {Label: "IoT", Color: "#94e2d5"},
	"printer":  {Label: "Printer", Color: "#fab387"},
	"device":   {Label: "Device", Color: "#9399b2"},
}

var wantCatppuccinMochaCategories = map[string]string{
	"web":    "#89dceb",
	"media":  "#cba6f7",
	"data":   "#fab387",
	"remote": "#a6e3a1",
	"infra":  "#89b4fa",
	"iot":    "#94e2d5",
	"other":  "#9399b2",
}

var wantCatppuccinMochaEffects = map[string]any{
	"vignette":       0.55,
	"deviceGlow":     0.22,
	"devicePulse":    0.3,
	"serviceGlow":    0.16,
	"territoryBlur":  26.0,
	"ribbonBlur":     16.0,
	"lensBlur":       14.0,
	"glassBlur":      16.0,
	"panelBlur":      22.0,
	"territoryBlend": "screen",
	"territoryAlpha": 1.0,
	"glowStrength":   1.0,
	"labelHalo":      1.0,
	"shadowStrength": 1.0,
}

var wantTokyoNightPalette = map[string]string{
	"bg":           "#1a1b26",
	"ink":          "#c0caf5",
	"muted":        "#a9b1d6",
	"accent":       "#7aa2f7",
	"line":         "rgba(192, 202, 245, 0.14)",
	"glass":        "rgba(26, 27, 38, 0.74)",
	"canvasTop":    "#292e42",
	"canvasMiddle": "#1a1b26",
	"canvasBottom": "#16161e",
	"grid":         "rgba(122, 162, 247, 0.06)",
	"ambientGlow":  "rgba(122, 162, 247, 0.12)",
	"online":       "#9ece6a",
	"warning":      "#e0af68",
	"danger":       "#f7768e",
	"history":      "#565f89",
	"vignette":     "rgba(22, 22, 30, 0.55)",
	"labelHalo":    "rgba(22, 22, 30, 0.88)",
	"bubble":       "rgba(26, 27, 38, 0.94)",
	"panel":        "rgba(26, 27, 38, 0.95)",
}

var wantTokyoNightKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#e0af68"},
	"server":   {Label: "Server", Color: "#7dcfff"},
	"computer": {Label: "Computer", Color: "#7aa2f7"},
	"phone":    {Label: "Phone", Color: "#ff007c"},
	"nas":      {Label: "NAS", Color: "#9ece6a"},
	"media":    {Label: "Media", Color: "#bb9af7"},
	"iot":      {Label: "IoT", Color: "#73daca"},
	"printer":  {Label: "Printer", Color: "#ff9e64"},
	"device":   {Label: "Device", Color: "#737aa2"},
}

var wantTokyoNightCategories = map[string]string{
	"web":    "#7dcfff",
	"media":  "#bb9af7",
	"data":   "#ff9e64",
	"remote": "#9ece6a",
	"infra":  "#7aa2f7",
	"iot":    "#73daca",
	"other":  "#737aa2",
}

var wantTokyoNightEffects = map[string]any{
	"vignette":       0.55,
	"deviceGlow":     0.22,
	"devicePulse":    0.3,
	"serviceGlow":    0.16,
	"territoryBlur":  26.0,
	"ribbonBlur":     16.0,
	"lensBlur":       14.0,
	"glassBlur":      16.0,
	"panelBlur":      22.0,
	"territoryBlend": "screen",
	"territoryAlpha": 1.0,
	"glowStrength":   1.0,
	"labelHalo":      1.0,
	"shadowStrength": 1.0,
}

var requiredPaletteTokens = []string{
	"bg", "ink", "muted", "accent", "line", "glass",
	"canvasTop", "canvasMiddle", "canvasBottom", "grid", "ambientGlow",
	"online", "warning", "danger", "history",
	"vignette", "labelHalo", "bubble", "panel",
}

var requiredStringEffects = []string{"territoryBlend"}

var requiredNumberEffects = []string{
	"vignette", "deviceGlow", "devicePulse", "serviceGlow",
	"territoryBlur", "ribbonBlur", "lensBlur", "glassBlur", "panelBlur",
	"territoryAlpha", "glowStrength", "labelHalo", "shadowStrength",
}

func readCatalog(t *testing.T) ([]byte, themeCatalog) {
	t.Helper()
	raw, err := os.ReadFile("themes.json")
	if err != nil {
		t.Fatalf("read themes.json: %v", err)
	}
	var catalog themeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatalf("parse themes.json: %v", err)
	}
	return raw, catalog
}

func TestThemeCatalog(t *testing.T) {
	_, catalog := readCatalog(t)

	if catalog.SchemaVersion != 1 {
		t.Fatalf("schemaVersion = %d, want 1", catalog.SchemaVersion)
	}
	if len(catalog.Families) == 0 {
		t.Fatal("themes.json has no families")
	}

	family := findThemeFamily(t, catalog, "hearthlight")
	if family.Name != "Hearthlight" {
		t.Fatalf("family = %+v, want Hearthlight", family)
	}
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	dark, light := family.Variants[0], family.Variants[1]
	if dark.ID != "hearthlight-dark" || dark.Name != "Hearthlight Dark" || dark.Mode != "dark" {
		t.Fatalf("variant = %+v, want Hearthlight Dark", dark)
	}
	if light.ID != "hearthlight-light" || light.Name != "Hearthlight Light" || light.Mode != "light" {
		t.Fatalf("variant = %+v, want Hearthlight Light", light)
	}

	if !reflect.DeepEqual(dark.Palette, wantDarkPalette) {
		t.Fatalf("palette = %#v, want %#v", dark.Palette, wantDarkPalette)
	}
	if !reflect.DeepEqual(dark.Kinds, wantKinds) {
		t.Fatalf("kinds = %#v, want %#v", dark.Kinds, wantKinds)
	}
	if !reflect.DeepEqual(dark.Categories, wantCategories) {
		t.Fatalf("categories = %#v, want %#v", dark.Categories, wantCategories)
	}
	if !reflect.DeepEqual(dark.Effects, wantDarkEffects) {
		t.Fatalf("effects = %#v, want %#v", dark.Effects, wantDarkEffects)
	}

	if len(light.Palette) != len(dark.Palette) {
		t.Fatalf("light palette has %d tokens, want %d", len(light.Palette), len(dark.Palette))
	}
	if len(light.Kinds) != len(dark.Kinds) {
		t.Fatalf("light kinds = %d, want %d", len(light.Kinds), len(dark.Kinds))
	}
	if len(light.Categories) != len(dark.Categories) {
		t.Fatalf("light categories = %d, want %d", len(light.Categories), len(dark.Categories))
	}
	if len(light.Effects) != len(dark.Effects) {
		t.Fatalf("light effects = %d, want %d", len(light.Effects), len(dark.Effects))
	}

	for _, variant := range family.Variants {
		for _, token := range requiredPaletteTokens {
			value, ok := variant.Palette[token]
			if !ok {
				t.Errorf("%s: missing palette token %q", variant.ID, token)
				continue
			}
			if strings.TrimSpace(value) == "" {
				t.Errorf("%s: palette token %q is empty", variant.ID, token)
			}
		}
		for name := range dark.Palette {
			if _, ok := variant.Palette[name]; !ok {
				t.Errorf("%s: missing palette token %q", variant.ID, name)
			}
		}
		for name, kind := range variant.Kinds {
			if strings.TrimSpace(kind.Label) == "" || strings.TrimSpace(kind.Color) == "" {
				t.Errorf("%s: kind %q has an empty label or colour", variant.ID, name)
			}
		}
		for name := range dark.Kinds {
			if _, ok := variant.Kinds[name]; !ok {
				t.Errorf("%s: missing kind %q", variant.ID, name)
			}
		}
		for name, colour := range variant.Categories {
			if strings.TrimSpace(colour) == "" {
				t.Errorf("%s: category %q has an empty colour", variant.ID, name)
			}
		}
		for name := range dark.Categories {
			if _, ok := variant.Categories[name]; !ok {
				t.Errorf("%s: missing category %q", variant.ID, name)
			}
		}
		for _, token := range requiredStringEffects {
			value, ok := variant.Effects[token]
			if !ok {
				t.Errorf("%s: missing effect token %q", variant.ID, token)
				continue
			}
			text, ok := value.(string)
			if !ok || strings.TrimSpace(text) == "" {
				t.Errorf("%s: effect token %q = %#v, want a non-empty string", variant.ID, token, value)
			}
		}
		for _, token := range requiredNumberEffects {
			value, ok := variant.Effects[token]
			if !ok {
				t.Errorf("%s: missing effect token %q", variant.ID, token)
				continue
			}
			number, ok := value.(float64)
			if !ok {
				t.Errorf("%s: effect token %q = %#v, want a number", variant.ID, token, value)
				continue
			}
			if math.IsNaN(number) || math.IsInf(number, 0) || number < 0 {
				t.Errorf("%s: effect token %q = %v, want a finite non-negative number", variant.ID, token, number)
			}
		}
		for name := range dark.Effects {
			if _, ok := variant.Effects[name]; !ok {
				t.Errorf("%s: missing effect token %q", variant.ID, name)
			}
		}
	}
}

func findThemeFamily(t *testing.T, catalog themeCatalog, id string) themeFamily {
	t.Helper()
	for _, family := range catalog.Families {
		if family.ID == id {
			return family
		}
	}
	t.Fatalf("themes.json has no family %q", id)
	return themeFamily{}
}

func TestCatppuccinMochaVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "catppuccin")
	if family.Name != "Catppuccin" {
		t.Fatalf("family name = %q, want Catppuccin", family.Name)
	}
	if family.Source.Name != "Catppuccin" {
		t.Errorf("source name = %q, want Catppuccin", family.Source.Name)
	}
	if !strings.HasPrefix(family.Source.URL, "https://") {
		t.Errorf("source url = %q, want an https:// URL", family.Source.URL)
	}
	if len(family.Variants) != 1 {
		t.Fatalf("variants = %d, want 1", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "catppuccin-mocha" || variant.Name != "Catppuccin Mocha" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Catppuccin Mocha dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantCatppuccinMochaPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantCatppuccinMochaPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantCatppuccinMochaKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantCatppuccinMochaKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantCatppuccinMochaCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantCatppuccinMochaCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantCatppuccinMochaEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantCatppuccinMochaEffects)
	}
}

func TestTokyoNightVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "tokyo-night")
	if family.Name != "Tokyo Night" {
		t.Fatalf("family name = %q, want Tokyo Night", family.Name)
	}
	if family.Source.Name != "Tokyo Night" {
		t.Errorf("source name = %q, want Tokyo Night", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/folke/tokyonight.nvim" {
		t.Errorf("source url = %q, want the Tokyo Night project URL", family.Source.URL)
	}
	if len(family.Variants) != 1 {
		t.Fatalf("variants = %d, want 1", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "tokyo-night" || variant.Name != "Tokyo Night" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Tokyo Night dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantTokyoNightPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantTokyoNightPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantTokyoNightKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantTokyoNightKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantTokyoNightCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantTokyoNightCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantTokyoNightEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantTokyoNightEffects)
	}
}

func TestDarkVariantsKeepMapSeparation(t *testing.T) {
	_, catalog := readCatalog(t)
	for _, family := range catalog.Families {
		for _, variant := range family.Variants {
			if variant.Mode != "dark" {
				continue
			}
			checkDistinctColours(t, variant.ID, "kind", kindColours(variant.Kinds))
			checkDistinctColours(t, variant.ID, "category", variant.Categories)

			warning := parseHexColour(t, variant.ID, "warning", variant.Palette["warning"])
			accent := parseHexColour(t, variant.ID, "accent", variant.Palette["accent"])
			media := parseHexColour(t, variant.ID, "media category", variant.Categories["media"])
			online := parseHexColour(t, variant.ID, "online", variant.Palette["online"])
			history := parseHexColour(t, variant.ID, "history", variant.Palette["history"])

			if hue := hueOf(warning); hue < 20 || hue > 70 {
				t.Errorf("%s: warning hue = %.0f, want a warm hue between 20 and 70", variant.ID, hue)
			}
			for name, colour := range map[string][3]float64{"accent": accent, "media category": media} {
				if hue := hueOf(colour); hue < 180 || hue > 290 {
					t.Errorf("%s: %s hue = %.0f, want a cool hue between 180 and 290", variant.ID, name, hue)
				}
			}
			if saturationOf(online) <= saturationOf(history) {
				t.Errorf("%s: online saturation = %.2f, history saturation = %.2f, want online more saturated",
					variant.ID, saturationOf(online), saturationOf(history))
			}
		}
	}
}

func kindColours(kinds map[string]themeKind) map[string]string {
	colours := make(map[string]string, len(kinds))
	for name, kind := range kinds {
		colours[name] = kind.Color
	}
	return colours
}

func checkDistinctColours(t *testing.T, variant, group string, colours map[string]string) {
	t.Helper()
	seen := make(map[string]string, len(colours))
	for name, colour := range colours {
		if other, ok := seen[colour]; ok {
			t.Errorf("%s: %s %q and %q share the colour %s", variant, group, other, name, colour)
		}
		seen[colour] = name
	}
}

func parseHexColour(t *testing.T, variant, token, value string) [3]float64 {
	t.Helper()
	colour, ok := hexColour(value)
	if !ok {
		t.Fatalf("%s: %s = %q, want a hex colour", variant, token, value)
	}
	return colour
}

func hueOf(rgb [3]float64) float64 {
	high, low := math.Max(rgb[0], math.Max(rgb[1], rgb[2])), math.Min(rgb[0], math.Min(rgb[1], rgb[2]))
	if high == low {
		return 0
	}
	delta := high - low
	hue := 0.0
	switch high {
	case rgb[0]:
		hue = math.Mod((rgb[1]-rgb[2])/delta, 6)
	case rgb[1]:
		hue = (rgb[2]-rgb[0])/delta + 2
	default:
		hue = (rgb[0]-rgb[1])/delta + 4
	}
	hue *= 60
	if hue < 0 {
		hue += 360
	}
	return hue
}

func saturationOf(rgb [3]float64) float64 {
	high, low := math.Max(rgb[0], math.Max(rgb[1], rgb[2])), math.Min(rgb[0], math.Min(rgb[1], rgb[2]))
	lightness := (high + low) / 2
	if high == low {
		return 0
	}
	delta := high - low
	if lightness > 0.5 {
		return delta / (2 - high - low)
	}
	return delta / (high + low)
}

func TestThemeIDsAreKebabCase(t *testing.T) {
	_, catalog := readCatalog(t)
	pattern := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	for _, family := range catalog.Families {
		checkThemeID(t, pattern, family.ID)
		for _, variant := range family.Variants {
			checkThemeID(t, pattern, variant.ID)
		}
	}
}

func checkThemeID(t *testing.T, pattern *regexp.Regexp, id string) {
	t.Helper()
	if !pattern.MatchString(id) {
		t.Errorf("theme id %q is not kebab-case", id)
	}
	seen := make(map[string]bool)
	for _, word := range strings.Split(id, "-") {
		if seen[word] {
			t.Errorf("theme id %q repeats the word %q", id, word)
		}
		seen[word] = true
	}
}

func TestLightPaletteContrast(t *testing.T) {
	_, catalog := readCatalog(t)
	family := catalog.Families[0]
	var light themeVariant
	for _, variant := range family.Variants {
		if variant.Mode == "light" {
			light = variant
			break
		}
	}
	if light.ID == "" {
		t.Fatal("themes.json has no light variant")
	}

	ink, ok := hexColour(light.Palette["ink"])
	if !ok {
		t.Fatalf("ink = %q, want a hex colour", light.Palette["ink"])
	}
	for _, token := range []string{"bg", "labelHalo"} {
		surface, ok := hexColour(light.Palette[token])
		if !ok {
			t.Errorf("%s = %q, want a hex colour", token, light.Palette[token])
			continue
		}
		if ratio := contrastRatio(ink, surface); ratio < 7 {
			t.Errorf("ink on %s contrast ratio = %.2f:1, want at least 7:1", token, ratio)
		}
	}
}

func hexColour(value string) (rgb [3]float64, ok bool) {
	if len(value) != 7 || value[0] != '#' {
		return rgb, false
	}
	for i := 0; i < 3; i++ {
		channel, err := strconv.ParseUint(value[1+i*2:3+i*2], 16, 8)
		if err != nil {
			return rgb, false
		}
		rgb[i] = float64(channel) / 255
	}
	return rgb, true
}

func relativeLuminance(rgb [3]float64) float64 {
	linear := func(channel float64) float64 {
		if channel <= 0.03928 {
			return channel / 12.92
		}
		return math.Pow((channel+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(rgb[0]) + 0.7152*linear(rgb[1]) + 0.0722*linear(rgb[2])
}

func contrastRatio(a, b [3]float64) float64 {
	high, low := relativeLuminance(a), relativeLuminance(b)
	if high < low {
		high, low = low, high
	}
	return (high + 0.05) / (low + 0.05)
}

func TestGeneratedThemesFileIsFresh(t *testing.T) {
	raw, _ := readCatalog(t)
	generated, err := os.ReadFile("themes.js")
	if err != nil {
		t.Fatalf("read themes.js: %v", err)
	}
	want := themesPrefix + string(raw) + themesSuffix
	if string(generated) != want {
		t.Fatal("web/themes.js is stale; run make themes")
	}
}

func TestThemeFilesAreEmbedded(t *testing.T) {
	for _, name := range []string{"themes.json", "themes.js"} {
		onDisk, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		embedded, err := Files.ReadFile(name)
		if err != nil {
			t.Fatalf("embedded %s: %v", name, err)
		}
		if string(embedded) != string(onDisk) {
			t.Fatalf("embedded %s does not match the checked-in file", name)
		}
	}
}
