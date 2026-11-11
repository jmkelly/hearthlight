package web

// Omarchy A covers the eight stock themes assigned to this branch. The
// expected palettes pin the mapping decisions recorded in each source note.

import (
	"reflect"
	"strings"
	"testing"
)

var wantOmarchyDarkEffects = map[string]any{
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

var wantEtherealPalette = map[string]string{
	"bg":           "#060B1E",
	"ink":          "#ffcead",
	"muted":        "#919dd6",
	"accent":       "#7d82d9",
	"line":         "rgba(255, 206, 173, 0.14)",
	"glass":        "rgba(6, 11, 30, 0.74)",
	"canvasTop":    "#131a3a",
	"canvasMiddle": "#060B1E",
	"canvasBottom": "#030610",
	"grid":         "rgba(125, 130, 217, 0.06)",
	"ambientGlow":  "rgba(125, 130, 217, 0.12)",
	"online":       "#8fd6a4",
	"warning":      "#E9BB4F",
	"danger":       "#ED5B5A",
	"history":      "#7f8da8",
	"vignette":     "rgba(3, 6, 16, 0.55)",
	"labelHalo":    "rgba(3, 6, 16, 0.88)",
	"bubble":       "rgba(6, 11, 30, 0.94)",
	"panel":        "rgba(6, 11, 30, 0.95)",
}

var wantEtherealKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#E9BB4F"},
	"server":   {Label: "Server", Color: "#7d82d9"},
	"computer": {Label: "Computer", Color: "#a3bfd1"},
	"phone":    {Label: "Phone", Color: "#c89dc1"},
	"nas":      {Label: "NAS", Color: "#92a593"},
	"media":    {Label: "Media", Color: "#c2c4f0"},
	"iot":      {Label: "IoT", Color: "#dfeaf0"},
	"printer":  {Label: "Printer", Color: "#eb8b54"},
	"device":   {Label: "Device", Color: "#c9b8a6"},
}

var wantEtherealCategories = map[string]string{
	"web":    "#a3bfd1",
	"media":  "#c2c4f0",
	"data":   "#eb8b54",
	"remote": "#92a593",
	"infra":  "#7d82d9",
	"iot":    "#c89dc1",
	"other":  "#c9b8a6",
}

func TestEtherealVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "ethereal")
	if family.Name != "Ethereal" {
		t.Fatalf("family name = %q, want Ethereal", family.Name)
	}
	if family.Source.Name != "Omarchy" {
		t.Errorf("source name = %q, want Omarchy", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/ethereal/colors.toml" {
		t.Errorf("source url = %q, want the pinned Omarchy colors.toml URL", family.Source.URL)
	}
	if !strings.Contains(family.Source.Note, "748fa3c") {
		t.Error("source note does not record the pinned Omarchy commit")
	}
	if len(family.Variants) != 1 {
		t.Fatalf("variants = %d, want 1", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "ethereal" || variant.Name != "Ethereal" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Ethereal dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantEtherealPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantEtherealPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantEtherealKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantEtherealKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantEtherealCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantEtherealCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantOmarchyDarkEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantOmarchyDarkEffects)
	}
}

var wantHackermanPalette = map[string]string{
	"bg":           "#0B0C16",
	"ink":          "#ddf7ff",
	"muted":        "#8f99c4",
	"accent":       "#829dd4",
	"line":         "rgba(221, 247, 255, 0.14)",
	"glass":        "rgba(11, 12, 22, 0.74)",
	"canvasTop":    "#151828",
	"canvasMiddle": "#0B0C16",
	"canvasBottom": "#06060c",
	"grid":         "rgba(130, 157, 212, 0.06)",
	"ambientGlow":  "rgba(130, 157, 212, 0.12)",
	"online":       "#82FB9C",
	"warning":      "#e0b36a",
	"danger":       "#ff7a7a",
	"history":      "#7c85ad",
	"vignette":     "rgba(6, 6, 12, 0.55)",
	"labelHalo":    "rgba(6, 6, 12, 0.88)",
	"bubble":       "rgba(11, 12, 22, 0.94)",
	"panel":        "rgba(11, 12, 22, 0.95)",
}

var wantHackermanKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#e0b36a"},
	"server":   {Label: "Server", Color: "#829dd4"},
	"computer": {Label: "Computer", Color: "#86a7df"},
	"phone":    {Label: "Phone", Color: "#cddbf4"},
	"nas":      {Label: "NAS", Color: "#82FB9C"},
	"media":    {Label: "Media", Color: "#a4ffec"},
	"iot":      {Label: "IoT", Color: "#7cf8f7"},
	"printer":  {Label: "Printer", Color: "#50f7a3"},
	"device":   {Label: "Device", Color: "#8f99c4"},
}

