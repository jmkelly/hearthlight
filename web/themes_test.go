package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"regexp"
	"sort"
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
	Note string `json:"note"`
}

type themeVariant struct {
	ID                 string                   `json:"id"`
	Name               string                   `json:"name"`
	Mode               string                   `json:"mode"`
	Palette            map[string]string        `json:"palette"`
	Kinds              map[string]themeKind     `json:"kinds"`
	Categories         map[string]string        `json:"categories"`
	Effects            map[string]any           `json:"effects"`
	ContrastExceptions []themeContrastException `json:"contrastExceptions"`
}

type themeContrastException struct {
	Pair   string `json:"pair"`
	Reason string `json:"reason"`
}

// A variant may exempt a non-text graphical pair such as
// "kind.phone/canvasTop" with a reason. Text pairs are never exempt.

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
	"history":      "#707aa8",
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

var wantRosePineMoonPalette = map[string]string{
	"bg":           "#232136",
	"ink":          "#e0def4",
	"muted":        "#9591b2",
	"accent":       "#9ccfd8",
	"line":         "rgba(224, 222, 244, 0.14)",
	"glass":        "rgba(35, 33, 54, 0.74)",
	"canvasTop":    "#393552",
	"canvasMiddle": "#232136",
	"canvasBottom": "#1b1929",
	"grid":         "rgba(156, 207, 216, 0.06)",
	"ambientGlow":  "rgba(196, 167, 231, 0.12)",
	"online":       "#86c79a",
	"warning":      "#f6c177",
	"danger":       "#eb6f92",
	"history":      "#908caa",
	"vignette":     "rgba(35, 33, 54, 0.55)",
	"labelHalo":    "rgba(35, 33, 54, 0.88)",
	"bubble":       "rgba(42, 39, 63, 0.94)",
	"panel":        "rgba(35, 33, 54, 0.95)",
}

var wantRosePineMoonKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#f6c177"},
	"server":   {Label: "Server", Color: "#9ccfd8"},
	"computer": {Label: "Computer", Color: "#3e8fb0"},
	"phone":    {Label: "Phone", Color: "#eb6f92"},
	"nas":      {Label: "NAS", Color: "#86c79a"},
	"media":    {Label: "Media", Color: "#c4a7e7"},
	"iot":      {Label: "IoT", Color: "#7dc4b8"},
	"printer":  {Label: "Printer", Color: "#ea9a97"},
	"device":   {Label: "Device", Color: "#908caa"},
}

var wantRosePineMoonCategories = map[string]string{
	"web":    "#9ccfd8",
	"media":  "#c4a7e7",
	"data":   "#ea9a97",
	"remote": "#86c79a",
	"infra":  "#3e8fb0",
	"iot":    "#7dc4b8",
	"other":  "#908caa",
}

var wantRosePineMoonEffects = map[string]any{
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

var wantKanagawaWavePalette = map[string]string{
	"bg":           "#1F1F28",
	"ink":          "#DCD7BA",
	"muted":        "#9E9B82",
	"accent":       "#7E9CD8",
	"line":         "rgba(220, 215, 186, 0.14)",
	"glass":        "rgba(31, 31, 40, 0.74)",
	"canvasTop":    "#2A2A37",
	"canvasMiddle": "#1F1F28",
	"canvasBottom": "#16161D",
	"grid":         "rgba(126, 156, 216, 0.06)",
	"ambientGlow":  "rgba(126, 156, 216, 0.12)",
	"online":       "#98BB6C",
	"warning":      "#E6C384",
	"danger":       "#E46876",
	"history":      "#9E9B82",
	"vignette":     "rgba(22, 22, 29, 0.55)",
	"labelHalo":    "rgba(22, 22, 29, 0.88)",
	"bubble":       "rgba(31, 31, 40, 0.94)",
	"panel":        "rgba(31, 31, 40, 0.95)",
}

var wantKanagawaWaveKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#E6C384"},
	"server":   {Label: "Server", Color: "#7E9CD8"},
	"computer": {Label: "Computer", Color: "#7FB4CA"},
	"phone":    {Label: "Phone", Color: "#D27E99"},
	"nas":      {Label: "NAS", Color: "#98BB6C"},
	"media":    {Label: "Media", Color: "#957FB8"},
	"iot":      {Label: "IoT", Color: "#7AA89F"},
	"printer":  {Label: "Printer", Color: "#FFA066"},
	"device":   {Label: "Device", Color: "#9E9B82"},
}

var wantKanagawaWaveCategories = map[string]string{
	"web":    "#7E9CD8",
	"media":  "#957FB8",
	"data":   "#FFA066",
	"remote": "#98BB6C",
	"infra":  "#7FB4CA",
	"iot":    "#7AA89F",
	"other":  "#9E9B82",
}

var wantKanagawaWaveEffects = map[string]any{
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

var wantKanagawaLotusPalette = map[string]string{
	"bg":           "#F2ECBC",
	"ink":          "#545464",
	"muted":        "#686659",
	"accent":       "#4D699B",
	"line":         "rgba(84, 84, 100, 0.18)",
	"glass":        "rgba(242, 236, 188, 0.82)",
	"canvasTop":    "#F7F2CD",
	"canvasMiddle": "#EDE7B7",
	"canvasBottom": "#E4D794",
	"grid":         "rgba(77, 105, 155, 0.08)",
	"ambientGlow":  "rgba(77, 105, 155, 0.12)",
	"online":       "#677F49",
	"warning":      "#77713F",
	"danger":       "#C84053",
	"history":      "#716E61",
	"vignette":     "rgba(213, 206, 163, 0.55)",
	"labelHalo":    "#F7F2CD",
	"bubble":       "rgba(247, 242, 205, 0.94)",
	"panel":        "rgba(242, 236, 188, 0.97)",
}

var wantKanagawaLotusKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#77713F"},
	"server":   {Label: "Server", Color: "#4D699B"},
	"computer": {Label: "Computer", Color: "#477F94"},
	"phone":    {Label: "Phone", Color: "#B35B79"},
	"nas":      {Label: "NAS", Color: "#677F49"},
	"media":    {Label: "Media", Color: "#624C83"},
	"iot":      {Label: "IoT", Color: "#597B75"},
	"printer":  {Label: "Printer", Color: "#B66200"},
	"device":   {Label: "Device", Color: "#716E61"},
}

var wantKanagawaLotusCategories = map[string]string{
	"web":    "#4D699B",
	"media":  "#624C83",
	"data":   "#B66200",
	"remote": "#677F49",
	"infra":  "#477F94",
	"iot":    "#597B75",
	"other":  "#716E61",
}

var wantKanagawaLotusEffects = map[string]any{
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

var wantEverforestDarkPalette = map[string]string{
	"bg":           "#2d353b",
	"ink":          "#d3c6aa",
	"muted":        "#9da9a0",
	"accent":       "#7fb8cc",
	"line":         "rgba(211, 198, 170, 0.14)",
	"glass":        "rgba(45, 53, 59, 0.74)",
	"canvasTop":    "#343f44",
	"canvasMiddle": "#2d353b",
	"canvasBottom": "#232a2e",
	"grid":         "rgba(127, 184, 204, 0.06)",
	"ambientGlow":  "rgba(127, 184, 204, 0.12)",
	"online":       "#a7c080",
	"warning":      "#dbbc7f",
	"danger":       "#e67e80",
	"history":      "#859289",
	"vignette":     "rgba(35, 42, 46, 0.55)",
	"labelHalo":    "rgba(35, 42, 46, 0.88)",
	"bubble":       "rgba(45, 53, 59, 0.94)",
	"panel":        "rgba(45, 53, 59, 0.95)",
}

var wantEverforestDarkKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#dbbc7f"},
	"server":   {Label: "Server", Color: "#7fb8cc"},
	"computer": {Label: "Computer", Color: "#7fbbb3"},
	"phone":    {Label: "Phone", Color: "#d699b6"},
	"nas":      {Label: "NAS", Color: "#a7c080"},
	"media":    {Label: "Media", Color: "#b8a1d6"},
	"iot":      {Label: "IoT", Color: "#83c092"},
	"printer":  {Label: "Printer", Color: "#e69875"},
	"device":   {Label: "Device", Color: "#859289"},
}

var wantEverforestDarkCategories = map[string]string{
	"web":    "#7fb8cc",
	"media":  "#b8a1d6",
	"data":   "#e69875",
	"remote": "#a7c080",
	"infra":  "#7fbbb3",
	"iot":    "#83c092",
	"other":  "#859289",
}

