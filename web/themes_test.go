package web

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
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
	Variants []themeVariant `json:"variants"`
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
	if len(catalog.Families) != 1 {
		t.Fatalf("families = %d, want 1", len(catalog.Families))
	}
	family := catalog.Families[0]
	if family.ID != "hearthlight" || family.Name != "Hearthlight" {
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
