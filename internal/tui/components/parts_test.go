package components

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPartType_Constants(t *testing.T) {
	assert.Equal(t, PartType("text"), PartText)
	assert.Equal(t, PartType("reasoning"), PartReasoning)
}

func TestMessagePart_Creation(t *testing.T) {
	p := MessagePart{Type: PartText, Content: "hello"}
	assert.Equal(t, PartText, p.Type)
	assert.Equal(t, "hello", p.Content)
}

func TestMessagePart_ReasoningCreation(t *testing.T) {
	p := MessagePart{Type: PartReasoning, Content: "thinking..."}
	assert.Equal(t, PartReasoning, p.Type)
	assert.Equal(t, "thinking...", p.Content)
}

// --- PartType.Valid tests ---

func TestPartType_Valid_Text(t *testing.T) {
	assert.True(t, PartText.Valid())
}

func TestPartType_Valid_Reasoning(t *testing.T) {
	assert.True(t, PartReasoning.Valid())
}

func TestPartType_Valid_Unknown(t *testing.T) {
	assert.False(t, PartType("unknown").Valid())
}

func TestPartType_Valid_Empty(t *testing.T) {
	assert.False(t, PartType("").Valid())
}

// --- MessagePart.Valid tests ---

func TestMessagePart_Valid_TextOK(t *testing.T) {
	p := MessagePart{Type: PartText, Content: "hello"}
	assert.NoError(t, p.Valid())
}

func TestMessagePart_Valid_ReasoningOK(t *testing.T) {
	p := MessagePart{Type: PartReasoning, Content: "think"}
	assert.NoError(t, p.Valid())
}

func TestMessagePart_Valid_InvalidType(t *testing.T) {
	p := MessagePart{Type: "bogus", Content: "x"}
	err := p.Valid()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid part type")
}

func TestMessagePart_Valid_EmptyType(t *testing.T) {
	p := MessagePart{Type: "", Content: "x"}
	assert.Error(t, p.Valid())
}

// --- StreamPart validation tests ---

func TestChat_StreamPart_InvalidType_StillAccumulates(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	c := NewChat()
	c.SetSize(80, 20)

	c.StreamPart(PartType("bogus"), "content")

	// Should still accumulate — logging is best-effort, not blocking
	assert.Len(t, c.streamingParts, 1)
	assert.Equal(t, PartType("bogus"), c.streamingParts[0].Type)
	assert.Equal(t, "content", c.streamingParts[0].Content)
}

// --- Unknown PartType rendering ---

func TestChat_RenderAssistantParts_UnknownType_RenderedAsText(t *testing.T) {
	t.Setenv("RIPCODE_DIR", t.TempDir())
	c := NewChat()
	c.SetSize(80, 20)

	c.AddEntry(ChatEntry{
		Role: RoleAssistant,
		Parts: []MessagePart{
			{Type: PartType("future_type"), Content: "future content"},
		},
	})

	view := c.View()
	assert.Contains(t, view, "future content", "unknown type should render as text")
}

// --- StreamPart tests ---

func TestChat_StreamPart_TextAccumulates(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.StreamPart(PartText, "Hello ")
	c.StreamPart(PartText, "world")

	assert.Len(t, c.streamingParts, 1)
	assert.Equal(t, PartText, c.streamingParts[0].Type)
	assert.Equal(t, "Hello world", c.streamingParts[0].Content)
}

func TestChat_StreamPart_DifferentTypeStartsNew(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.StreamPart(PartText, "hello")
	c.StreamPart(PartReasoning, "thinking")

	assert.Len(t, c.streamingParts, 2)
	assert.Equal(t, PartText, c.streamingParts[0].Type)
	assert.Equal(t, "hello", c.streamingParts[0].Content)
	assert.Equal(t, PartReasoning, c.streamingParts[1].Type)
	assert.Equal(t, "thinking", c.streamingParts[1].Content)
}

func TestChat_StreamPart_InterleavedProducesThreeParts(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.StreamPart(PartText, "before ")
	c.StreamPart(PartReasoning, "think ")
	c.StreamPart(PartText, "after")

	assert.Len(t, c.streamingParts, 3)
	assert.Equal(t, PartText, c.streamingParts[0].Type)
	assert.Equal(t, "before ", c.streamingParts[0].Content)
	assert.Equal(t, PartReasoning, c.streamingParts[1].Type)
	assert.Equal(t, "think ", c.streamingParts[1].Content)
	assert.Equal(t, PartText, c.streamingParts[2].Type)
	assert.Equal(t, "after", c.streamingParts[2].Content)
}

