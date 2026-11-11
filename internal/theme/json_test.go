package theme

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func validPalette() map[string]any {
	palette := make(map[string]any, len(RequiredPaletteTokens))
	for _, token := range RequiredPaletteTokens {
		palette[token] = "#112233"
	}
	palette["line"] = "rgba(17, 34, 51, 0.14)"
	return palette
}

func validVariant(id, name, mode string) map[string]any {
	return map[string]any{
		"id":      id,
		"name":    name,
		"mode":    mode,
		"palette": validPalette(),
		"effects": map[string]any{
			"vignette": 0.4,
		},
	}
}

func validFamily(id, name string, variants ...map[string]any) map[string]any {
	list := make([]any, 0, len(variants))
	for _, variant := range variants {
		list = append(list, variant)
	}
	return map[string]any{
		"id":       id,
		"name":     name,
		"variants": list,
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return raw
}

func requireCategory(t *testing.T, err error, want ErrorCategory) *ValidationError {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want category %s", want)
	}
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, want a *ValidationError", err)
	}
	if validation.Category != want {
		t.Fatalf("category = %s, want %s (error: %v)", validation.Category, want, err)
	}
	return validation
}

func TestParseJSONSingleVariant(t *testing.T) {
	tests := []struct {
		name string
		doc  map[string]any
	}{
		{
			name: "direct",
			doc: func() map[string]any {
				variant := validVariant("my-theme-dark", "My Theme", "dark")
				delete(variant, "palette")
				variant["colors"] = validPalette()
				variant["effects"] = map[string]any{}
				return variant
			}(),
		},
		{
			name: "wrapped",
			doc: map[string]any{
				"schemaVersion": 1,
				"variant": func() map[string]any {
					variant := validVariant("my-theme-dark", "My Theme", "dark")
					delete(variant, "palette")
					variant["colors"] = validPalette()
					variant["effects"] = map[string]any{}
					return variant
				}(),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalog, err := ParseJSON(mustJSON(t, test.doc))
			if err != nil {
				t.Fatalf("ParseJSON: %v", err)
			}
			if catalog.SchemaVersion != 1 {
				t.Fatalf("schemaVersion = %d, want 1", catalog.SchemaVersion)
			}
			if len(catalog.Families) != 1 {
				t.Fatalf("families = %d, want 1", len(catalog.Families))
			}
			family := catalog.Families[0]
			if family.ID != "my-theme" || family.Name != "My Theme" {
				t.Fatalf("family = %+v, want my-theme/My Theme", family)
			}
			if len(family.Variants) != 1 {
				t.Fatalf("variants = %d, want 1", len(family.Variants))
			}
			variant := family.Variants[0]
			if variant.ID != "my-theme-dark" || variant.Mode != "dark" {
				t.Fatalf("variant = %+v, want my-theme-dark dark", variant)
			}
			if variant.Palette["accent"] != "#112233" {
				t.Fatalf("accent = %q, want #112233", variant.Palette["accent"])
			}
			if variant.Effects.Vignette != 0.52 {
				t.Fatalf("vignette = %v, want the dark default 0.52", variant.Effects.Vignette)
			}
		})
	}
}

func TestParseJSONFamily(t *testing.T) {
	dark := validVariant("my-theme-dark", "My Theme", "dark")
	light := validVariant("my-theme-light", "My Theme Light", "light")
	delete(dark, "mode")
	delete(light, "mode")
	doc := map[string]any{
		"schemaVersion": 1,
		"family": map[string]any{
			"id":     "my-theme",
			"name":   "My Theme",
			"source": map[string]any{"name": "Theme Author", "url": "https://example.com/theme"},
			"variants": map[string]any{
				"dark":  dark,
				"light": light,
			},
		},
	}
	catalog, err := ParseJSON(mustJSON(t, doc))
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	if len(catalog.Families) != 1 {
		t.Fatalf("families = %d, want 1", len(catalog.Families))
	}
	family := catalog.Families[0]
	if family.Source == nil || family.Source.URL != "https://example.com/theme" {
		t.Fatalf("source = %+v, want the imported credit", family.Source)
	}
	if len(family.Variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(family.Variants))
	}
	if family.Variants[0].Mode != "dark" || family.Variants[1].Mode != "light" {
		t.Fatalf("variant modes = %q, %q, want dark then light", family.Variants[0].Mode, family.Variants[1].Mode)
	}
}

func TestParseJSONFamilyVariantArray(t *testing.T) {
	doc := map[string]any{
		"schemaVersion": 1,
		"id":            "my-theme",
		"name":          "My Theme",
		"variants": []any{
			validVariant("my-theme-dark", "My Theme", "dark"),
			validVariant("my-theme-light", "My Theme Light", "light"),
		},
	}
	catalog, err := ParseJSON(mustJSON(t, doc))
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	if len(catalog.Families) != 1 || len(catalog.Families[0].Variants) != 2 {
		t.Fatalf("catalog = %+v, want one family with two variants", catalog)
	}
}