var wantEverforestDarkEffects = map[string]any{
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

var wantEverforestLightPalette = map[string]string{
	"bg":           "#fdf6e3",
	"ink":          "#5c6a72",
	"muted":        "#657567",
	"accent":       "#378dbc",
	"line":         "rgba(92, 106, 114, 0.18)",
	"glass":        "rgba(253, 246, 227, 0.82)",
	"canvasTop":    "#fffbef",
	"canvasMiddle": "#fdf6e3",
	"canvasBottom": "#efebd4",
	"grid":         "rgba(55, 141, 188, 0.08)",
	"ambientGlow":  "rgba(55, 141, 188, 0.12)",
	"online":       "#7c8e01",
	"warning":      "#ae7d00",
	"danger":       "#f85552",
	"history":      "#6f7d70",
	"vignette":     "rgba(230, 226, 204, 0.55)",
	"labelHalo":    "#fffbef",
	"bubble":       "rgba(255, 251, 239, 0.94)",
	"panel":        "rgba(253, 246, 227, 0.97)",
}

var wantEverforestLightKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#ae7d00"},
	"server":   {Label: "Server", Color: "#378dbc"},
	"computer": {Label: "Computer", Color: "#309670"},
	"phone":    {Label: "Phone", Color: "#da4fae"},
	"nas":      {Label: "NAS", Color: "#7c8e01"},
	"media":    {Label: "Media", Color: "#8a6fbf"},
	"iot":      {Label: "IoT", Color: "#d94a48"},
	"printer":  {Label: "Printer", Color: "#da620a"},
	"device":   {Label: "Device", Color: "#6f7d70"},
}

var wantEverforestLightCategories = map[string]string{
	"web":    "#378dbc",
	"media":  "#8a6fbf",
	"data":   "#da620a",
	"remote": "#7c8e01",
	"infra":  "#309670",
	"iot":    "#da4fae",
	"other":  "#6f7d70",
}

var wantEverforestLightEffects = map[string]any{
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

var wantDraculaPalette = map[string]string{
	"bg":           "#282a36",
	"ink":          "#f8f8f2",
	"muted":        "#8b97bc",
	"accent":       "#8be9fd",
	"line":         "rgba(248, 248, 242, 0.14)",
	"glass":        "rgba(40, 42, 54, 0.74)",
	"canvasTop":    "#343746",
	"canvasMiddle": "#282a36",
	"canvasBottom": "#1e1f29",
	"grid":         "rgba(139, 233, 253, 0.06)",
	"ambientGlow":  "rgba(139, 233, 253, 0.12)",
	"online":       "#50fa7b",
	"warning":      "#f1fa8c",
	"danger":       "#ff5555",
	"history":      "#8b97bc",
	"vignette":     "rgba(30, 31, 41, 0.55)",
	"labelHalo":    "rgba(30, 31, 41, 0.88)",
	"bubble":       "rgba(40, 42, 54, 0.94)",
	"panel":        "rgba(40, 42, 54, 0.95)",
}

var wantDraculaKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#f1fa8c"},
	"server":   {Label: "Server", Color: "#8be9fd"},
	"computer": {Label: "Computer", Color: "#8fa1e8"},
	"phone":    {Label: "Phone", Color: "#ff79c6"},
	"nas":      {Label: "NAS", Color: "#50fa7b"},
	"media":    {Label: "Media", Color: "#bd93f9"},
	"iot":      {Label: "IoT", Color: "#4ee6c4"},
	"printer":  {Label: "Printer", Color: "#ffb86c"},
	"device":   {Label: "Device", Color: "#8b97bc"},
}

var wantDraculaCategories = map[string]string{
	"web":    "#8be9fd",
	"media":  "#bd93f9",
	"data":   "#ffb86c",
	"remote": "#50fa7b",
	"infra":  "#8fa1e8",
	"iot":    "#4ee6c4",
	"other":  "#8b97bc",
}

var wantDraculaEffects = map[string]any{
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

var wantDraculaAlucardPalette = map[string]string{
	"bg":           "#fffbeb",
	"ink":          "#1f1f1f",
	"muted":        "#6c664b",
	"accent":       "#036a96",
	"line":         "rgba(31, 31, 31, 0.16)",
	"glass":        "rgba(255, 251, 235, 0.82)",
	"canvasTop":    "#fffdf5",
	"canvasMiddle": "#fffbeb",
	"canvasBottom": "#f6efd9",
	"grid":         "rgba(3, 106, 150, 0.08)",
	"ambientGlow":  "rgba(3, 106, 150, 0.12)",
	"online":       "#14710a",
	"warning":      "#846e15",
	"danger":       "#cb3a2a",
	"history":      "#6c664b",
	"vignette":     "rgba(207, 207, 222, 0.55)",
	"labelHalo":    "#fffdf5",
	"bubble":       "rgba(255, 253, 245, 0.94)",
	"panel":        "rgba(255, 251, 235, 0.97)",
}

var wantDraculaAlucardKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#846e15"},
	"server":   {Label: "Server", Color: "#036a96"},
	"computer": {Label: "Computer", Color: "#3a5fbf"},
	"phone":    {Label: "Phone", Color: "#a3144d"},
	"nas":      {Label: "NAS", Color: "#14710a"},
	"media":    {Label: "Media", Color: "#644ac9"},
	"iot":      {Label: "IoT", Color: "#0f7d6c"},
	"printer":  {Label: "Printer", Color: "#a34d14"},
	"device":   {Label: "Device", Color: "#6c664b"},
}

var wantDraculaAlucardCategories = map[string]string{
	"web":    "#036a96",
	"media":  "#644ac9",
	"data":   "#a34d14",
	"remote": "#14710a",
	"infra":  "#3a5fbf",
	"iot":    "#a3144d",
	"other":  "#6c664b",
}

var wantDraculaAlucardEffects = map[string]any{
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

var wantOneDarkPalette = map[string]string{
	"bg":           "#282c34",
	"ink":          "#abb2bf",
	"muted":        "#8d94a1",
	"accent":       "#61afef",
	"line":         "rgba(171, 178, 191, 0.14)",
	"glass":        "rgba(40, 44, 52, 0.74)",
	"canvasTop":    "#313640",
	"canvasMiddle": "#282c34",
	"canvasBottom": "#21252b",
	"grid":         "rgba(97, 175, 239, 0.06)",
	"ambientGlow":  "rgba(97, 175, 239, 0.12)",
	"online":       "#98c379",
	"warning":      "#e5c07b",
	"danger":       "#e06c75",
	"history":      "#77808f",
	"vignette":     "rgba(33, 37, 43, 0.55)",
	"labelHalo":    "rgba(33, 37, 43, 0.88)",
	"bubble":       "rgba(40, 44, 52, 0.94)",
	"panel":        "rgba(40, 44, 52, 0.95)",
}

var wantOneDarkKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#e5c07b"},
	"server":   {Label: "Server", Color: "#61afef"},
	"computer": {Label: "Computer", Color: "#528bff"},
	"phone":    {Label: "Phone", Color: "#e06c75"},
	"nas":      {Label: "NAS", Color: "#98c379"},
	"media":    {Label: "Media", Color: "#c678dd"},
	"iot":      {Label: "IoT", Color: "#56b6c2"},
	"printer":  {Label: "Printer", Color: "#d19a66"},
	"device":   {Label: "Device", Color: "#818896"},
}

var wantOneDarkCategories = map[string]string{
	"web":    "#61afef",
	"media":  "#c678dd",
	"data":   "#d19a66",
	"remote": "#98c379",
	"infra":  "#56b6c2",
	"iot":    "#528bff",
	"other":  "#818896",
}

var wantOneDarkEffects = map[string]any{
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

var wantOneLightPalette = map[string]string{
	"bg":           "#fafafa",
	"ink":          "#383a42",
	"muted":        "#696c77",
	"accent":       "#4078f2",
	"line":         "rgba(56, 58, 66, 0.16)",
	"glass":        "rgba(250, 250, 250, 0.82)",
	"canvasTop":    "#ffffff",
	"canvasMiddle": "#fafafa",
	"canvasBottom": "#f0f0f0",
	"grid":         "rgba(64, 120, 242, 0.08)",
	"ambientGlow":  "rgba(64, 120, 242, 0.12)",
	"online":       "#4c9a4c",
	"warning":      "#986801",
	"danger":       "#e45649",
	"history":      "#83858d",
	"vignette":     "rgba(229, 229, 230, 0.55)",
	"labelHalo":    "#ffffff",
	"bubble":       "rgba(255, 255, 255, 0.94)",
	"panel":        "rgba(250, 250, 250, 0.97)",
}

var wantOneLightKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#986801"},
	"server":   {Label: "Server", Color: "#4078f2"},
	"computer": {Label: "Computer", Color: "#526fff"},
	"phone":    {Label: "Phone", Color: "#e45649"},
	"nas":      {Label: "NAS", Color: "#4c9a4c"},
	"media":    {Label: "Media", Color: "#a626a4"},
	"iot":      {Label: "IoT", Color: "#0184bc"},
	"printer":  {Label: "Printer", Color: "#b87e01"},
	"device":   {Label: "Device", Color: "#696c77"},
}

var wantOneLightCategories = map[string]string{
	"web":    "#4078f2",
	"media":  "#a626a4",
	"data":   "#986801",
	"remote": "#4c9a4c",
	"infra":  "#0184bc",
	"iot":    "#526fff",
	"other":  "#83858d",
}

