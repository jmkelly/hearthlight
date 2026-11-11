package web

import (
	"os/exec"
	"strings"
	"testing"
)

func TestThemeImportOmarchyTomlBehavior(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed; skipping browser parser execution")
	}
	output, err := exec.Command(node, "testdata/theme-import.test.js").CombinedOutput()
	if err != nil {
		t.Fatalf("theme-import.js parser tests failed: %v\n%s", err, output)
	}
}

func TestThemeImportOmarchyTomlParserHooks(t *testing.T) {
	source := readAccessibilityFile(t, "theme-import.js")
	requireContains(t, source,
		"const OMARCHY_COLOR_KEYS = new Set([",
		"const OMARCHY_REQUIRED_SOURCES = Object.freeze([",
		"const parseOmarchyDocument = (raw) => {",
		"const looksLikeJSONDocument = (raw) => {",
		"const stripTomlComment = (line) => {",
		`format: "toml"`,
		`pushError(errors, "forbidden_key"`,
		`ignored.push(key)`,
		`missing.push("mode")`,
	)
	for _, forbidden := range []string{"eval(", "new Function", "fetch(", "XMLHttpRequest"} {
		if strings.Contains(source, forbidden) {
			t.Errorf("theme-import.js must not use %s", forbidden)
		}
	}
}

func TestThemeImportOmarchyTomlMarkup(t *testing.T) {
	html := readAccessibilityFile(t, "index.html")
	dialog := markupBetween(t, html, `<dialog id="themeImportDialog"`, `</dialog>`)
	requireContains(t, dialog,
		`accept=".json,.toml,application/json,text/plain"`,
		`Paste JSON or TOML`,
		`id="themeImportNotes"`,
		`role="status"`,
		`Omarchy colors.toml`,
	)
}

func TestThemeImportOmarchyTomlAppWiring(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	notes := functionBody(t, js, "renderThemeImportNotes")
	requireContains(t, notes,
		`result.format === "toml"`,
		`Resolved `,
		`result.missing.join`,
		`result.ignored.join`,
	)

	parse := functionBody(t, js, "validateThemeImport")
	requireContains(t, parse, `renderThemeImportNotes(ok ? result : null)`)

	reset := functionBody(t, js, "resetThemeImport")
	requireContains(t, reset, `renderThemeImportNotes(null)`)

	failure := functionBody(t, js, "showThemeImportFailure")
	requireContains(t, failure, `renderThemeImportNotes(null)`)
}