func TestChat_StreamPart_EmptyDeltaNoOp(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.StreamPart(PartText, "")
	assert.Empty(t, c.streamingParts, "empty delta should not add parts")
}

func TestChat_StreamPart_EmptyDeltaAfterExistingNoOp(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.StreamPart(PartText, "hello")
	c.StreamPart(PartText, "")

	assert.Len(t, c.streamingParts, 1)
	assert.Equal(t, "hello", c.streamingParts[0].Content)
}

// --- CommitStream with parts ---

func TestChat_CommitStream_WithParts_CreatesPartsEntry(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.StreamPart(PartText, "hello ")
	c.StreamPart(PartReasoning, "thinking")
	c.CommitStream()

	entries := c.Entries()
	assert.Len(t, entries, 1)
	assert.Equal(t, RoleAssistant, entries[0].Role)
	assert.Len(t, entries[0].Parts, 2)
	assert.Equal(t, PartText, entries[0].Parts[0].Type)
	assert.Equal(t, "hello ", entries[0].Parts[0].Content)
	assert.Equal(t, PartReasoning, entries[0].Parts[1].Type)
	assert.Equal(t, "thinking", entries[0].Parts[1].Content)
	assert.Equal(t, "hello ", entries[0].Content, "content should preserve text parts for copy/export compatibility")
}

func TestChat_CommitStream_WithInterleavedParts_PopulatesContentFromTextParts(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.StreamPart(PartText, "alpha ")
	c.StreamPart(PartReasoning, "internal")
	c.StreamPart(PartText, "omega")
	c.CommitStream()

	entries := c.Entries()
	assert.Len(t, entries, 1)
	assert.Equal(t, RoleAssistant, entries[0].Role)
	assert.Equal(t, "alpha omega", entries[0].Content)
	assert.Len(t, entries[0].Parts, 3)
}

func TestChat_CommitStream_SingleTextPart_FallsBackToContent(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.StreamPart(PartText, "simple response")
	c.CommitStream()

	entries := c.Entries()
	assert.Len(t, entries, 1)
	assert.Equal(t, RoleAssistant, entries[0].Role)
	assert.Equal(t, "simple response", entries[0].Content)
	assert.Nil(t, entries[0].Parts, "single text part should use Content field")
}

func TestChat_CommitStream_Empty_NoEntry(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.CommitStream()
	assert.Empty(t, c.Entries(), "empty commit should not add entries")
}

func TestChat_CommitStream_ClearsStreamingParts(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	c.StreamPart(PartText, "hello")
	c.CommitStream()

	assert.Empty(t, c.streamingParts)
}

func TestChat_CommitStream_SetsCreatedAt(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	before := time.Now()
	c.StreamPart(PartText, "hello")
	c.CommitStream()

	entries := c.Entries()
	assert.Len(t, entries, 1)
	assert.False(t, entries[0].CreatedAt.IsZero(), "committed stream entries should have CreatedAt")
	assert.False(t, entries[0].CreatedAt.Before(before), "CreatedAt should be set at commit time")
}

func TestChat_CommitStream_LegacyStreamContent_StillWorks(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	// Use legacy StreamContent path
	c.StreamContent("legacy content")
	c.CommitStream()

	entries := c.Entries()
	assert.Len(t, entries, 1)
	assert.Equal(t, "legacy content", entries[0].Content)
}

// --- CopyableContent tests ---

// --- FullContent tests ---

func TestChatEntry_FullContent_NoParts_ReturnsContent(t *testing.T) {
	e := ChatEntry{Role: RoleAssistant, Content: "plain text"}
	assert.Equal(t, "plain text", e.FullContent())
}

func TestChatEntry_FullContent_WithParts_IncludesReasoning(t *testing.T) {
	e := ChatEntry{
		Role:    RoleAssistant,
		Content: "visible only",
		Parts: []MessagePart{
			{Type: PartReasoning, Content: "thinking deeply"},
			{Type: PartText, Content: "visible only"},
		},
	}
	assert.Equal(t, "thinking deeplyvisible only", e.FullContent())
}

func TestChatEntry_FullContent_Empty(t *testing.T) {
	e := ChatEntry{Role: RoleAssistant}
	assert.Equal(t, "", e.FullContent())
}

// --- CopyableContent tests ---

func TestChatEntry_CopyableContent_UsesContent(t *testing.T) {
	e := ChatEntry{Role: RoleAssistant, Content: "hello world"}
	assert.Equal(t, "hello world", e.CopyableContent())
}

