package web

import (
	"bytes"
	"os"
	"regexp"
	"strings"
	"testing"
)

var (
	scriptElementPattern = regexp.MustCompile(`(?is)<script\b([^>]*)>(.*?)</script>`)
	scriptSourcePattern  = regexp.MustCompile(`(?i)\bsrc\s*=\s*"([^"]+)"`)
)

func TestThemeBootLoadsBeforeStylesheet(t *testing.T) {
	html := readAccessibilityFile(t, "index.html")
	head := markupBetween(t, html, "<head>", "</head>")

	themes := strings.Index(head, `<script src="/themes.js"></script>`)
	boot := strings.Index(head, `<script src="/theme-boot.js"></script>`)
	stylesheet := strings.Index(head, `<link rel="stylesheet" href="/style.css">`)
	if themes < 0 {
		t.Fatal("index.html does not load /themes.js")
	}
	if boot < 0 {
		t.Fatal("index.html does not load /theme-boot.js")
	}
	if stylesheet < 0 {
		t.Fatal("index.html does not load /style.css")
	}
	if themes > boot {
		t.Error("/themes.js must load before /theme-boot.js")
	}
	if boot > stylesheet {
		t.Error("/theme-boot.js must load before /style.css")
	}

	requireContains(t, html,
		`data-theme="hearthlight"`,
		`data-mode="dark"`,
	)
	requireContains(t, head,
		`<meta name="theme-color" content="#050a12">`,
		`<meta name="color-scheme" content="dark">`,
	)
}

func TestIndexScriptsAreExternalAndEmbedded(t *testing.T) {
	html := readAccessibilityFile(t, "index.html")
	scripts := scriptElementPattern.FindAllStringSubmatch(html, -1)
	if len(scripts) == 0 {
		t.Fatal("index.html has no scripts")
	}
	for _, script := range scripts {
		attrs, body := script[1], script[2]
		src := scriptSourcePattern.FindStringSubmatch(attrs)
		if len(src) != 2 {
			t.Errorf("<script%s> is inline; the CSP allows external same-origin scripts only", attrs)
			continue
		}
		if strings.TrimSpace(body) != "" {
			t.Errorf("%s carries an inline script body", src[1])
		}
		name := strings.TrimPrefix(src[1], "/")
		if _, err := Files.ReadFile(name); err != nil {
			t.Errorf("embedded files do not contain %s: %v", name, err)
		}
	}
}

func TestThemeBootIsEmbedded(t *testing.T) {
	onDisk, err := os.ReadFile("theme-boot.js")
	if err != nil {
		t.Fatalf("read theme-boot.js: %v", err)
	}
	embedded, err := Files.ReadFile("theme-boot.js")
	if err != nil {
		t.Fatalf("embedded theme-boot.js: %v", err)
	}
	if !bytes.Equal(embedded, onDisk) {
		t.Fatal("embedded theme-boot.js does not match the checked-in file")
	}
}

func TestThemeBootWiresLiveThemeChanges(t *testing.T) {
	source := readAccessibilityFile(t, "theme-boot.js")
	requireContains(t, source,
		`"hearthlight.theme.v1"`,
		`{ family: "hearthlight", mode: "dark" }`,
		`window.matchMedia("(prefers-color-scheme: light)")`,
		`window.addEventListener("storage"`,
		`root.dataset.theme =`,
		`root.dataset.mode =`,
		`.setProperty(`,
		`"theme-color"`,
		`"color-scheme"`,
	)
}

func TestSettingsDialogMarkup(t *testing.T) {
	html := readAccessibilityFile(t, "index.html")
	gear := markupBetween(t, html, `<button id="settingsBtn"`, `</button>`)
	requireContains(t, gear,
		`aria-haspopup="dialog"`,
		`aria-controls="settingsDialog"`,
		`aria-expanded="false"`,
		`aria-label="Settings"`,
	)

	dialog := markupBetween(t, html, `<dialog id="settingsDialog"`, `</dialog>`)
	requireContains(t, dialog,
		`aria-labelledby="settingsTitle"`,
		`<h2 id="settingsTitle">Settings</h2>`,
		`id="settingsClose"`,
		`aria-labelledby="settingsMapHeading"`,
		`<h3 id="settingsMapHeading">Map</h3>`,
		`id="settingsHistoryBtn"`,
		`id="settingsTailnetBtn"`,
		`aria-pressed="true"`,
	)
}

