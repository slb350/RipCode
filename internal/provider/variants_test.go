package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelVariants_ReturnsVariantsForKnownModel(t *testing.T) {
	variants := VariantsFor("anthropic/claude-sonnet-4-thinking")
	assert.Equal(t, []string{"low", "medium", "high"}, variants)
}

func TestModelVariants_ReturnsNilForUnknownModel(t *testing.T) {
	variants := VariantsFor("openai/gpt-4o")
	assert.Nil(t, variants)
}

func TestCycleVariant_AdvancesForward(t *testing.T) {
	next := CycleVariant("anthropic/claude-sonnet-4-thinking", "low")
	assert.Equal(t, "medium", next)
}

func TestCycleVariant_WrapsAround(t *testing.T) {
	// medium -> high
	next := CycleVariant("anthropic/claude-sonnet-4-thinking", "medium")
	assert.Equal(t, "high", next)
}

func TestCycleVariant_EmptyToFirst(t *testing.T) {
	next := CycleVariant("anthropic/claude-sonnet-4-thinking", "")
	assert.Equal(t, "low", next)
}

func TestCycleVariant_LastToEmpty(t *testing.T) {
	next := CycleVariant("anthropic/claude-sonnet-4-thinking", "high")
	assert.Equal(t, "", next, "cycling past last should return to none")
}
