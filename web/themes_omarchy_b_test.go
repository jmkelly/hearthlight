package web

import (
	"reflect"
	"strings"
	"testing"
)

var wantFlexokiLightPalette = map[string]string{
	"bg":           "#FFFCF0",
	"ink":          "#100F0F",
	"muted":        "#6F6D64",
	"accent":       "#205EA6",
	"line":         "rgba(16, 15, 15, 0.16)",
	"glass":        "rgba(255, 252, 240, 0.82)",
	"canvasTop":    "#FFFEF8",
	"canvasMiddle": "#FFFCF0",
	"canvasBottom": "#E5E2D8",
	"grid":         "rgba(32, 94, 166, 0.07)",
	"ambientGlow":  "rgba(32, 94, 166, 0.12)",
	"online":       "#6E7D2E",
	"warning":      "#8F6E0E",
	"danger":       "#D14D41",
	"history":      "#7C7A72",
	"vignette":     "rgba(229, 226, 216, 0.55)",
	"labelHalo":    "#FFFEF8",
	"bubble":       "rgba(255, 254, 248, 0.94)",
	"panel":        "rgba(255, 252, 240, 0.97)",
}

var wantFlexokiLightKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#8F6E0E"},
	"server":   {Label: "Server", Color: "#205EA6"},
	"computer": {Label: "Computer", Color: "#2F6FA8"},
	"phone":    {Label: "Phone", Color: "#C7508D"},
	"nas":      {Label: "NAS", Color: "#6E7D2E"},
	"media":    {Label: "Media", Color: "#6F4FA0"},
	"iot":      {Label: "IoT", Color: "#2B8A81"},
	"printer":  {Label: "Printer", Color: "#B35F17"},
	"device":   {Label: "Device", Color: "#7C7A72"},
}

var wantFlexokiLightCategories = map[string]string{
	"web":    "#205EA6",
	"media":  "#6F4FA0",
	"data":   "#B35F17",
	"remote": "#6E7D2E",
	"infra":  "#2F6FA8",
	"iot":    "#2B8A81",
	"other":  "#7C7A72",
}

var wantLupinePalette = map[string]string{
	"bg":           "#FAFAFA",
	"ink":          "#212121",
	"muted":        "#656565",
	"accent":       "#3264EB",
	"line":         "rgba(33, 33, 33, 0.16)",
	"glass":        "rgba(250, 250, 250, 0.82)",
	"canvasTop":    "#FFFFFF",
	"canvasMiddle": "#FAFAFA",
	"canvasBottom": "#ECECEC",
	"grid":         "rgba(50, 100, 235, 0.07)",
	"ambientGlow":  "rgba(50, 100, 235, 0.12)",
	"online":       "#4A2FD0",
	"warning":      "#8F5410",
	"danger":       "#C900C4",
	"history":      "#747474",
	"vignette":     "rgba(236, 236, 236, 0.55)",
	"labelHalo":    "#FFFFFF",
	"bubble":       "rgba(255, 255, 255, 0.94)",
	"panel":        "rgba(250, 250, 250, 0.97)",
}

var wantLupineKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#8F5410"},
	"server":   {Label: "Server", Color: "#3264EB"},
	"computer": {Label: "Computer", Color: "#013A6F"},
	"phone":    {Label: "Phone", Color: "#C900C4"},
	"nas":      {Label: "NAS", Color: "#4A2FD0"},
	"media":    {Label: "Media", Color: "#8A4AD7"},
	"iot":      {Label: "IoT", Color: "#0C67DE"},
	"printer":  {Label: "Printer", Color: "#B86A1E"},
	"device":   {Label: "Device", Color: "#747474"},
}

var wantLupineCategories = map[string]string{
	"web":    "#3264EB",
	"media":  "#8A4AD7",
	"data":   "#B86A1E",
	"remote": "#4A2FD0",
	"infra":  "#013A6F",
	"iot":    "#0C67DE",
	"other":  "#747474",
}