func TestParseJSONFullCatalog(t *testing.T) {
	raw, err := os.ReadFile("../../web/themes.json")
	if err != nil {
		t.Fatalf("read web/themes.json: %v", err)
	}
	catalog, err := ParseJSON(raw)
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	if catalog.SchemaVersion != 1 {
		t.Fatalf("schemaVersion = %d, want 1", catalog.SchemaVersion)
	}
	if len(catalog.Families) < 2 {
		t.Fatalf("families = %d, want at least 2", len(catalog.Families))
	}
	var hearthlight *Family
	for index := range catalog.Families {
		if catalog.Families[index].ID == "hearthlight" {
			hearthlight = &catalog.Families[index]
			break
		}
	}
	if hearthlight == nil {
		t.Fatal("hearthlight family is missing")
	}
	if len(hearthlight.Variants) != 2 {
		t.Fatalf("hearthlight variants = %d, want 2", len(hearthlight.Variants))
	}
	if hearthlight.Variants[0].Mode != "dark" || hearthlight.Variants[1].Mode != "light" {
		t.Fatalf("hearthlight modes = %q, %q", hearthlight.Variants[0].Mode, hearthlight.Variants[1].Mode)
	}
	for _, family := range catalog.Families {
		for _, variant := range family.Variants {
			for _, token := range RequiredPaletteTokens {
				if variant.Palette[token] == "" {
					t.Errorf("%s: missing token %q", variant.ID, token)
				}
			}
		}
	}
}

func TestDefaultsMatchBuiltinHearthlight(t *testing.T) {
	raw, err := os.ReadFile("../../web/themes.json")
	if err != nil {
		t.Fatalf("read web/themes.json: %v", err)
	}
	catalog, err := ParseJSON(raw)
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	for _, family := range catalog.Families {
		if family.ID != "hearthlight" {
			continue
		}
		for _, variant := range family.Variants {
			if !reflect.DeepEqual(variant.Palette, defaultPalette(variant.Mode)) {
				t.Errorf("%s: built-in palette differs from the import default", variant.ID)
			}
			if variant.Effects != defaultEffects(variant.Mode) {
				t.Errorf("%s: built-in effects differ from the import default", variant.ID)
			}
		}
		return
	}
	t.Fatal("hearthlight family is missing")
}

