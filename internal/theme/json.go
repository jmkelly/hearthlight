package theme

import (
	"bytes"
	"encoding/json"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// ParseJSON parses a Hearthlight theme import. Accepted documents are a single
// variant, a family with one or two variants, or a full web/themes.json-shaped
// catalog. Variants may be a list or an object keyed by dark and light.
func ParseJSON(data []byte) (Catalog, error) {
	raw, sizeErr := boundedInput(data)
	if sizeErr != nil {
		return Catalog{}, sizeErr
	}
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})

	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return Catalog{}, fail(ErrMalformed, "", "invalid JSON: %v", err)
	}
	if err := rejectForbiddenKeys(root, "$"); err != nil {
		return Catalog{}, err
	}
	object, ok := root.(map[string]any)
	if !ok {
		return Catalog{}, fail(ErrInvalidShape, "", "top-level JSON value must be an object")
	}
	catalog, validationErr := parseJSONDocument(object)
	if validationErr != nil {
		return Catalog{}, validationErr
	}
	if validationErr := validateCatalog(catalog); validationErr != nil {
		return Catalog{}, validationErr
	}
	return catalog, nil
}

func boundedInput(data []byte) ([]byte, *ValidationError) {
	if len(data) > MaxImportBytes {
		return nil, fail(ErrTooLarge, "", "import is %d bytes, maximum is %d", len(data), MaxImportBytes)
	}
	return data, nil
}

