package theme

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

const fullOmarchyTOML = `# Tokyo Night, pasted from a third-party theme
mode = "dark"
background = "#1a1b26"
dark_background = "#16161e"
lighter_background = "#292e42"
foreground = "#c0caf5"
dark_foreground = "#a9b1d6"
accent = "#7aa2f7"
green = "#9ece6a"
yellow = "#e0af68"
red = "#f7768e"
blue = "#7aa2f7"
magenta = "#bb9af7"
cursor = "#c0caf5"
selection_background = "#283457" # inline comment
unknown_array = [1, 2, 3]
`

func TestParseOmarchyFull(t *testing.T) {
	result, err := ParseOmarchy([]byte(fullOmarchyTOML))
	if err != nil {
		t.Fatalf("ParseOmarchy: %v", err)
	}
	if len(result.Missing) != 0 {
		t.Fatalf("missing = %v, want none", result.Missing)
	}
	wantIgnored := []string{"cursor", "selection_background", "unknown_array"}
	if !reflect.DeepEqual(result.Ignored, wantIgnored) {
		t.Fatalf("ignored = %v, want %v", result.Ignored, wantIgnored)
	}
	if len(result.Catalog.Families) != 1 {
		t.Fatalf("families = %d, want 1", len(result.Catalog.Families))
	}
	family := result.Catalog.Families[0]
	if !strings.HasPrefix(family.ID, "omarchy-") {
		t.Fatalf("family id = %q, want an omarchy- prefix", family.ID)
	}
	if family.Name != "Omarchy Import" {
		t.Fatalf("family name = %q", family.Name)
	}
	if len(family.Variants) != 1 {
		t.Fatalf("variants = %d, want 1", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != family.ID+"-dark" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want the dark variant", variant)
	}
	wantPalette := map[string]string{
		"bg":           "#1a1b26",
		"canvasMiddle": "#1a1b26",
		"canvasTop":    "#292e42",
		"canvasBottom": "#16161e",
		"ink":          "#c0caf5",
		"muted":        "#a9b1d6",
		"history":      "#a9b1d6",
		"accent":       "#7aa2f7",
		"online":       "#9ece6a",
		"warning":      "#e0af68",
		"danger":       "#f7768e",
		"line":         "rgba(192, 202, 245, 0.14)",
	}
	for token, want := range wantPalette {
		if variant.Palette[token] != want {
			t.Errorf("%s = %q, want %q", token, variant.Palette[token], want)
		}
	}
	for _, token := range RequiredPaletteTokens {
		if !validColor(variant.Palette[token]) {
			t.Errorf("%s = %q, want a valid colour", token, variant.Palette[token])
		}
	}
	if variant.Effects != defaultEffects("dark") {
		t.Errorf("effects = %+v, want the dark defaults", variant.Effects)
	}
	if variant.Kinds["router"].Color != "#e0af68" {
		t.Errorf("router kind = %+v, want the yellow warning colour", variant.Kinds["router"])
	}
	if variant.Kinds["server"].Color != "#7aa2f7" {
		t.Errorf("server kind = %+v, want the accent", variant.Kinds["server"])
	}
	if variant.Categories["remote"] != "#9ece6a" {
		t.Errorf("remote category = %q, want green", variant.Categories["remote"])
	}
}

func TestParseOmarchyMissingKeys(t *testing.T) {
	input := "background = \"#101010\"\nforeground = \"#f0f0f0\"\n"
	result, err := ParseOmarchy([]byte(input))
	if err != nil {
		t.Fatalf("ParseOmarchy: %v", err)
	}
	wantMissing := []string{"accent", "green", "yellow", "red", "blue", "magenta", "mode"}
	if !reflect.DeepEqual(result.Missing, wantMissing) {
		t.Fatalf("missing = %v, want %v", result.Missing, wantMissing)
	}
	variant := result.Catalog.Families[0].Variants[0]
	if variant.Mode != "dark" {
		t.Fatalf("mode = %q, want inferred dark", variant.Mode)
	}
	if variant.Palette["bg"] != "#101010" || variant.Palette["ink"] != "#f0f0f0" {
		t.Fatalf("palette = %+v, want the supplied background and foreground", variant.Palette)
	}
	for _, token := range RequiredPaletteTokens {
		if !validColor(variant.Palette[token]) {
			t.Errorf("%s = %q, want a derived valid colour", token, variant.Palette[token])
		}
	}
	if variant.Palette["accent"] != defaultPalette("dark")["accent"] {
		t.Errorf("accent = %q, want the Hearthlight fallback", variant.Palette["accent"])
	}
}

func TestParseOmarchyLight(t *testing.T) {
	input := `mode = "light"
background = "#ffffff"
foreground = "#000000"
accent = "#205ea6"
green = "#0a7448"
yellow = "#8f5c00"
red = "#b02a22"
blue = "#3264eb"
cyan = "#007a8a"
magenta = "#8a2be2"
`
	result, err := ParseOmarchy([]byte(input))
	if err != nil {
		t.Fatalf("ParseOmarchy: %v", err)
	}
	variant := result.Catalog.Families[0].Variants[0]
	if variant.Mode != "light" {
		t.Fatalf("mode = %q, want light", variant.Mode)
	}
	if variant.Effects != defaultEffects("light") {
		t.Fatalf("effects = %+v, want the light defaults", variant.Effects)
	}
	if variant.Effects.Vignette != 0.3 || variant.Effects.TerritoryBlend != "multiply" {
		t.Fatalf("effects = %+v, want light rendering defaults", variant.Effects)
	}
	if variant.Palette["bg"] != "#ffffff" || variant.Palette["ink"] != "#000000" {
		t.Fatalf("palette = %+v, want the supplied light colours", variant.Palette)
	}
}

func TestParseOmarchyIgnoresNestedTables(t *testing.T) {
	input := `background = "#101010"
foreground = "#f0f0f0"
[colors]
color0 = "#000000"
color1 = "#ffffff"
`
	result, err := ParseOmarchy([]byte(input))
	if err != nil {
		t.Fatalf("ParseOmarchy: %v", err)
	}
	if len(result.Ignored) != 0 {
		t.Fatalf("ignored = %v, want keys inside a nested table to be skipped", result.Ignored)
	}
	if len(result.Catalog.Families) != 1 {
		t.Fatal("nested table import did not produce a family")
	}
}

func TestParseOmarchyDeterministicIDs(t *testing.T) {
	first, err := ParseOmarchy([]byte(fullOmarchyTOML))
	if err != nil {
		t.Fatalf("first parse: %v", err)
	}
	second, err := ParseOmarchy([]byte(fullOmarchyTOML))
	if err != nil {
		t.Fatalf("second parse: %v", err)
	}
	if first.Catalog.Families[0].ID != second.Catalog.Families[0].ID {
		t.Fatalf("ids differ across parses: %q and %q", first.Catalog.Families[0].ID, second.Catalog.Families[0].ID)
	}
	commented, err := ParseOmarchy([]byte(strings.Replace(fullOmarchyTOML, "# Tokyo Night, pasted from a third-party theme", "# a different cosmetic comment", 1)))
	if err != nil {
		t.Fatalf("commented parse: %v", err)
	}
	if first.Catalog.Families[0].ID != commented.Catalog.Families[0].ID {
		t.Fatalf("comment-only edit changed the id from %q to %q", first.Catalog.Families[0].ID, commented.Catalog.Families[0].ID)
	}
	other, err := ParseOmarchy([]byte(strings.Replace(fullOmarchyTOML, `background = "#1a1b26"`, `background = "#202020"`, 1)))
	if err != nil {
		t.Fatalf("other parse: %v", err)
	}
	if first.Catalog.Families[0].ID == other.Catalog.Families[0].ID {
		t.Fatal("different palettes produced the same family id")
	}
}

func TestParseOmarchyRejects(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		want ErrorCategory
	}{
		{name: "oversized", raw: bytes.Repeat([]byte("#"), MaxImportBytes+1), want: ErrTooLarge},
		{name: "only comments", raw: []byte("# nothing here\n"), want: ErrMalformed},
		{name: "missing equals", raw: []byte("background \"#fff\"\n"), want: ErrMalformed},
		{name: "unterminated table", raw: []byte("[colors\nbackground = \"#fff\"\n"), want: ErrMalformed},
		{name: "unquoted value", raw: []byte("background = #fff\n"), want: ErrMalformed},
		{name: "unclosed quote", raw: []byte("background = \"#fff\n"), want: ErrMalformed},
		{name: "bad colour", raw: []byte("background = \"red\"\n"), want: ErrInvalidColor},
		{name: "short hex", raw: []byte("background = \"#12345\"\n"), want: ErrInvalidColor},
		{name: "bad mode", raw: []byte("mode = \"dim\"\nbackground = \"#fff\"\n"), want: ErrInvalidMode},
		{name: "forbidden key", raw: []byte("__proto__ = \"#fff\"\nbackground = \"#fff\"\n"), want: ErrForbiddenKey},
		{name: "quoted forbidden key", raw: []byte("\"__proto__\" = \"#fff\"\nbackground = \"#fff\"\n"), want: ErrForbiddenKey},
		{name: "constructor key", raw: []byte("constructor = \"#fff\"\nbackground = \"#fff\"\n"), want: ErrForbiddenKey},
		{name: "prototype key", raw: []byte("prototype = \"#fff\"\nbackground = \"#fff\"\n"), want: ErrForbiddenKey},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseOmarchy(test.raw)
			requireCategory(t, err, test.want)
		})
	}
}
