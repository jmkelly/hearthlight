package theme

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// OmarchyResult carries the parsed catalog plus import notes for the UI.
// Missing lists source keys that were derived or defaulted. Ignored lists
// unknown top-level keys that were skipped without failing the import.
type OmarchyResult struct {
	Catalog Catalog
	Missing []string
	Ignored []string
}

var omarchyColorKeys = map[string]bool{
	"background":         true,
	"dark_background":    true,
	"darker_background":  true,
	"lighter_background": true,
	"foreground":         true,
	"bright_foreground":  true,
	"light_foreground":   true,
	"dark_foreground":    true,
	"accent":             true,
	"red":                true,
	"green":              true,
	"yellow":             true,
	"orange":             true,
	"blue":               true,
	"cyan":               true,
	"magenta":            true,
}

var omarchyRequiredSources = []struct {
	name string
	keys []string
}{
	{"background", []string{"background", "dark_background", "darker_background", "lighter_background"}},
	{"foreground", []string{"foreground", "bright_foreground", "light_foreground", "dark_foreground"}},
	{"accent", []string{"accent", "blue", "cyan"}},
	{"green", []string{"green"}},
	{"yellow", []string{"yellow", "orange"}},
	{"red", []string{"red"}},
	{"blue", []string{"blue", "cyan"}},
	{"magenta", []string{"magenta"}},
}

var (
	tomlDoubleQuoted = regexp.MustCompile(`^"((?:[^"\\]|\\.)*)"\s*(?:#.*)?$`)
	tomlSingleQuoted = regexp.MustCompile(`^'([^']*)'\s*(?:#.*)?$`)
)

// ParseOmarchy parses the simple Omarchy colors.toml subset. It accepts
// quoted hex or rgb/rgba colours and an optional mode. Comments, unknown keys,
// and nested tables are skipped. Missing source keys are derived from the
// supplied palette or from the matching built-in Hearthlight defaults, and
// reported in OmarchyResult.Missing instead of failing the import.
func ParseOmarchy(data []byte) (OmarchyResult, error) {
	raw, sizeErr := boundedInput(data)
	if sizeErr != nil {
		return OmarchyResult{}, sizeErr
	}
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})

	colors := make(map[string]string)
	ignored := make([]string, 0)
	ignoredSeen := make(map[string]bool)
	mode := ""
	inTable := false

	for lineNumber, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSuffix(line, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			if !strings.HasSuffix(trimmed, "]") {
				return OmarchyResult{}, fail(ErrMalformed, linePath(lineNumber), "unterminated table header")
			}
			inTable = true
			continue
		}
		if inTable {
			continue
		}
		equals := strings.IndexByte(trimmed, '=')
		if equals < 0 {
			return OmarchyResult{}, fail(ErrMalformed, linePath(lineNumber), "expected key = value")
		}
		key, ok := tomlKey(trimmed[:equals])
		if !ok {
			return OmarchyResult{}, fail(ErrMalformed, linePath(lineNumber), "invalid key")
		}
		if key == "" {
			return OmarchyResult{}, fail(ErrMalformed, linePath(lineNumber), "key must not be empty")
		}
		if forbiddenKeys[key] {
			return OmarchyResult{}, fail(ErrForbiddenKey, linePath(lineNumber), "key %q is not allowed", key)
		}
		if key != "mode" && !omarchyColorKeys[key] {
			if !ignoredSeen[key] {
				ignoredSeen[key] = true
				ignored = append(ignored, key)
			}
			continue
		}
		value, ok := tomlString(trimmed[equals+1:])
		if !ok {
			return OmarchyResult{}, fail(ErrMalformed, linePath(lineNumber), "value for %s must be a quoted string", key)
		}
		if key == "mode" {
			if value != "dark" && value != "light" {
				return OmarchyResult{}, fail(ErrInvalidMode, linePath(lineNumber)+".mode", "mode %q must be dark or light", value)
			}
			mode = value
			continue
		}
		if !validColor(value) {
			return OmarchyResult{}, fail(ErrInvalidColor, linePath(lineNumber)+"."+key, "%q is not a valid hex or rgb/rgba colour", value)
		}
		colors[key] = strings.TrimSpace(value)
	}
	if len(colors) == 0 {
		return OmarchyResult{}, fail(ErrMalformed, "$", "no colour keys found")
	}

	missing := missingOmarchySources(colors)
	background, hasBackground := firstOmarchyColor(colors, "background", "dark_background", "darker_background", "lighter_background")
	if mode == "" {
		if hasBackground && relativeLuminance(background) > 0.5 {
			mode = "light"
		} else {
			mode = "dark"
		}
		missing = append(missing, "mode")
	}

	palette := omarchyPalette(colors, mode, background, hasBackground)
	hash := sha256.Sum256([]byte(omarchyIdentity(colors, mode)))
	familyID := fmt.Sprintf("omarchy-%x", hash[:6])
	variantID := familyID + "-" + mode
	familyName := "Omarchy Import"
	variantName := "Omarchy Import " + strings.ToUpper(mode[:1]) + mode[1:]

	catalog := Catalog{
		SchemaVersion: SchemaVersion,
		Families: []Family{{
			ID:   familyID,
			Name: familyName,
			Variants: []Variant{{
				ID:         variantID,
				Name:       variantName,
				Mode:       mode,
				Palette:    palette,
				Kinds:      omarchyKinds(colors, palette),
				Categories: omarchyCategories(colors, palette),
				Effects:    defaultEffects(mode),
			}},
		}},
	}
	if err := validateCatalog(catalog); err != nil {
		return OmarchyResult{}, err
	}
	return OmarchyResult{Catalog: catalog, Missing: missing, Ignored: ignored}, nil
}