func rejectForbiddenKeys(value any, path string) *ValidationError {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			childPath := path + "." + key
			if forbiddenKeys[key] {
				return fail(ErrForbiddenKey, childPath, "key %q is not allowed", key)
			}
			if err := rejectForbiddenKeys(typed[key], childPath); err != nil {
				return err
			}
		}
	case []any:
		for index, child := range typed {
			if err := rejectForbiddenKeys(child, path+"["+strconv.Itoa(index)+"]"); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseJSONDocument(object map[string]any) (Catalog, *ValidationError) {
	if _, err := schemaVersionOf(object, "$"); err != nil {
		return Catalog{}, err
	}

	hasFamilies := hasKey(object, "families")
	hasFamily := hasKey(object, "family")
	hasVariant := hasKey(object, "variant")
	hasVariants := hasKey(object, "variants")
	hasPalette := hasKey(object, "palette") || hasKey(object, "colors")

	switch {
	case hasFamilies:
		if hasFamily || hasVariant || hasVariants || hasPalette {
			return Catalog{}, fail(ErrInvalidShape, "$", "catalog document must not mix families with family, variant, variants, or palette")
		}
		families, err := parseCatalogFamilies(object["families"], "$.families")
		if err != nil {
			return Catalog{}, err
		}
		return Catalog{SchemaVersion: SchemaVersion, Families: families}, nil
	case hasFamily:
		if hasVariant || hasVariants || hasPalette {
			return Catalog{}, fail(ErrInvalidShape, "$", "family document must not mix family with variant, variants, or palette")
		}
		rawFamily, ok := object["family"].(map[string]any)
		if !ok {
			return Catalog{}, fail(ErrInvalidShape, "$.family", "family must be an object")
		}
		family, err := parseFamily(rawFamily, "$.family")
		if err != nil {
			return Catalog{}, err
		}
		return Catalog{SchemaVersion: SchemaVersion, Families: []Family{family}}, nil
	case hasVariant:
		if hasVariants || hasPalette {
			return Catalog{}, fail(ErrInvalidShape, "$", "variant document must not mix variant with variants or palette")
		}
		rawVariant, ok := object["variant"].(map[string]any)
		if !ok {
			return Catalog{}, fail(ErrInvalidShape, "$.variant", "variant must be an object")
		}
		variant, err := parseVariant(rawVariant, "$.variant", "")
		if err != nil {
			return Catalog{}, err
		}
		return catalogForVariant(variant, "$.variant")
	case hasVariants:
		if hasPalette {
			return Catalog{}, fail(ErrInvalidShape, "$", "family document must not mix variants with palette")
		}
		family, err := parseFamily(object, "$")
		if err != nil {
			return Catalog{}, err
		}
		return Catalog{SchemaVersion: SchemaVersion, Families: []Family{family}}, nil
	case hasPalette:
		variant, err := parseVariant(object, "$", "")
		if err != nil {
			return Catalog{}, err
		}
		return catalogForVariant(variant, "$")
	default:
		return Catalog{}, fail(ErrInvalidShape, "$", "document must contain families, family, variant, variants, or a palette")
	}
}

func hasKey(object map[string]any, key string) bool {
	value, ok := object[key]
	return ok && value != nil
}

func schemaVersionOf(object map[string]any, path string) (int, *ValidationError) {
	value, ok := object["schemaVersion"]
	if !ok || value == nil {
		return SchemaVersion, nil
	}
	number, ok := value.(float64)
	if !ok || math.IsNaN(number) || math.IsInf(number, 0) || number != math.Trunc(number) {
		return 0, fail(ErrMalformed, path+".schemaVersion", "schemaVersion must be an integer")
	}
	if int(number) != SchemaVersion {
		return 0, fail(ErrUnsupportedSchema, path+".schemaVersion", "schemaVersion %d is not supported, expected %d", int(number), SchemaVersion)
	}
	return SchemaVersion, nil
}

func parseCatalogFamilies(value any, path string) ([]Family, *ValidationError) {
	list, ok := value.([]any)
	if !ok {
		return nil, fail(ErrInvalidShape, path, "families must be an array")
	}
	if len(list) == 0 {
		return nil, fail(ErrInvalidShape, path, "families must contain at least one family")
	}
	families := make([]Family, 0, len(list))
	for index, item := range list {
		rawFamily, ok := item.(map[string]any)
		if !ok {
			return nil, fail(ErrInvalidShape, path+"["+strconv.Itoa(index)+"]", "family must be an object")
		}
		family, err := parseFamily(rawFamily, path+"["+strconv.Itoa(index)+"]")
		if err != nil {
			return nil, err
		}
		families = append(families, family)
	}
	return families, nil
}

func parseFamily(object map[string]any, path string) (Family, *ValidationError) {
	id, err := requiredString(object, "id", path, ErrInvalidID)
	if err != nil {
		return Family{}, err
	}
	if err := validateID(id, path+".id"); err != nil {
		return Family{}, err
	}
	name, err := requiredString(object, "name", path, ErrInvalidName)
	if err != nil {
		return Family{}, err
	}
	if err := validateName(name, path+".name"); err != nil {
		return Family{}, err
	}
	source, err := parseSource(object["source"], path+".source")
	if err != nil {
		return Family{}, err
	}
	variants, err := parseFamilyVariants(object["variants"], path+".variants")
	if err != nil {
		return Family{}, err
	}
	return Family{
		ID:       strings.TrimSpace(id),
		Name:     strings.TrimSpace(name),
		Source:   source,
		Variants: variants,
	}, nil
}

func parseFamilyVariants(value any, path string) ([]Variant, *ValidationError) {
	switch typed := value.(type) {
	case []any:
		if len(typed) == 0 {
			return nil, fail(ErrInvalidShape, path, "variants must contain at least one variant")
		}
		variants := make([]Variant, 0, len(typed))
		for index, item := range typed {
			rawVariant, ok := item.(map[string]any)
			if !ok {
				return nil, fail(ErrInvalidShape, path+"["+strconv.Itoa(index)+"]", "variant must be an object")
			}
			variant, err := parseVariant(rawVariant, path+"["+strconv.Itoa(index)+"]", "")
			if err != nil {
				return nil, err
			}
			variants = append(variants, variant)
		}
		return variants, nil
	case map[string]any:
		if len(typed) == 0 {
			return nil, fail(ErrInvalidShape, path, "variants must contain at least one variant")
		}
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		variants := make([]Variant, 0, len(typed))
		for _, key := range keys {
			if key != "dark" && key != "light" {
				return nil, fail(ErrInvalidMode, path+"."+key, "variant object keys must be dark or light")
			}
			rawVariant, ok := typed[key].(map[string]any)
			if !ok {
				return nil, fail(ErrInvalidShape, path+"."+key, "variant must be an object")
			}
			variant, err := parseVariant(rawVariant, path+"."+key, key)
			if err != nil {
				return nil, err
			}
			variants = append(variants, variant)
		}
		return variants, nil
	default:
		return nil, fail(ErrInvalidShape, path, "variants must be an array or an object keyed by dark and light")
	}
}

func parseVariant(object map[string]any, path, impliedMode string) (Variant, *ValidationError) {
	id, err := requiredString(object, "id", path, ErrInvalidID)
	if err != nil {
		return Variant{}, err
	}
	if err := validateID(id, path+".id"); err != nil {
		return Variant{}, err
	}
	name, err := requiredString(object, "name", path, ErrInvalidName)
	if err != nil {
		return Variant{}, err
	}
	if err := validateName(name, path+".name"); err != nil {
		return Variant{}, err
	}

	mode := impliedMode
	if value, ok := object["mode"]; ok && value != nil {
		text, ok := value.(string)
		if !ok {
			return Variant{}, fail(ErrInvalidMode, path+".mode", "mode must be a string")
		}
		if text != "dark" && text != "light" {
			return Variant{}, fail(ErrInvalidMode, path+".mode", "mode %q must be dark or light", text)
		}
		if impliedMode != "" && text != impliedMode {
			return Variant{}, fail(ErrInvalidMode, path+".mode", "mode %q conflicts with the %q variant key", text, impliedMode)
		}
		mode = text
	}
	if mode == "" {
		return Variant{}, fail(ErrInvalidMode, path+".mode", "mode must be dark or light")
	}

	palette, err := parsePaletteField(object, path)
	if err != nil {
		return Variant{}, err
	}
	effects, err := parseEffects(object["effects"], path+".effects", mode)
	if err != nil {
		return Variant{}, err
	}
	kinds, err := parseKinds(object["kinds"], path+".kinds")
	if err != nil {
		return Variant{}, err
	}
	categories, err := parseCategories(object["categories"], path+".categories")
	if err != nil {
		return Variant{}, err
	}
	return Variant{
		ID:         strings.TrimSpace(id),
		Name:       strings.TrimSpace(name),
		Mode:       mode,
		Palette:    palette,
		Kinds:      kinds,
		Categories: categories,
		Effects:    effects,
	}, nil
}

func requiredString(object map[string]any, key, path string, category ErrorCategory) (string, *ValidationError) {
	value, ok := object[key]
	if !ok || value == nil {
		return "", fail(category, path+"."+key, "%s is required", key)
	}
	text, ok := value.(string)
	if !ok {
		return "", fail(category, path+"."+key, "%s must be a string", key)
	}
	return text, nil
}

func parsePaletteField(object map[string]any, path string) (map[string]string, *ValidationError) {
	paletteValue, hasPalette := object["palette"]
	colorsValue, hasColors := object["colors"]
	if hasPalette && hasColors && paletteValue != nil && colorsValue != nil {
		return nil, fail(ErrInvalidShape, path, "use either palette or colors, not both")
	}
	value := paletteValue
	field := "palette"
	if value == nil && colorsValue != nil {
		value = colorsValue
		field = "colors"
	}
	if value == nil {
		return nil, fail(ErrMissingToken, path+".palette", "palette or colors is required")
	}
	rawPalette, ok := value.(map[string]any)
	if !ok {
		return nil, fail(ErrInvalidShape, path+"."+field, "%s must be an object", field)
	}
	palette := make(map[string]string, len(rawPalette))
	for _, key := range sortedKeys(rawPalette) {
		keyPath := path + "." + field + "." + key
		if err := validateMapKey(key, keyPath, paletteKeyPattern); err != nil {
			return nil, err
		}
		text, ok := rawPalette[key].(string)
		if !ok {
			return nil, fail(ErrInvalidColor, keyPath, "colour must be a string")
		}
		if !validColor(text) {
			return nil, fail(ErrInvalidColor, keyPath, "%q is not a valid hex or rgb/rgba colour", text)
		}
		palette[key] = strings.TrimSpace(text)
	}
	for _, token := range RequiredPaletteTokens {
		if _, ok := palette[token]; !ok {
			return nil, fail(ErrMissingToken, path+"."+field+"."+token, "required palette token %q is missing", token)
		}
	}
	return palette, nil
}

func parseEffects(value any, path, mode string) (Effects, *ValidationError) {
	effects := defaultEffects(mode)
	if value == nil {
		return effects, nil
	}
	rawEffects, ok := value.(map[string]any)
	if !ok {
		return Effects{}, fail(ErrInvalidEffect, path, "effects must be an object")
	}
	for _, key := range sortedKeys(rawEffects) {
		keyPath := path + "." + key
		item := rawEffects[key]
		if key == "territoryBlend" {
			text, ok := item.(string)
			if !ok || !blendModes[text] {
				return Effects{}, fail(ErrInvalidEffect, keyPath, "territoryBlend must be a supported blend mode")
			}
			effects.TerritoryBlend = text
			continue
		}
		limits, known := numericEffectRanges[key]
		if !known {
			return Effects{}, fail(ErrInvalidEffect, keyPath, "unknown effect %q", key)
		}
		number, ok := item.(float64)
		if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
			return Effects{}, fail(ErrInvalidEffect, keyPath, "effect must be a finite number")
		}
		if number < limits.min || number > limits.max {
			return Effects{}, fail(ErrInvalidEffect, keyPath, "effect %v is outside the allowed range %v to %v", number, limits.min, limits.max)
		}
		setNumericEffect(&effects, key, number)
	}
	return effects, nil
}