var wantRistrettoPalette = map[string]string{
	"bg":           "#2C2525",
	"ink":          "#E6D9DB",
	"muted":        "#A89D9E",
	"accent":       "#A8A9EB",
	"line":         "rgba(230, 217, 219, 0.14)",
	"glass":        "rgba(44, 37, 37, 0.74)",
	"canvasTop":    "#3D2F2A",
	"canvasMiddle": "#2C2525",
	"canvasBottom": "#211B1B",
	"grid":         "rgba(168, 169, 235, 0.06)",
	"ambientGlow":  "rgba(168, 169, 235, 0.12)",
	"online":       "#ADDA78",
	"warning":      "#F9CC6C",
	"danger":       "#FD6883",
	"history":      "#B0989A",
	"vignette":     "rgba(33, 27, 27, 0.55)",
	"labelHalo":    "rgba(33, 27, 27, 0.88)",
	"bubble":       "rgba(44, 37, 37, 0.94)",
	"panel":        "rgba(44, 37, 37, 0.95)",
}

var wantRistrettoKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#F9CC6C"},
	"server":   {Label: "Server", Color: "#A8A9EB"},
	"computer": {Label: "Computer", Color: "#85DACC"},
	"phone":    {Label: "Phone", Color: "#FF8297"},
	"nas":      {Label: "NAS", Color: "#ADDA78"},
	"media":    {Label: "Media", Color: "#BEBFFD"},
	"iot":      {Label: "IoT", Color: "#9BF1E1"},
	"printer":  {Label: "Printer", Color: "#FB9A77"},
	"device":   {Label: "Device", Color: "#B0989A"},
}

var wantRistrettoCategories = map[string]string{
	"web":    "#A8A9EB",
	"media":  "#BEBFFD",
	"data":   "#FB9A77",
	"remote": "#ADDA78",
	"infra":  "#85DACC",
	"iot":    "#FF8297",
	"other":  "#B0989A",
}

var wantSolitudePalette = map[string]string{
	"bg":           "#101315",
	"ink":          "#CACCCC",
	"muted":        "#8A9097",
	"accent":       "#798186",
	"line":         "rgba(202, 204, 204, 0.14)",
	"glass":        "rgba(16, 19, 21, 0.74)",
	"canvasTop":    "#1E2225",
	"canvasMiddle": "#101315",
	"canvasBottom": "#0C0E10",
	"grid":         "rgba(121, 129, 134, 0.06)",
	"ambientGlow":  "rgba(121, 129, 134, 0.12)",
	"online":       "#8FB59F",
	"warning":      "#D8CDBD",
	"danger":       "#DE6145",
	"history":      "#9A9A9A",
	"vignette":     "rgba(12, 14, 16, 0.55)",
	"labelHalo":    "rgba(12, 14, 16, 0.88)",
	"bubble":       "rgba(16, 19, 21, 0.94)",
	"panel":        "rgba(16, 19, 21, 0.95)",
}

var wantSolitudeKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#D8CDBD"},
	"server":   {Label: "Server", Color: "#798186"},
	"computer": {Label: "Computer", Color: "#A9B4BC"},
	"phone":    {Label: "Phone", Color: "#B59AA6"},
	"nas":      {Label: "NAS", Color: "#8FB59F"},
	"media":    {Label: "Media", Color: "#A2A7C4"},
	"iot":      {Label: "IoT", Color: "#8FAEB0"},
	"printer":  {Label: "Printer", Color: "#C08B72"},
	"device":   {Label: "Device", Color: "#9A9A9A"},
}

var wantSolitudeCategories = map[string]string{
	"web":    "#798186",
	"media":  "#A2A7C4",
	"data":   "#C08B72",
	"remote": "#8FB59F",
	"infra":  "#A9B4BC",
	"iot":    "#8FAEB0",
	"other":  "#9A9A9A",
}

var wantStarshipPalette = map[string]string{
	"bg":           "#1C1C1E",
	"ink":          "#F5F5F7",
	"muted":        "#8E8E93",
	"accent":       "#8FB8DE",
	"line":         "rgba(245, 245, 247, 0.14)",
	"glass":        "rgba(28, 28, 30, 0.74)",
	"canvasTop":    "#2C2C2E",
	"canvasMiddle": "#1C1C1E",
	"canvasBottom": "#161618",
	"grid":         "rgba(143, 184, 222, 0.06)",
	"ambientGlow":  "rgba(143, 184, 222, 0.12)",
	"online":       "#FF9F0A",
	"warning":      "#FFC27A",
	"danger":       "#FF5F4A",
	"history":      "#98989D",
	"vignette":     "rgba(22, 22, 24, 0.55)",
	"labelHalo":    "rgba(22, 22, 24, 0.88)",
	"bubble":       "rgba(28, 28, 30, 0.94)",
	"panel":        "rgba(28, 28, 30, 0.95)",
}

var wantStarshipKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#FFC27A"},
	"server":   {Label: "Server", Color: "#8FB8DE"},
	"computer": {Label: "Computer", Color: "#B5D0EA"},
	"phone":    {Label: "Phone", Color: "#FF8FA3"},
	"nas":      {Label: "NAS", Color: "#FF9F0A"},
	"media":    {Label: "Media", Color: "#A6DCEF"},
	"iot":      {Label: "IoT", Color: "#CDEEFA"},
	"printer":  {Label: "Printer", Color: "#FF6A3D"},
	"device":   {Label: "Device", Color: "#98989D"},
}

var wantStarshipCategories = map[string]string{
	"web":    "#8FB8DE",
	"media":  "#A6DCEF",
	"data":   "#FF6A3D",
	"remote": "#FF9F0A",
	"infra":  "#B5D0EA",
	"iot":    "#CDEEFA",
	"other":  "#98989D",
}

var wantVantablackPalette = map[string]string{
	"bg":           "#000000",
	"ink":          "#FFFFFF",
	"muted":        "#8A8A8A",
	"accent":       "#8D9BA3",
	"line":         "rgba(255, 255, 255, 0.14)",
	"glass":        "rgba(0, 0, 0, 0.74)",
	"canvasTop":    "#1A1A1A",
	"canvasMiddle": "#090909",
	"canvasBottom": "#070707",
	"grid":         "rgba(141, 155, 163, 0.06)",
	"ambientGlow":  "rgba(141, 155, 163, 0.12)",
	"online":       "#86B386",
	"warning":      "#C9BDA3",
	"danger":       "#C08A8A",
	"history":      "#8F8F8F",
	"vignette":     "rgba(7, 7, 7, 0.55)",
	"labelHalo":    "rgba(7, 7, 7, 0.88)",
	"bubble":       "rgba(9, 9, 9, 0.94)",
	"panel":        "rgba(0, 0, 0, 0.95)",
}

var wantVantablackKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#C9BDA3"},
	"server":   {Label: "Server", Color: "#8D9BA3"},
	"computer": {Label: "Computer", Color: "#A7B4C2"},
	"phone":    {Label: "Phone", Color: "#B09FAE"},
	"nas":      {Label: "NAS", Color: "#86B386"},
	"media":    {Label: "Media", Color: "#9A9AAD"},
	"iot":      {Label: "IoT", Color: "#8FA8A4"},
	"printer":  {Label: "Printer", Color: "#B9A58F"},
	"device":   {Label: "Device", Color: "#8F8F8F"},
}

var wantVantablackCategories = map[string]string{
	"web":    "#8D9BA3",
	"media":  "#9A9AAD",
	"data":   "#B9A58F",
	"remote": "#86B386",
	"infra":  "#A7B4C2",
	"iot":    "#8FA8A4",
	"other":  "#8F8F8F",
}

var wantWhitePalette = map[string]string{
	"bg":           "#FFFFFF",
	"ink":          "#000000",
	"muted":        "#666666",
	"accent":       "#626B73",
	"line":         "rgba(0, 0, 0, 0.16)",
	"glass":        "rgba(255, 255, 255, 0.82)",
	"canvasTop":    "#FFFFFF",
	"canvasMiddle": "#FFFFFF",
	"canvasBottom": "#E8E8E8",
	"grid":         "rgba(98, 107, 115, 0.07)",
	"ambientGlow":  "rgba(98, 107, 115, 0.12)",
	"online":       "#2F6B3A",
	"warning":      "#7A694D",
	"danger":       "#A02F2F",
	"history":      "#6F6F6F",
	"vignette":     "rgba(232, 232, 232, 0.55)",
	"labelHalo":    "#FFFFFF",
	"bubble":       "rgba(255, 255, 255, 0.94)",
	"panel":        "rgba(255, 255, 255, 0.97)",
}

var wantWhiteKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#7A694D"},
	"server":   {Label: "Server", Color: "#626B73"},
	"computer": {Label: "Computer", Color: "#3F566B"},
	"phone":    {Label: "Phone", Color: "#7A4058"},
	"nas":      {Label: "NAS", Color: "#2F6B3A"},
	"media":    {Label: "Media", Color: "#5A4B7A"},
	"iot":      {Label: "IoT", Color: "#2F6B67"},
	"printer":  {Label: "Printer", Color: "#8A5A2B"},
	"device":   {Label: "Device", Color: "#6F6F6F"},
}

