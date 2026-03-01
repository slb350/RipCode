package provider

// KnownVariants maps model IDs to their available thinking budget variants.
// NOTE: hardcoded; update when new thinking-budget models are added.
// TODO: parse from API metadata when OpenRouter exposes variant info.
var KnownVariants = map[string][]string{
	"anthropic/claude-sonnet-4-thinking": {"low", "medium", "high"},
	"anthropic/claude-opus-4-thinking":   {"low", "medium", "high"},
}

// VariantsFor returns the available variants for a model, or nil if unknown.
func VariantsFor(modelID string) []string {
	return KnownVariants[modelID]
}

// VariantBadge returns the status bar badge for a variant, or "" if empty.
func VariantBadge(variant string) string {
	if variant == "" {
		return ""
	}
	return "[thinking:" + variant + "]"
}

// CycleVariant cycles through variants: none -> first -> ... -> last -> none.
// Returns empty string to disable the variant, then first variant to re-enable.
// If the model has no variants, always returns empty string.
// If the current variant is unknown, resets to the first variant.
func CycleVariant(modelID, current string) string {
	variants := VariantsFor(modelID)
	if len(variants) == 0 {
		return ""
	}
	if current == "" {
		return variants[0]
	}
	for i, v := range variants {
		if v == current {
			if i == len(variants)-1 {
				return "" // cycle past last -> none
			}
			return variants[i+1]
		}
	}
	return variants[0] // unknown current -> start over
}
