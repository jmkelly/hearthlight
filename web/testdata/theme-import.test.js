"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

const source = fs.readFileSync(path.join(__dirname, "..", "theme-import.js"), "utf8");

function createThemeImport() {
  const store = new Map();
  const window = {
    localStorage: {
      getItem: (key) => (store.has(key) ? store.get(key) : null),
      setItem: (key, value) => store.set(key, String(value)),
      removeItem: (key) => store.delete(key),
    },
    addEventListener() {},
    HearthlightThemes: { families: [] },
  };
  const context = vm.createContext({ window, console, TextEncoder, URL });
  vm.runInContext(source, context, { filename: "theme-import.js" });
  return window.HearthlightThemeImport;
}

const api = createThemeImport();

const plain = (value) => JSON.parse(JSON.stringify(value));

const fullDark = `# Tokyo Night, pasted from a third-party theme
mode = "dark"
background = "#1a1b26"
dark_background = "#16161e"
lighter_background = "#292e42"
foreground = "#c0caf5"
dark_foreground = "#a9b1d6"
accent = "#7aa2f7"
green = "#9ece6a"
yellow = "#e0af68"
red = "#f7768e"
blue = "#7aa2f7"
magenta = "#bb9af7"
cursor = "#c0caf5"
unknown_array = [1, 2, 3]

[colors]
color0 = "#000000"
color1 = "#ffffff"
`;

const fullLight = `mode = "light"
background = "#fdf6e3"
foreground = "#657b83"
accent = "#268bd2"
green = "#859900"
yellow = "#b58900"
red = "#dc322f"
blue = "#268bd2"
cyan = "#2aa198"
magenta = "#d33682"
`;

function variantOf(result) {
  assert.equal(result.ok, true, JSON.stringify(result.errors));
  assert.equal(result.families.length, 1);
  assert.equal(result.families[0].variants.length, 1);
  return result.families[0].variants[0];
}

function requirePalette(palette, expected) {
  for (const [token, value] of Object.entries(expected)) {
    assert.equal(palette[token], value, "palette." + token);
  }
  for (const token of api.REQUIRED_PALETTE_TOKENS) {
    assert.equal(typeof palette[token], "string", "missing palette token " + token);
    assert.ok(palette[token].length > 0, "empty palette token " + token);
  }
}

let failures = 0;
function test(name, run) {
  try {
    run();
    console.log("ok - " + name);
  } catch (error) {
    failures += 1;
    console.error("not ok - " + name);
    console.error(error && error.stack ? error.stack : error);
  }
}

test("parses a full dark colors.toml", () => {
  const result = api.parse(fullDark);
  assert.equal(result.format, "toml");
  assert.equal(result.mode, "dark");
  assert.deepEqual(plain(result.missing), []);
  assert.deepEqual(plain(result.ignored), ["cursor", "unknown_array"]);
  const variant = variantOf(result);
  assert.equal(result.families[0].name, "Omarchy Import");
  assert.match(result.families[0].id, /^omarchy-[0-9a-f]{12}$/);
  assert.equal(variant.id, result.families[0].id + "-dark");
  assert.equal(variant.mode, "dark");
  requirePalette(variant.palette, {
    bg: "#1a1b26",
    canvasMiddle: "#1a1b26",
    canvasTop: "#292e42",
    canvasBottom: "#16161e",
    ink: "#c0caf5",
    muted: "#a9b1d6",
    history: "#a9b1d6",
    accent: "#7aa2f7",
    online: "#9ece6a",
    warning: "#e0af68",
    danger: "#f7768e",
    line: "rgba(192, 202, 245, 0.14)",
    glass: "rgba(26, 27, 38, 0.74)",
    grid: "rgba(122, 162, 247, 0.06)",
    ambientGlow: "rgba(122, 162, 247, 0.12)",
    vignette: "rgba(22, 22, 30, 0.55)",
    labelHalo: "rgba(22, 22, 30, 0.88)",
  });
  assert.equal(variant.kinds.router.color, "#e0af68");
  assert.equal(variant.kinds.server.color, "#7aa2f7");
  assert.equal(variant.kinds.phone.color, "#bb9af7");
  assert.equal(variant.categories.remote, "#9ece6a");
  assert.equal(Object.keys(variant.kinds).length, 9);
  assert.equal(Object.keys(variant.categories).length, 7);
  assert.equal(variant.effects.territoryBlend, "screen");
  assert.equal(variant.effects.vignette, 0.52);
  assert.equal(variant.effects.territoryAlpha, 1);
});

test("parses a full light colors.toml", () => {
  const result = api.parse(fullLight);
  assert.equal(result.format, "toml");
  assert.equal(result.mode, "light");
  assert.deepEqual(plain(result.missing), []);
  const variant = variantOf(result);
  assert.equal(variant.mode, "light");
  requirePalette(variant.palette, {
    bg: "#fdf6e3",
    ink: "#657b83",
    accent: "#268bd2",
    online: "#859900",
    warning: "#b58900",
    danger: "#dc322f",
    line: "rgba(101, 123, 131, 0.16)",
    glass: "rgba(253, 246, 227, 0.82)",
    grid: "rgba(38, 139, 210, 0.07)",
    ambientGlow: "rgba(38, 139, 210, 0.16)",
    labelHalo: "rgba(253, 247, 229, 0.88)",
  });
  assert.equal(variant.effects.territoryBlend, "multiply");
  assert.equal(variant.effects.vignette, 0.3);
  assert.equal(variant.effects.territoryAlpha, 0.72);
});