func TestChatEntry_CopyableContent_FallsBackToParts(t *testing.T) {
	e := ChatEntry{
		Role: RoleAssistant,
		Parts: []MessagePart{
			{Type: PartReasoning, Content: "thinking deeply"},
		},
	}
	assert.Equal(t, "thinking deeply", e.CopyableContent())
}

func TestChatEntry_CopyableContent_MixedParts(t *testing.T) {
	e := ChatEntry{
		Role:    RoleAssistant,
		Content: "visible text",
		Parts: []MessagePart{
			{Type: PartReasoning, Content: "thinking"},
			{Type: PartText, Content: "visible text"},
		},
	}
	assert.Equal(t, "visible text", e.CopyableContent())
}

func TestChatEntry_CopyableContent_Empty(t *testing.T) {
	e := ChatEntry{Role: RoleAssistant}
	assert.Equal(t, "", e.CopyableContent())
}

// --- ChatEntry.Valid tests ---

func TestChatEntry_Valid_SimpleUser(t *testing.T) {
	e := ChatEntry{Role: RoleUser, Content: "hello"}
	assert.NoError(t, e.Valid())
}

func TestChatEntry_Valid_SimpleAssistant(t *testing.T) {
	e := ChatEntry{Role: RoleAssistant, Content: "reply"}
	assert.NoError(t, e.Valid())
}

func TestChatEntry_Valid_AssistantWithParts(t *testing.T) {
	e := ChatEntry{
		Role:    RoleAssistant,
		Content: "visible",
		Parts: []MessagePart{
			{Type: PartReasoning, Content: "thinking"},
			{Type: PartText, Content: "visible"},
		},
	}
	assert.NoError(t, e.Valid())
}

func TestChatEntry_Valid_UnknownRole(t *testing.T) {
	e := ChatEntry{Role: "bogus"}
	err := e.Valid()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid role")
}

func TestChatEntry_Valid_EmptyRole(t *testing.T) {
	e := ChatEntry{Role: ""}
	assert.Error(t, e.Valid())
}

func TestChatEntry_Valid_ToolRequiresToolID(t *testing.T) {
	e := ChatEntry{Role: RoleTool, ToolName: "bash", ToolID: ""}
	err := e.Valid()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ToolID")
}

func TestChatEntry_Valid_ToolWithID(t *testing.T) {
	e := ChatEntry{Role: RoleTool, ToolName: "bash", ToolID: "t1", ToolStatus: StatusPending}
	assert.NoError(t, e.Valid())
}

func TestChatEntry_Valid_CompleteRequiresMeta(t *testing.T) {
	e := ChatEntry{Role: RoleComplete}
	err := e.Valid()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Meta")
}

func TestChatEntry_Valid_CompleteWithMeta(t *testing.T) {
	e := ChatEntry{Role: RoleComplete, Meta: &CompleteMeta{Mode: "build", Model: "gpt-4"}}
	assert.NoError(t, e.Valid())
}

func TestChatEntry_Valid_PartsContentMismatch(t *testing.T) {
	e := ChatEntry{
		Role:    RoleAssistant,
		Content: "wrong content",
		Parts: []MessagePart{
			{Type: PartText, Content: "correct content"},
		},
	}
	err := e.Valid()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Content mismatch")
}

func TestChatEntry_Valid_InvalidPart(t *testing.T) {
	e := ChatEntry{
		Role: RoleAssistant,
		Parts: []MessagePart{
			{Type: PartType("bogus"), Content: "x"},
		},
	}
	err := e.Valid()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "part[0]")
}

// --- CommitStream dual-path test ---

func TestChat_CommitStream_BothPathsPopulated_PartsWin(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	// Populate both paths — parts should take precedence
	c.StreamPart(PartText, "part-based content")
	c.streaming = "legacy content" // force legacy path too

	c.CommitStream()

	entries := c.Entries()
	assert.Len(t, entries, 1)
	assert.Equal(t, "part-based content", entries[0].Content)
	assert.Empty(t, c.streaming, "legacy streaming should be cleared")
}

// --- View with empty entries + streaming parts ---

func TestChat_View_EmptyEntriesWithStreamingParts(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	// No committed entries, only streaming parts
	c.StreamPart(PartText, "streaming text")

	view := c.View()
	assert.Contains(t, view, "streaming text")
}

// --- Rendering with parts ---

func TestChat_RenderAssistantParts_TextOnly(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowThinking(true)

	c.AddEntry(ChatEntry{
		Role: RoleAssistant,
		Parts: []MessagePart{
			{Type: PartText, Content: "hello world"},
		},
	})

	view := c.View()
	assert.Contains(t, view, "hello world")
}