var wantOneLightEffects = map[string]any{
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

var wantSolarizedDarkPalette = map[string]string{
	"bg":           "#002b36",
	"ink":          "#839496",
	"muted":        "#79929a",
	"accent":       "#268bd2",
	"line":         "rgba(131, 148, 150, 0.14)",
	"glass":        "rgba(0, 43, 54, 0.74)",
	"canvasTop":    "#073642",
	"canvasMiddle": "#002b36",
	"canvasBottom": "#001f27",
	"grid":         "rgba(38, 139, 210, 0.06)",
	"ambientGlow":  "rgba(38, 139, 210, 0.12)",
	"online":       "#859900",
	"warning":      "#b58900",
	"danger":       "#dc322f",
	"history":      "#657e86",
	"vignette":     "rgba(0, 31, 39, 0.55)",
	"labelHalo":    "rgba(0, 31, 39, 0.88)",
	"bubble":       "rgba(0, 43, 54, 0.94)",
	"panel":        "rgba(0, 43, 54, 0.95)",
}

var wantSolarizedDarkKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#b58900"},
	"server":   {Label: "Server", Color: "#268bd2"},
	"computer": {Label: "Computer", Color: "#6e73c5"},
	"phone":    {Label: "Phone", Color: "#d53f88"},
	"nas":      {Label: "NAS", Color: "#859900"},
	"media":    {Label: "Media", Color: "#2aa198"},
	"iot":      {Label: "IoT", Color: "#de403d"},
	"printer":  {Label: "Printer", Color: "#d44e17"},
	"device":   {Label: "Device", Color: "#93a1a1"},
}

var wantSolarizedDarkCategories = map[string]string{
	"web":    "#268bd2",
	"media":  "#6e73c5",
	"data":   "#d44e17",
	"remote": "#859900",
	"infra":  "#2aa198",
	"iot":    "#d53f88",
	"other":  "#93a1a1",
}

var wantSolarizedDarkEffects = map[string]any{
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

var wantSolarizedLightPalette = map[string]string{
	"bg":           "#fdf6e3",
	"ink":          "#586e75",
	"muted":        "#5f747c",
	"accent":       "#268bd2",
	"line":         "rgba(88, 110, 117, 0.18)",
	"glass":        "rgba(253, 246, 227, 0.82)",
	"canvasTop":    "#fffcf0",
	"canvasMiddle": "#fdf6e3",
	"canvasBottom": "#eee8d5",
	"grid":         "rgba(38, 139, 210, 0.08)",
	"ambientGlow":  "rgba(38, 139, 210, 0.12)",
	"online":       "#7a8c00",
	"warning":      "#a67e00",
	"danger":       "#dc322f",
	"history":      "#758787",
	"vignette":     "rgba(238, 232, 213, 0.55)",
	"labelHalo":    "#fffcf0",
	"bubble":       "rgba(255, 252, 240, 0.94)",
	"panel":        "rgba(253, 246, 227, 0.97)",
}

var wantSolarizedLightKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#a67e00"},
	"server":   {Label: "Server", Color: "#268bd2"},
	"computer": {Label: "Computer", Color: "#6c71c4"},
	"phone":    {Label: "Phone", Color: "#d33682"},
	"nas":      {Label: "NAS", Color: "#7a8c00"},
	"media":    {Label: "Media", Color: "#26928a"},
	"iot":      {Label: "IoT", Color: "#dc322f"},
	"printer":  {Label: "Printer", Color: "#cb4b16"},
	"device":   {Label: "Device", Color: "#758787"},
}

var wantSolarizedLightCategories = map[string]string{
	"web":    "#268bd2",
	"media":  "#6c71c4",
	"data":   "#cb4b16",
	"remote": "#7a8c00",
	"infra":  "#26928a",
	"iot":    "#d33682",
	"other":  "#758787",
}

var wantSolarizedLightEffects = map[string]any{
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

var wantGruvboxDarkPalette = map[string]string{
	"bg":           "#282828",
	"ink":          "#ebdbb2",
	"muted":        "#bdae93",
	"accent":       "#7fb0b5",
	"line":         "rgba(235, 219, 178, 0.14)",
	"glass":        "rgba(40, 40, 40, 0.74)",
	"canvasTop":    "#3c3836",
	"canvasMiddle": "#282828",
	"canvasBottom": "#1d2021",
	"grid":         "rgba(131, 165, 152, 0.06)",
	"ambientGlow":  "rgba(131, 165, 152, 0.12)",
	"online":       "#b8bb26",
	"warning":      "#fabd2f",
	"danger":       "#fb4934",
	"history":      "#928374",
	"vignette":     "rgba(29, 32, 33, 0.55)",
	"labelHalo":    "rgba(29, 32, 33, 0.88)",
	"bubble":       "rgba(50, 48, 47, 0.94)",
	"panel":        "rgba(40, 40, 40, 0.95)",
}

var wantGruvboxDarkKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#fabd2f"},
	"server":   {Label: "Server", Color: "#7fb0b5"},
	"computer": {Label: "Computer", Color: "#83a598"},
	"phone":    {Label: "Phone", Color: "#d3869b"},
	"nas":      {Label: "NAS", Color: "#b8bb26"},
	"media":    {Label: "Media", Color: "#b986d3"},
	"iot":      {Label: "IoT", Color: "#8ec07c"},
	"printer":  {Label: "Printer", Color: "#fe8019"},
	"device":   {Label: "Device", Color: "#928374"},
}

var wantGruvboxDarkCategories = map[string]string{
	"web":    "#7fb0b5",
	"media":  "#b986d3",
	"data":   "#fe8019",
	"remote": "#b8bb26",
	"infra":  "#83a598",
	"iot":    "#8ec07c",
	"other":  "#928374",
}

var wantGruvboxDarkEffects = map[string]any{
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

var wantNordDarkPalette = map[string]string{
	"bg":           "#2e3440",
	"ink":          "#d8dee9",
	"muted":        "#a7b1c2",
	"accent":       "#88c0d0",
	"line":         "rgba(216, 222, 233, 0.14)",
	"glass":        "rgba(46, 52, 64, 0.74)",
	"canvasTop":    "#3b4252",
	"canvasMiddle": "#2e3440",
	"canvasBottom": "#242933",
	"grid":         "rgba(136, 192, 208, 0.06)",
	"ambientGlow":  "rgba(136, 192, 208, 0.12)",
	"online":       "#a3be8c",
	"warning":      "#ebcb8b",
	"danger":       "#bf616a",
	"history":      "#8a96ab",
	"vignette":     "rgba(36, 41, 51, 0.55)",
	"labelHalo":    "rgba(36, 41, 51, 0.88)",
	"bubble":       "rgba(46, 52, 64, 0.94)",
	"panel":        "rgba(46, 52, 64, 0.95)",
}

var wantNordDarkKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#ebcb8b"},
	"server":   {Label: "Server", Color: "#88c0d0"},
	"computer": {Label: "Computer", Color: "#81a1c1"},
	"phone":    {Label: "Phone", Color: "#b48ead"},
	"nas":      {Label: "NAS", Color: "#a3be8c"},
	"media":    {Label: "Media", Color: "#a18eb4"},
	"iot":      {Label: "IoT", Color: "#8fbcbb"},
	"printer":  {Label: "Printer", Color: "#d08770"},
	"device":   {Label: "Device", Color: "#8a96ab"},
}

var wantNordDarkCategories = map[string]string{
	"web":    "#88c0d0",
	"media":  "#a18eb4",
	"data":   "#d08770",
	"remote": "#a3be8c",
	"infra":  "#81a1c1",
	"iot":    "#8fbcbb",
	"other":  "#8a96ab",
}

var wantNordDarkEffects = map[string]any{
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

func findThemeVariant(t *testing.T, family themeFamily, id string) themeVariant {
	t.Helper()
	for _, variant := range family.Variants {
		if variant.ID == id {
			return variant
		}
	}
	t.Fatalf("theme family %q has no variant %q", family.ID, id)
	return themeVariant{}
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
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
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
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
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

func TestRosePineMoonVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "rose-pine")
	if family.Name != "Rosé Pine" {
		t.Fatalf("family name = %q, want Rosé Pine", family.Name)
	}
	if family.Source.Name != "Rosé Pine" {
		t.Errorf("source name = %q, want Rosé Pine", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/rose-pine/rose-pine" {
		t.Errorf("source url = %q, want the Rosé Pine project URL", family.Source.URL)
	}
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "rose-pine-moon" || variant.Name != "Rosé Pine Moon" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Rosé Pine Moon dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantRosePineMoonPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantRosePineMoonPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantRosePineMoonKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantRosePineMoonKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantRosePineMoonCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantRosePineMoonCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantRosePineMoonEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantRosePineMoonEffects)
	}
}

func TestKanagawaWaveVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "kanagawa")
	if family.Name != "Kanagawa" {
		t.Fatalf("family name = %q, want Kanagawa", family.Name)
	}
	if family.Source.Name != "Kanagawa" {
		t.Errorf("source name = %q, want Kanagawa", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/rebelot/kanagawa.nvim" {
		t.Errorf("source url = %q, want the Kanagawa project URL", family.Source.URL)
	}
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "kanagawa-wave" || variant.Name != "Kanagawa Wave" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Kanagawa Wave dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantKanagawaWavePalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantKanagawaWavePalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantKanagawaWaveKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantKanagawaWaveKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantKanagawaWaveCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantKanagawaWaveCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantKanagawaWaveEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantKanagawaWaveEffects)
	}
}

func TestKanagawaLotusVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "kanagawa")
	if family.Name != "Kanagawa" {
		t.Fatalf("family name = %q, want Kanagawa", family.Name)
	}
	if family.Source.Name != "Kanagawa" {
		t.Errorf("source name = %q, want Kanagawa", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/rebelot/kanagawa.nvim" {
		t.Errorf("source url = %q, want the Kanagawa project URL", family.Source.URL)
	}
	if strings.TrimSpace(family.Source.Note) == "" {
		t.Error("source note is empty, want the contrast shifts documented")
	}
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := findThemeVariant(t, family, "kanagawa-lotus")
	if variant.Name != "Kanagawa Lotus" || variant.Mode != "light" {
		t.Fatalf("variant = %+v, want Kanagawa Lotus light", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantKanagawaLotusPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantKanagawaLotusPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantKanagawaLotusKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantKanagawaLotusKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantKanagawaLotusCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantKanagawaLotusCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantKanagawaLotusEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantKanagawaLotusEffects)
	}
}

func TestEverforestDarkVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "everforest")
	if family.Name != "Everforest" {
		t.Fatalf("family name = %q, want Everforest", family.Name)
	}
	if family.Source.Name != "Everforest" {
		t.Errorf("source name = %q, want Everforest", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/sainnhe/everforest" {
		t.Errorf("source url = %q, want the Everforest project URL", family.Source.URL)
	}
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "everforest-dark" || variant.Name != "Everforest Dark" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Everforest Dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantEverforestDarkPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantEverforestDarkPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantEverforestDarkKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantEverforestDarkKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantEverforestDarkCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantEverforestDarkCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantEverforestDarkEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantEverforestDarkEffects)
	}
}

func TestEverforestLightVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "everforest")
	if family.Name != "Everforest" {
		t.Fatalf("family name = %q, want Everforest", family.Name)
	}
	if family.Source.Name != "Everforest" {
		t.Errorf("source name = %q, want Everforest", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/sainnhe/everforest" {
		t.Errorf("source url = %q, want the Everforest project URL", family.Source.URL)
	}
	if strings.TrimSpace(family.Source.Note) == "" {
		t.Error("source note is empty, want the contrast shifts documented")
	}
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := findThemeVariant(t, family, "everforest-light")
	if variant.Name != "Everforest Light" || variant.Mode != "light" {
		t.Fatalf("variant = %+v, want Everforest Light", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantEverforestLightPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantEverforestLightPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantEverforestLightKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantEverforestLightKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantEverforestLightCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantEverforestLightCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantEverforestLightEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantEverforestLightEffects)
	}
}

func TestDraculaVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "dracula")
	if family.Name != "Dracula" {
		t.Fatalf("family name = %q, want Dracula", family.Name)
	}
	if family.Source.Name != "Dracula" {
		t.Errorf("source name = %q, want Dracula", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/dracula/dracula-theme" {
		t.Errorf("source url = %q, want the Dracula project URL", family.Source.URL)
	}
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "dracula" || variant.Name != "Dracula" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Dracula dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantDraculaPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantDraculaPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantDraculaKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantDraculaKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantDraculaCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantDraculaCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantDraculaEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantDraculaEffects)
	}
}

func TestDraculaAlucardVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "dracula")
	if family.Name != "Dracula" {
		t.Fatalf("family name = %q, want Dracula", family.Name)
	}
	if family.Source.Name != "Dracula" {
		t.Errorf("source name = %q, want Dracula", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/dracula/dracula-theme" {
		t.Errorf("source url = %q, want the Dracula project URL", family.Source.URL)
	}
	if strings.TrimSpace(family.Source.Note) == "" {
		t.Error("source note is empty, want the provenance documented")
	}
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := findThemeVariant(t, family, "dracula-alucard")
	if variant.Name != "Dracula Alucard" || variant.Mode != "light" {
		t.Fatalf("variant = %+v, want Dracula Alucard light", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantDraculaAlucardPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantDraculaAlucardPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantDraculaAlucardKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantDraculaAlucardKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantDraculaAlucardCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantDraculaAlucardCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantDraculaAlucardEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantDraculaAlucardEffects)
	}
}

func TestOneDarkVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "one")
	if family.Name != "One" {
		t.Fatalf("family name = %q, want One", family.Name)
	}
	if family.Source.Name != "Atom One Dark" {
		t.Errorf("source name = %q, want Atom One Dark", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/atom/one-dark-syntax" {
		t.Errorf("source url = %q, want the One Dark project URL", family.Source.URL)
	}
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "one-dark" || variant.Name != "One Dark" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want One Dark dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantOneDarkPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantOneDarkPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantOneDarkKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantOneDarkKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantOneDarkCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantOneDarkCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantOneDarkEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantOneDarkEffects)
	}
}

func TestOneLightVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "one")
	if family.Name != "One" {
		t.Fatalf("family name = %q, want One", family.Name)
	}
	if family.Source.Name != "Atom One Dark" {
		t.Errorf("source name = %q, want Atom One Dark", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/atom/one-dark-syntax" {
		t.Errorf("source url = %q, want the One Dark project URL", family.Source.URL)
	}
	if strings.TrimSpace(family.Source.Note) == "" {
		t.Error("source note is empty, want the contrast shifts documented")
	}
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := findThemeVariant(t, family, "one-light")
	if variant.Name != "One Light" || variant.Mode != "light" {
		t.Fatalf("variant = %+v, want One Light", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantOneLightPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantOneLightPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantOneLightKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantOneLightKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantOneLightCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantOneLightCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantOneLightEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantOneLightEffects)
	}
}

func TestSolarizedDarkVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "solarized")
	if family.Name != "Solarized" {
		t.Fatalf("family name = %q, want Solarized", family.Name)
	}
	if family.Source.Name != "Solarized" {
		t.Errorf("source name = %q, want Solarized", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/altercation/solarized" {
		t.Errorf("source url = %q, want the Solarized project URL", family.Source.URL)
	}
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "solarized-dark" || variant.Name != "Solarized Dark" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Solarized Dark dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantSolarizedDarkPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantSolarizedDarkPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantSolarizedDarkKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantSolarizedDarkKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantSolarizedDarkCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantSolarizedDarkCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantSolarizedDarkEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantSolarizedDarkEffects)
	}
}

func TestSolarizedLightVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "solarized")
	if family.Name != "Solarized" {
		t.Fatalf("family name = %q, want Solarized", family.Name)
	}
	if family.Source.Name != "Solarized" {
		t.Errorf("source name = %q, want Solarized", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/altercation/solarized" {
		t.Errorf("source url = %q, want the Solarized project URL", family.Source.URL)
	}
	if strings.TrimSpace(family.Source.Note) == "" {
		t.Error("source note is empty, want the contrast shifts documented")
	}
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := findThemeVariant(t, family, "solarized-light")
	if variant.Name != "Solarized Light" || variant.Mode != "light" {
		t.Fatalf("variant = %+v, want Solarized Light", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantSolarizedLightPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantSolarizedLightPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantSolarizedLightKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantSolarizedLightKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantSolarizedLightCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantSolarizedLightCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantSolarizedLightEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantSolarizedLightEffects)
	}
}

func TestGruvboxVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "gruvbox")
	if family.Name != "Gruvbox" {
		t.Fatalf("family name = %q, want Gruvbox", family.Name)
	}
	if family.Source.Name != "Gruvbox" {
		t.Errorf("source name = %q, want Gruvbox", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/morhetz/gruvbox" {
		t.Errorf("source url = %q, want the Gruvbox project URL", family.Source.URL)
	}
	if strings.TrimSpace(family.Source.Note) == "" {
		t.Error("source note is empty, want the separation-driven shifts documented")
	}
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "gruvbox-dark" || variant.Name != "Gruvbox Dark" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Gruvbox Dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantGruvboxDarkPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantGruvboxDarkPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantGruvboxDarkKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantGruvboxDarkKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantGruvboxDarkCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantGruvboxDarkCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantGruvboxDarkEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantGruvboxDarkEffects)
	}
}

func TestNordVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "nord")
	if family.Name != "Nord" {
		t.Fatalf("family name = %q, want Nord", family.Name)
	}
	if family.Source.Name != "Nord" {
		t.Errorf("source name = %q, want Nord", family.Source.Name)
	}
	if family.Source.URL != "https://github.com/nordtheme/nord" {
		t.Errorf("source url = %q, want the Nord project URL", family.Source.URL)
	}
	if strings.TrimSpace(family.Source.Note) == "" {
		t.Error("source note is empty, want the separation-driven shifts documented")
	}
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := family.Variants[0]
	if variant.ID != "nord-dark" || variant.Name != "Nord Dark" || variant.Mode != "dark" {
		t.Fatalf("variant = %+v, want Nord Dark", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantNordDarkPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantNordDarkPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantNordDarkKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantNordDarkKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantNordDarkCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantNordDarkCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantNordDarkEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantNordDarkEffects)
	}
}