test("reports missing keys and derives the mode", () => {
  const input = `# only two top-level colours
background = "#101010" # nearly black
foreground = "#f0f0f0"

[colors]
accent = "#00ff00"
`;
  const result = api.parse(input);
  assert.equal(result.format, "toml");
  assert.equal(result.mode, "dark");
  assert.deepEqual(plain(result.missing), ["accent", "green", "yellow", "red", "blue", "magenta", "mode"]);
  assert.deepEqual(plain(result.ignored), []);
  const variant = variantOf(result);
  assert.equal(variant.palette.bg, "#101010");
  assert.equal(variant.palette.ink, "#f0f0f0");
  assert.equal(variant.palette.accent, "#5ee7ff");
  assert.equal(variant.palette.online, "#5ee7a5");
  assert.equal(variant.palette.warning, "#ffd166");
  assert.equal(variant.palette.danger, "#ff7b72");
  assert.equal(variant.palette.muted, "#8b8b8b");
  requirePalette(variant.palette, {});
});

test("ignores comments, quoted keys, and nested tables", () => {
  const input = `"background" = "#ffffff" # light surface
'foreground' = '#111111'
[misc]
background = "#000000"
unknown = "#abcdef"
`;
  const result = api.parse(input);
  assert.equal(result.ok, true);
  assert.equal(result.mode, "light");
  assert.deepEqual(plain(result.ignored), []);
  assert.equal(variantOf(result).palette.bg, "#ffffff");
  assert.equal(variantOf(result).palette.ink, "#111111");
});

test("accepts #rrggbbaa colours", () => {
  const input = 'background = "#101010"\nforeground = "#f0f0f0"\naccent = "#7aa2f780"\n';
  const result = api.parse(input);
  const variant = variantOf(result);
  assert.equal(variant.palette.accent, "#7aa2f780");
});

test("rejects malformed input with useful categories", () => {
  const cases = [
    ["missing equals", "background \"#fff\"\n", "malformed"],
    ["unterminated table", "[colors\nbackground = \"#fff\"\n", "malformed"],
    ["unquoted value", "background = #fff\n", "malformed"],
    ["unclosed quote", "background = \"#fff\n", "malformed"],
    ["bad colour", "background = \"red\"\n", "invalid_color"],
    ["short hex", "background = \"#12345\"\n", "invalid_color"],
    ["bad mode", "mode = \"dim\"\nbackground = \"#fff\"\n", "invalid_mode"],
    ["only comments", "# nothing here\n", "malformed"],
    ["forbidden key", "__proto__ = \"#fff\"\nbackground = \"#fff\"\n", "forbidden_key"],
    ["constructor key", "constructor = \"#fff\"\nbackground = \"#fff\"\n", "forbidden_key"],
    ["prototype key", "prototype = \"#fff\"\nbackground = \"#fff\"\n", "forbidden_key"],
    ["too large", "#".repeat(api.MAX_IMPORT_BYTES + 1), "too_large"],
  ];
  for (const [name, input, category] of cases) {
    const result = api.parse(input);
    assert.equal(result.ok, false, name);
    assert.ok(result.errors.some((error) => error.category === category), name + ": " + JSON.stringify(result.errors));
  }
});

test("ignores unknown keys with unquoted values instead of failing", () => {
  const result = api.parse('background = "#101010"\nforeground = "#f0f0f0"\nsetting = 42\n');
  assert.equal(result.ok, true);
  assert.deepEqual(plain(result.ignored), ["setting"]);
});

test("round-trips a TOML import through JSON export", () => {
  const first = api.parse(fullDark);
  const variant = variantOf(first);
  const exported = api.exportDocument(first.families);
  const second = api.parse(exported);
  assert.equal(second.format, "json");
  assert.equal(second.ok, true);
  const roundTrip = variantOf(second);
  assert.equal(roundTrip.id, variant.id);
  assert.equal(roundTrip.name, variant.name);
  assert.equal(roundTrip.mode, variant.mode);
  assert.deepEqual(plain(roundTrip.palette), plain(variant.palette));
  assert.deepEqual(plain(roundTrip.kinds), plain(variant.kinds));
  assert.deepEqual(plain(roundTrip.categories), plain(variant.categories));
  assert.deepEqual(plain(roundTrip.effects), plain(variant.effects));
});

test("keeps JSON detection for existing imports", () => {
  const doc = JSON.stringify({
    schemaVersion: 1,
    family: {
      id: "manual",
      name: "Manual",
      variants: {
        dark: {
          id: "manual-dark",
          name: "Manual",
          mode: "dark",
          palette: {
            bg: "#000000", ink: "#ffffff", muted: "#888888", accent: "#00ffff",
            line: "rgba(255, 255, 255, 0.14)", glass: "rgba(0, 0, 0, 0.74)",
            canvasTop: "#111111", canvasMiddle: "#000000", canvasBottom: "#000000",
            grid: "rgba(0, 255, 255, 0.06)", ambientGlow: "rgba(0, 255, 255, 0.12)",
            online: "#00ff00", warning: "#ffff00", danger: "#ff0000",
            history: "#888888", vignette: "rgba(0, 0, 0, 0.55)",
            labelHalo: "rgba(0, 0, 0, 0.88)", bubble: "rgba(0, 0, 0, 0.94)",
            panel: "rgba(0, 0, 0, 0.95)",
          },
        },
      },
    },
  });
  const result = api.parse(doc);
  assert.equal(result.format, "json");
  assert.equal(result.ok, true);
  assert.equal(result.families.length, 1);
});

if (failures) {
  console.error(failures + " test(s) failed");
  process.exit(1);
}
console.log("all theme import TOML tests passed");