func setNumericEffect(effects *Effects, key string, value float64) {
	switch key {
	case "vignette":
		effects.Vignette = value
	case "deviceGlow":
		effects.DeviceGlow = value
	case "devicePulse":
		effects.DevicePulse = value
	case "serviceGlow":
		effects.ServiceGlow = value
	case "territoryBlur":
		effects.TerritoryBlur = value
	case "ribbonBlur":
		effects.RibbonBlur = value
	case "lensBlur":
		effects.LensBlur = value
	case "glassBlur":
		effects.GlassBlur = value
	case "panelBlur":
		effects.PanelBlur = value
	case "territoryAlpha":
		effects.TerritoryAlpha = value
	case "glowStrength":
		effects.GlowStrength = value
	case "labelHalo":
		effects.LabelHalo = value
	case "shadowStrength":
		effects.ShadowStrength = value
	}
}

func parseKinds(value any, path string) (map[string]Kind, *ValidationError) {
	if value == nil {
		return nil, nil
	}
	rawKinds, ok := value.(map[string]any)
	if !ok {
		return nil, fail(ErrInvalidShape, path, "kinds must be an object")
	}
	kinds := make(map[string]Kind, len(rawKinds))
	for _, key := range sortedKeys(rawKinds) {
		keyPath := path + "." + key
		if err := validateMapKey(key, keyPath, mapKeyPattern); err != nil {
			return nil, err
		}
		object, ok := rawKinds[key].(map[string]any)
		if !ok {
			return nil, fail(ErrInvalidShape, keyPath, "kind must be an object")
		}
		label, err := requiredString(object, "label", keyPath, ErrInvalidName)
		if err != nil {
			return nil, err
		}
		if err := validateName(label, keyPath+".label"); err != nil {
			return nil, err
		}
		color, err := requiredString(object, "color", keyPath, ErrInvalidColor)
		if err != nil {
			return nil, err
		}
		if !validColor(color) {
			return nil, fail(ErrInvalidColor, keyPath+".color", "%q is not a valid hex or rgb/rgba colour", color)
		}
		kinds[key] = Kind{Label: strings.TrimSpace(label), Color: strings.TrimSpace(color)}
	}
	return kinds, nil
}