func TestThemeFamiliesCreditSources(t *testing.T) {
	_, catalog := readCatalog(t)
	for _, family := range catalog.Families {
		if strings.TrimSpace(family.Source.Name) == "" {
			t.Errorf("family %q has an empty source name", family.ID)
		}
		if !strings.HasPrefix(family.Source.URL, "https://") {
			t.Errorf("family %q source url = %q, want an https:// URL", family.ID, family.Source.URL)
		}
	}
	hearthlight := findThemeFamily(t, catalog, "hearthlight")
	if hearthlight.Source.URL != "https://github.com/jmkelly/hearthlight" {
		t.Errorf("hearthlight source url = %q, want the Hearthlight repository URL", hearthlight.Source.URL)
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

const (
	textContrastMinimum      = 4.5
	graphicalContrastMinimum = 3.0
)

type textContrastPair struct {
	token string
	role  string
}

// themes.json calls the muted text token "muted"; the issue and the design
// spec call the same role "inkMuted". Check the token the renderer applies.
var themeTextPairs = []textContrastPair{
	{token: "ink", role: "ink"},
	{token: "muted", role: "inkMuted"},
}

type themeColour struct {
	r, g, b, a float64
}

func parseThemeColour(value string) (themeColour, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "#") {
		hex := value[1:]
		switch len(hex) {
		case 3, 4:
			var expanded strings.Builder
			for _, digit := range hex {
				expanded.WriteRune(digit)
				expanded.WriteRune(digit)
			}
			hex = expanded.String()
		case 6, 8:
		default:
			return themeColour{}, fmt.Errorf("%q is not a 3, 4, 6, or 8 digit hex colour", value)
		}
		if len(hex) == 6 {
			hex += "ff"
		}
		var channels [4]float64
		for i := range channels {
			channel, err := strconv.ParseUint(hex[i*2:2+i*2], 16, 8)
			if err != nil {
				return themeColour{}, fmt.Errorf("%q has an invalid hex channel: %w", value, err)
			}
			channels[i] = float64(channel) / 255
		}
		return themeColour{r: channels[0], g: channels[1], b: channels[2], a: channels[3]}, nil
	}

	open := strings.Index(value, "(")
	if open < 0 || !strings.HasSuffix(value, ")") {
		return themeColour{}, fmt.Errorf("unsupported colour %q", value)
	}
	function := strings.TrimSpace(value[:open])
	if function != "rgb" && function != "rgba" {
		return themeColour{}, fmt.Errorf("unsupported colour %q", value)
	}
	parts := strings.Split(value[open+1:len(value)-1], ",")
	if len(parts) != 3 && len(parts) != 4 {
		return themeColour{}, fmt.Errorf("%q has %d channels, want 3 or 4", value, len(parts))
	}
	channels := [4]float64{0, 0, 0, 1}
	for i, part := range parts {
		part = strings.TrimSpace(part)
		percent := strings.HasSuffix(part, "%")
		part = strings.TrimSuffix(part, "%")
		number, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return themeColour{}, fmt.Errorf("%q has an invalid channel %q: %w", value, part, err)
		}
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return themeColour{}, fmt.Errorf("%q has a non-finite channel %q", value, part)
		}
		if percent {
			channels[i] = number / 100
		} else if i == 3 {
			channels[i] = number
		} else {
			channels[i] = number / 255
		}
	}
	for _, channel := range channels {
		if channel < 0 || channel > 1 {
			return themeColour{}, fmt.Errorf("%q has a channel outside 0..1", value)
		}
	}
	return themeColour{r: channels[0], g: channels[1], b: channels[2], a: channels[3]}, nil
}

func compositeColour(foreground, background themeColour) themeColour {
	alpha := foreground.a
	return themeColour{
		r: foreground.r*alpha + background.r*(1-alpha),
		g: foreground.g*alpha + background.g*(1-alpha),
		b: foreground.b*alpha + background.b*(1-alpha),
		a: 1,
	}
}

func contrastRatioColours(a, b themeColour) float64 {
	return contrastRatio([3]float64{a.r, a.g, a.b}, [3]float64{b.r, b.g, b.b})
}

type contrastCheck struct {
	key        string
	token      string
	role       string
	surface    string
	foreground themeColour
	background themeColour
	minimum    float64
	text       bool
}

var contrastPaletteTokens = []string{
	"bg", "ink", "muted", "panel", "bubble", "glass",
	"canvasTop", "canvasMiddle", "canvasBottom", "labelHalo",
	"accent", "online", "history",
}

func loadContrastColours(variant themeVariant) (map[string]themeColour, []string) {
	colours := make(map[string]themeColour, len(contrastPaletteTokens))
	var failures []string
	for _, token := range contrastPaletteTokens {
		value, ok := variant.Palette[token]
		if !ok {
			failures = append(failures, fmt.Sprintf("%s: palette is missing %q", variant.ID, token))
			continue
		}
		colour, err := parseThemeColour(value)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: palette %q: %v", variant.ID, token, err))
			continue
		}
		colours[token] = colour
	}
	return colours, failures
}

// Panel and glass chrome sit over the page background. Device bubbles sit on
// the canvas, and the label halo is drawn over every canvas stop.
func contrastSurfaces(colours map[string]themeColour) map[string]themeColour {
	background := colours["bg"]
	halo := colours["labelHalo"]
	bubble := colours["bubble"]
	surfaces := map[string]themeColour{
		"bg":           background,
		"canvasTop":    colours["canvasTop"],
		"canvasMiddle": colours["canvasMiddle"],
		"canvasBottom": colours["canvasBottom"],
		"panel":        compositeColour(colours["panel"], background),
		"glass":        compositeColour(colours["glass"], background),
		"bubble":       compositeColour(bubble, colours["canvasMiddle"]),
	}
	for _, canvas := range []string{"canvasTop", "canvasMiddle", "canvasBottom"} {
		surfaces["labelHalo."+canvas] = compositeColour(halo, colours[canvas])
	}
	surfaces["bubble.canvasTop"] = compositeColour(bubble, colours["canvasTop"])
	surfaces["bubble.canvasBottom"] = compositeColour(bubble, colours["canvasBottom"])
	return surfaces
}

func themeContrastChecks(variant themeVariant) ([]contrastCheck, []string) {
	colours, failures := loadContrastColours(variant)
	if len(failures) > 0 {
		return nil, failures
	}
	surfaces := contrastSurfaces(colours)
	checks := make([]contrastCheck, 0, 2*7+3+5+3*len(variant.Kinds)+3*len(variant.Categories))
	seen := make(map[string]bool)
	add := func(key, token, role, surface string, foreground themeColour, minimum float64, text bool) {
		if seen[key] {
			return
		}
		seen[key] = true
		checks = append(checks, contrastCheck{
			key: key, token: token, role: role, surface: surface,
			foreground: foreground, background: surfaces[surface],
			minimum: minimum, text: text,
		})
	}

	for _, pair := range themeTextPairs {
		for _, surface := range []string{"bg", "canvasMiddle", "panel", "bubble", "glass", "bubble.canvasTop", "bubble.canvasBottom"} {
			role := pair.role
			if role != pair.token {
				role = fmt.Sprintf("%s (%s)", pair.role, pair.token)
			}
			add(pair.token+"/"+surface, pair.token, role, surface, colours[pair.token], textContrastMinimum, true)
		}
	}
	for _, canvas := range []string{"canvasTop", "canvasMiddle", "canvasBottom"} {
		surface := "labelHalo." + canvas
		add("ink/"+surface, "ink", "ink label text", surface, colours["ink"], textContrastMinimum, true)
	}

	addGraphical := func(token, surface string, foreground themeColour) {
		add(token+"/"+surface, token, token, surface, foreground, graphicalContrastMinimum, false)
	}
	addGraphical("accent", "bg", colours["accent"])
	addGraphical("accent", "panel", colours["accent"])
	addGraphical("online", "canvasMiddle", colours["online"])
	addGraphical("history", "canvasMiddle", colours["history"])
	// Keep the earlier all-canvas coverage in addition to the exact adjacent
	// pairs, so a mark cannot regress at a different canvas stop.
	for _, token := range []string{"accent", "online", "history"} {
		for _, surface := range []string{"canvasTop", "canvasMiddle", "canvasBottom", "panel"} {
			addGraphical(token, surface, colours[token])
		}
	}

	kindNames := make([]string, 0, len(variant.Kinds))
	for name := range variant.Kinds {
		kindNames = append(kindNames, name)
	}
	sort.Strings(kindNames)
	for _, name := range kindNames {
		colour, err := parseThemeColour(variant.Kinds[name].Color)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: kind %q colour: %v", variant.ID, name, err))
			continue
		}
		token := "kind." + name
		for _, surface := range []string{"bubble", "canvasTop", "canvasMiddle", "canvasBottom"} {
			addGraphical(token, surface, colour)
		}
	}

	categoryNames := make([]string, 0, len(variant.Categories))
	for name := range variant.Categories {
		categoryNames = append(categoryNames, name)
	}
	sort.Strings(categoryNames)
	for _, name := range categoryNames {
		colour, err := parseThemeColour(variant.Categories[name])
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: category %q colour: %v", variant.ID, name, err))
			continue
		}
		token := "category." + name
		for _, surface := range []string{"canvasTop", "canvasMiddle", "canvasBottom"} {
			addGraphical(token, surface, colour)
		}
	}
	return checks, failures
}

