(() => {
  "use strict";

  const STORAGE_KEY = "hearthlight.theme.v1";
  const CONTRAST_STORAGE_KEY = "hearthlight.contrast.v1";
  const DEFAULT_THEME = Object.freeze({ family: "hearthlight", mode: "dark" });
  const AUTO_MODES = new Set(["auto", "system"]);
  const MODES = new Set(["dark", "light"]);
  const QUERY_MODES = new Set(["dark", "light", "auto"]);

  const root = document.documentElement;
  const lightQuery = window.matchMedia("(prefers-color-scheme: light)");
  const contrastQuery = window.matchMedia("(prefers-contrast: more)");
  const catalog = window.HearthlightThemes;
  const families = catalog && Array.isArray(catalog.families) ? catalog.families : [];
  const appliedTokens = new Set();

  const kebabCase = (name) => name.replace(/[A-Z]/g, (letter) => "-" + letter.toLowerCase());

  const findFamily = (id) => families.find((family) => family && family.id === id) || null;

  const findVariant = (family, mode) => {
    if (!family || !Array.isArray(family.variants)) return null;
    return family.variants.find((variant) => variant && variant.mode === mode) || null;
  };

  const findVariantById = (id) => {
    if (typeof id !== "string" || !id) return null;
    for (const family of families) {
      if (!family || !Array.isArray(family.variants)) continue;
      const variant = family.variants.find((candidate) => candidate && candidate.id === id);
      if (variant) return { family, variant };
    }
    return null;
  };

  const firstVariant = (family) => {
    if (!family || !Array.isArray(family.variants)) return null;
    return family.variants.find((variant) => variant && typeof variant === "object") || null;
  };

  const normalizePreference = (preference) => {
    if (!preference || typeof preference !== "object" || Array.isArray(preference)) {
      return { ...DEFAULT_THEME };
    }
    let family =
      typeof preference.family === "string" && findFamily(preference.family)
        ? preference.family
        : null;
    if (!family && typeof preference.variant === "string") {
      const match = findVariantById(preference.variant);
      if (match) family = match.family.id;
    }
    const mode =
      typeof preference.mode === "string" &&
      (MODES.has(preference.mode) || AUTO_MODES.has(preference.mode))
        ? preference.mode
        : DEFAULT_THEME.mode;
    return { family: family || DEFAULT_THEME.family, mode };
  };

  const readPreference = () => {
    let stored = null;
    try {
      stored = JSON.parse(window.localStorage.getItem(STORAGE_KEY) || "null");
    } catch {
      stored = null;
    }
    return normalizePreference(stored);
  };

  const readQueryOverride = () => {
    let params;
    try {
      params = new URLSearchParams(window.location.search);
    } catch {
      return null;
    }
    const theme = params.get("theme");
    const mode = params.get("mode");
    if (theme === null && mode === null) return null;
    const override = {};
    if (theme !== null) {
      const match = findVariantById(theme);
      if (!match) return null;
      override.family = match.family.id;
      override.variant = match.variant.id;
      override.mode = match.variant.mode;
    }
    if (mode !== null) {
      if (!QUERY_MODES.has(mode)) return null;
      override.mode = mode;
    }
    return override;
  };

  const queryPreference = readQueryOverride();

  const mergePreference = (stored, override) => {
    if (!override) return { ...stored };
    const merged = {
      family: override.family || stored.family,
      mode: override.mode || stored.mode,
    };
    if (override.variant) merged.variant = override.variant;
    return merged;
  };

  const initialPreference = () => {
    return mergePreference(readPreference(), queryPreference);
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

  const resolve = (preference) => {
    const source = preference && typeof preference === "object" ? preference : initialPreference();
    const normalized = normalizePreference(source);
    const requestedMode = normalized.mode;
    const resolvedMode = resolveMode(requestedMode);
    let family = findFamily(normalized.family) || findFamily(DEFAULT_THEME.family);
    const requestedVariant =
      typeof source.variant === "string" ? findVariantById(source.variant) : null;
    let variant = requestedVariant ? requestedVariant.variant : null;
    let fallback = false;
    if (requestedVariant) {
      family = requestedVariant.family;
    } else {
      variant = findVariant(family, resolvedMode);
      if (!variant) {
        variant = findVariant(family, DEFAULT_THEME.mode) || firstVariant(family);
        fallback = true;
      }
      if (typeof source.variant === "string") fallback = true;
    }
    if (!family) fallback = true;
    const palette =
      variant && variant.palette && typeof variant.palette === "object" ? variant.palette : null;
    return {
      family: family ? family.id : DEFAULT_THEME.family,
      variant: variant ? variant.id : null,
      requestedMode,
      resolvedMode,
      fallback,
      palette,
      kinds: variant && variant.kinds ? variant.kinds : null,
      categories: variant && variant.categories ? variant.categories : null,
      effects: variant && variant.effects ? variant.effects : null,
    };
  };

  const apply = (theme) => {
    const resolved =
      theme && typeof theme === "object" && typeof theme.resolvedMode === "string"
        ? theme
        : resolve(theme || initialPreference());
    const family =
      typeof resolved.family === "string"
        ? resolved.family
        : (resolved.family && resolved.family.id) || DEFAULT_THEME.family;
    const mode = resolved.resolvedMode || DEFAULT_THEME.mode;
    root.dataset.theme = family;
    root.dataset.mode = mode;
    applyPalette(resolved.palette);
    setMeta("theme-color", resolved.palette && resolved.palette.bg);
    setMeta("color-scheme", mode);
    return resolved;
  };

  const dispatchTheme = (theme) => {
    if (typeof window.CustomEvent !== "function") return;
    window.dispatchEvent(new CustomEvent("hearthlight:theme", { detail: theme }));
  };

  const update = (preference) => {
    const theme = apply(resolve(preference));
    dispatchTheme(theme);
    return theme;
  };

  const persist = (preference) => {
    const normalized = normalizePreference(preference);
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(normalized));
    } catch {
      // Private browsing can disable localStorage; the live theme still applies.
    }
    return update(normalized);
  };

  // The OS asks for more contrast unless the user chose either value explicitly.
  const readContrastOverride = () => {
    let stored = null;
    try {
      stored = window.localStorage.getItem(CONTRAST_STORAGE_KEY);
    } catch {
      return null;
    }
    if (stored === "high") return true;
    if (stored === "normal") return false;
    return null;
  };

  const contrastPreference = () => {
    const override = readContrastOverride();
    return override === null ? contrastQuery.matches : override;
  };

  const dispatchContrast = (enabled) => {
    if (typeof window.CustomEvent !== "function") return;
    window.dispatchEvent(new CustomEvent("hearthlight:contrast", { detail: { enabled } }));
  };

  const applyContrast = (enabled) => {
    const high = Boolean(enabled);
    root.dataset.contrast = high ? "high" : "normal";
    dispatchContrast(high);
    return high;
  };

  apply(resolve(initialPreference()));
  applyContrast(contrastPreference());

  window.addEventListener("storage", (event) => {
    if (event.key === STORAGE_KEY || event.key === null) update(initialPreference());
    if (event.key === CONTRAST_STORAGE_KEY || event.key === null) {
      applyContrast(contrastPreference());
    }
  });

  if (typeof lightQuery.addEventListener === "function") {
    lightQuery.addEventListener("change", () => update(initialPreference()));
  } else if (typeof lightQuery.addListener === "function") {
    lightQuery.addListener(() => update(initialPreference()));
  }

  const onContrastChange = () => applyContrast(contrastPreference());
  if (typeof contrastQuery.addEventListener === "function") {
    contrastQuery.addEventListener("change", onContrastChange);
  } else if (typeof contrastQuery.addListener === "function") {
    contrastQuery.addListener(onContrastChange);
  }

  window.HearthlightTheme = Object.freeze({
    STORAGE_KEY,
    CONTRAST_STORAGE_KEY,
    DEFAULT_THEME,
    initialPreference,
    contrastPreference,
    applyContrast,
    resolve,
    apply,
    persist,
  });
})();