func parseCategories(value any, path string) (map[string]string, *ValidationError) {
	if value == nil {
		return nil, nil
	}
	rawCategories, ok := value.(map[string]any)
	if !ok {
		return nil, fail(ErrInvalidShape, path, "categories must be an object")
	}
	categories := make(map[string]string, len(rawCategories))
	for _, key := range sortedKeys(rawCategories) {
		keyPath := path + "." + key
		if err := validateMapKey(key, keyPath, mapKeyPattern); err != nil {
			return nil, err
		}
		color, ok := rawCategories[key].(string)
		if !ok {
			return nil, fail(ErrInvalidColor, keyPath, "colour must be a string")
		}
		if !validColor(color) {
			return nil, fail(ErrInvalidColor, keyPath, "%q is not a valid hex or rgb/rgba colour", color)
		}
		categories[key] = strings.TrimSpace(color)
	}
	return categories, nil
}

func parseSource(value any, path string) (*Source, *ValidationError) {
	if value == nil {
		return nil, nil
	}
	if text, ok := value.(string); ok {
		if err := validateName(text, path); err != nil {
			return nil, err
		}
		return &Source{Name: strings.TrimSpace(text)}, nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fail(ErrInvalidShape, path, "source must be a string or an object")
	}
	source := &Source{}
	if rawName, ok := object["name"]; ok && rawName != nil {
		name, ok := rawName.(string)
		if !ok {
			return nil, fail(ErrInvalidName, path+".name", "name must be a string")
		}
		if err := validateName(name, path+".name"); err != nil {
			return nil, err
		}
		source.Name = strings.TrimSpace(name)
	}
	if rawURL, ok := object["url"]; ok && rawURL != nil {
		urlText, ok := rawURL.(string)
		if !ok || len(urlText) > 512 {
			return nil, fail(ErrInvalidShape, path+".url", "url must be a string no longer than 512 bytes")
		}
		urlText = strings.TrimSpace(urlText)
		if urlText != "" {
			parsed, err := url.Parse(urlText)
			if err != nil {
				return nil, fail(ErrInvalidShape, path+".url", "url must be an http or https URL")
			}
			scheme := strings.ToLower(parsed.Scheme)
			if (scheme != "http" && scheme != "https") || parsed.Host == "" {
				return nil, fail(ErrInvalidShape, path+".url", "url must be an http or https URL")
			}
		}
		source.URL = urlText
	}
	if source.Name == "" && source.URL == "" {
		return nil, nil
	}
	return source, nil
}