func contrastFailures(variant themeVariant) []string {
	checks, failures := themeContrastChecks(variant)
	if len(failures) > 0 {
		return failures
	}

	textPairs := make(map[string]bool)
	graphicalPairs := make(map[string]bool)
	for _, check := range checks {
		if check.text {
			textPairs[check.key] = true
		} else {
			graphicalPairs[check.key] = true
		}
	}

	exceptions := make(map[string]themeContrastException, len(variant.ContrastExceptions))
	order := make([]string, 0, len(variant.ContrastExceptions))
	for _, exception := range variant.ContrastExceptions {
		key := strings.TrimSpace(exception.Pair)
		if key == "" {
			failures = append(failures, fmt.Sprintf("%s: contrast exception has an empty pair", variant.ID))
			continue
		}
		if _, exists := exceptions[key]; exists {
			failures = append(failures, fmt.Sprintf("%s: duplicate contrast exception %q", variant.ID, key))
			continue
		}
		exceptions[key] = exception
		order = append(order, key)
	}

	exempted := make(map[string]bool, len(exceptions))
	for _, key := range order {
		exception := exceptions[key]
		if textPairs[key] {
			failures = append(failures, fmt.Sprintf("%s: text pair %s declares an exception (%q); text pairs must never be exempt",
				variant.ID, key, exception.Reason))
			continue
		}
		if strings.TrimSpace(exception.Reason) == "" {
			failures = append(failures, fmt.Sprintf("%s: contrast exception %q has no reason", variant.ID, key))
		}
		if graphicalPairs[key] {
			exempted[key] = true
		} else {
			failures = append(failures, fmt.Sprintf("%s: contrast exception %q does not match a checked pair", variant.ID, key))
		}
	}

	for _, check := range checks {
		if exempted[check.key] {
			continue
		}
		ratio := contrastRatioColours(check.foreground, check.background)
		if ratio < check.minimum {
			failures = append(failures, fmt.Sprintf("%s: %s on %s contrast is %.2f:1, want at least %.1f:1",
				variant.ID, check.role, check.surface, ratio, check.minimum))
		}
	}
	return failures
}

func TestThemeContrast(t *testing.T) {
	_, catalog := readCatalog(t)
	for _, family := range catalog.Families {
		for _, variant := range family.Variants {
			variant := variant
			t.Run(variant.ID, func(t *testing.T) {
				for _, failure := range contrastFailures(variant) {
					t.Error(failure)
				}
			})
		}
	}
}

func TestThemeContrastCoversRequiredPairs(t *testing.T) {
	_, catalog := readCatalog(t)
	for _, family := range catalog.Families {
		for _, variant := range family.Variants {
			checks, failures := themeContrastChecks(variant)
			if len(failures) > 0 {
				t.Fatalf("%s: theme contrast checks failed: %v", variant.ID, failures)
			}
			keys := make(map[string]bool, len(checks))
			for _, check := range checks {
				keys[check.key] = true
			}

			required := []string{
				"ink/bg", "ink/canvasMiddle", "ink/panel", "ink/bubble", "ink/glass",
				"muted/bg", "muted/canvasMiddle", "muted/panel", "muted/bubble", "muted/glass",
				"ink/labelHalo.canvasTop", "ink/labelHalo.canvasMiddle", "ink/labelHalo.canvasBottom",
				"accent/bg", "accent/panel", "online/canvasMiddle", "history/canvasMiddle",
			}
			for name := range variant.Kinds {
				required = append(required, "kind."+name+"/bubble")
			}
			for name := range variant.Categories {
				required = append(required, "category."+name+"/canvasMiddle")
			}
			sort.Strings(required)
			for _, key := range required {
				if !keys[key] {
					t.Errorf("%s: missing contrast pair %q", variant.ID, key)
				}
			}
		}
	}
}

func syntheticContrastVariant(t *testing.T, id string) themeVariant {
	t.Helper()
	_, catalog := readCatalog(t)
	base := findThemeVariant(t, findThemeFamily(t, catalog, "hearthlight"), "hearthlight-dark")
	variant := themeVariant{
		ID:         id,
		Name:       id,
		Mode:       base.Mode,
		Palette:    make(map[string]string, len(base.Palette)),
		Kinds:      make(map[string]themeKind, len(base.Kinds)),
		Categories: make(map[string]string, len(base.Categories)),
		Effects:    make(map[string]any, len(base.Effects)),
	}
	for name, value := range base.Palette {
		variant.Palette[name] = value
	}
	for name, value := range base.Kinds {
		variant.Kinds[name] = value
	}
	for name, value := range base.Categories {
		variant.Categories[name] = value
	}
	for name, value := range base.Effects {
		variant.Effects[name] = value
	}
	return variant
}

func accentContrastExceptions(t *testing.T, variant themeVariant, reason string) []themeContrastException {
	t.Helper()
	checks, failures := themeContrastChecks(variant)
	if len(failures) > 0 {
		t.Fatalf("theme contrast checks failed before exceptions: %v", failures)
	}
	var exceptions []themeContrastException
	for _, check := range checks {
		if check.token == "accent" {
			exceptions = append(exceptions, themeContrastException{Pair: check.key, Reason: reason})
		}
	}
	return exceptions
}

func TestThemeContrastFailureNamesLowMuted(t *testing.T) {
	variant := syntheticContrastVariant(t, "synthetic-low-muted")
	variant.Palette["muted"] = "#232323"
	failures := contrastFailures(variant)
	message := strings.Join(failures, "\n")
	for _, want := range []string{
		"synthetic-low-muted",
		"inkMuted (muted)",
		"on bg",
		"want at least 4.5:1",
	} {
		if !strings.Contains(message, want) {
			t.Errorf("failure message %q does not name %q", message, want)
		}
	}
	if !regexp.MustCompile(`contrast is [0-9]+\.[0-9]+:1`).MatchString(message) {
		t.Errorf("failure message %q does not name the measured ratio", message)
	}
}

func TestThemeContrastRejectsTextException(t *testing.T) {
	variant := syntheticContrastVariant(t, "synthetic-text-exception")
	variant.ContrastExceptions = []themeContrastException{{Pair: "muted/bg", Reason: "quiet labels are decorative"}}
	message := strings.Join(contrastFailures(variant), "\n")
	for _, want := range []string{
		"text pair muted/bg declares an exception",
		"text pairs must never be exempt",
	} {
		if !strings.Contains(message, want) {
			t.Errorf("failure message %q does not name %q", message, want)
		}
	}
}

func TestThemeContrastAcceptsGraphicalException(t *testing.T) {
	variant := syntheticContrastVariant(t, "synthetic-graphical-exception")
	variant.Palette["accent"] = "#101010"
	variant.ContrastExceptions = accentContrastExceptions(t, variant, "accent is decorative in this synthetic case")
	if failures := contrastFailures(variant); len(failures) > 0 {
		t.Fatalf("valid graphical exceptions were not accepted:\n%s", strings.Join(failures, "\n"))
	}
}

func TestThemeContrastRejectsInvalidGraphicalExceptions(t *testing.T) {
	t.Run("empty reason", func(t *testing.T) {
		variant := syntheticContrastVariant(t, "synthetic-empty-reason")
		variant.Palette["accent"] = "#101010"
		variant.ContrastExceptions = accentContrastExceptions(t, variant, "")
		message := strings.Join(contrastFailures(variant), "\n")
		if !strings.Contains(message, "has no reason") {
			t.Errorf("failure message %q does not reject the empty reason", message)
		}
	})

	t.Run("unmatched pair", func(t *testing.T) {
		variant := syntheticContrastVariant(t, "synthetic-unmatched-exception")
		variant.ContrastExceptions = []themeContrastException{{Pair: "kind.router/nowhere", Reason: "not a checked pair"}}
		message := strings.Join(contrastFailures(variant), "\n")
		if !strings.Contains(message, "does not match a checked pair") {
			t.Errorf("failure message %q does not reject the unmatched pair", message)
		}
	})

	t.Run("duplicate pair", func(t *testing.T) {
		variant := syntheticContrastVariant(t, "synthetic-duplicate-exception")
		variant.ContrastExceptions = []themeContrastException{
			{Pair: "accent/bg", Reason: "first declaration"},
			{Pair: "accent/bg", Reason: "second declaration"},
		}
		message := strings.Join(contrastFailures(variant), "\n")
		if !strings.Contains(message, "duplicate contrast exception") {
			t.Errorf("failure message %q does not reject the duplicate pair", message)
		}
	})

	t.Run("empty pair", func(t *testing.T) {
		variant := syntheticContrastVariant(t, "synthetic-empty-pair")
		variant.ContrastExceptions = []themeContrastException{{Pair: "  ", Reason: "no pair named"}}
		message := strings.Join(contrastFailures(variant), "\n")
		if !strings.Contains(message, "has an empty pair") {
			t.Errorf("failure message %q does not reject the empty pair", message)
		}
	})
}