func linePath(lineNumber int) string {
	return "line " + strconv.Itoa(lineNumber+1)
}

func omarchyIdentity(colors map[string]string, mode string) string {
	keys := make([]string, 0, len(colors))
	for key := range colors {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var identity strings.Builder
	identity.WriteString("mode=")
	identity.WriteString(mode)
	identity.WriteByte('\n')
	for _, key := range keys {
		color, _ := parseColor(colors[key])
		identity.WriteString(key)
		identity.WriteByte('=')
		identity.WriteString(formatHex(color))
		identity.WriteByte('\n')
	}
	return identity.String()
}

func tomlString(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if match := tomlDoubleQuoted.FindStringSubmatch(value); match != nil {
		text, err := strconv.Unquote(`"` + match[1] + `"`)
		if err != nil {
			return "", false
		}
		return text, true
	}
	if match := tomlSingleQuoted.FindStringSubmatch(value); match != nil {
		return match[1], true
	}
	return "", false
}

func tomlKey(raw string) (string, bool) {
	key := strings.TrimSpace(raw)
	if key == "" {
		return "", true
	}
	if key[0] == '"' {
		if len(key) < 2 || key[len(key)-1] != '"' {
			return "", false
		}
		text, err := strconv.Unquote(key)
		if err != nil {
			return "", false
		}
		return text, true
	}
	if key[0] == '\'' {
		if len(key) < 2 || key[len(key)-1] != '\'' {
			return "", false
		}
		return key[1 : len(key)-1], true
	}
	return key, true
}

func missingOmarchySources(colors map[string]string) []string {
	missing := make([]string, 0)
	for _, requirement := range omarchyRequiredSources {
		found := false
		for _, key := range requirement.keys {
			if _, ok := colors[key]; ok {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, requirement.name)
		}
	}
	return missing
}

func firstOmarchyColor(colors map[string]string, keys ...string) (rgbaColor, bool) {
	for _, key := range keys {
		value, ok := colors[key]
		if !ok {
			continue
		}
		color, ok := parseColor(value)
		if ok {
			return color, true
		}
	}
	return rgbaColor{}, false
}

func omarchyPalette(colors map[string]string, mode string, background rgbaColor, hasBackground bool) map[string]string {
	palette := defaultPalette(mode)
	bg := colorOr(palette["bg"])
	if hasBackground {
		bg = background
	}
	palette["bg"] = formatHex(bg)
	palette["canvasMiddle"] = formatHex(bg)

	canvasTop := lighten(bg, 0.08)
	if color, ok := firstOmarchyColor(colors, "lighter_background"); ok {
		canvasTop = color
	}
	canvasBottom := darken(bg, 0.12)
	if color, ok := firstOmarchyColor(colors, "darker_background", "dark_background"); ok {
		canvasBottom = color
	}
	palette["canvasTop"] = formatHex(canvasTop)
	palette["canvasBottom"] = formatHex(canvasBottom)

	ink := colorOr(palette["ink"])
	if color, ok := firstOmarchyColor(colors, "foreground", "bright_foreground", "light_foreground", "dark_foreground"); ok {
		ink = color
	}
	palette["ink"] = formatHex(ink)

	muted := mixColor(ink, bg, 0.45)
	if color, ok := firstOmarchyColor(colors, "dark_foreground"); ok {
		muted = color
	}
	palette["muted"] = formatHex(muted)
	palette["history"] = formatHex(muted)

	accent := colorOr(palette["accent"])
	if color, ok := firstOmarchyColor(colors, "accent", "blue", "cyan"); ok {
		accent = color
	}
	palette["accent"] = formatHex(accent)

	online := colorOr(palette["online"])
	if color, ok := firstOmarchyColor(colors, "green"); ok {
		online = color
	}
	palette["online"] = formatHex(online)

	warning := colorOr(palette["warning"])
	if color, ok := firstOmarchyColor(colors, "yellow", "orange"); ok {
		warning = color
	}
	palette["warning"] = formatHex(warning)

	danger := colorOr(palette["danger"])
	if color, ok := firstOmarchyColor(colors, "red"); ok {
		danger = color
	}
	palette["danger"] = formatHex(danger)

	lineAlpha := 0.14
	glassAlpha := 0.74
	gridAlpha := 0.06
	glowAlpha := 0.12
	bubbleAlpha := 0.94
	panelAlpha := 0.95
	if mode == "light" {
		lineAlpha = 0.16
		glassAlpha = 0.82
		gridAlpha = 0.07
		glowAlpha = 0.16
		bubbleAlpha = 0.92
		panelAlpha = 0.97
	}
	palette["line"] = rgbaString(ink, lineAlpha)
	palette["glass"] = rgbaString(bg, glassAlpha)
	palette["grid"] = rgbaString(accent, gridAlpha)
	palette["ambientGlow"] = rgbaString(accent, glowAlpha)
	palette["vignette"] = rgbaString(canvasBottom, 0.55)
	if mode == "light" {
		palette["labelHalo"] = rgbaString(canvasTop, 0.88)
	} else {
		palette["labelHalo"] = rgbaString(canvasBottom, 0.88)
	}
	palette["bubble"] = rgbaString(bg, bubbleAlpha)
	palette["panel"] = rgbaString(bg, panelAlpha)
	return palette
}

func omarchyKinds(colors map[string]string, palette map[string]string) map[string]Kind {
	warning := namedOmarchyColor(colors, "yellow", colorOr(palette["warning"]))
	orange := namedOmarchyColor(colors, "orange", warning)
	green := namedOmarchyColor(colors, "green", colorOr(palette["online"]))
	cyan := namedOmarchyColor(colors, "cyan", colorOr(palette["accent"]))
	blue := namedOmarchyColor(colors, "blue", colorOr(palette["accent"]))
	magenta := namedOmarchyColor(colors, "magenta", colorOr(palette["accent"]))
	return map[string]Kind{
		"router":   {Label: "Router", Color: formatHex(warning)},
		"server":   {Label: "Server", Color: palette["accent"]},
		"computer": {Label: "Computer", Color: formatHex(blue)},
		"phone":    {Label: "Phone", Color: formatHex(magenta)},
		"nas":      {Label: "NAS", Color: formatHex(green)},
		"media":    {Label: "Media", Color: formatHex(magenta)},
		"iot":      {Label: "IoT", Color: formatHex(cyan)},
		"printer":  {Label: "Printer", Color: formatHex(orange)},
		"device":   {Label: "Device", Color: palette["muted"]},
	}
}

func omarchyCategories(colors map[string]string, palette map[string]string) map[string]string {
	warning := namedOmarchyColor(colors, "yellow", colorOr(palette["warning"]))
	orange := namedOmarchyColor(colors, "orange", warning)
	green := namedOmarchyColor(colors, "green", colorOr(palette["online"]))
	cyan := namedOmarchyColor(colors, "cyan", colorOr(palette["accent"]))
	blue := namedOmarchyColor(colors, "blue", colorOr(palette["accent"]))
	magenta := namedOmarchyColor(colors, "magenta", colorOr(palette["accent"]))
	return map[string]string{
		"web":    formatHex(cyan),
		"media":  formatHex(magenta),
		"data":   formatHex(orange),
		"remote": formatHex(green),
		"infra":  formatHex(blue),
		"iot":    formatHex(cyan),
		"other":  palette["muted"],
	}
}

func namedOmarchyColor(colors map[string]string, key string, fallback rgbaColor) rgbaColor {
	if color, ok := firstOmarchyColor(colors, key); ok {
		return color
	}
	return fallback
}

func colorOr(value string) rgbaColor {
	color, _ := parseColor(value)
	return color
}