func TestSettingsAppearanceMarkup(t *testing.T) {
	html := readAccessibilityFile(t, "index.html")
	dialog := markupBetween(t, html, `<dialog id="settingsDialog"`, `</dialog>`)
	appearance := markupBetween(t, dialog, `<section class="settings-section settings-appearance"`, `</section>`)
	requireContains(t, appearance,
		`aria-labelledby="settingsAppearanceHeading"`,
		`<h3 id="settingsAppearanceHeading">Appearance</h3>`,
		`id="themeFamilies"`,
		`class="theme-families"`,
		`role="group"`,
		`aria-label="Theme family"`,
		`aria-label="Theme mode"`,
		`id="themeModeDark"`,
		`id="themeModeLight"`,
		`id="themeModeAuto"`,
		`>Dark<`,
		`>Light<`,
		`>Auto<`,
		`id="themeFallbackNote"`,
		`hidden`,
		`id="themeReset"`,
		`>Reset to default<`,
	)
	if count := strings.Count(appearance, `aria-pressed=`); count != 3 {
		t.Errorf("appearance segmented control has %d aria-pressed buttons, want 3", count)
	}
}

func TestAppearanceControlsApplyAndPersist(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	requireContains(t, js,
		`document.getElementById("themeModeDark")`,
		`document.getElementById("themeModeLight")`,
		`document.getElementById("themeModeAuto")`,
		`document.getElementById("themeReset")`,
		`document.getElementById("themeDefaultBadge")`,
		`document.getElementById("themeFallbackNote")`,
		`themeModeDark.addEventListener("click", () => setThemeMode("dark"));`,
		`themeModeLight.addEventListener("click", () => setThemeMode("light"));`,
		`themeModeAuto.addEventListener("click", () => setThemeMode("auto"));`,
		`themeReset.addEventListener("click", resetTheme);`,
	)
	set := functionBody(t, js, "setThemeMode")
	requireContains(t, set,
		"window.HearthlightTheme",
		"THEME_MODES.has(mode)",
		"control.initialPreference()",
		"control.persist({ family, mode })",
		"applyTheme(",
	)
	reset := functionBody(t, js, "resetTheme")
	requireContains(t, reset,
		"window.HearthlightTheme",
		"control.DEFAULT_THEME",
		"control.persist(",
		"applyTheme(",
	)
	sync := functionBody(t, js, "syncThemeControls")
	requireContains(t, sync,
		"aria-pressed",
		"requestedMode",
		"themeDefaultBadge.hidden",
		"themeFallbackNote.hidden",
		"fallback",
		"themeFallbackNote.textContent =",
	)
	apply := functionBody(t, js, "applyTheme")
	requireContains(t, apply, "syncThemeControls(view.theme)")
	open := functionBody(t, js, "openSettings")
	requireContains(t, open, "syncThemeControls()")
}

func TestThemeFamilyCardsRenderFromCatalog(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	render := functionBody(t, js, "renderThemeFamilies")
	requireContains(t, render,
		"window.HearthlightThemes",
		"Array.isArray(catalog.families)",
		"catalog.families.filter(",
		"themeFamilyPreview(family)",
		`<button type="button" class="theme-card theme-family"`,
		`data-family="${escapeHTML(family.id)}"`,
		`aria-pressed="false"`,
		`tabindex="-1"`,
		`class="theme-swatch theme-swatch-dark"`,
		`class="theme-swatch theme-swatch-light"`,
		`class="theme-card-name">${escapeHTML(name)}<`,
		"family.id === defaultFamily",
		`id="themeDefaultBadge"`,
		`querySelectorAll(".theme-family")`,
		`button.addEventListener("click", () => setThemeFamily(button.dataset.family))`,
	)
}

func TestThemeFamilyKeyboardNavigation(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	requireContains(t, js, `themeFamilies.addEventListener("keydown", onThemeFamilyKeydown)`)

	move := functionBody(t, js, "onThemeFamilyKeydown")
	requireContains(t, move,
		`event.target.closest(".theme-family")`,
		`event.key === "ArrowRight"`,
		`event.key === "ArrowDown"`,
		`event.key === "ArrowLeft"`,
		`event.key === "ArrowUp"`,
		`event.key === "Home"`,
		`event.key === "End"`,
		"event.preventDefault()",
		"button.tabIndex =",
		"target.focus()",
	)
}

