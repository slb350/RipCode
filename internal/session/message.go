package session

import (
	"fmt"
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

// Valid checks that the record has required fields and a valid message.
func (r MessageRecord) Valid() error {
	if r.ID == "" {
		return fmt.Errorf("message record missing ID")
	}
	if r.CreatedAt.IsZero() {
		return fmt.Errorf("message record %s missing CreatedAt", r.ID)
	}
	if err := r.Message.Valid(); err != nil {
		return fmt.Errorf("message record %s: %w", r.ID, err)
	}
	return nil
}

func generateMessageID() string {
	return generatePrefixedID("msg", 6)
}
