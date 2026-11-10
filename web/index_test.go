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
	if strings.Contains(strings.ToLower(dialog), "theme") {
		t.Error("the settings dialog must not add theme controls")
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
