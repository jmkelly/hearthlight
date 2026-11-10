package web

import (
	"os"
	"strings"
	"testing"
)

func readAccessibilityFile(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(raw)
}

func requireContains(t *testing.T, source string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(source, want) {
			t.Errorf("missing %s", want)
		}
	}
}

func markupBetween(t *testing.T, source, start, end string) string {
	t.Helper()
	from := strings.Index(source, start)
	if from < 0 {
		t.Fatalf("missing %q", start)
	}
	rest := source[from+len(start):]
	to := strings.Index(rest, end)
	if to < 0 {
		t.Fatalf("missing %q after %q", end, start)
	}
	return rest[:to]
}

func functionBody(t *testing.T, source, name string) string {
	t.Helper()
	start := strings.Index(source, "function "+name+"(")
	if start < 0 {
		t.Fatalf("missing function %s", name)
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
	t.Fatalf("unterminated function %s", name)
	return ""
}

func cssRule(t *testing.T, source, selector string) string {
	t.Helper()
	start := strings.Index(source, selector)
	if start < 0 {
		t.Fatalf("missing CSS rule %s", selector)
	}
	open := strings.Index(source[start:], "{")
	if open < 0 {
		t.Fatalf("missing body for CSS rule %s", selector)
	}
	open += start
	end := strings.Index(source[open:], "}")
	if end < 0 {
		t.Fatalf("unterminated CSS rule %s", selector)
	}
	return source[open+1 : open+end]
}

func TestDeviceListMarkup(t *testing.T) {
	html := readAccessibilityFile(t, "index.html")
	section := markupBetween(t, html, `<section id="deviceList"`, `</section>`)
	requireContains(t, section,
		`aria-labelledby="deviceListHeading"`,
		`<h2 id="deviceListHeading">Devices</h2>`,
		`<ul id="deviceListItems"></ul>`,
	)
	summary := markupBetween(t, section, `<p id="deviceSummary"`, `</p>`)
	requireContains(t, summary,
		`class="visually-hidden"`,
		`role="status"`,
		`aria-live="polite"`,
		`aria-atomic="true"`,
	)
}

func TestDeviceListIsFedBySnapshots(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	apply := functionBody(t, js, "applySnapshot")
	requireContains(t, apply, "renderDeviceList(snapshot)", "announceDeviceSummary(snapshot)")

	render := functionBody(t, js, "renderDeviceList")
	requireContains(t, render,
		"displayName(device)",
		"device.online",
		"device.ip",
		"KIND[device.kind]",
		"(device.services || []).length",
		`class="device-entry"`,
		`data-device-id="${escapeHTML(device.id)}"`,
		`document.activeElement.closest(".device-entry")`,
		"entry.focus()",
	)
	requireContains(t, js,
		`event.target.closest(".device-entry")`,
		"entry.dataset.deviceId",
		"selectNode(node)",
	)
}

func TestDeviceSummaryIsThrottled(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	requireContains(t, js, "const DEVICE_SUMMARY_THROTTLE_MS = 5000;")

	announce := functionBody(t, js, "announceDeviceSummary")
	requireContains(t, announce,
		"deviceSummaryState.announced",
		"DEVICE_SUMMARY_THROTTLE_MS",
		"window.setTimeout(flushDeviceSummary",
	)
	flush := functionBody(t, js, "flushDeviceSummary")
	requireContains(t, flush, "deviceSummary.textContent = text;")
	if count := strings.Count(js, "deviceSummary.textContent ="); count != 1 {
		t.Errorf("device summary is written from %d places, want the throttled flush only", count)
	}
	for _, name := range []string{"frame", "physicsStep", "draw"} {
		body := functionBody(t, js, name)
		if strings.Contains(body, "deviceSummary") || strings.Contains(body, "announceDeviceSummary") {
			t.Errorf("%s touches the live region and would repeat announcements", name)
		}
	}
}

func TestDeviceListStaysFindableAndHidden(t *testing.T) {
	css := readAccessibilityFile(t, "style.css")
	hidden := cssRule(t, css, ".device-list {")
	requireContains(t, hidden,
		"position: fixed;",
		"width: 1px;",
		"height: 1px;",
		"overflow: hidden;",
		"clip: rect(0, 0, 0, 0);",
		"clip-path: inset(50%);",
	)
	if strings.Contains(hidden, "display: none;") || strings.Contains(hidden, "visibility: hidden;") {
		t.Error("device list must stay in the document so Ctrl+F and screen readers can find names")
	}
	focus := cssRule(t, css, ".device-list:focus-within {")
	requireContains(t, focus, "clip: auto;", "clip-path: none;", "width: min(")

	summary := cssRule(t, css, ".visually-hidden {")
	requireContains(t, summary, "overflow: hidden;", "clip: rect(0, 0, 0, 0);", "clip-path: inset(50%);")
}

func TestDeviceSheetMarkup(t *testing.T) {
	html := readAccessibilityFile(t, "index.html")
	button := markupBetween(t, html, `<button id="devicesBtn"`, `</button>`)
	requireContains(t, button,
		`aria-label="Devices"`,
		`aria-expanded="false"`,
		`aria-controls="deviceList"`,
		`aria-keyshortcuts="l"`,
		`<span>Devices</span>`,
	)

	section := markupBetween(t, html, `<section id="deviceList"`, `</section>`)
	requireContains(t, section,
		`aria-labelledby="deviceListHeading"`,
		`aria-describedby="deviceListHint"`,
		`tabindex="-1"`,
		`<p id="deviceListHint"`,
		`<button id="deviceListClose"`,
		`aria-label="Close devices"`,
	)
	hint := markupBetween(t, section, `<p id="deviceListHint"`, `</p>`)
	requireContains(t, hint, "Arrow keys", "Enter", "Escape")
}

func TestCanvasIsNotADuplicateFocusStop(t *testing.T) {
	html := readAccessibilityFile(t, "index.html")
	canvas := markupBetween(t, html, `<canvas id="map"`, `</canvas>`)
	requireContains(t, canvas, `aria-hidden="true"`)
	if strings.Contains(canvas, "tabindex") {
		t.Error("the canvas must not be a second keyboard focus stop beside the device sheet")
	}
}

func TestDeviceSheetKeyboardBehaviour(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	requireContains(t, js,
		`devicesBtn.addEventListener("click", toggleDeviceSheet)`,
		`devicesBtn.setAttribute("aria-expanded"`,
		`case "ArrowDown":`,
		`case "ArrowUp":`,
		`case "Home":`,
		`case "End":`,
		`event.key === "Tab"`,
		`event.key === "Escape"`,
		`event.key.toLowerCase() === "l"`,
		`toggleDeviceSheet();`,
		`event.stopPropagation()`,
		`closeDeviceSheet(false);`,
		`selectNode(node, devicesBtn);`,
	)

	open := functionBody(t, js, "openDeviceSheet")
	requireContains(t, open, "view.deviceSheetOpen = true", "syncDeviceSheet()", "target.focus()")
	close := functionBody(t, js, "closeDeviceSheet")
	requireContains(t, close, "view.deviceSheetOpen = false", "syncDeviceSheet()", "devicesBtn.focus()")
	move := functionBody(t, js, "moveDeviceSheetFocus")
	requireContains(t, move, `document.activeElement.closest(".device-entry")`, "entries[next].focus()")
	trap := functionBody(t, js, "trapDeviceSheetFocus")
	requireContains(t, trap, "FOCUSABLE_SELECTOR", "event.shiftKey", "last.focus()", "first.focus()")
}

func TestDeviceSheetReusesTheVisibleDeviceList(t *testing.T) {
	js := readAccessibilityFile(t, "app.js")
	entries := functionBody(t, js, "deviceEntries")
	requireContains(t, entries, `querySelectorAll(".device-entry")`)
	sync := functionBody(t, js, "syncDeviceSheet")
	requireContains(t, sync, `classList.toggle("open"`, `aria-expanded`)
}

func TestDeviceSheetStyling(t *testing.T) {
	css := readAccessibilityFile(t, "style.css")
	sheet := cssRule(t, css, ".device-list.open,")
	requireContains(t, sheet,
		"z-index: 25;",
		"width: min(",
		"overflow-y: auto;",
		"clip: auto;",
		"clip-path: none;",
	)
	close := cssRule(t, css, ".device-list.open .device-list-close,")
	requireContains(t, close, "display: inline-flex;")
	hint := cssRule(t, css, ".device-list-hint {")
	requireContains(t, hint, "font-size:", "line-height:")
}
