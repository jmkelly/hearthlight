package web

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

var (
	cssColourPattern = regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b|(?:rgba?|hsla?)\([^)]*\)`)
	cssTokenPattern  = regexp.MustCompile(`(?m)^\s*(--theme-[a-z0-9-]+):\s*([^;]+);`)
	cssTokenUse      = regexp.MustCompile(`var\((--theme-[a-z0-9-]+)\)`)
)

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

	rootEnd := strings.Index(css, "\n}\n")
	if rootEnd < 0 {
		t.Fatal("style.css has no root block")
	}
	root, body := css[:rootEnd+3], css[rootEnd+3:]

	if found := cssColourPattern.FindAllString(body, -1); len(found) > 0 {
		t.Fatalf("style.css uses literal colours outside the theme defaults: %v", found)
	}

	tokens := make(map[string]string)
	for _, match := range cssTokenPattern.FindAllStringSubmatch(root, -1) {
		tokens[match[1]] = strings.TrimSpace(match[2])
	}
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
		if !want.MatchString(root) {
			t.Errorf("%s does not resolve through %s", legacy, token)
		}
	}
}