func TestParseJSONRejects(t *testing.T) {
	forbiddenDoc := func(key string) map[string]any {
		variant := validVariant("bad-dark", "Bad", "dark")
		variant[key] = map[string]any{"polluted": true}
		return variant
	}
	catalogWith := func(families ...map[string]any) map[string]any {
		list := make([]any, 0, len(families))
		for _, family := range families {
			list = append(list, family)
		}
		return map[string]any{"schemaVersion": 1, "families": list}
	}
	tests := []struct {
		name string
		raw  []byte
		doc  any
		want ErrorCategory
	}{
		{name: "malformed JSON", raw: []byte(`{"family":`), want: ErrMalformed},
		{name: "empty JSON", raw: []byte(``), want: ErrMalformed},
		{name: "trailing garbage", raw: []byte(`{"variant": {}} {}`), want: ErrMalformed},
		{name: "top-level array", raw: []byte(`[]`), want: ErrInvalidShape},
		{name: "unknown shape", raw: []byte(`{}`), want: ErrInvalidShape},
		{name: "oversized", raw: bytes.Repeat([]byte(" "), MaxImportBytes+1), want: ErrTooLarge},
		{name: "unsupported schema", doc: map[string]any{"schemaVersion": 2, "families": []any{}}, want: ErrUnsupportedSchema},
		{name: "missing palette", doc: map[string]any{
			"variant": map[string]any{"id": "x", "name": "X", "mode": "dark"},
		}, want: ErrMissingToken},
		{name: "missing token", doc: func() map[string]any {
			variant := validVariant("x-dark", "X", "dark")
			delete(variant["palette"].(map[string]any), "panel")
			return variant
		}(), want: ErrMissingToken},
		{name: "bad colour", doc: func() map[string]any {
			variant := validVariant("x-dark", "X", "dark")
			variant["palette"].(map[string]any)["accent"] = "blue"
			return variant
		}(), want: ErrInvalidColor},
		{name: "invalid palette key", doc: func() map[string]any {
			variant := validVariant("x-dark", "X", "dark")
			variant["palette"].(map[string]any)["bad key"] = "#112233"
			return variant
		}(), want: ErrInvalidKey},
		{name: "short hex", doc: func() map[string]any {
			variant := validVariant("x-dark", "X", "dark")
			variant["palette"].(map[string]any)["accent"] = "#12345"
			return variant
		}(), want: ErrInvalidColor},
		{name: "bad mode", doc: func() map[string]any {
			variant := validVariant("x-dark", "X", "dim")
			return variant
		}(), want: ErrInvalidMode},
		{name: "prototype pollution __proto__", doc: forbiddenDoc("__proto__"), want: ErrForbiddenKey},
		{name: "prototype pollution constructor", doc: forbiddenDoc("constructor"), want: ErrForbiddenKey},
		{name: "prototype pollution prototype", doc: forbiddenDoc("prototype"), want: ErrForbiddenKey},
		{name: "duplicate family ids", doc: catalogWith(
			validFamily("same", "One", validVariant("one-dark", "One", "dark")),
			validFamily("same", "Two", validVariant("two-dark", "Two", "dark")),
		), want: ErrDuplicateID},
		{name: "duplicate variant ids", doc: validFamily("family", "Family",
			validVariant("same-id", "Dark", "dark"),
			validVariant("same-id", "Light", "light"),
		), want: ErrDuplicateID},
		{name: "duplicate mode", doc: validFamily("family", "Family",
			validVariant("one-dark", "One", "dark"),
			validVariant("two-dark", "Two", "dark"),
		), want: ErrDuplicateMode},
		{name: "effect above range", doc: func() map[string]any {
			variant := validVariant("x-dark", "X", "dark")
			variant["effects"].(map[string]any)["vignette"] = 1.5
			return variant
		}(), want: ErrInvalidEffect},
		{name: "negative blur", doc: func() map[string]any {
			variant := validVariant("x-dark", "X", "dark")
			variant["effects"].(map[string]any)["panelBlur"] = -1
			return variant
		}(), want: ErrInvalidEffect},
		{name: "unknown effect", doc: func() map[string]any {
			variant := validVariant("x-dark", "X", "dark")
			variant["effects"].(map[string]any)["mystery"] = 1
			return variant
		}(), want: ErrInvalidEffect},
		{name: "bad blend mode", doc: func() map[string]any {
			variant := validVariant("x-dark", "X", "dark")
			variant["effects"].(map[string]any)["territoryBlend"] = "url(javascript:alert(1))"
			return variant
		}(), want: ErrInvalidEffect},
		{name: "long family name", doc: validFamily(
			"family", strings.Repeat("a", MaxNameLength+1), validVariant("one-dark", "One", "dark"),
		), want: ErrInvalidName},
		{name: "long variant id", doc: validVariant(
			strings.Repeat("a", MaxIDLength+1), "X", "dark",
		), want: ErrInvalidID},
		{name: "both palette and colors", doc: func() map[string]any {
			variant := validVariant("x-dark", "X", "dark")
			variant["colors"] = validPalette()
			return variant
		}(), want: ErrInvalidShape},
		{name: "variant object key is not a mode", doc: map[string]any{
			"id":   "family",
			"name": "Family",
			"variants": map[string]any{
				"dim": validVariant("one", "One", ""),
			},
		}, want: ErrInvalidMode},
		{name: "variant key and mode disagree", doc: map[string]any{
			"id":   "family",
			"name": "Family",
			"variants": map[string]any{
				"light": validVariant("one-dark", "One", "dark"),
			},
		}, want: ErrInvalidMode},
		{name: "bad category colour", doc: func() map[string]any {
			variant := validVariant("x-dark", "X", "dark")
			variant["categories"] = map[string]any{"web": "not-a-colour"}
			return variant
		}(), want: ErrInvalidColor},
		{name: "bad kind label", doc: func() map[string]any {
			variant := validVariant("x-dark", "X", "dark")
			variant["kinds"] = map[string]any{"router": map[string]any{"label": "", "color": "#112233"}}
			return variant
		}(), want: ErrInvalidName},
		{name: "javascript source url", doc: func() map[string]any {
			family := validFamily("family", "Family", validVariant("one-dark", "One", "dark"))
			family["source"] = map[string]any{"name": "Credit", "url": "javascript:alert(1)"}
			return family
		}(), want: ErrInvalidShape},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw := test.raw
			if test.doc != nil {
				raw = mustJSON(t, test.doc)
			}
			_, err := ParseJSON(raw)
			requireCategory(t, err, test.want)
		})
	}
}

func TestParseJSONForbiddenKeyInNestedPalette(t *testing.T) {
	variant := validVariant("x-dark", "X", "dark")
	variant["palette"].(map[string]any)["__proto__"] = "#ffffff"
	_, err := ParseJSON(mustJSON(t, variant))
	validation := requireCategory(t, err, ErrForbiddenKey)
	if !strings.Contains(validation.Path, "__proto__") {
		t.Fatalf("path = %q, want it to name the forbidden key", validation.Path)
	}
}
