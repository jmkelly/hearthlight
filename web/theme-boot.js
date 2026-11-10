(() => {
  "use strict";

  const STORAGE_KEY = "hearthlight.theme.v1";
  const DEFAULT_THEME = Object.freeze({ family: "hearthlight", mode: "dark" });
  const AUTO_MODES = new Set(["auto", "system"]);
  const MODES = new Set(["dark", "light"]);

  const root = document.documentElement;
  const lightQuery = window.matchMedia("(prefers-color-scheme: light)");
  const catalog = window.HearthlightThemes;
  const families = catalog && Array.isArray(catalog.families) ? catalog.families : [];
  const appliedTokens = new Set();

  const kebabCase = (name) => name.replace(/[A-Z]/g, (letter) => "-" + letter.toLowerCase());

  const findFamily = (id) => families.find((family) => family && family.id === id) || null;

  const findVariant = (family, mode) => {
    if (!family || !Array.isArray(family.variants)) return null;
    return family.variants.find((variant) => variant && variant.mode === mode) || null;
  };

  const readPreference = () => {
    let stored = null;
    try {
      stored = JSON.parse(window.localStorage.getItem(STORAGE_KEY) || "null");
    } catch {
      stored = null;
    }
    if (!stored || typeof stored !== "object" || Array.isArray(stored)) {
      return { ...DEFAULT_THEME };
    }
    const family =
      typeof stored.family === "string" && findFamily(stored.family)
        ? stored.family
        : DEFAULT_THEME.family;
    const mode =
      typeof stored.mode === "string" && (MODES.has(stored.mode) || AUTO_MODES.has(stored.mode))
        ? stored.mode
        : DEFAULT_THEME.mode;
    return { family, mode };
  };

  const resolveMode = (mode) => {
    if (AUTO_MODES.has(mode)) return lightQuery.matches ? "light" : "dark";
    return MODES.has(mode) ? mode : DEFAULT_THEME.mode;
  };

  const applyPalette = (palette) => {
    for (const property of appliedTokens) root.style.removeProperty(property);
    appliedTokens.clear();
    if (!palette || typeof palette !== "object") return;
    for (const [name, value] of Object.entries(palette)) {
      if (typeof value !== "string" || !value) continue;
      const property = "--theme-" + kebabCase(name);
      root.style.setProperty(property, value);
      appliedTokens.add(property);
    }
  };

  const setMeta = (name, content) => {
    const tag = document.querySelector('meta[name="' + name + '"]');
    if (tag && content) tag.setAttribute("content", content);
  };

  const apply = () => {
    const preference = readPreference();
    const family = findFamily(preference.family) || findFamily(DEFAULT_THEME.family);
    const mode = resolveMode(preference.mode);
    const variant =
      findVariant(family, mode) ||
      findVariant(family, DEFAULT_THEME.mode) ||
      (family && Array.isArray(family.variants) ? family.variants[0] : null);
    const palette =
      variant && variant.palette && typeof variant.palette === "object" ? variant.palette : null;
    const familyID = (family && family.id) || DEFAULT_THEME.family;
    const variantMode = (variant && variant.mode) || DEFAULT_THEME.mode;

    root.dataset.theme = familyID;
    root.dataset.mode = variantMode;
    applyPalette(palette);
    setMeta("theme-color", palette && palette.bg);
    setMeta("color-scheme", variantMode);
  };

  apply();

  window.addEventListener("storage", (event) => {
    if (event.key === STORAGE_KEY || event.key === null) apply();
  });

  if (typeof lightQuery.addEventListener === "function") {
    lightQuery.addEventListener("change", apply);
  } else if (typeof lightQuery.addListener === "function") {
    lightQuery.addListener(apply);
  }
})();