func TestChat_RenderAssistantParts_ReasoningVisible(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowThinking(true)

	c.AddEntry(ChatEntry{
		Role: RoleAssistant,
		Parts: []MessagePart{
			{Type: PartReasoning, Content: "let me think"},
			{Type: PartText, Content: "answer"},
		},
	})

	view := c.View()
	assert.Contains(t, view, "let me think")
	assert.Contains(t, view, "answer")
}

func TestChat_RenderAssistantParts_ReasoningHidden(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowThinking(false)

	c.AddEntry(ChatEntry{
		Role: RoleAssistant,
		Parts: []MessagePart{
			{Type: PartReasoning, Content: "secret thinking"},
			{Type: PartText, Content: "visible answer"},
		},
	})

	view := c.View()
	assert.NotContains(t, view, "secret thinking")
	assert.Contains(t, view, "visible answer")
}

func TestChat_RenderAssistantParts_Interleaved(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowThinking(true)

	c.AddEntry(ChatEntry{
		Role: RoleAssistant,
		Parts: []MessagePart{
			{Type: PartText, Content: "part one"},
			{Type: PartReasoning, Content: "hmm"},
			{Type: PartText, Content: "part two"},
		},
	})

	view := c.View()
	assert.Contains(t, view, "part one")
	assert.Contains(t, view, "hmm")
	assert.Contains(t, view, "part two")
}

func TestChat_RenderAssistantParts_BackwardCompat_ContentField(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)

	// Legacy entry with no Parts — should still render via Content
	c.AddEntry(ChatEntry{Role: RoleAssistant, Content: "legacy text"})

	view := c.View()
	assert.Contains(t, view, "legacy text")
}

// --- Hidden reasoning indicator ---

func TestChat_RenderAssistantParts_AllReasoningHidden_ShowsIndicator(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowThinking(false)

	c.AddEntry(ChatEntry{
		Role: RoleAssistant,
		Parts: []MessagePart{
			{Type: PartReasoning, Content: "all reasoning, no text"},
		},
	})

	view := c.View()
	assert.Contains(t, view, "thinking", "should show a [thinking] indicator when all parts are hidden reasoning")
	assert.NotContains(t, view, "all reasoning, no text")
}

func TestChat_RenderAssistantParts_AllReasoningVisible_NoIndicator(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowThinking(true)

	c.AddEntry(ChatEntry{
		Role: RoleAssistant,
		Parts: []MessagePart{
			{Type: PartReasoning, Content: "reasoning content"},
		},
	})

	view := c.View()
	assert.Contains(t, view, "reasoning content")
	// Should NOT show the [thinking] indicator when thinking is visible
	assert.NotContains(t, view, "[thinking]")
}

// --- Streaming parts in View ---

func TestChat_StreamingParts_VisibleInView(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowThinking(true)

	c.StreamPart(PartText, "streaming ")
	c.StreamPart(PartText, "content")

	view := c.View()
	assert.Contains(t, view, "streaming content")
}

func TestChat_StreamingParts_ReasoningVisibleWhenThinkingOn(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowThinking(true)

	c.StreamPart(PartReasoning, "thinking stream")

	view := c.View()
	assert.Contains(t, view, "thinking stream")
}

func TestChat_StreamingParts_ReasoningHiddenWhenThinkingOff(t *testing.T) {
	c := NewChat()
	c.SetSize(80, 20)
	c.SetShowThinking(false)

	c.StreamPart(PartReasoning, "hidden thinking")
	c.StreamPart(PartText, "visible text")

	view := c.View()
	assert.NotContains(t, view, "hidden thinking")
	assert.Contains(t, view, "visible text")
}

// --- Toggle setters ---

func TestChat_SetShowThinking(t *testing.T) {
	c := NewChat()
	assert.False(t, c.showThinking)

	c.SetShowThinking(true)
	assert.True(t, c.showThinking)

	c.SetShowThinking(false)
	assert.False(t, c.showThinking)
}

func TestChat_SetShowDetails(t *testing.T) {
	c := NewChat()
	assert.False(t, c.showDetails)

	c.SetShowDetails(true)
	assert.True(t, c.showDetails)
}

func TestChat_SetShowTimestamps(t *testing.T) {
	c := NewChat()
	assert.False(t, c.showTimestamps)

	c.SetShowTimestamps(true)
	assert.True(t, c.showTimestamps)
}

func TestChat_SetShowCodeBlocks(t *testing.T) {
	c := NewChat()
	assert.True(t, c.showCodeBlocks, "default should be true (show code)")

	c.SetShowCodeBlocks(false)
	assert.False(t, c.showCodeBlocks)
}