var wantHackermanCategories = map[string]string{
	"web":    "#7cf8f7",
	"media":  "#86a7df",
	"data":   "#50f7a3",
	"remote": "#82FB9C",
	"infra":  "#829dd4",
	"iot":    "#cddbf4",
	"other":  "#8f99c4",
}

func TestHackermanVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "hackerman")
	if family.Name != "Hackerman" {
		t.Fatalf("family name = %q, want Hackerman", family.Name)
	}
	if family.Source.Name != "Omarchy" {
		t.Errorf("source name = %q, want Omarchy", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/hackerman/colors.toml" {
		t.Errorf("source url = %q, want the pinned Omarchy colors.toml URL", family.Source.URL)
	}
	if !strings.Contains(family.Source.Note, "748fa3c") {
		t.Error("source note does not record the pinned Omarchy commit")
	}
	if len(family.Variants) != 1 {
		t.Fatalf("variants = %d, want 1", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "hackerman" || variant.Name != "Hackerman" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Hackerman dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantHackermanPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantHackermanPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantHackermanKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantHackermanKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantHackermanCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantHackermanCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantOmarchyDarkEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantOmarchyDarkEffects)
	}
}

var wantJadeHorsePalette = map[string]string{
	"bg":           "#142620",
	"ink":          "#dfede5",
	"muted":        "#93bfae",
	"accent":       "#96b7d0",
	"line":         "rgba(223, 237, 229, 0.14)",
	"glass":        "rgba(20, 38, 32, 0.74)",
	"canvasTop":    "#203b31",
	"canvasMiddle": "#142620",
	"canvasBottom": "#0a1713",
	"grid":         "rgba(150, 183, 208, 0.06)",
	"ambientGlow":  "rgba(150, 183, 208, 0.12)",
	"online":       "#8bcbb1",
	"warning":      "#c5cc91",
	"danger":       "#d88d91",
	"history":      "#91ad9f",
	"vignette":     "rgba(10, 23, 19, 0.55)",
	"labelHalo":    "rgba(10, 23, 19, 0.88)",
	"bubble":       "rgba(20, 38, 32, 0.94)",
	"panel":        "rgba(20, 38, 32, 0.95)",
}

var wantJadeHorseKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#c5cc91"},
	"server":   {Label: "Server", Color: "#96b7d0"},
	"computer": {Label: "Computer", Color: "#86c7c5"},
	"phone":    {Label: "Phone", Color: "#b7a1c8"},
	"nas":      {Label: "NAS", Color: "#8bcbb1"},
	"media":    {Label: "Media", Color: "#cfbce0"},
	"iot":      {Label: "IoT", Color: "#a2dcda"},
	"printer":  {Label: "Printer", Color: "#d2ab8d"},
	"device":   {Label: "Device", Color: "#91ad9f"},
}

var wantJadeHorseCategories = map[string]string{
	"web":    "#86c7c5",
	"media":  "#b7a1c8",
	"data":   "#d2ab8d",
	"remote": "#8bcbb1",
	"infra":  "#96b7d0",
	"iot":    "#a2dcda",
	"other":  "#91ad9f",
}

func TestJadeHorseVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "jade-horse")
	if family.Name != "Jade Horse" {
		t.Fatalf("family name = %q, want Jade Horse", family.Name)
	}
	if family.Source.Name != "Omarchy" {
		t.Errorf("source name = %q, want Omarchy", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/jade-horse/colors.toml" {
		t.Errorf("source url = %q, want the pinned Omarchy colors.toml URL", family.Source.URL)
	}
	if !strings.Contains(family.Source.Note, "748fa3c") {
		t.Error("source note does not record the pinned Omarchy commit")
	}
	if len(family.Variants) != 1 {
		t.Fatalf("variants = %d, want 1", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "jade-horse" || variant.Name != "Jade Horse" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Jade Horse dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantJadeHorsePalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantJadeHorsePalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantJadeHorseKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantJadeHorseKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantJadeHorseCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantJadeHorseCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantOmarchyDarkEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantOmarchyDarkEffects)
	}
}

