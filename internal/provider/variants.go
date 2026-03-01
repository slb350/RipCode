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

// CycleVariant advances to the next variant: none → first → ... → last → none.
// Does not wrap; returns empty string after the last variant or if the model
// has no variants.
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
