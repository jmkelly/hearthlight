(() => {
  "use strict";

  const STORAGE_KEY = "hearthlight.theme.v1";
  const CONTRAST_STORAGE_KEY = "hearthlight.contrast.v1";
  const DEFAULT_THEME = Object.freeze({ family: "hearthlight", mode: "dark" });
  const AUTO_MODES = new Set(["auto", "system"]);
  const MODES = new Set(["dark", "light"]);
  const QUERY_MODES = new Set(["dark", "light", "auto"]);

  const FORBIDDEN_KEYS = new Set(["__proto__", "constructor", "prototype"]);
  const ID_PATTERN = /^[A-Za-z0-9][A-Za-z0-9._:-]*$/;
  const MAX_ID_LENGTH = 64;
  const MAX_NAME_LENGTH = 64;

  const root = document.documentElement;
  const lightQuery = window.matchMedia("(prefers-color-scheme: light)");
  const contrastQuery = window.matchMedia("(prefers-contrast: more)");
  const catalog = window.HearthlightThemes;
  const builtinFamilies = catalog && Array.isArray(catalog.families)
    ? catalog.families.slice()
    : [];
  let families = builtinFamilies.slice();
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

  const isPlainObject = (value) =>
    value !== null && typeof value === "object" && !Array.isArray(value);

  const hasForbiddenKeys = (value) => {
    if (Array.isArray(value)) return value.some(hasForbiddenKeys);
    if (!isPlainObject(value)) return false;
    for (const key of Object.keys(value)) {
      if (FORBIDDEN_KEYS.has(key) || hasForbiddenKeys(value[key])) return true;
    }
    return false;
  };

  const copyStringMap = (value) => {
    if (!isPlainObject(value)) return null;
    const copy = Object.create(null);
    for (const key of Object.keys(value)) {
      if (FORBIDDEN_KEYS.has(key) || typeof value[key] !== "string") return null;
      copy[key] = value[key];
    }
    return copy;
  };

  const copyKinds = (value) => {
    if (value === undefined || value === null) return null;
    if (!isPlainObject(value)) return null;
    const copy = Object.create(null);
    for (const key of Object.keys(value)) {
      if (FORBIDDEN_KEYS.has(key)) return null;
      const kind = value[key];
      if (!isPlainObject(kind)) return null;
      if (typeof kind.label !== "string" || typeof kind.color !== "string") return null;
      copy[key] = { label: kind.label, color: kind.color };
    }
    return copy;
  };

  const copyEffects = (value) => {
    if (value === undefined || value === null) return null;
    if (!isPlainObject(value)) return null;
    const copy = {};
    for (const key of Object.keys(value)) {
      if (FORBIDDEN_KEYS.has(key)) return null;
      const item = value[key];
      if (typeof item !== "number" && typeof item !== "string") return null;
      copy[key] = item;
    }
    return copy;
  };

  const copySource = (value) => {
    if (typeof value === "string" && value.trim()) return { name: value.trim() };
    if (!isPlainObject(value)) return null;
    const copy = {};
    if (typeof value.name === "string" && value.name.trim()) copy.name = value.name.trim();
    if (typeof value.url === "string" && value.url.trim()) copy.url = value.url.trim();
    return Object.keys(copy).length ? copy : null;
  };

  const cleanName = (value) => {
    if (typeof value !== "string") return null;
    const name = value.trim();
    if (!name || Array.from(name).length > MAX_NAME_LENGTH) return null;
    if (/[\u0000-\u001f\u007f]/.test(name)) return null;
    return name;
  };

  const cleanID = (value) => {
    if (typeof value !== "string") return null;
    const id = value.trim();
    if (!id || id.length > MAX_ID_LENGTH || !ID_PATTERN.test(id)) return null;
    return id;
  };

  const copyVariant = (variant) => {
    if (!isPlainObject(variant) || hasForbiddenKeys(variant)) return null;
    const id = cleanID(variant.id);
    const name = cleanName(variant.name);
    const mode = variant.mode === "dark" || variant.mode === "light" ? variant.mode : null;
    const palette = copyStringMap(variant.palette);
    if (!id || !name || !mode || !palette || !Object.keys(palette).length) return null;
    const result = { id, name, mode, palette };
    const kinds = copyKinds(variant.kinds);
    if (kinds) result.kinds = kinds;
    const categories = copyStringMap(variant.categories);
    if (categories) result.categories = categories;
    const effects = copyEffects(variant.effects);
    if (effects) result.effects = effects;
    return result;
  };

  const copyFamily = (family) => {
    if (!isPlainObject(family) || hasForbiddenKeys(family)) return null;
    const id = cleanID(family.id);
    const name = cleanName(family.name);
    if (!id || !name || !Array.isArray(family.variants) || !family.variants.length) return null;
    const variants = [];
    for (const rawVariant of family.variants) {
      const variant = copyVariant(rawVariant);
      if (!variant) return null;
      variants.push(variant);
    }
    const result = { id, name, custom: true, variants };
    const source = copySource(family.source);
    if (source) result.source = source;
    return result;
  };

  // Replaces the custom family set. Built-ins always stay in the catalog, and
  // a family that reuses a built-in or custom ID is rejected instead of merged.
  const registerFamilies = (input) => {
    const familyIDs = new Set(builtinFamilies.map((family) => family && family.id).filter(Boolean));
    const variantIDs = new Set();
    for (const family of builtinFamilies) {
      if (!family || !Array.isArray(family.variants)) continue;
      for (const variant of family.variants) {
        if (variant && variant.id) variantIDs.add(variant.id);
      }
    }
    const custom = [];
    const rejected = [];
    const list = Array.isArray(input) ? input : [];
    for (const rawFamily of list) {
      const family = copyFamily(rawFamily);
      if (!family) {
        rejected.push({ id: rawFamily && rawFamily.id, reason: "invalid family shape" });
        continue;
      }
      if (familyIDs.has(family.id)) {
        rejected.push({ id: family.id, reason: "family id is already in use" });
        continue;
      }
      const pendingVariants = new Set();
      const pendingModes = new Set();
      let conflict = "";
      for (const variant of family.variants) {
        if (variantIDs.has(variant.id) || pendingVariants.has(variant.id)) {
          conflict = "variant id " + variant.id + " is already in use";
          break;
        }
        if (pendingModes.has(variant.mode)) {
          conflict = "mode " + variant.mode + " appears twice";
          break;
        }
        pendingVariants.add(variant.id);
        pendingModes.add(variant.mode);
      }
      if (conflict) {
        rejected.push({ id: family.id, reason: conflict });
        continue;
      }
      familyIDs.add(family.id);
      for (const variant of family.variants) variantIDs.add(variant.id);
      custom.push(family);
    }
    families = builtinFamilies.slice().concat(custom);
    update(initialPreference());
    return { families: custom.slice(), rejected };
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
    families: () => families.slice(),
    registerFamilies,
  });
})();