func TestThemeFamilySelectionPersists(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	choose := functionBody(t, js, "setThemeFamily")
	requireContains(t, choose,
		"window.HearthlightTheme",
		"control.initialPreference()",
		"family: familyId",
		"mode",
		"control.persist({ family: familyId, mode })",
		"applyTheme(",
	)

	sync := functionBody(t, js, "syncThemeFamilies")
	requireContains(t, sync,
		"themeFamilyButtons",
		`button.dataset.family === selected`,
		`button.setAttribute("aria-pressed"`,
		"button.tabIndex =",
	)
}

func TestAppearanceStylingUsesThemeTokens(t *testing.T) {
	css := readAccessibilityFile(t, "style.css")
	for _, selector := range []string{
		".theme-card {",
		`.theme-card[aria-pressed="true"] {`,
		".theme-swatch-dark {",
		".theme-swatch-light {",
		".theme-modes {",
		`.theme-mode[aria-pressed="true"] {`,
		".theme-default-badge {",
		".theme-fallback-note {",
	} {
		body := cssRule(t, css, selector)
		if !strings.Contains(body, "var(--theme-") {
			t.Errorf("%s does not use theme variables", selector)
		}
	}
}

func TestSettingsDialogWiresModalBehaviour(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	requireContains(t, js,
		`settingsDialog.showModal()`,
		`settingsDialog.close()`,
		`settingsBtn.focus()`,
		`settingsDialog.addEventListener("cancel"`,
		`event.preventDefault()`,
		`if (event.target === settingsDialog) closeSettings()`,
		`if (event.key !== "Tab") return;`,
		`last.focus()`,
		`first.focus()`,
		`event.target.matches("input, textarea, select, [contenteditable]")`,
		`if (settingsDialog.open) return;`,
		`event.key.toLowerCase() === "s"`,
		`openSettings();`,
	)
}

func TestSettingsMapControlsUseExistingPreferences(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	history := functionBody(t, js, "setHistoryVisible")
	requireContains(t, history,
		`historyBtn.setAttribute("aria-pressed"`,
		`window.localStorage.setItem("hearthlight.history"`,
		`syncSettingsControls()`,
	)
	tailnet := functionBody(t, js, "setTailnetVisible")
	requireContains(t, tailnet,
		`tailnetBtn.setAttribute("aria-pressed"`,
		`window.localStorage.setItem("hearthlight.tailnet"`,
		`syncSettingsControls()`,
	)
	sync := functionBody(t, js, "syncSettingsControls")
	requireContains(t, sync, "settingsHistoryBtn", "settingsTailnetBtn")
}

