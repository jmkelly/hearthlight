// Package theme parses and validates Hearthlight theme imports.
//
// It is deliberately local-only. Nothing here reads files, fetches remote
// content, or evaluates input as code. Parsers cap input size and reject keys
// that could pollute JavaScript object prototypes when a theme is stored and
// later merged in the browser.
package theme

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// MaxImportBytes caps a single imported theme document at 256 KiB.
	MaxImportBytes = 256 * 1024
	// MaxNameLength caps family, variant, source, and kind labels.
	MaxNameLength = 64
	// MaxIDLength caps family and variant identifiers.
	MaxIDLength = 64
	// SchemaVersion is the only Hearthlight catalog schema this parser accepts.
	SchemaVersion = 1
)

// RequiredPaletteTokens are the semantic colours every variant must define.
var RequiredPaletteTokens = [...]string{
	"bg", "ink", "muted", "accent", "line", "glass",
	"canvasTop", "canvasMiddle", "canvasBottom", "grid", "ambientGlow",
	"online", "warning", "danger", "history",
	"vignette", "labelHalo", "bubble", "panel",
}

// ErrorCategory classifies an import or validation failure so callers can show
// a useful message without matching on error strings.
type ErrorCategory string

const (
	ErrTooLarge          ErrorCategory = "too_large"
	ErrMalformed         ErrorCategory = "malformed"
	ErrInvalidShape      ErrorCategory = "invalid_shape"
	ErrForbiddenKey      ErrorCategory = "forbidden_key"
	ErrMissingToken      ErrorCategory = "missing_token"
	ErrInvalidColor      ErrorCategory = "invalid_color"
	ErrInvalidMode       ErrorCategory = "invalid_mode"
	ErrInvalidEffect     ErrorCategory = "invalid_effect"
	ErrInvalidID         ErrorCategory = "invalid_id"
	ErrInvalidName       ErrorCategory = "invalid_name"
	ErrInvalidKey        ErrorCategory = "invalid_key"
	ErrDuplicateID       ErrorCategory = "duplicate_id"
	ErrDuplicateMode     ErrorCategory = "duplicate_mode"
	ErrUnsupportedSchema ErrorCategory = "unsupported_schema"
)

// ValidationError is returned for every rejected import.
type ValidationError struct {
	Category ErrorCategory
	Path     string
	Detail   string
}

func (e *ValidationError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("theme %s at %s: %s", e.Category, e.Path, e.Detail)
	}
	return fmt.Sprintf("theme %s: %s", e.Category, e.Detail)
}

// Catalog is the validated import result. It matches the shape of
// web/themes.json with variants always represented as a list.
type Catalog struct {
	SchemaVersion int      `json:"schemaVersion"`
	Families      []Family `json:"families"`
}

// Family is one theme family with one or two variants.
type Family struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Source   *Source   `json:"source,omitempty"`
	Variants []Variant `json:"variants"`
}

// Source records optional theme credit. It is never fetched or resolved.
type Source struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

// Variant is one dark or light palette.
type Variant struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Mode       string            `json:"mode"`
	Palette    map[string]string `json:"palette"`
	Kinds      map[string]Kind   `json:"kinds,omitempty"`
	Categories map[string]string `json:"categories,omitempty"`
	Effects    Effects           `json:"effects"`
}

// Kind is a device kind label and colour.
type Kind struct {
	Label string `json:"label"`
	Color string `json:"color"`
}

// Effects holds the canvas effect values the renderer understands.
type Effects struct {
	Vignette       float64 `json:"vignette"`
	DeviceGlow     float64 `json:"deviceGlow"`
	DevicePulse    float64 `json:"devicePulse"`
	ServiceGlow    float64 `json:"serviceGlow"`
	TerritoryBlur  float64 `json:"territoryBlur"`
	RibbonBlur     float64 `json:"ribbonBlur"`
	LensBlur       float64 `json:"lensBlur"`
	GlassBlur      float64 `json:"glassBlur"`
	PanelBlur      float64 `json:"panelBlur"`
	TerritoryBlend string  `json:"territoryBlend"`
	TerritoryAlpha float64 `json:"territoryAlpha"`
	GlowStrength   float64 `json:"glowStrength"`
	LabelHalo      float64 `json:"labelHalo"`
	ShadowStrength float64 `json:"shadowStrength"`
}

type effectRange struct {
	min float64
	max float64
}