var wantLastHorizonPalette = map[string]string{
	"bg":           "#0c0b0c",
	"ink":          "#FAFCFB",
	"muted":        "#a89da1",
	"accent":       "#a5a0b6",
	"line":         "rgba(250, 252, 251, 0.14)",
	"glass":        "rgba(12, 11, 12, 0.74)",
	"canvasTop":    "#0c0b0c",
	"canvasMiddle": "#0c0b0c",
	"canvasBottom": "#060606",
	"grid":         "rgba(165, 160, 182, 0.06)",
	"ambientGlow":  "rgba(165, 160, 182, 0.12)",
	"online":       "#87a9b0",
	"warning":      "#d1a36f",
	"danger":       "#c38b7b",
	"history":      "#6f6a70",
	"vignette":     "rgba(6, 6, 6, 0.55)",
	"labelHalo":    "rgba(6, 6, 6, 0.88)",
	"bubble":       "rgba(12, 11, 12, 0.94)",
	"panel":        "rgba(12, 11, 12, 0.95)",
}

var wantLastHorizonKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#d1a36f"},
	"server":   {Label: "Server", Color: "#a5a0b6"},
	"computer": {Label: "Computer", Color: "#87a9b0"},
	"phone":    {Label: "Phone", Color: "#c4d8e2"},
	"nas":      {Label: "NAS", Color: "#9fc9b4"},
	"media":    {Label: "Media", Color: "#c9b8d4"},
	"iot":      {Label: "IoT", Color: "#b8d8db"},
	"printer":  {Label: "Printer", Color: "#d8b08f"},
	"device":   {Label: "Device", Color: "#9b9499"},
}

var wantLastHorizonCategories = map[string]string{
	"web":    "#87a9b0",
	"media":  "#a5a0b6",
	"data":   "#d1a36f",
	"remote": "#9fc9b4",
	"infra":  "#b59790",
	"iot":    "#c4d8e2",
	"other":  "#9b9499",
}

func TestLastHorizonVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "last-horizon")
	if family.Name != "Last Horizon" {
		t.Fatalf("family name = %q, want Last Horizon", family.Name)
	}
	if family.Source.Name != "Omarchy" {
		t.Errorf("source name = %q, want Omarchy", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/last-horizon/colors.toml" {
		t.Errorf("source url = %q, want the pinned Omarchy colors.toml URL", family.Source.URL)
	}
	if !strings.Contains(family.Source.Note, "748fa3c") {
		t.Error("source note does not record the pinned Omarchy commit")
	}
	if len(family.Variants) != 1 {
		t.Fatalf("variants = %d, want 1", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "last-horizon" || variant.Name != "Last Horizon" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Last Horizon dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantLastHorizonPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantLastHorizonPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantLastHorizonKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantLastHorizonKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantLastHorizonCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantLastHorizonCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantOmarchyDarkEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantOmarchyDarkEffects)
	}
}

var wantLumonPalette = map[string]string{
	"bg":           "#16242d",
	"ink":          "#d6e2ee",
	"muted":        "#8fb0cc",
	"accent":       "#8bc9eb",
	"line":         "rgba(214, 226, 238, 0.14)",
	"glass":        "rgba(22, 36, 45, 0.74)",
	"canvasTop":    "#1b2d40",
	"canvasMiddle": "#16242d",
	"canvasBottom": "#0b1216",
	"grid":         "rgba(139, 201, 235, 0.06)",
	"ambientGlow":  "rgba(139, 201, 235, 0.12)",
	"online":       "#7fd1b0",
	"warning":      "#d9b36a",
	"danger":       "#c96b6b",
	"history":      "#6f86a0",
	"vignette":     "rgba(11, 18, 22, 0.55)",
	"labelHalo":    "rgba(11, 18, 22, 0.88)",
	"bubble":       "rgba(22, 36, 45, 0.94)",
	"panel":        "rgba(22, 36, 45, 0.95)",
}

var wantLumonKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#d9b36a"},
	"server":   {Label: "Server", Color: "#8bc9eb"},
	"computer": {Label: "Computer", Color: "#6fb8e3"},
	"phone":    {Label: "Phone", Color: "#b1d8ee"},
	"nas":      {Label: "NAS", Color: "#7fd1b0"},
	"media":    {Label: "Media", Color: "#b4e4f6"},
	"iot":      {Label: "IoT", Color: "#d1eef8"},
	"printer":  {Label: "Printer", Color: "#73a6cb"},
	"device":   {Label: "Device", Color: "#8fb0cc"},
}

var wantLumonCategories = map[string]string{
	"web":    "#8bc9eb",
	"media":  "#6fb8e3",
	"data":   "#d9b36a",
	"remote": "#7fd1b0",
	"infra":  "#73a6cb",
	"iot":    "#b1d8ee",
	"other":  "#8fb0cc",
}

func TestLumonVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "lumon")
	if family.Name != "Lumon" {
		t.Fatalf("family name = %q, want Lumon", family.Name)
	}
	if family.Source.Name != "Omarchy" {
		t.Errorf("source name = %q, want Omarchy", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/lumon/colors.toml" {
		t.Errorf("source url = %q, want the pinned Omarchy colors.toml URL", family.Source.URL)
	}
	if !strings.Contains(family.Source.Note, "748fa3c") {
		t.Error("source note does not record the pinned Omarchy commit")
	}
	if len(family.Variants) != 1 {
		t.Fatalf("variants = %d, want 1", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "lumon" || variant.Name != "Lumon" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Lumon dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantLumonPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantLumonPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantLumonKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantLumonKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantLumonCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantLumonCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantOmarchyDarkEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantOmarchyDarkEffects)
	}
}

var wantMatteBlackPalette = map[string]string{
	"bg":           "#121212",
	"ink":          "#bebebe",
	"muted":        "#9e9e9e",
	"accent":       "#6da8d8",
	"line":         "rgba(190, 190, 190, 0.14)",
	"glass":        "rgba(18, 18, 18, 0.74)",
	"canvasTop":    "#1e1e1e",
	"canvasMiddle": "#121212",
	"canvasBottom": "#090909",
	"grid":         "rgba(109, 168, 216, 0.06)",
	"ambientGlow":  "rgba(109, 168, 216, 0.12)",
	"online":       "#7fc97f",
	"warning":      "#e68e0d",
	"danger":       "#D35F5F",
	"history":      "#8a8a8d",
	"vignette":     "rgba(9, 9, 9, 0.55)",
	"labelHalo":    "rgba(9, 9, 9, 0.88)",
	"bubble":       "rgba(18, 18, 18, 0.94)",
	"panel":        "rgba(18, 18, 18, 0.95)",
}

var wantMatteBlackKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#e68e0d"},
	"server":   {Label: "Server", Color: "#6da8d8"},
	"computer": {Label: "Computer", Color: "#8ab8de"},
	"phone":    {Label: "Phone", Color: "#d38fa8"},
	"nas":      {Label: "NAS", Color: "#7fc97f"},
	"media":    {Label: "Media", Color: "#b79ad8"},
	"iot":      {Label: "IoT", Color: "#8fd8c8"},
	"printer":  {Label: "Printer", Color: "#c63d3d"},
	"device":   {Label: "Device", Color: "#9e9e9e"},
}

var wantMatteBlackCategories = map[string]string{
	"web":    "#6da8d8",
	"media":  "#b79ad8",
	"data":   "#c63d3d",
	"remote": "#7fc97f",
	"infra":  "#8ab8de",
	"iot":    "#8fd8c8",
	"other":  "#9e9e9e",
}

func TestMatteBlackVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "matte-black")
	if family.Name != "Matte Black" {
		t.Fatalf("family name = %q, want Matte Black", family.Name)
	}
	if family.Source.Name != "Omarchy" {
		t.Errorf("source name = %q, want Omarchy", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/matte-black/colors.toml" {
		t.Errorf("source url = %q, want the pinned Omarchy colors.toml URL", family.Source.URL)
	}
	if !strings.Contains(family.Source.Note, "748fa3c") {
		t.Error("source note does not record the pinned Omarchy commit")
	}
	if len(family.Variants) != 1 {
		t.Fatalf("variants = %d, want 1", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "matte-black" || variant.Name != "Matte Black" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Matte Black dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantMatteBlackPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantMatteBlackPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantMatteBlackKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantMatteBlackKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantMatteBlackCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantMatteBlackCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantOmarchyDarkEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantOmarchyDarkEffects)
	}
}

var wantMiasmaPalette = map[string]string{
	"bg":           "#222222",
	"ink":          "#c2c2b0",
	"muted":        "#a8a894",
	"accent":       "#7f93b3",
	"line":         "rgba(194, 194, 176, 0.14)",
	"glass":        "rgba(34, 34, 34, 0.74)",
	"canvasTop":    "#2c2c2c",
	"canvasMiddle": "#222222",
	"canvasBottom": "#121212",
	"grid":         "rgba(127, 147, 179, 0.06)",
	"ambientGlow":  "rgba(127, 147, 179, 0.12)",
	"online":       "#5f875f",
	"warning":      "#b36d43",
	"danger":       "#b35a4a",
	"history":      "#85857a",
	"vignette":     "rgba(18, 18, 18, 0.55)",
	"labelHalo":    "rgba(18, 18, 18, 0.88)",
	"bubble":       "rgba(34, 34, 34, 0.94)",
	"panel":        "rgba(34, 34, 34, 0.95)",
}

var wantMiasmaKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#c9a554"},
	"server":   {Label: "Server", Color: "#7f93b3"},
	"computer": {Label: "Computer", Color: "#9fb3c9"},
	"phone":    {Label: "Phone", Color: "#bb8a6a"},
	"nas":      {Label: "NAS", Color: "#5f875f"},
	"media":    {Label: "Media", Color: "#a58fc0"},
	"iot":      {Label: "IoT", Color: "#7fae9a"},
	"printer":  {Label: "Printer", Color: "#b36d43"},
	"device":   {Label: "Device", Color: "#a8a894"},
}

var wantMiasmaCategories = map[string]string{
	"web":    "#9fb3c9",
	"media":  "#a58fc0",
	"data":   "#b36d43",
	"remote": "#5f875f",
	"infra":  "#7f93b3",
	"iot":    "#7fae9a",
	"other":  "#a8a894",
}

func TestMiasmaVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "miasma")
	if family.Name != "Miasma" {
		t.Fatalf("family name = %q, want Miasma", family.Name)
	}
	if family.Source.Name != "Omarchy" {
		t.Errorf("source name = %q, want Omarchy", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/miasma/colors.toml" {
		t.Errorf("source url = %q, want the pinned Omarchy colors.toml URL", family.Source.URL)
	}
	if !strings.Contains(family.Source.Note, "748fa3c") {
		t.Error("source note does not record the pinned Omarchy commit")
	}
	if len(family.Variants) != 1 {
		t.Fatalf("variants = %d, want 1", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "miasma" || variant.Name != "Miasma" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Miasma dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantMiasmaPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantMiasmaPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantMiasmaKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantMiasmaKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantMiasmaCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantMiasmaCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantOmarchyDarkEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantOmarchyDarkEffects)
	}
}

var wantRetro82Palette = map[string]string{
	"bg":           "#05182e",
	"ink":          "#f6dcac",
	"muted":        "#7fb8c2",
	"accent":       "#028391",
	"line":         "rgba(246, 220, 172, 0.14)",
	"glass":        "rgba(5, 24, 46, 0.74)",
	"canvasTop":    "#0a2540",
	"canvasMiddle": "#05182e",
	"canvasBottom": "#020c17",
	"grid":         "rgba(2, 131, 145, 0.06)",
	"ambientGlow":  "rgba(2, 131, 145, 0.12)",
	"online":       "#8cbfb8",
	"warning":      "#faa968",
	"danger":       "#f85525",
	"history":      "#7f98a0",
	"vignette":     "rgba(2, 12, 23, 0.55)",
	"labelHalo":    "rgba(2, 12, 23, 0.88)",
	"bubble":       "rgba(5, 24, 46, 0.94)",
	"panel":        "rgba(5, 24, 46, 0.95)",
}

var wantRetro82Kinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#faa968"},
	"server":   {Label: "Server", Color: "#028391"},
	"computer": {Label: "Computer", Color: "#3f8f8a"},
	"phone":    {Label: "Phone", Color: "#8cbfb8"},
	"nas":      {Label: "NAS", Color: "#5fc7a4"},
	"media":    {Label: "Media", Color: "#a8c4d8"},
	"iot":      {Label: "IoT", Color: "#6fd1b8"},
	"printer":  {Label: "Printer", Color: "#e97b3c"},
	"device":   {Label: "Device", Color: "#7f98a0"},
}

var wantRetro82Categories = map[string]string{
	"web":    "#8cbfb8",
	"media":  "#a8c4d8",
	"data":   "#e97b3c",
	"remote": "#6fd1b8",
	"infra":  "#3f8f8a",
	"iot":    "#028391",
	"other":  "#7f98a0",
}

func TestRetro82Variant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "retro-82")
	if family.Name != "Retro 82" {
		t.Fatalf("family name = %q, want Retro 82", family.Name)
	}
	if family.Source.Name != "Omarchy" {
		t.Errorf("source name = %q, want Omarchy", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/retro-82/colors.toml" {
		t.Errorf("source url = %q, want the pinned Omarchy colors.toml URL", family.Source.URL)
	}
	if !strings.Contains(family.Source.Note, "748fa3c") {
		t.Error("source note does not record the pinned Omarchy commit")
	}
	if len(family.Variants) != 1 {
		t.Fatalf("variants = %d, want 1", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "retro-82" || variant.Name != "Retro 82" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Retro 82 dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantRetro82Palette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantRetro82Palette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantRetro82Kinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantRetro82Kinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantRetro82Categories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantRetro82Categories)
	}
	if !reflect.DeepEqual(variant.Effects, wantOmarchyDarkEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantOmarchyDarkEffects)
	}
}