type effectRange struct {
	min, max float64
}

const effectBlurMaximum = 64.0

var themeEffectRanges = map[string]effectRange{
	"vignette":       {min: 0, max: 1},
	"deviceGlow":     {min: 0, max: 1},
	"devicePulse":    {min: 0, max: 1},
	"serviceGlow":    {min: 0, max: 1},
	"territoryBlur":  {min: 0, max: effectBlurMaximum},
	"ribbonBlur":     {min: 0, max: effectBlurMaximum},
	"lensBlur":       {min: 0, max: effectBlurMaximum},
	"glassBlur":      {min: 0, max: effectBlurMaximum},
	"panelBlur":      {min: 0, max: effectBlurMaximum},
	"territoryAlpha": {min: 0, max: 1},
	"glowStrength":   {min: 0, max: 1},
	"labelHalo":      {min: 0, max: 1},
	"shadowStrength": {min: 0, max: 1},
}

var themeEffectChoices = map[string]map[string]bool{
	"territoryBlend": {"screen": true, "multiply": true},
}

var themeEffectFlags = map[string]bool{
	"highContrast": true,
}

func TestThemeEffectsAreInRange(t *testing.T) {
	_, catalog := readCatalog(t)
	for _, family := range catalog.Families {
		for _, variant := range family.Variants {
			for name, value := range variant.Effects {
				switch typed := value.(type) {
				case float64:
					limits, ok := themeEffectRanges[name]
					if !ok {
						t.Errorf("%s: effect %q has no documented range", variant.ID, name)
						continue
					}
					if math.IsNaN(typed) || math.IsInf(typed, 0) || typed < limits.min || typed > limits.max {
						t.Errorf("%s: effect %q = %v, want between %v and %v",
							variant.ID, name, typed, limits.min, limits.max)
					}
				case string:
					allowed, ok := themeEffectChoices[name]
					if !ok {
						t.Errorf("%s: effect %q has no documented choices", variant.ID, name)
						continue
					}
					if !allowed[typed] {
						t.Errorf("%s: effect %q = %q, want one of %v", variant.ID, name, typed, sortedChoices(allowed))
					}
				case bool:
					if !themeEffectFlags[name] {
						t.Errorf("%s: effect %q = %v, want a documented flag", variant.ID, name, typed)
					}
				default:
					t.Errorf("%s: effect %q = %#v, want a number, string, or flag", variant.ID, name, value)
				}
			}
		}
	}
}