var numericEffectRanges = map[string]effectRange{
	"vignette":       {0, 1},
	"deviceGlow":     {0, 1},
	"devicePulse":    {0, 1},
	"serviceGlow":    {0, 1},
	"territoryBlur":  {0, 64},
	"ribbonBlur":     {0, 64},
	"lensBlur":       {0, 64},
	"glassBlur":      {0, 64},
	"panelBlur":      {0, 64},
	"territoryAlpha": {0, 1},
	"glowStrength":   {0, 4},
	"labelHalo":      {0, 1},
	"shadowStrength": {0, 4},
}

var blendModes = map[string]bool{
	"normal":      true,
	"source-over": true,
	"multiply":    true,
	"screen":      true,
	"overlay":     true,
	"darken":      true,
	"lighten":     true,
	"color-dodge": true,
	"color-burn":  true,
	"hard-light":  true,
	"soft-light":  true,
	"difference":  true,
	"exclusion":   true,
	"hue":         true,
	"saturation":  true,
	"color":       true,
	"luminosity":  true,
}

var forbiddenKeys = map[string]bool{
	"__proto__":   true,
	"constructor": true,
	"prototype":   true,
}

var (
	idPattern          = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`)
	mapKeyPattern      = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
	paletteKeyPattern  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
	rgbFunctionPattern = regexp.MustCompile(`^rgba?\(\s*([0-9]{1,3})\s*,\s*([0-9]{1,3})\s*,\s*([0-9]{1,3})\s*(?:,\s*([0-9]*\.?[0-9]+)\s*)?\)$`)
	hexColorPattern    = regexp.MustCompile(`^[0-9A-Fa-f]+$`)
)

type rgbaColor struct {
	r uint8
	g uint8
	b uint8
	a float64
}

func fail(category ErrorCategory, path, format string, args ...any) *ValidationError {
	return &ValidationError{Category: category, Path: path, Detail: fmt.Sprintf(format, args...)}
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func validateName(value, path string) *ValidationError {
	name := strings.TrimSpace(value)
	if name == "" {
		return fail(ErrInvalidName, path, "name must not be empty")
	}
	if utf8.RuneCountInString(name) > MaxNameLength {
		return fail(ErrInvalidName, path, "name is %d characters, maximum is %d", utf8.RuneCountInString(name), MaxNameLength)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fail(ErrInvalidName, path, "name must not contain control characters")
		}
	}
	return nil
}

func validateID(value, path string) *ValidationError {
	id := strings.TrimSpace(value)
	if id == "" {
		return fail(ErrInvalidID, path, "id must not be empty")
	}
	if len(id) > MaxIDLength {
		return fail(ErrInvalidID, path, "id is %d bytes, maximum is %d", len(id), MaxIDLength)
	}
	if !idPattern.MatchString(id) {
		return fail(ErrInvalidID, path, "id %q must start with a letter or digit and contain only letters, digits, dot, underscore, hyphen, or colon", id)
	}
	return nil
}

func validateMapKey(value, path string, pattern *regexp.Regexp) *ValidationError {
	if forbiddenKeys[value] {
		return fail(ErrForbiddenKey, path, "key %q is not allowed", value)
	}
	if len(value) > MaxNameLength || !pattern.MatchString(value) {
		return fail(ErrInvalidKey, path, "key %q is not a valid token name", value)
	}
	return nil
}

func validColor(value string) bool {
	_, ok := parseColor(value)
	return ok
}

func parseColor(value string) (rgbaColor, bool) {
	text := strings.TrimSpace(value)
	if text == "" {
		return rgbaColor{}, false
	}
	if text[0] == '#' {
		return parseHexColor(text)
	}
	return parseFunctionColor(text)
}

func parseHexColor(value string) (rgbaColor, bool) {
	digits := value[1:]
	if !hexColorPattern.MatchString(digits) {
		return rgbaColor{}, false
	}
	switch len(digits) {
	case 3, 4:
		expanded := make([]byte, 0, len(digits)*2)
		for i := 0; i < len(digits); i++ {
			expanded = append(expanded, digits[i], digits[i])
		}
		digits = string(expanded)
	case 6, 8:
	default:
		return rgbaColor{}, false
	}
	channel := func(offset int) (uint8, bool) {
		raw, err := strconv.ParseUint(digits[offset:offset+2], 16, 8)
		if err != nil {
			return 0, false
		}
		return uint8(raw), true
	}
	r, ok := channel(0)
	if !ok {
		return rgbaColor{}, false
	}
	g, ok := channel(2)
	if !ok {
		return rgbaColor{}, false
	}
	b, ok := channel(4)
	if !ok {
		return rgbaColor{}, false
	}
	alpha := 1.0
	if len(digits) == 8 {
		raw, err := strconv.ParseUint(digits[6:8], 16, 8)
		if err != nil {
			return rgbaColor{}, false
		}
		alpha = float64(raw) / 255
	}
	return rgbaColor{r: r, g: g, b: b, a: alpha}, true
}

func parseFunctionColor(value string) (rgbaColor, bool) {
	match := rgbFunctionPattern.FindStringSubmatch(value)
	if match == nil {
		return rgbaColor{}, false
	}
	channels := [3]uint8{}
	for i := 0; i < 3; i++ {
		number, err := strconv.ParseUint(match[i+1], 10, 8)
		if err != nil || number > 255 {
			return rgbaColor{}, false
		}
		channels[i] = uint8(number)
	}
	alpha := 1.0
	if match[4] != "" {
		number, err := strconv.ParseFloat(match[4], 64)
		if err != nil || math.IsNaN(number) || number < 0 || number > 1 {
			return rgbaColor{}, false
		}
		alpha = number
	}
	return rgbaColor{r: channels[0], g: channels[1], b: channels[2], a: alpha}, true
}

func formatHex(color rgbaColor) string {
	if color.a < 1 {
		return fmt.Sprintf("#%02x%02x%02x%02x", color.r, color.g, color.b, uint8(math.Round(color.a*255)))
	}
	return fmt.Sprintf("#%02x%02x%02x", color.r, color.g, color.b)
}

func rgbaString(color rgbaColor, alpha float64) string {
	value := color.a * alpha
	return fmt.Sprintf("rgba(%d, %d, %d, %s)", color.r, color.g, color.b, strconv.FormatFloat(value, 'f', -1, 64))
}

func mixColor(a, b rgbaColor, amount float64) rgbaColor {
	channel := func(left, right uint8) uint8 {
		return uint8(math.Round(float64(left)*(1-amount) + float64(right)*amount))
	}
	return rgbaColor{
		r: channel(a.r, b.r),
		g: channel(a.g, b.g),
		b: channel(a.b, b.b),
		a: a.a*(1-amount) + b.a*amount,
	}
}

func lighten(color rgbaColor, amount float64) rgbaColor {
	return mixColor(color, rgbaColor{r: 255, g: 255, b: 255, a: 1}, amount)
}

func darken(color rgbaColor, amount float64) rgbaColor {
	return mixColor(color, rgbaColor{a: 1}, amount)
}

func relativeLuminance(color rgbaColor) float64 {
	linear := func(channel uint8) float64 {
		value := float64(channel) / 255
		if value <= 0.03928 {
			return value / 12.92
		}
		return math.Pow((value+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(color.r) + 0.7152*linear(color.g) + 0.0722*linear(color.b)
}

func defaultEffects(mode string) Effects {
	if mode == "light" {
		return Effects{
			Vignette:       0.3,
			DeviceGlow:     0.14,
			DevicePulse:    0.22,
			ServiceGlow:    0.08,
			TerritoryBlur:  26,
			RibbonBlur:     16,
			LensBlur:       14,
			GlassBlur:      16,
			PanelBlur:      22,
			TerritoryBlend: "multiply",
			TerritoryAlpha: 0.72,
			GlowStrength:   0.6,
			LabelHalo:      0.9,
			ShadowStrength: 0.5,
		}
	}
	return Effects{
		Vignette:       0.52,
		DeviceGlow:     0.22,
		DevicePulse:    0.3,
		ServiceGlow:    0.16,
		TerritoryBlur:  26,
		RibbonBlur:     16,
		LensBlur:       14,
		GlassBlur:      16,
		PanelBlur:      22,
		TerritoryBlend: "screen",
		TerritoryAlpha: 1,
		GlowStrength:   1,
		LabelHalo:      1,
		ShadowStrength: 1,
	}
}

func defaultPalette(mode string) map[string]string {
	if mode == "light" {
		return map[string]string{
			"bg":           "#f2f5f9",
			"ink":          "#0f1c2b",
			"muted":        "#4f6275",
			"accent":       "#076b84",
			"line":         "rgba(21, 54, 84, 0.16)",
			"glass":        "rgba(250, 253, 255, 0.82)",
			"canvasTop":    "#f8fbff",
			"canvasMiddle": "#eef3f9",
			"canvasBottom": "#e2eaf3",
			"grid":         "rgba(32, 71, 110, 0.07)",
			"ambientGlow":  "rgba(90, 170, 210, 0.16)",
			"online":       "#0a7448",
			"warning":      "#8f5c00",
			"danger":       "#b02a22",
			"history":      "#5d6c7f",
			"vignette":     "rgba(203, 217, 232, 0.55)",
			"labelHalo":    "#fbfdff",
			"bubble":       "rgba(255, 255, 255, 0.92)",
			"panel":        "rgba(250, 252, 255, 0.97)",
		}
	}
	return map[string]string{
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
}