var wantWhiteCategories = map[string]string{
	"web":    "#626B73",
	"media":  "#5A4B7A",
	"data":   "#8A5A2B",
	"remote": "#2F6B3A",
	"infra":  "#3F566B",
	"iot":    "#2F6B67",
	"other":  "#6F6F6F",
}

var wantOmarchyDarkEffectsB = map[string]any{
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

var wantOmarchyLightEffects = map[string]any{
	"vignette":       0.3,
	"deviceGlow":     0.14,
	"devicePulse":    0.22,
	"serviceGlow":    0.08,
	"territoryBlur":  26.0,
	"ribbonBlur":     16.0,
	"lensBlur":       14.0,
	"glassBlur":      16.0,
	"panelBlur":      22.0,
	"territoryBlend": "multiply",
	"territoryAlpha": 0.72,
	"glowStrength":   0.6,
	"labelHalo":      0.9,
	"shadowStrength": 0.5,
}

func checkOmarchyPalette(t *testing.T, familyID, name, mode, sourceURL string, wantPalette map[string]string, wantKinds map[string]themeKind, wantCategories map[string]string) {
	t.Helper()
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, familyID)
	if family.Name != name {
		t.Fatalf("family name = %q, want %q", family.Name, name)
	}
	if family.Source.Name != "Omarchy" {
		t.Errorf("source name = %q, want Omarchy", family.Source.Name)
	}
	if family.Source.URL != sourceURL {
		t.Errorf("source url = %q, want %q", family.Source.URL, sourceURL)
	}
	if !strings.Contains(family.Source.Note, "748fa3c0d0418d7a199818d94c1f87d495d7e804") {
		t.Errorf("source note = %q, want the pinned Omarchy commit", family.Source.Note)
	}
	if len(family.Variants) != 1 {
		t.Fatalf("variants = %d, want 1", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != familyID || variant.Name != name || variant.Mode != mode {
		t.Fatalf("variant = %+v, want %s %s", variant, name, mode)
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
	wantEffects := wantOmarchyDarkEffectsB
	if mode == "light" {
		wantEffects = wantOmarchyLightEffects
	}
	if !reflect.DeepEqual(variant.Effects, wantEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantEffects)
	}
}

func TestFlexokiLightVariant(t *testing.T) {
	checkOmarchyPalette(t, "flexoki-light", "Flexoki Light", "light", "https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/flexoki-light/colors.toml", wantFlexokiLightPalette, wantFlexokiLightKinds, wantFlexokiLightCategories)
}

func TestLupineVariant(t *testing.T) {
	checkOmarchyPalette(t, "lupine", "Lupine", "light", "https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/lupine/colors.toml", wantLupinePalette, wantLupineKinds, wantLupineCategories)
}

func TestRistrettoVariant(t *testing.T) {
	checkOmarchyPalette(t, "ristretto", "Ristretto", "dark", "https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/ristretto/colors.toml", wantRistrettoPalette, wantRistrettoKinds, wantRistrettoCategories)
}

func TestSolitudeVariant(t *testing.T) {
	checkOmarchyPalette(t, "solitude", "Solitude", "dark", "https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/solitude/colors.toml", wantSolitudePalette, wantSolitudeKinds, wantSolitudeCategories)
}

func TestStarshipVariant(t *testing.T) {
	checkOmarchyPalette(t, "starship", "Starship", "dark", "https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/starship/colors.toml", wantStarshipPalette, wantStarshipKinds, wantStarshipCategories)
}

func TestVantablackVariant(t *testing.T) {
	checkOmarchyPalette(t, "vantablack", "Vantablack", "dark", "https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/vantablack/colors.toml", wantVantablackPalette, wantVantablackKinds, wantVantablackCategories)
}

func TestWhiteVariant(t *testing.T) {
	checkOmarchyPalette(t, "white", "White", "light", "https://github.com/omacom/omarchy/blob/748fa3c0d0418d7a199818d94c1f87d495d7e804/themes/white/colors.toml", wantWhitePalette, wantWhiteKinds, wantWhiteCategories)
}