func arrowBody(t *testing.T, source, name string) string {
	t.Helper()
	start := strings.Index(source, "const "+name+" =")
	if start < 0 {
		t.Fatalf("missing const %s", name)
	}
	open := strings.Index(source[start:], "{")
	if open < 0 {
		t.Fatalf("missing body for %s", name)
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
	t.Fatalf("unterminated arrow %s", name)
	return ""
}

func TestThemeControlSurfaceIsExposed(t *testing.T) {
	source := readAccessibilityFile(t, "theme-boot.js")
	api := markupBetween(t, source, "window.HearthlightTheme = Object.freeze({", "});")
	for _, name := range []string{
		"STORAGE_KEY",
		"DEFAULT_THEME",
		"initialPreference",
		"resolve",
		"apply",
		"persist",
	} {
		pattern := regexp.MustCompile(`(?m)^\s*` + name + `,?$`)
		if !pattern.MatchString(api) {
			t.Errorf("HearthlightTheme does not expose %s", name)
		}
	}
	requireContains(t, source,
		`const STORAGE_KEY = "hearthlight.theme.v1";`,
		`const DEFAULT_THEME = Object.freeze({ family: "hearthlight", mode: "dark" });`,
	)
}

func TestThemeResolveCarriesPaletteAndModes(t *testing.T) {
	source := readAccessibilityFile(t, "theme-boot.js")
	resolve := arrowBody(t, source, "resolve")
	requireContains(t, resolve,
		"family:",
		"variant:",
		"requestedMode",
		"resolvedMode",
		"fallback",
		"palette,",
		"kinds:",
		"categories:",
		"effects:",
	)
}

func TestThemeQueryOverride(t *testing.T) {
	source := readAccessibilityFile(t, "theme-boot.js")
	requireContains(t, source,
		`const QUERY_MODES = new Set(["dark", "light", "auto"]);`,
		`new URLSearchParams(window.location.search)`,
	)
	query := arrowBody(t, source, "readQueryOverride")
	requireContains(t, query,
		`params.get("theme")`,
		`params.get("mode")`,
		"findVariantById(theme)",
		"if (!match) return null;",
		"if (!QUERY_MODES.has(mode)) return null;",
		"override.variant = match.variant.id;",
	)
	initial := arrowBody(t, source, "initialPreference")
	requireContains(t, initial, "readPreference()", "queryPreference", "mergePreference(")
}

func TestThemeQueryOverrideNeverWritesStorage(t *testing.T) {
	source := readAccessibilityFile(t, "theme-boot.js")
	if count := strings.Count(source, ".setItem("); count != 1 {
		t.Fatalf("theme-boot.js writes localStorage from %d places, want persist() only", count)
	}
	persist := arrowBody(t, source, "persist")
	requireContains(t, persist, ".setItem(STORAGE_KEY, JSON.stringify(normalized))")
	for _, name := range []string{"readPreference", "readQueryOverride", "initialPreference", "resolve", "apply"} {
		if strings.Contains(arrowBody(t, source, name), ".setItem(") {
			t.Errorf("%s writes localStorage", name)
		}
	}
}

func TestThemePersistWritesStoredPreference(t *testing.T) {
	source := readAccessibilityFile(t, "theme-boot.js")
	persist := arrowBody(t, source, "persist")
	requireContains(t, persist,
		"normalizePreference(preference)",
		"JSON.stringify(normalized)",
		"update(normalized)",
	)
}

func TestThemeControlDispatchesAndListens(t *testing.T) {
	source := readAccessibilityFile(t, "theme-boot.js")
	dispatch := arrowBody(t, source, "dispatchTheme")
	requireContains(t, dispatch,
		`new CustomEvent("hearthlight:theme"`,
		"detail: theme",
		"window.dispatchEvent(",
	)
	update := arrowBody(t, source, "update")
	requireContains(t, update, "apply(resolve(preference))", "dispatchTheme(theme)")
	requireContains(t, source,
		`window.addEventListener("storage"`,
		`if (event.key === STORAGE_KEY || event.key === null) update(initialPreference());`,
		`lightQuery.addEventListener("change"`,
		`lightQuery.addListener(`,
	)
}

func TestThemeControlAppliesBeforeFirstPaint(t *testing.T) {
	source := readAccessibilityFile(t, "theme-boot.js")
	applyCall := strings.Index(source, "apply(resolve(initialPreference()));")
	storageListener := strings.Index(source, `window.addEventListener("storage"`)
	exposed := strings.Index(source, "window.HearthlightTheme =")
	if applyCall < 0 {
		t.Fatal("theme-boot.js does not resolve and apply the initial theme")
	}
	if storageListener < 0 || exposed < 0 || applyCall > storageListener || applyCall > exposed {
		t.Error("theme-boot.js must apply the initial theme before wiring listeners and exposing the API")
	}
	apply := arrowBody(t, source, "apply")
	requireContains(t, apply,
		"root.dataset.theme =",
		"root.dataset.mode =",
		`setMeta("theme-color"`,
		`setMeta("color-scheme"`,
	)
}

func TestColorSchemeFollowsResolvedMode(t *testing.T) {
	source := readAccessibilityFile(t, "theme-boot.js")
	apply := arrowBody(t, source, "apply")
	requireContains(t, apply,
		"resolved.resolvedMode",
		"root.dataset.mode = mode",
		`setMeta("color-scheme", mode)`,
	)

	html := readAccessibilityFile(t, "index.html")
	requireContains(t, html,
		`<meta name="color-scheme" content="dark">`,
	)

	css := readAccessibilityFile(t, "style.css")
	dark, light, _ := splitThemeCSS(t, css)
	requireContains(t, dark, "color-scheme: dark;")
	requireContains(t, light, "color-scheme: light;")
}

func TestCanvasThemeBootstrapsFromControl(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	requireContains(t, js,
		"theme: null,",
		`window.addEventListener("hearthlight:theme"`,
		"applyTheme(event.detail)",
		"\n  applyTheme();\n",
	)
	apply := functionBody(t, js, "applyTheme")
	requireContains(t, apply,
		"window.HearthlightTheme",
		"control.resolve(preference)",
		"view.theme = control.apply(resolved)",
		`view.territorySig = ""`,
		"view.territoryLayer = null",
		"renderLegend()",
		"requestDraw()",
	)
	draw := functionBody(t, js, "requestDraw")
	requireContains(t, draw, "window.requestAnimationFrame", "draw(now)")
}

func TestCanvasRenderersReadThemeTokens(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	readers := []string{
		"drawBackground",
		"drawTerritoryFill",
		"drawAuroraRibbon",
		"drawOverlapLens",
		"drawTerritoryBoundaries",
		"drawRegionPill",
		"drawReachBadge",
		"drawDevice",
		"drawService",
		"drawTraffic",
		"strokeLabel",
	}
	tokenPattern := regexp.MustCompile(`theme(Color|Kind|Category|Effect)\(`)
	colorLiteral := regexp.MustCompile(`#[0-9a-fA-F]{3,8}`)
	rgbLiteral := regexp.MustCompile(`rgba?\(\s*[0-9]`)
	for _, name := range readers {
		body := functionBody(t, js, name)
		if !tokenPattern.MatchString(body) {
			t.Errorf("%s does not read a theme token", name)
		}
		if colorLiteral.MatchString(body) {
			t.Errorf("%s still carries a hard-coded colour literal", name)
		}
		if rgbLiteral.MatchString(body) {
			t.Errorf("%s still carries a hard-coded rgb colour", name)
		}
	}

	for _, name := range []string{"themeColor", "themeKind", "themeCategory", "themeEffect"} {
		if body := arrowBody(t, js, name); !strings.Contains(body, "view.theme") {
			t.Errorf("%s does not read view.theme", name)
		}
	}
}

func TestThemeConstantsCarryNoColours(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	definitions := markupBetween(t, js, "  const KIND = {", "  // Glyph shapes live")
	requireContains(t, definitions,
		`const CATEGORY = ["web", "media", "data", "remote", "infra", "iot", "other"];`,
		`token: "warning"`,
		`token: "accent"`,
	)
	for _, pattern := range []*regexp.Regexp{
		regexp.MustCompile(`#[0-9a-fA-F]{3,8}`),
		regexp.MustCompile(`rgba?\(\s*[0-9]`),
	} {
		if pattern.MatchString(definitions) {
			t.Errorf("KIND, CATEGORY or TRAFFIC still carries a hard-coded colour: %s", pattern)
		}
	}
}

func TestLegendReadsThemeTokens(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	legend := functionBody(t, js, "renderLegend")
	requireContains(t, legend,
		`themeKind(`,
		`themeCategory(`,
		`themeColor("online")`,
		`themeColor("history")`,
		`themeColor(TRAFFIC.lan.token)`,
		`themeColor(TRAFFIC.tailnet.token)`,
	)
}

func TestCanvasEffectsComeFromTheme(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	requireContains(t, functionBody(t, js, "drawTerritoryFill"), `themeEffect("territoryBlur")`)
	requireContains(t, functionBody(t, js, "drawAuroraRibbon"), `themeEffect("ribbonBlur")`)
	requireContains(t, functionBody(t, js, "drawOverlapLens"), `themeEffect("lensBlur")`)
	requireContains(t, functionBody(t, js, "drawDevice"), `themeEffect("deviceGlow")`, `themeEffect("devicePulse")`)
	requireContains(t, functionBody(t, js, "drawService"), `themeEffect("serviceGlow")`)
}

func TestCanvasTerritoryCompositingReadsTheme(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	layer := functionBody(t, js, "renderTerritoryLayer")
	requireContains(t, layer,
		`themeEffect("territoryBlend")`,
		`view.theme.variant`,
		`view.theme.resolvedMode`,
	)
	for _, name := range []string{"drawTerritoryFill", "drawAuroraRibbon", "drawOverlapLens"} {
		requireContains(t, functionBody(t, js, name), `themeEffect("territoryAlpha")`)
	}
	requireContains(t, functionBody(t, js, "drawOverlapLens"),
		`themeEffect("territoryBlend")`,
		`themeEffect("glowStrength")`,
	)
	requireContains(t, functionBody(t, js, "drawBoundary"),
		`themeEffect("glowStrength")`,
		`themeEffect("shadowStrength")`,
	)
	requireContains(t, functionBody(t, js, "strokeLabel"),
		`themeColor("labelHalo")`,
		`themeEffect("labelHalo")`,
	)
	requireContains(t, functionBody(t, js, "drawDevice"), `themeEffect("glowStrength")`)
	requireContains(t, functionBody(t, js, "drawService"), `themeEffect("glowStrength")`)
	requireContains(t, functionBody(t, js, "draw"), `themeEffect("territoryBlend")`)
	if strings.Contains(js, `globalCompositeOperation = "screen"`) {
		t.Error("app.js hard-codes screen compositing instead of reading the territory blend token")
	}
}

func TestCanvasGridReadsThemeToken(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	requireContains(t, functionBody(t, js, "drawBackground"), `themeColor("grid")`)
}
