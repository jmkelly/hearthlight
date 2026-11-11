package web

import (
	"strings"
	"testing"
)

func cssMedia(t *testing.T, source, query string) string {
	t.Helper()
	start := strings.Index(source, query)
	if start < 0 {
		t.Fatalf("missing CSS media query %s", query)
	}
	open := strings.Index(source[start:], "{")
	if open < 0 {
		t.Fatalf("missing body for CSS media query %s", query)
	}
	open += start
	depth := 0
	for i := open; i < len(source); i++ {
		switch source[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[open+1 : i]
			}
		}
	}
	t.Fatalf("unterminated CSS media query %s", query)
	return ""
}

func TestCompactStatsMarkup(t *testing.T) {
	html := readAccessibilityFile(t, "index.html")
	compact := markupBetween(t, html, `<div id="statsCompact"`, `</div>`)
	requireContains(t, compact,
		`class="stats-compact glass"`,
		`role="status"`,
		`aria-live="polite"`,
		`id="statsCompactVisual"`,
		`class="stats-compact-visual"`,
		`aria-hidden="true"`,
	)
}

func TestCompactStatsSyncChrome(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	requireContains(t, js,
		`const statsCompact = document.getElementById("statsCompact")`,
		`const statsCompactVisual = document.getElementById("statsCompactVisual")`,
	)
	sync := functionBody(t, js, "syncChrome")
	requireContains(t, sync,
		"stats.innerHTML =",
		"statsCompactVisual.innerHTML =",
		`statsCompact.setAttribute("aria-label"`,
		"summary.devicesOnline",
		"summary.devicesTotal",
		"summary.servicesOnline",
		"summary.servicesTotal",
		"of ${summary.devicesTotal} devices up",
		"of ${summary.servicesTotal} services up",
		"tailnet.peersOnline",
		"tailnet.peersTotal",
	)
	if count := strings.Count(sync, "stats.innerHTML ="); count != 1 {
		t.Errorf("syncChrome writes the full stats card from %d places, want 1", count)
	}
	if count := strings.Count(sync, "statsCompactVisual.innerHTML ="); count != 1 {
		t.Errorf("syncChrome writes the compact stats from %d places, want 1", count)
	}
}

func TestCompactStatsBreakpoints(t *testing.T) {
	css := readAccessibilityFile(t, "style.css")

	full := cssRule(t, css, ".stats {")
	requireContains(t, full, "display: flex;")

	base := cssRule(t, css, ".stats-compact {")
	requireContains(t, base, "display: none;")

	narrow := cssMedia(t, css, "@media (max-width: 1259.98px)")
	requireContains(t, narrow,
		".stats {",
		"display: none;",
		".stats-compact {",
		"display: flex;",
	)

	reduced := cssMedia(t, css, "@media (max-width: 960px)")
	requireContains(t, reduced,
		".stats-compact .compact-label,",
		".stats-compact .compact-sep,",
		".stats-compact .compact-tailnet {",
		"display: none;",
	)

	mobile := cssMedia(t, css, "@media (max-width: 760px)")
	requireContains(t, mobile, "flex-wrap: wrap;")
}
