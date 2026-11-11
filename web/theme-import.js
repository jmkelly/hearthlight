(() => {
  "use strict";

  const STORAGE_KEY = "hearthlight.themes.custom.v1";
  const MAX_IMPORT_BYTES = 256 * 1024;
  const MAX_CUSTOM_FAMILIES = 50;
  const MAX_NAME_LENGTH = 64;
  const MAX_ID_LENGTH = 64;
  const SCHEMA_VERSION = 1;

  const REQUIRED_PALETTE_TOKENS = Object.freeze([
    "bg", "ink", "muted", "accent", "line", "glass",
    "canvasTop", "canvasMiddle", "canvasBottom", "grid", "ambientGlow",
    "online", "warning", "danger", "history",
    "vignette", "labelHalo", "bubble", "panel",
  ]);

  const DEFAULT_EFFECTS = Object.freeze({
    dark: Object.freeze({
      vignette: 0.52,
      deviceGlow: 0.22,
      devicePulse: 0.3,
      serviceGlow: 0.16,
      territoryBlur: 26,
      ribbonBlur: 16,
      lensBlur: 14,
      glassBlur: 16,
      panelBlur: 22,
      territoryBlend: "screen",
      territoryAlpha: 1,
      glowStrength: 1,
      labelHalo: 1,
      shadowStrength: 1,
    }),
    light: Object.freeze({
      vignette: 0.3,
      deviceGlow: 0.14,
      devicePulse: 0.22,
      serviceGlow: 0.08,
      territoryBlur: 26,
      ribbonBlur: 16,
      lensBlur: 14,
      glassBlur: 16,
      panelBlur: 22,
      territoryBlend: "multiply",
      territoryAlpha: 0.72,
      glowStrength: 0.6,
      labelHalo: 0.9,
      shadowStrength: 0.5,
    }),
  });

  const NUMERIC_EFFECT_RANGES = Object.freeze({
    vignette: [0, 1],
    deviceGlow: [0, 1],
    devicePulse: [0, 1],
    serviceGlow: [0, 1],
    territoryBlur: [0, 64],
    ribbonBlur: [0, 64],
    lensBlur: [0, 64],
    glassBlur: [0, 64],
    panelBlur: [0, 64],
    territoryAlpha: [0, 1],
    glowStrength: [0, 4],
    labelHalo: [0, 1],
    shadowStrength: [0, 4],
  });

  const BLEND_MODES = new Set([
    "normal", "source-over", "multiply", "screen", "overlay", "darken", "lighten",
    "color-dodge", "color-burn", "hard-light", "soft-light", "difference",
    "exclusion", "hue", "saturation", "color", "luminosity",
  ]);

  const FORBIDDEN_KEYS = new Set(["__proto__", "constructor", "prototype"]);
  const ID_PATTERN = /^[A-Za-z0-9][A-Za-z0-9._:-]*$/;
  const MAP_KEY_PATTERN = /^[A-Za-z][A-Za-z0-9_-]*$/;
  const PALETTE_KEY_PATTERN = /^[A-Za-z][A-Za-z0-9]*$/;
  const HEX_COLOR_PATTERN = /^[0-9A-Fa-f]+$/;
  const RGB_COLOR_PATTERN = /^rgba?\(\s*([0-9]{1,3})\s*,\s*([0-9]{1,3})\s*,\s*([0-9]{1,3})\s*(?:,\s*([0-9]*\.?[0-9]+)\s*)?\)$/;

  let storageMode = "unknown";

  const isPlainObject = (value) =>
    value !== null && typeof value === "object" && !Array.isArray(value);

  const hasValue = (object, key) =>
    Object.prototype.hasOwnProperty.call(object, key) && object[key] !== null && object[key] !== undefined;

  const byteLength = (text) => {
    if (typeof TextEncoder === "function") return new TextEncoder().encode(text).length;
    return text.length;
  };

  const isQuotaError = (error) =>
    Boolean(error) &&
    (error.name === "QuotaExceededError" ||
      error.name === "NS_ERROR_DOM_QUOTA_REACHED" ||
      error.code === 22 ||
      error.code === 1014);

  const pushError = (errors, category, path, message) => {
    errors.push({ category, path, message });
  };

  const rejectForbiddenKeys = (value, path, errors) => {
    if (Array.isArray(value)) {
      for (let index = 0; index < value.length; index += 1) {
        if (rejectForbiddenKeys(value[index], path + "[" + index + "]", errors)) return true;
      }
      return false;
    }
    if (!isPlainObject(value)) return false;
    for (const key of Object.keys(value)) {
      const childPath = path + "." + key;
      if (FORBIDDEN_KEYS.has(key)) {
        pushError(errors, "forbidden_key", childPath, "Key \"" + key + "\" is not allowed.");
        return true;
      }
      if (rejectForbiddenKeys(value[key], childPath, errors)) return true;
    }
    return false;
  };

  const validateName = (value, path, errors) => {
    if (typeof value !== "string") {
      pushError(errors, "invalid_name", path, "Name must be a string.");
      return null;
    }
    const name = value.trim();
    if (!name) {
      pushError(errors, "invalid_name", path, "Name must not be empty.");
      return null;
    }
    if (Array.from(name).length > MAX_NAME_LENGTH) {
      pushError(errors, "invalid_name", path, "Name is longer than " + MAX_NAME_LENGTH + " characters.");
      return null;
    }
    if (/[\u0000-\u001f\u007f]/.test(name)) {
      pushError(errors, "invalid_name", path, "Name must not contain control characters.");
      return null;
    }
    return name;
  };

  const validateID = (value, path, errors) => {
    if (typeof value !== "string") {
      pushError(errors, "invalid_id", path, "ID must be a string.");
      return null;
    }
    const id = value.trim();
    if (!id) {
      pushError(errors, "invalid_id", path, "ID must not be empty.");
      return null;
    }
    if (byteLength(id) > MAX_ID_LENGTH) {
      pushError(errors, "invalid_id", path, "ID is longer than " + MAX_ID_LENGTH + " bytes.");
      return null;
    }
    if (!ID_PATTERN.test(id)) {
      pushError(errors, "invalid_id", path, "ID must start with a letter or digit and contain only letters, digits, dot, underscore, hyphen, or colon.");
      return null;
    }
    return id;
  };

  const validateMapKey = (key, path, pattern, errors) => {
    if (FORBIDDEN_KEYS.has(key)) {
      pushError(errors, "forbidden_key", path, "Key \"" + key + "\" is not allowed.");
      return false;
    }
    if (byteLength(key) > MAX_NAME_LENGTH || !pattern.test(key)) {
      pushError(errors, "invalid_key", path, "Key \"" + key + "\" is not a valid token name.");
      return false;
    }
    return true;
  };

  const parseColor = (value) => {
    if (typeof value !== "string") return false;
    const text = value.trim();
    if (!text) return false;
    if (text[0] === "#") {
      const digits = text.slice(1);
      if (!HEX_COLOR_PATTERN.test(digits)) return false;
      return digits.length === 3 || digits.length === 4 || digits.length === 6 || digits.length === 8;
    }
    const match = text.match(RGB_COLOR_PATTERN);
    if (!match) return false;
    for (let index = 1; index <= 3; index += 1) {
      if (Number(match[index]) > 255) return false;
    }
    if (match[4] !== undefined && match[4] !== "") {
      const alpha = Number(match[4]);
      if (!Number.isFinite(alpha) || alpha < 0 || alpha > 1) return false;
    }
    return true;
  };

  const validateColor = (value, path, errors) => {
    if (typeof value !== "string") {
      pushError(errors, "invalid_color", path, "Colour must be a string.");
      return null;
    }
    if (!parseColor(value)) {
      pushError(errors, "invalid_color", path, "\"" + value + "\" is not a valid hex or rgb/rgba colour.");
      return null;
    }
    return value.trim();
  };

  const sortedKeys = (object) => Object.keys(object).sort();

  const requiredString = (object, key, path, category, errors) => {
    if (!hasValue(object, key)) {
      pushError(errors, category, path + "." + key, key + " is required.");
      return null;
    }
    if (typeof object[key] !== "string") {
      pushError(errors, category, path + "." + key, key + " must be a string.");
      return null;
    }
    return object[key];
  };

  const parsePalette = (object, path, errors) => {
    const paletteValue = object.palette;
    const colorsValue = object.colors;
    const hasPalette = paletteValue !== null && paletteValue !== undefined;
    const hasColors = colorsValue !== null && colorsValue !== undefined;
    if (hasPalette && hasColors) {
      pushError(errors, "invalid_shape", path, "Use either palette or colors, not both.");
      return Object.create(null);
    }
    const field = hasColors ? "colors" : "palette";
    const value = hasColors ? colorsValue : paletteValue;
    const fieldPath = path + "." + field;
    if (!hasValue(object, field)) {
      pushError(errors, "missing_token", fieldPath, "palette or colors is required.");
      return Object.create(null);
    }
    if (!isPlainObject(value)) {
      pushError(errors, "invalid_shape", fieldPath, field + " must be an object.");
      return Object.create(null);
    }
    const palette = Object.create(null);
    for (const key of sortedKeys(value)) {
      const keyPath = fieldPath + "." + key;
      if (!validateMapKey(key, keyPath, PALETTE_KEY_PATTERN, errors)) continue;
      if (typeof value[key] !== "string") {
        pushError(errors, "invalid_color", keyPath, "Colour must be a string.");
        continue;
      }
      if (!parseColor(value[key])) {
        pushError(errors, "invalid_color", keyPath, "\"" + value[key] + "\" is not a valid hex or rgb/rgba colour.");
        continue;
      }
      palette[key] = value[key].trim();
    }
    for (const token of REQUIRED_PALETTE_TOKENS) {
      if (!Object.prototype.hasOwnProperty.call(palette, token)) {
        pushError(errors, "missing_token", fieldPath + "." + token, "Required palette token \"" + token + "\" is missing.");
      }
    }
    return palette;
  };

  const parseEffects = (value, path, mode, errors) => {
    const effects = Object.assign({}, DEFAULT_EFFECTS[mode === "light" ? "light" : "dark"]);
    if (value === null || value === undefined) return effects;
    if (!isPlainObject(value)) {
      pushError(errors, "invalid_effect", path, "effects must be an object.");
      return effects;
    }
    for (const key of sortedKeys(value)) {
      const keyPath = path + "." + key;
      const item = value[key];
      if (key === "territoryBlend") {
        if (typeof item !== "string" || !BLEND_MODES.has(item)) {
          pushError(errors, "invalid_effect", keyPath, "territoryBlend must be a supported blend mode.");
          continue;
        }
        effects.territoryBlend = item;
        continue;
      }
      const limits = NUMERIC_EFFECT_RANGES[key];
      if (!limits) {
        pushError(errors, "invalid_effect", keyPath, "Unknown effect \"" + key + "\".");
        continue;
      }
      if (typeof item !== "number" || !Number.isFinite(item)) {
        pushError(errors, "invalid_effect", keyPath, "Effect must be a finite number.");
        continue;
      }
      if (item < limits[0] || item > limits[1]) {
        pushError(errors, "invalid_effect", keyPath, "Effect " + item + " is outside the allowed range " + limits[0] + " to " + limits[1] + ".");
        continue;
      }
      effects[key] = item;
    }
    return effects;
  };

  const parseKinds = (value, path, errors) => {
    if (value === null || value === undefined) return null;
    if (!isPlainObject(value)) {
      pushError(errors, "invalid_shape", path, "kinds must be an object.");
      return null;
    }
    const kinds = Object.create(null);
    for (const key of sortedKeys(value)) {
      const keyPath = path + "." + key;
      if (!validateMapKey(key, keyPath, MAP_KEY_PATTERN, errors)) continue;
      const kind = value[key];
      if (!isPlainObject(kind)) {
        pushError(errors, "invalid_shape", keyPath, "kind must be an object.");
        continue;
      }
      const label = requiredString(kind, "label", keyPath, "invalid_name", errors);
      const cleanLabel = label === null ? null : validateName(label, keyPath + ".label", errors);
      const color = requiredString(kind, "color", keyPath, "invalid_color", errors);
      const cleanColor = color === null ? null : validateColor(color, keyPath + ".color", errors);
      if (cleanLabel && cleanColor) kinds[key] = { label: cleanLabel, color: cleanColor };
    }
    return kinds;
  };

  const parseCategories = (value, path, errors) => {
    if (value === null || value === undefined) return null;
    if (!isPlainObject(value)) {
      pushError(errors, "invalid_shape", path, "categories must be an object.");
      return null;
    }
    const categories = Object.create(null);
    for (const key of sortedKeys(value)) {
      const keyPath = path + "." + key;
      if (!validateMapKey(key, keyPath, MAP_KEY_PATTERN, errors)) continue;
      const color = validateColor(value[key], keyPath, errors);
      if (color) categories[key] = color;
    }
    return categories;
  };

  const parseSource = (value, path, errors) => {
    if (value === null || value === undefined) return null;
    if (typeof value === "string") {
      const name = validateName(value, path, errors);
      return name ? { name } : null;
    }
    if (!isPlainObject(value)) {
      pushError(errors, "invalid_shape", path, "source must be a string or an object.");
      return null;
    }
    const source = {};
    if (hasValue(value, "name")) {
      const name = validateName(value.name, path + ".name", errors);
      if (name) source.name = name;
    }
    if (hasValue(value, "url")) {
      if (typeof value.url !== "string" || value.url.length > 512) {
        pushError(errors, "invalid_shape", path + ".url", "url must be a string no longer than 512 bytes.");
      } else {
        const text = value.url.trim();
        if (text) {
          try {
            const parsed = new URL(text);
            if ((parsed.protocol !== "http:" && parsed.protocol !== "https:") || !parsed.hostname) {
              pushError(errors, "invalid_shape", path + ".url", "url must be an http or https URL.");
            } else {
              source.url = text;
            }
          } catch (_error) {
            pushError(errors, "invalid_shape", path + ".url", "url must be an http or https URL.");
          }
        }
      }
    }
    return Object.keys(source).length ? source : null;
  };

  const parseVariant = (object, path, impliedMode, errors) => {
    if (!isPlainObject(object)) {
      pushError(errors, "invalid_shape", path, "variant must be an object.");
      return null;
    }
    const idText = requiredString(object, "id", path, "invalid_id", errors);
    const id = idText === null ? null : validateID(idText, path + ".id", errors);
    const nameText = requiredString(object, "name", path, "invalid_name", errors);
    const name = nameText === null ? null : validateName(nameText, path + ".name", errors);

    let mode = impliedMode || "";
    if (hasValue(object, "mode")) {
      if (typeof object.mode !== "string") {
        pushError(errors, "invalid_mode", path + ".mode", "mode must be a string.");
      } else if (object.mode !== "dark" && object.mode !== "light") {
        pushError(errors, "invalid_mode", path + ".mode", "mode must be dark or light.");
      } else if (impliedMode && object.mode !== impliedMode) {
        pushError(errors, "invalid_mode", path + ".mode", "mode " + object.mode + " conflicts with the " + impliedMode + " variant key.");
      } else {
        mode = object.mode;
      }
    }
    if (!mode) pushError(errors, "invalid_mode", path + ".mode", "mode must be dark or light.");

    const palette = parsePalette(object, path, errors);
    const effects = parseEffects(object.effects, path + ".effects", mode, errors);
    const kinds = parseKinds(object.kinds, path + ".kinds", errors);
    const categories = parseCategories(object.categories, path + ".categories", errors);

    return {
      id: id || "",
      name: name || "",
      mode,
      palette,
      kinds,
      categories,
      effects,
    };
  };

  const parseFamilyVariants = (value, path, errors) => {
    if (Array.isArray(value)) {
      if (!value.length) {
        pushError(errors, "invalid_shape", path, "variants must contain at least one variant.");
        return [];
      }
      const variants = [];
      for (let index = 0; index < value.length; index += 1) {
        const variant = parseVariant(value[index], path + "[" + index + "]", "", errors);
        if (variant) variants.push(variant);
      }
      return variants;
    }
    if (!isPlainObject(value)) {
      pushError(errors, "invalid_shape", path, "variants must be an array or an object keyed by dark and light.");
      return [];
    }
    const keys = sortedKeys(value);
    if (!keys.length) {
      pushError(errors, "invalid_shape", path, "variants must contain at least one variant.");
      return [];
    }
    const variants = [];
    for (const key of keys) {
      if (key !== "dark" && key !== "light") {
        pushError(errors, "invalid_mode", path + "." + key, "variant object keys must be dark or light.");
        continue;
      }
      const variant = parseVariant(value[key], path + "." + key, key, errors);
      if (variant) variants.push(variant);
    }
    return variants;
  };

  const parseFamily = (object, path, errors) => {
    if (!isPlainObject(object)) {
      pushError(errors, "invalid_shape", path, "family must be an object.");
      return null;
    }
    const idText = requiredString(object, "id", path, "invalid_id", errors);
    const id = idText === null ? null : validateID(idText, path + ".id", errors);
    const nameText = requiredString(object, "name", path, "invalid_name", errors);
    const name = nameText === null ? null : validateName(nameText, path + ".name", errors);
    const source = parseSource(object.source, path + ".source", errors);
    const variants = parseFamilyVariants(object.variants, path + ".variants", errors);
    const modes = new Map();
    for (let index = 0; index < variants.length; index += 1) {
      const mode = variants[index].mode;
      if (!mode) continue;
      if (modes.has(mode)) {
        pushError(errors, "duplicate_mode", path + ".variants[" + index + "].mode", "variant mode \"" + mode + "\" is already used.");
      } else {
        modes.set(mode, index);
      }
    }
    return { id: id || "", name: name || "", source, variants };
  };

  const schemaVersionOf = (object, path, errors) => {
    if (!hasValue(object, "schemaVersion")) return SCHEMA_VERSION;
    const value = object.schemaVersion;
    if (typeof value !== "number" || !Number.isInteger(value)) {
      pushError(errors, "malformed", path + ".schemaVersion", "schemaVersion must be an integer.");
      return null;
    }
    if (value !== SCHEMA_VERSION) {
      pushError(errors, "unsupported_schema", path + ".schemaVersion", "schemaVersion " + value + " is not supported, expected " + SCHEMA_VERSION + ".");
      return null;
    }
    return value;
  };

  const parseDocument = (object, errors) => {
    if (schemaVersionOf(object, "$", errors) === null) return [];

    const hasFamilies = hasValue(object, "families");
    const hasFamily = hasValue(object, "family");
    const hasVariant = hasValue(object, "variant");
    const hasVariants = hasValue(object, "variants");
    const hasPalette = hasValue(object, "palette") || hasValue(object, "colors");

    if (hasFamilies) {
      if (hasFamily || hasVariant || hasVariants || hasPalette) {
        pushError(errors, "invalid_shape", "$", "Catalog document must not mix families with family, variant, variants, or palette.");
        return [];
      }
      if (!Array.isArray(object.families)) {
        pushError(errors, "invalid_shape", "$.families", "families must be an array.");
        return [];
      }
      if (!object.families.length) {
        pushError(errors, "invalid_shape", "$.families", "families must contain at least one family.");
        return [];
      }
      const families = [];
      for (let index = 0; index < object.families.length; index += 1) {
        const family = parseFamily(object.families[index], "$.families[" + index + "]", errors);
        if (family) families.push(family);
      }
      return families;
    }
    if (hasFamily) {
      if (hasVariant || hasVariants || hasPalette) {
        pushError(errors, "invalid_shape", "$", "Family document must not mix family with variant, variants, or palette.");
        return [];
      }
      const family = parseFamily(object.family, "$.family", errors);
      return family ? [family] : [];
    }
    if (hasVariant) {
      if (hasVariants || hasPalette) {
        pushError(errors, "invalid_shape", "$", "Variant document must not mix variant with variants or palette.");
        return [];
      }
      const variant = parseVariant(object.variant, "$.variant", "", errors);
      return variant ? [familyForVariant(variant)] : [];
    }
    if (hasVariants) {
      if (hasPalette) {
        pushError(errors, "invalid_shape", "$", "Family document must not mix variants with palette.");
        return [];
      }
      const family = parseFamily(object, "$", errors);
      return family ? [family] : [];
    }
    if (hasPalette) {
      const variant = parseVariant(object, "$", "", errors);
      return variant ? [familyForVariant(variant)] : [];
    }
    pushError(errors, "invalid_shape", "$", "Document must contain families, family, variant, variants, or a palette.");
    return [];
  };

  const familyForVariant = (variant) => {
    let familyID = variant.id;
    for (const suffix of ["-dark", "-light"]) {
      if (familyID.endsWith(suffix) && familyID.length > suffix.length) {
        familyID = familyID.slice(0, -suffix.length);
        break;
      }
    }
    return { id: familyID, name: variant.name, variants: [variant] };
  };

  const reservedIDs = () => {
    const familyIDs = new Set();
    const variantIDs = new Set();
    const catalog = window.HearthlightThemes;
    const families = catalog && Array.isArray(catalog.families) ? catalog.families : [];
    for (const family of families) {
      if (!family) continue;
      if (family.id) familyIDs.add(family.id);
      if (!Array.isArray(family.variants)) continue;
      for (const variant of family.variants) {
        if (variant && variant.id) variantIDs.add(variant.id);
      }
    }
    return { familyIDs, variantIDs };
  };

  const validateCatalog = (families, errors) => {
    const reserved = reservedIDs();
    const familyIDs = new Map();
    const variantIDs = new Map();
    for (let familyIndex = 0; familyIndex < families.length; familyIndex += 1) {
      const family = families[familyIndex];
      const familyPath = "$.families[" + familyIndex + "]";
      if (family.id && reserved.familyIDs.has(family.id)) {
        pushError(errors, "duplicate_id", familyPath + ".id", "Family id \"" + family.id + "\" is already used by a built-in theme.");
      } else if (family.id && familyIDs.has(family.id)) {
        pushError(errors, "duplicate_id", familyPath + ".id", "Family id \"" + family.id + "\" is already used by " + familyIDs.get(family.id) + ".");
      } else if (family.id) {
        familyIDs.set(family.id, familyPath);
      }
      const modes = new Map();
      for (let variantIndex = 0; variantIndex < family.variants.length; variantIndex += 1) {
        const variant = family.variants[variantIndex];
        const variantPath = familyPath + ".variants[" + variantIndex + "]";
        if (variant.id && reserved.variantIDs.has(variant.id)) {
          pushError(errors, "duplicate_id", variantPath + ".id", "Variant id \"" + variant.id + "\" is already used by a built-in theme.");
        } else if (variant.id && variantIDs.has(variant.id)) {
          pushError(errors, "duplicate_id", variantPath + ".id", "Variant id \"" + variant.id + "\" is already used by " + variantIDs.get(variant.id) + ".");
        } else if (variant.id) {
          variantIDs.set(variant.id, variantPath);
        }
        if (variant.mode && modes.has(variant.mode)) {
          pushError(errors, "duplicate_mode", variantPath + ".mode", "Variant mode \"" + variant.mode + "\" is already used by " + modes.get(variant.mode) + ".");
        } else if (variant.mode) {
          modes.set(variant.mode, variantPath);
        }
      }
    }
  };

  const parse = (text) => {
    const errors = [];
    const warnings = [];
    if (typeof text !== "string") {
      pushError(errors, "malformed", "$", "Theme input must be text.");
      return { ok: false, families: [], errors, warnings };
    }
    const bytes = byteLength(text);
    if (bytes > MAX_IMPORT_BYTES) {
      pushError(errors, "too_large", "$", "Import is " + Math.ceil(bytes / 1024) + " KiB, maximum is 256 KiB.");
      return { ok: false, families: [], errors, warnings };
    }
    const raw = text.replace(/^\uFEFF/, "");
    if (!raw.trim()) {
      pushError(errors, "malformed", "$", "Paste a theme or choose a file first.");
      return { ok: false, families: [], errors, warnings };
    }
    let root;
    try {
      root = JSON.parse(raw);
    } catch (error) {
      pushError(errors, "malformed", "$", "Invalid JSON: " + (error && error.message ? error.message : "could not parse input") + ".");
      return { ok: false, families: [], errors, warnings };
    }
    if (rejectForbiddenKeys(root, "$", errors)) {
      return { ok: false, families: [], errors, warnings };
    }
    if (!isPlainObject(root)) {
      pushError(errors, "invalid_shape", "$", "Top-level JSON value must be an object.");
      return { ok: false, families: [], errors, warnings };
    }
    const families = parseDocument(root, errors);
    validateCatalog(families, errors);
    return { ok: errors.length === 0 && families.length > 0, families, errors, warnings };
  };

  const markCustomFamily = (family) => Object.assign({}, family, { custom: true });

  const normalizeSourceForExport = (source) => {
    if (typeof source === "string" && source.trim()) return { name: source.trim() };
    if (!isPlainObject(source)) return null;
    const copy = {};
    if (typeof source.name === "string" && source.name.trim()) copy.name = source.name.trim();
    if (typeof source.url === "string" && source.url.trim()) copy.url = source.url.trim();
    return Object.keys(copy).length ? copy : null;
  };

  const sanitizeVariantForExport = (variant) => {
    const copy = {
      id: variant.id,
      name: variant.name,
      mode: variant.mode,
      palette: Object.assign({}, variant.palette),
      effects: Object.assign({}, variant.effects),
    };
    if (variant.kinds) {
      copy.kinds = {};
      for (const key of Object.keys(variant.kinds)) {
        copy.kinds[key] = {
          label: variant.kinds[key].label,
          color: variant.kinds[key].color,
        };
      }
    }
    if (variant.categories) copy.categories = Object.assign({}, variant.categories);
    return copy;
  };

  const sanitizeFamilyForExport = (family) => {
    const copy = {
      id: family.id,
      name: family.name,
      source: normalizeSourceForExport(family.source) || { name: family.name },
      variants: family.variants.map(sanitizeVariantForExport),
    };
    return copy;
  };

  const exportDocument = (families) =>
    JSON.stringify({
      schemaVersion: SCHEMA_VERSION,
      source: {
        name: "Hearthlight",
        url: "https://github.com/jmkelly/hearthlight",
      },
      families: families.map(sanitizeFamilyForExport),
    }, null, 2) + "\n";

  const probeStorage = () => {
    if (storageMode !== "unknown") return storageMode;
    try {
      const probe = "__hearthlight_theme_probe__";
      window.localStorage.setItem(probe, "1");
      window.localStorage.removeItem(probe);
      storageMode = "ok";
    } catch (error) {
      storageMode = isQuotaError(error) ? "quota" : "disabled";
    }
    return storageMode;
  };

  const loadStoredFamilies = () => {
    let raw = null;
    try {
      raw = window.localStorage.getItem(STORAGE_KEY);
    } catch (error) {
      storageMode = isQuotaError(error) ? "quota" : "disabled";
      return [];
    }
    probeStorage();
    if (!raw) return [];
    let root;
    try {
      root = JSON.parse(raw);
    } catch (error) {
      console.warn("Hearthlight custom themes: stored JSON is invalid.", error);
      return [];
    }
    if (!isPlainObject(root) || !Array.isArray(root.families)) {
      console.warn("Hearthlight custom themes: stored document has no families array.");
      return [];
    }
    const versionErrors = [];
    if (schemaVersionOf(root, "$", versionErrors) === null) {
      console.warn("Hearthlight custom themes: stored schema is not supported.", versionErrors);
      return [];
    }
    const reserved = reservedIDs();
    const familyIDs = new Set(reserved.familyIDs);
    const variantIDs = new Set(reserved.variantIDs);
    const families = [];
    for (let index = 0; index < root.families.length && families.length < MAX_CUSTOM_FAMILIES; index += 1) {
      const errors = [];
      const family = parseFamily(root.families[index], "$.families[" + index + "]", errors);
      if (!family || errors.length) {
        console.warn("Hearthlight custom theme skipped:", root.families[index] && root.families[index].id, errors);
        continue;
      }
      if (familyIDs.has(family.id)) {
        console.warn("Hearthlight custom theme skipped: family id is already in use.", family.id);
        continue;
      }
      let conflict = "";
      const pendingVariants = new Set();
      const pendingModes = new Set();
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
        console.warn("Hearthlight custom theme skipped:", family.id, conflict);
        continue;
      }
      familyIDs.add(family.id);
      for (const variant of family.variants) variantIDs.add(variant.id);
      families.push(markCustomFamily(family));
    }
    return families;
  };

  const persistFamilies = (families) => {
    const serialized = exportDocument(families);
    try {
      window.localStorage.setItem(STORAGE_KEY, serialized);
      storageMode = "ok";
      return { ok: true };
    } catch (error) {
      const quota = isQuotaError(error);
      storageMode = quota ? "quota" : "disabled";
      return {
        ok: false,
        quota,
        disabled: !quota,
        message: quota
          ? "Browser storage is full. The previous theme was kept."
          : "Browser storage is disabled. The import lasts for this session.",
      };
    }
  };

  const validateNameValue = (value) => {
    const errors = [];
    const name = validateName(value, "$.name", errors);
    return errors.length ? { ok: false, message: errors[0].message } : { ok: true, name };
  };

  const uniqueID = (base, used) => {
    let candidate = base.slice(0, MAX_ID_LENGTH);
    let index = 2;
    while (used.has(candidate)) {
      const suffix = "-" + index;
      index += 1;
      candidate = base.slice(0, MAX_ID_LENGTH - suffix.length) + suffix;
    }
    used.add(candidate);
    return candidate;
  };

  const duplicateName = (name) => {
    const suffix = " copy";
    return Array.from(name).slice(0, MAX_NAME_LENGTH - suffix.length).join("").trimEnd() + suffix;
  };

  const allCustomIDs = (families) => {
    const familyIDs = new Set(reservedIDs().familyIDs);
    const variantIDs = new Set(reservedIDs().variantIDs);
    for (const family of families) {
      if (!family) continue;
      if (family.id) familyIDs.add(family.id);
      if (!Array.isArray(family.variants)) continue;
      for (const variant of family.variants) {
        if (variant && variant.id) variantIDs.add(variant.id);
      }
    }
    return { familyIDs, variantIDs };
  };

  const renameFamily = (families, id, name) => {
    const validated = validateNameValue(name);
    if (!validated.ok) return { ok: false, message: validated.message };
    const family = families.find((candidate) => candidate && candidate.id === id);
    if (!family) return { ok: false, message: "Theme not found." };
    const next = families.map((candidate) =>
      (candidate && candidate.id === id ? Object.assign({}, candidate, { name: validated.name }) : candidate));
    return { ok: true, families: next, family: next.find((candidate) => candidate && candidate.id === id) };
  };

  const duplicateFamily = (families, id) => {
    const source = families.find((candidate) => candidate && candidate.id === id);
    if (!source) return { ok: false, message: "Theme not found." };
    const used = allCustomIDs(families);
    const familyID = uniqueID(source.id + "-copy", used.familyIDs);
    const duplicate = {
      id: familyID,
      name: duplicateName(source.name),
      custom: true,
      variants: source.variants.map((variant) => {
        const copy = {
          id: uniqueID(variant.id + "-copy", used.variantIDs),
          name: variant.name,
          mode: variant.mode,
          palette: Object.assign({}, variant.palette),
          effects: Object.assign({}, variant.effects),
        };
        if (variant.kinds) {
          copy.kinds = {};
          for (const key of Object.keys(variant.kinds)) {
            copy.kinds[key] = { label: variant.kinds[key].label, color: variant.kinds[key].color };
          }
        }
        if (variant.categories) copy.categories = Object.assign({}, variant.categories);
        return copy;
      }),
    };
    const sourceCopy = normalizeSourceForExport(source.source);
    if (sourceCopy) duplicate.source = sourceCopy;
    return { ok: true, families: families.concat([duplicate]), family: duplicate };
  };

  const storageState = () => probeStorage();

  window.HearthlightThemeImport = Object.freeze({
    STORAGE_KEY,
    MAX_IMPORT_BYTES,
    MAX_CUSTOM_FAMILIES,
    SCHEMA_VERSION,
    REQUIRED_PALETTE_TOKENS,
    parse,
    loadStoredFamilies,
    persist: persistFamilies,
    exportDocument,
    storageState,
    renameFamily,
    duplicateFamily,
    markCustomFamily,
  });

  const storedFamilies = loadStoredFamilies();
  if (storedFamilies.length && window.HearthlightTheme &&
      typeof window.HearthlightTheme.registerFamilies === "function") {
    window.HearthlightTheme.registerFamilies(storedFamilies);
  }

  window.addEventListener("storage", (event) => {
    if (event.key !== STORAGE_KEY && event.key !== null) return;
    const families = loadStoredFamilies();
    if (window.HearthlightTheme && typeof window.HearthlightTheme.registerFamilies === "function") {
      window.HearthlightTheme.registerFamilies(families);
    }
    if (typeof window.CustomEvent === "function") {
      window.dispatchEvent(new CustomEvent("hearthlight:custom-themes", { detail: { families } }));
    }
  });
})();