func sortedChoices(choices map[string]bool) []string {
	names := make([]string, 0, len(choices))
	for name := range choices {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func TestThemeCatalogIDsAndModes(t *testing.T) {
	_, catalog := readCatalog(t)

	familyIDs := make(map[string]bool, len(catalog.Families))
	variantIDs := make(map[string]bool)
	for _, family := range catalog.Families {
		if strings.TrimSpace(family.ID) == "" {
			t.Error("theme family has an empty id")
		}
		if familyIDs[family.ID] {
			t.Errorf("theme family id %q is not unique", family.ID)
		}
		familyIDs[family.ID] = true
		if len(family.Variants) == 0 {
			t.Errorf("theme family %q has no variants", family.ID)
		}
		for _, variant := range family.Variants {
			if strings.TrimSpace(variant.ID) == "" {
				t.Errorf("theme family %q has a variant with an empty id", family.ID)
			}
			if variantIDs[variant.ID] {
				t.Errorf("theme variant id %q is not unique", variant.ID)
			}
			variantIDs[variant.ID] = true
			if variant.Mode != "dark" && variant.Mode != "light" {
				t.Errorf("%s: mode = %q, want dark or light", variant.ID, variant.Mode)
			}
		}
	}
}

type omarchyMapping struct {
	SchemaVersion int               `json:"schemaVersion"`
	Themes        map[string]string `json:"themes"`
	Modes         map[string]string `json:"modes"`
}

func TestOmarchyMapping(t *testing.T) {
	raw, err := os.ReadFile("omarchy.json")
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("web/omarchy.json is not present")
	}
	if err != nil {
		t.Fatalf("read omarchy.json: %v", err)
	}
	var mapping omarchyMapping
	if err := json.Unmarshal(raw, &mapping); err != nil {
		t.Fatalf("parse omarchy.json: %v", err)
	}
	if mapping.SchemaVersion != 1 {
		t.Fatalf("omarchy schemaVersion = %d, want 1", mapping.SchemaVersion)
	}
	if len(mapping.Themes) == 0 {
		t.Fatal("omarchy.json has no theme mappings")
	}

	_, catalog := readCatalog(t)
	variants := make(map[string]themeVariant)
	for _, family := range catalog.Families {
		for _, variant := range family.Variants {
			variants[variant.ID] = variant
		}
	}
	for name, id := range mapping.Themes {
		variant, ok := variants[id]
		if !ok {
			t.Errorf("omarchy theme %q maps to unknown variant %q", name, id)
			continue
		}
		mode, ok := mapping.Modes[name]
		if !ok {
			continue
		}
		if mode != "dark" && mode != "light" {
			t.Errorf("omarchy theme %q mode = %q, want dark or light", name, mode)
			continue
		}
		if variant.Mode != mode {
			t.Errorf("omarchy theme %q maps to %s (%s mode), want %s mode", name, id, variant.Mode, mode)
		}
	}
	for name := range mapping.Modes {
		if _, ok := mapping.Themes[name]; !ok {
			t.Errorf("omarchy mode %q has no theme mapping", name)
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

// Light variants use multiply blending and reduced glow so a pale canvas does
// not bloom; all batch-A light variants share these values.
var wantLightEffects = map[string]any{
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

var wantGruvboxLightPalette = map[string]string{
	"bg":           "#fbf1c7",
	"ink":          "#3c3836",
	"muted":        "#665c54",
	"accent":       "#076678",
	"line":         "rgba(60, 56, 54, 0.16)",
	"glass":        "rgba(251, 241, 199, 0.82)",
	"canvasTop":    "#fdf6d8",
	"canvasMiddle": "#fbf1c7",
	"canvasBottom": "#f5eac0",
	"grid":         "rgba(7, 102, 120, 0.07)",
	"ambientGlow":  "rgba(7, 102, 120, 0.12)",
	"online":       "#79740e",
	"warning":      "#b57614",
	"danger":       "#9d0006",
	"history":      "#928374",
	"vignette":     "rgba(213, 196, 161, 0.55)",
	"labelHalo":    "#fffdf2",
	"bubble":       "rgba(255, 251, 235, 0.92)",
	"panel":        "rgba(253, 246, 221, 0.97)",
}

var wantGruvboxLightKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#b57614"},
	"server":   {Label: "Server", Color: "#076678"},
	"computer": {Label: "Computer", Color: "#427b58"},
	"phone":    {Label: "Phone", Color: "#8f3f71"},
	"nas":      {Label: "NAS", Color: "#79740e"},
	"media":    {Label: "Media", Color: "#7a4a9e"},
	"iot":      {Label: "IoT", Color: "#2f6f5a"},
	"printer":  {Label: "Printer", Color: "#af3a03"},
	"device":   {Label: "Device", Color: "#928374"},
}

var wantGruvboxLightCategories = map[string]string{
	"web":    "#076678",
	"media":  "#7a4a9e",
	"data":   "#af3a03",
	"remote": "#79740e",
	"infra":  "#427b58",
	"iot":    "#8f3f71",
	"other":  "#928374",
}

var wantNordLightPalette = map[string]string{
	"bg":           "#eceff4",
	"ink":          "#2e3440",
	"muted":        "#4c566a",
	"accent":       "#5e81ac",
	"line":         "rgba(46, 52, 64, 0.16)",
	"glass":        "rgba(236, 239, 244, 0.82)",
	"canvasTop":    "#f4f6f9",
	"canvasMiddle": "#eceff4",
	"canvasBottom": "#e0e4eb",
	"grid":         "rgba(94, 129, 172, 0.07)",
	"ambientGlow":  "rgba(136, 192, 208, 0.14)",
	"online":       "#4c7a2a",
	"warning":      "#8a6a1f",
	"danger":       "#bf616a",
	"history":      "#5f6b7d",
	"vignette":     "rgba(216, 222, 233, 0.55)",
	"labelHalo":    "#fbfcfd",
	"bubble":       "rgba(250, 252, 254, 0.92)",
	"panel":        "rgba(244, 246, 249, 0.97)",
}

var wantNordLightKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#8a6a1f"},
	"server":   {Label: "Server", Color: "#5e81ac"},
	"computer": {Label: "Computer", Color: "#3f6f8f"},
	"phone":    {Label: "Phone", Color: "#8a5f86"},
	"nas":      {Label: "NAS", Color: "#4c7a2a"},
	"media":    {Label: "Media", Color: "#6f5a8a"},
	"iot":      {Label: "IoT", Color: "#2f736f"},
	"printer":  {Label: "Printer", Color: "#a35a3d"},
	"device":   {Label: "Device", Color: "#5f6b7d"},
}

var wantNordLightCategories = map[string]string{
	"web":    "#2f736f",
	"media":  "#6f5a8a",
	"data":   "#a35a3d",
	"remote": "#4c7a2a",
	"infra":  "#5e81ac",
	"iot":    "#4d817c",
	"other":  "#5f6b7d",
}

var wantCatppuccinLattePalette = map[string]string{
	"bg":           "#eff1f5",
	"ink":          "#4c4f69",
	"muted":        "#5c5f77",
	"accent":       "#1e66f5",
	"line":         "rgba(76, 79, 105, 0.16)",
	"glass":        "rgba(239, 241, 245, 0.82)",
	"canvasTop":    "#f7f8fb",
	"canvasMiddle": "#eff1f5",
	"canvasBottom": "#e6e9ef",
	"grid":         "rgba(30, 102, 245, 0.07)",
	"ambientGlow":  "rgba(114, 135, 253, 0.14)",
	"online":       "#2f7d20",
	"warning":      "#9a6400",
	"danger":       "#d20f39",
	"history":      "#7c7f93",
	"vignette":     "rgba(220, 224, 232, 0.55)",
	"labelHalo":    "#fbfcfe",
	"bubble":       "rgba(255, 255, 255, 0.92)",
	"panel":        "rgba(248, 249, 252, 0.97)",
}

var wantCatppuccinLatteKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#9a6400"},
	"server":   {Label: "Server", Color: "#179299"},
	"computer": {Label: "Computer", Color: "#1e66f5"},
	"phone":    {Label: "Phone", Color: "#c0266b"},
	"nas":      {Label: "NAS", Color: "#2f7d20"},
	"media":    {Label: "Media", Color: "#8839ef"},
	"iot":      {Label: "IoT", Color: "#1a7f6e"},
	"printer":  {Label: "Printer", Color: "#b34a00"},
	"device":   {Label: "Device", Color: "#6c6f85"},
}

var wantCatppuccinLatteCategories = map[string]string{
	"web":    "#179299",
	"media":  "#8839ef",
	"data":   "#b34a00",
	"remote": "#2f7d20",
	"infra":  "#1e66f5",
	"iot":    "#1a7f6e",
	"other":  "#6c6f85",
}

var wantTokyoNightDayPalette = map[string]string{
	"bg":           "#e1e2e7",
	"ink":          "#3760bf",
	"muted":        "#4e5c8f",
	"accent":       "#2e7de9",
	"line":         "rgba(55, 96, 191, 0.16)",
	"glass":        "rgba(225, 226, 231, 0.82)",
	"canvasTop":    "#eff0f4",
	"canvasMiddle": "#e1e2e7",
	"canvasBottom": "#e0e2e9",
	"grid":         "rgba(46, 125, 233, 0.07)",
	"ambientGlow":  "rgba(46, 125, 233, 0.12)",
	"online":       "#587539",
	"warning":      "#b15c00",
	"danger":       "#ef245f",
	"history":      "#5f6685",
	"vignette":     "rgba(208, 213, 227, 0.55)",
	"labelHalo":    "#fafbfd",
	"bubble":       "rgba(248, 249, 252, 0.92)",
	"panel":        "rgba(236, 237, 242, 0.97)",
}

var wantTokyoNightDayKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#8c6c3e"},
	"server":   {Label: "Server", Color: "#007197"},
	"computer": {Label: "Computer", Color: "#2e7de9"},
	"phone":    {Label: "Phone", Color: "#9854f1"},
	"nas":      {Label: "NAS", Color: "#587539"},
	"media":    {Label: "Media", Color: "#7847bd"},
	"iot":      {Label: "IoT", Color: "#118c74"},
	"printer":  {Label: "Printer", Color: "#b15c00"},
	"device":   {Label: "Device", Color: "#5f6685"},
}

var wantTokyoNightDayCategories = map[string]string{
	"web":    "#007197",
	"media":  "#7847bd",
	"data":   "#b15c00",
	"remote": "#587539",
	"infra":  "#2e7de9",
	"iot":    "#118c74",
	"other":  "#5f6685",
}

var wantRosePineDawnPalette = map[string]string{
	"bg":           "#faf4ed",
	"ink":          "#575279",
	"muted":        "#6e6a86",
	"accent":       "#286983",
	"line":         "rgba(87, 82, 121, 0.16)",
	"glass":        "rgba(250, 244, 237, 0.82)",
	"canvasTop":    "#fffaf3",
	"canvasMiddle": "#faf4ed",
	"canvasBottom": "#f7efe8",
	"grid":         "rgba(40, 105, 131, 0.07)",
	"ambientGlow":  "rgba(86, 148, 159, 0.14)",
	"online":       "#2f7a4a",
	"warning":      "#a36500",
	"danger":       "#b4637a",
	"history":      "#797593",
	"vignette":     "rgba(242, 233, 225, 0.55)",
	"labelHalo":    "#fffcf7",
	"bubble":       "rgba(255, 255, 255, 0.92)",
	"panel":        "rgba(255, 250, 243, 0.97)",
}

var wantRosePineDawnKinds = map[string]themeKind{
	"router":   {Label: "Router", Color: "#a36500"},
	"server":   {Label: "Server", Color: "#286983"},
	"computer": {Label: "Computer", Color: "#3e6f9e"},
	"phone":    {Label: "Phone", Color: "#b4637a"},
	"nas":      {Label: "NAS", Color: "#2f7a4a"},
	"media":    {Label: "Media", Color: "#907aa9"},
	"iot":      {Label: "IoT", Color: "#56949f"},
	"printer":  {Label: "Printer", Color: "#a35d43"},
	"device":   {Label: "Device", Color: "#6e6a86"},
}

var wantRosePineDawnCategories = map[string]string{
	"web":    "#357f8c",
	"media":  "#907aa9",
	"data":   "#a36500",
	"remote": "#2f7a4a",
	"infra":  "#286983",
	"iot":    "#56949f",
	"other":  "#6e6a86",
}

func TestGruvboxLightVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "gruvbox")
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := family.Variants[1]
	if variant.ID != "gruvbox-light" || variant.Name != "Gruvbox Light" || variant.Mode != "light" {
		t.Fatalf("variant = %+v, want Gruvbox Light light", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantGruvboxLightPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantGruvboxLightPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantGruvboxLightKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantGruvboxLightKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantGruvboxLightCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantGruvboxLightCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantLightEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantLightEffects)
	}
}

func TestNordLightVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "nord")
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := family.Variants[1]
	if variant.ID != "nord-light" || variant.Name != "Nord Light" || variant.Mode != "light" {
		t.Fatalf("variant = %+v, want Nord Light light", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantNordLightPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantNordLightPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantNordLightKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantNordLightKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantNordLightCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantNordLightCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantLightEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantLightEffects)
	}
}

func TestCatppuccinLatteVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "catppuccin")
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := family.Variants[1]
	if variant.ID != "catppuccin-latte" || variant.Name != "Catppuccin Latte" || variant.Mode != "light" {
		t.Fatalf("variant = %+v, want Catppuccin Latte light", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantCatppuccinLattePalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantCatppuccinLattePalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantCatppuccinLatteKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantCatppuccinLatteKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantCatppuccinLatteCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantCatppuccinLatteCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantLightEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantLightEffects)
	}
}

func TestTokyoNightDayVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "tokyo-night")
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := family.Variants[1]
	if variant.ID != "tokyo-night-day" || variant.Name != "Tokyo Night Day" || variant.Mode != "light" {
		t.Fatalf("variant = %+v, want Tokyo Night Day light", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantTokyoNightDayPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantTokyoNightDayPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantTokyoNightDayKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantTokyoNightDayKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantTokyoNightDayCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantTokyoNightDayCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantLightEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantLightEffects)
	}
}

func TestRosePineDawnVariant(t *testing.T) {
	_, catalog := readCatalog(t)
	family := findThemeFamily(t, catalog, "rose-pine")
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	variant := family.Variants[1]
	if variant.ID != "rose-pine-dawn" || variant.Name != "Rosé Pine Dawn" || variant.Mode != "light" {
		t.Fatalf("variant = %+v, want Rosé Pine Dawn light", variant)
	}
	if !reflect.DeepEqual(variant.Palette, wantRosePineDawnPalette) {
		t.Fatalf("palette = %#v, want %#v", variant.Palette, wantRosePineDawnPalette)
	}
	if !reflect.DeepEqual(variant.Kinds, wantRosePineDawnKinds) {
		t.Fatalf("kinds = %#v, want %#v", variant.Kinds, wantRosePineDawnKinds)
	}
	if !reflect.DeepEqual(variant.Categories, wantRosePineDawnCategories) {
		t.Fatalf("categories = %#v, want %#v", variant.Categories, wantRosePineDawnCategories)
	}
	if !reflect.DeepEqual(variant.Effects, wantLightEffects) {
		t.Fatalf("effects = %#v, want %#v", variant.Effects, wantLightEffects)
	}
}
