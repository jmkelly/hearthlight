package web

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

const (
	darkRootSelector  = ":root {"
	lightRootSelector = `:root[data-mode="light"] {`
)

var (
	cssColourPattern = regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b|(?:rgba?|hsla?)\([^)]*\)`)
	cssTokenPattern  = regexp.MustCompile(`(?m)^\s*(--theme-[a-z0-9-]+):\s*([^;]+);`)
	cssTokenUse      = regexp.MustCompile(`var\((--theme-[a-z0-9-]+)\)`)
)

// cssBlock extracts the block that starts with selector and returns it with
// the remaining source.
func cssBlock(source, selector string) (string, string, bool) {
	start := strings.Index(source, selector)
	if start < 0 {
		return "", source, false
	}
	open := strings.Index(source[start:], "{")
	if open < 0 {
		return "", source, false
	}
	open += start
	end := strings.Index(source[open:], "}")
	if end < 0 {
		return "", source, false
	}
	end += open + 1
	return source[start:end], source[end:], true
}

// splitThemeCSS returns the dark defaults, the light overrides and the rest of
// the stylesheet. Literal colours are only allowed inside the token blocks so
// every rendered colour resolves through a theme variable.
func splitThemeCSS(t *testing.T, css string) (string, string, string) {
	t.Helper()
	dark, rest, ok := cssBlock(css, darkRootSelector)
	if !ok {
		t.Fatalf("style.css has no %q block", darkRootSelector)
	}
	light, body, ok := cssBlock(rest, lightRootSelector)
	if !ok {
		t.Fatalf("style.css has no %q block", lightRootSelector)
	}
	return dark, light, body
}

func cssTokens(source string) map[string]string {
	tokens := make(map[string]string)
	for _, match := range cssTokenPattern.FindAllStringSubmatch(source, -1) {
		tokens[match[1]] = strings.TrimSpace(match[2])
	}
	return tokens
}

func kebabCase(name string) string {
	var out strings.Builder
	for _, r := range name {
		if r >= 'A' && r <= 'Z' {
			out.WriteByte('-')
			out.WriteRune(r + ('a' - 'A'))
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

func TestStyleThemeTokens(t *testing.T) {
	raw, err := os.ReadFile("style.css")
	if err != nil {
		t.Fatalf("read style.css: %v", err)
	}
	css := string(raw)
	dark, _, body := splitThemeCSS(t, css)

	if found := cssColourPattern.FindAllString(body, -1); len(found) > 0 {
		t.Fatalf("style.css uses literal colours outside the theme defaults: %v", found)
	}

	tokens := cssTokens(dark)
	for _, match := range cssTokenUse.FindAllStringSubmatch(body, -1) {
		if _, ok := tokens[match[1]]; !ok {
			t.Errorf("style.css uses undefined theme token %s", match[1])
		}
	}

	_, catalog := readCatalog(t)
	if len(catalog.Families) == 0 || len(catalog.Families[0].Variants) == 0 {
		t.Fatal("themes.json has no variants")
	}
	for name, want := range catalog.Families[0].Variants[0].Palette {
		token := "--theme-" + kebabCase(name)
		got, ok := tokens[token]
		if !ok {
			t.Errorf("style.css is missing %s", token)
			continue
		}
		if got != want {
			t.Errorf("%s = %q, want %q", token, got, want)
		}
	}

	for legacy, token := range map[string]string{
		"--bg":     "--theme-bg",
		"--ink":    "--theme-ink",
		"--muted":  "--theme-muted",
		"--accent": "--theme-accent",
		"--line":   "--theme-line",
		"--glass":  "--theme-glass",
	} {
		want := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(legacy) + `:\s*var\(` + regexp.QuoteMeta(token) + `\);`)
		if !want.MatchString(dark) {
			t.Errorf("%s does not resolve through %s", legacy, token)
		}
	}
}

func TestLightModeChromeTokens(t *testing.T) {
	css := readAccessibilityFile(t, "style.css")
	dark, light, _ := splitThemeCSS(t, css)
	darkTokens := cssTokens(dark)
	lightTokens := cssTokens(light)

	_, catalog := readCatalog(t)
	if len(catalog.Families) == 0 || len(catalog.Families[0].Variants) < 2 {
		t.Fatal("themes.json has no light variant")
	}
	lightVariant := catalog.Families[0].Variants[1]
	if lightVariant.Mode != "light" {
		t.Fatalf("variant %s has mode %q, want light", lightVariant.ID, lightVariant.Mode)
	}
	paletteTokens := make(map[string]bool)
	for name := range lightVariant.Palette {
		paletteTokens["--theme-"+kebabCase(name)] = true
	}

	// Palette tokens arrive inline from the theme boot script. Every chrome
	// token in the dark defaults needs an explicit light counterpart.
	for token, darkValue := range darkTokens {
		if paletteTokens[token] {
			continue
		}
		lightValue, ok := lightTokens[token]
		if !ok {
			t.Errorf("light mode does not override %s", token)
			continue
		}
		if lightValue == darkValue {
			t.Errorf("%s is identical in light and dark mode", token)
		}
	}

	if !strings.Contains(light, "color-scheme: light;") {
		t.Error("light mode does not declare color-scheme: light")
	}
	if !strings.Contains(dark, "color-scheme: dark;") {
		t.Error("dark mode does not declare color-scheme: dark")
	}
}

func TestChromeSurfacesUseThemeTokens(t *testing.T) {
	css := readAccessibilityFile(t, "style.css")
	for _, selector := range []string{
		".vignette {",
		".glass {",
		".device-list.open,",
		".panel {",
		".settings {",
		".settings::backdrop {",
		".tooltip {",
		".toast {",
		".hint {",
		".panel::-webkit-scrollbar-thumb {",
		".actions button {",
		".scan-btn {",
	} {
		body := cssRule(t, css, selector)
		if !strings.Contains(body, "var(--theme-") {
			t.Errorf("%s does not use theme variables", selector)
		}
	}
}
