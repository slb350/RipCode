package session

import (
	"time"

	"github.com/stephenbrandon/ripcode/internal/provider"
)

// MessageRecord wraps a provider.Message with session-level metadata.
type MessageRecord struct {
	ID        string
	Message   provider.Message
	CreatedAt time.Time
	Meta      *AssistantMeta // nil for non-assistant messages
}

// AssistantMeta holds metadata about an assistant response.
type AssistantMeta struct {
	Model        string
	Agent        string
	InputTokens  int
	OutputTokens int
	FinishReason string
	Duration     time.Duration
}

func generateMessageID() string {
	return generatePrefixedID("msg", 6)
}