func catalogForVariant(variant Variant, path string) (Catalog, *ValidationError) {
	familyID := variant.ID
	for _, suffix := range []string{"-dark", "-light"} {
		if strings.HasSuffix(familyID, suffix) && len(familyID) > len(suffix) {
			familyID = strings.TrimSuffix(familyID, suffix)
			break
		}
	}
	if err := validateID(familyID, path+".id"); err != nil {
		return Catalog{}, err
	}
	return Catalog{
		SchemaVersion: SchemaVersion,
		Families: []Family{{
			ID:       familyID,
			Name:     variant.Name,
			Variants: []Variant{variant},
		}},
	}, nil
}

func validateCatalog(catalog Catalog) *ValidationError {
	familyIDs := make(map[string]string, len(catalog.Families))
	variantIDs := make(map[string]string)
	for familyIndex, family := range catalog.Families {
		familyPath := "families[" + strconv.Itoa(familyIndex) + "]"
		if previous, ok := familyIDs[family.ID]; ok {
			return fail(ErrDuplicateID, familyPath+".id", "family id %q is already used by %s", family.ID, previous)
		}
		familyIDs[family.ID] = familyPath
		if len(family.Variants) == 0 {
			return fail(ErrInvalidShape, familyPath+".variants", "family must contain at least one variant")
		}
		modes := make(map[string]string, len(family.Variants))
		for variantIndex, variant := range family.Variants {
			variantPath := familyPath + ".variants[" + strconv.Itoa(variantIndex) + "]"
			if previous, ok := variantIDs[variant.ID]; ok {
				return fail(ErrDuplicateID, variantPath+".id", "variant id %q is already used by %s", variant.ID, previous)
			}
			variantIDs[variant.ID] = variantPath
			if previous, ok := modes[variant.Mode]; ok {
				return fail(ErrDuplicateMode, variantPath+".mode", "variant mode %q is already used by %s", variant.Mode, previous)
			}
			modes[variant.Mode] = variantPath
		}
	}
	return nil
}
