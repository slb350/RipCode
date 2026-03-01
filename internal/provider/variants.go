package provider

// KnownVariants maps model IDs to their available thinking budget variants.
var KnownVariants = map[string][]string{
	"anthropic/claude-sonnet-4-thinking": {"low", "medium", "high"},
	"anthropic/claude-opus-4-thinking":   {"low", "medium", "high"},
}

// VariantsFor returns the available variants for a model, or nil if unknown.
func VariantsFor(modelID string) []string {
	return KnownVariants[modelID]
}

// CycleVariant returns the next variant for a model, cycling through the list.
// If current is empty, returns the first variant.
// If current is the last variant, returns empty string (no variant).
// If the model has no variants, returns empty string.
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
