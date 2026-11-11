package web

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

const (
	omarchyPinnedCommit = "748fa3c0d0418d7a199818d94c1f87d495d7e804"
	omarchyPinnedSource = "https://github.com/omacom/omarchy/tree/" + omarchyPinnedCommit + "/themes"
)

// omarchyStockThemes pins the stock list from the source commit. A mapping
// change is a deliberate catalog decision, so the expected target and mode are
// both recorded here.
var omarchyStockThemes = map[string]struct {
	variant string
	mode    string
}{
	"catppuccin":       {"catppuccin-mocha", "dark"},
	"catppuccin-latte": {"catppuccin-latte", "light"},
	"ethereal":         {"ethereal", "dark"},
	"everforest":       {"everforest-dark", "dark"},
	"flexoki-light":    {"flexoki-light", "light"},
	"gruvbox":          {"gruvbox-dark", "dark"},
	"hackerman":        {"hackerman", "dark"},
	"jade-horse":       {"jade-horse", "dark"},
	"kanagawa":         {"kanagawa-wave", "dark"},
	"last-horizon":     {"last-horizon", "dark"},
	"lumon":            {"lumon", "dark"},
	"lupine":           {"lupine", "light"},
	"matte-black":      {"matte-black", "dark"},
	"miasma":           {"miasma", "dark"},
	"nord":             {"nord-dark", "dark"},
	"retro-82":         {"retro-82", "dark"},
	"ristretto":        {"ristretto", "dark"},
	"rose-pine":        {"rose-pine-dawn", "light"},
	"solitude":         {"solitude", "dark"},
	"starship":         {"starship", "dark"},
	"tokyo-night":      {"tokyo-night", "dark"},
	"vantablack":       {"vantablack", "dark"},
	"white":            {"white", "light"},
}

type omarchyCatalog struct {
	SchemaVersion int               `json:"schemaVersion"`
	Source        string            `json:"source"`
	Commit        string            `json:"commit"`
	Themes        map[string]string `json:"themes"`
	Modes         map[string]string `json:"modes"`
}

func TestOmarchyMappingMatchesPinnedStockThemes(t *testing.T) {
	mapping, catalog := readOmarchyFixtures(t)
	for _, problem := range validateOmarchyMapping(mapping, catalog) {
		t.Error(problem)
	}
}

func TestOmarchyMappingCatchesRosePineRegression(t *testing.T) {
	mapping, catalog := readOmarchyFixtures(t)
	if problems := validateOmarchyMapping(mapping, catalog); len(problems) != 0 {
		t.Fatalf("baseline mapping is invalid: %v", problems)
	}

	mapping.Themes["rose-pine"] = "rose-pine-moon"
	problems := validateOmarchyMapping(mapping, catalog)
	for _, problem := range problems {
		if strings.Contains(problem, `"rose-pine" maps to "rose-pine-moon"`) {
			return
		}
	}
	t.Fatalf("rose-pine -> rose-pine-moon regression not detected: %v", problems)
}

func readOmarchyFixtures(t *testing.T) (omarchyCatalog, themeCatalog) {
	t.Helper()
	rawCatalog, err := os.ReadFile("themes.json")
	if err != nil {
		t.Fatalf("read themes.json: %v", err)
	}
	var catalog themeCatalog
	if err := json.Unmarshal(rawCatalog, &catalog); err != nil {
		t.Fatalf("parse themes.json: %v", err)
	}
	rawMapping, err := os.ReadFile("omarchy.json")
	if err != nil {
		t.Fatalf("read omarchy.json: %v", err)
	}
	var mapping omarchyCatalog
	if err := json.Unmarshal(rawMapping, &mapping); err != nil {
		t.Fatalf("parse omarchy.json: %v", err)
	}
	return mapping, catalog
}

func validateOmarchyMapping(mapping omarchyCatalog, catalog themeCatalog) []string {
	var problems []string
	report := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf(format, args...))
	}

	if mapping.SchemaVersion != 1 {
		report("omarchy schemaVersion = %d, want 1", mapping.SchemaVersion)
	}
	if mapping.Source != omarchyPinnedSource {
		report("omarchy source = %q, want %q", mapping.Source, omarchyPinnedSource)
	}
	if mapping.Commit != omarchyPinnedCommit {
		report("omarchy commit = %q, want %q", mapping.Commit, omarchyPinnedCommit)
	}

	variants := make(map[string]themeVariant)
	for _, family := range catalog.Families {
		for _, variant := range family.Variants {
			variants[variant.ID] = variant
		}
	}

	if len(mapping.Themes) != len(omarchyStockThemes) {
		report("omarchy mapping has %d themes, want %d", len(mapping.Themes), len(omarchyStockThemes))
	}
	for name, want := range omarchyStockThemes {
		id, ok := mapping.Themes[name]
		if !ok {
			report("omarchy stock theme %q is not mapped", name)
			continue
		}
		if id != want.variant {
			report("omarchy theme %q maps to %q, want %q", name, id, want.variant)
		}
		variant, ok := variants[id]
		if !ok {
			report("omarchy theme %q maps to unknown variant %q", name, id)
			continue
		}
		mode, ok := mapping.Modes[name]
		if !ok {
			report("omarchy theme %q has no mode entry", name)
			continue
		}
		if mode != want.mode {
			report("omarchy theme %q mode = %q, want pinned %q", name, mode, want.mode)
		}
		if variant.Mode != mode {
			report("omarchy theme %q maps to %q (%s mode), want %s mode", name, id, variant.Mode, mode)
		}
	}
	for name := range mapping.Themes {
		if _, ok := omarchyStockThemes[name]; !ok {
			report("omarchy mapping contains unknown stock theme %q", name)
		}
	}
	for name := range mapping.Modes {
		if _, ok := mapping.Themes[name]; !ok {
			report("omarchy mode %q has no theme mapping", name)
		}
	}
	return problems
}
