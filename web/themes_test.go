package web

import (
	"encoding/json"
	"os"
	"reflect"
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
	Effects    map[string]float64   `json:"effects"`
}

type themeKind struct {
	Label string `json:"label"`
	Color string `json:"color"`
}

const (
	themesPrefix = "window.HearthlightThemes = "
	themesSuffix = ";\n"
)

var wantPalette = map[string]string{
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

var wantEffects = map[string]float64{
	"vignette":      0.52,
	"deviceGlow":    0.22,
	"devicePulse":   0.3,
	"serviceGlow":   0.16,
	"territoryBlur": 26,
	"ribbonBlur":    16,
	"lensBlur":      14,
	"glassBlur":     16,
	"panelBlur":     22,
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
	if len(family.Variants) != 1 {
		t.Fatalf("variants = %d, want 1", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "hearthlight-dark" || variant.Name != "Hearthlight Dark" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Hearthlight Dark", variant)
	}

	if !reflect.DeepEqual(variant.Palette, wantPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantEffects)
	}

	for token, value := range variant.Palette {
		if strings.TrimSpace(value) == "" {
			t.Errorf("palette token %q is empty", token)
		}
	}
	for name, kind := range variant.Kinds {
		if strings.TrimSpace(kind.Label) == "" || strings.TrimSpace(kind.Color) == "" {
			t.Errorf("kind %q has an empty label or colour", name)
		}
	}
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
