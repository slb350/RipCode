package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/stephenbrandon/ripcode/internal/provider"
	"github.com/stephenbrandon/ripcode/internal/session"
)

// SessionSummary holds minimal session info for listing.
type SessionSummary struct {
	ID           string
	Title        string
	WorkDir      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	MessageCount int
}

// sessionFile is the on-disk JSON format (v1).
type sessionFile struct {
	Version   int            `json:"version"`
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	ParentID  string         `json:"parentId,omitempty"`
	WorkDir   string         `json:"workDir"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Tokens    tokenCountFile `json:"tokens"`
	Messages  []messageFile  `json:"messages"`
}

type tokenCountFile struct {
	Input  int `json:"input"`
	Output int `json:"output"`
}

type messageFile struct {
	ID         string             `json:"id"`
	Role       string             `json:"role"`
	Content    string             `json:"content"`
	ToolCalls  []toolCallFile     `json:"toolCalls,omitempty"`
	ToolCallID string             `json:"toolCallId,omitempty"`
	CreatedAt  time.Time          `json:"createdAt"`
	Meta       *assistantMetaFile `json:"meta,omitempty"`
}

type toolCallFile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Args string `json:"args"`
}

type assistantMetaFile struct {
	Model        string        `json:"model"`
	Agent        string        `json:"agent,omitempty"`
	InputTokens  int           `json:"inputTokens"`
	OutputTokens int           `json:"outputTokens"`
	FinishReason string        `json:"finishReason"`
	Duration     time.Duration `json:"duration"`
}

// Save writes a session to disk as JSON.
func Save(s *session.Session) error {
	dir := SessionsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create sessions dir: %w", err)
	}

	f := sessionFile{
		Version:   1,
		ID:        s.ID,
		Title:     s.Title,
		ParentID:  s.ParentID,
		WorkDir:   s.WorkDir,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
		Tokens: tokenCountFile{
			Input:  s.Tokens.Input,
			Output: s.Tokens.Output,
		},
	}

	for _, rec := range s.Messages {
		mf := messageFile{
			ID:         rec.ID,
			Role:       rec.Message.Role,
			Content:    rec.Message.Content,
			ToolCallID: rec.Message.ToolCallID,
			CreatedAt:  rec.CreatedAt,
		}
		for _, tc := range rec.Message.ToolCalls {
			mf.ToolCalls = append(mf.ToolCalls, toolCallFile{
				ID:   tc.ID,
				Name: tc.Name,
				Args: tc.Args,
			})
		}
		if rec.Meta != nil {
			mf.Meta = &assistantMetaFile{
				Model:        rec.Meta.Model,
				Agent:        rec.Meta.Agent,
				InputTokens:  rec.Meta.InputTokens,
				OutputTokens: rec.Meta.OutputTokens,
				FinishReason: rec.Meta.FinishReason,
				Duration:     rec.Meta.Duration,
			}
		}
		f.Messages = append(f.Messages, mf)
	}

	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	path := filepath.Join(dir, s.ID+".json")
	return os.WriteFile(path, data, 0o644)
}

// Load reads a session from disk by ID.
func Load(id string) (*session.Session, error) {
	path := filepath.Join(SessionsDir(), id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read session %s: %w", id, err)
	}

	var f sessionFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("unmarshal session %s: %w", id, err)
	}

	s := &session.Session{
		ID:        f.ID,
		Title:     f.Title,
		ParentID:  f.ParentID,
		WorkDir:   f.WorkDir,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
		Tokens: session.TokenCount{
			Input:  f.Tokens.Input,
			Output: f.Tokens.Output,
		},
	}

	for _, mf := range f.Messages {
		rec := session.MessageRecord{
			ID: mf.ID,
			Message: provider.Message{
				Role:       mf.Role,
				Content:    mf.Content,
				ToolCallID: mf.ToolCallID,
			},
			CreatedAt: mf.CreatedAt,
		}
		for _, tc := range mf.ToolCalls {
			rec.Message.ToolCalls = append(rec.Message.ToolCalls, provider.ToolCall{
				ID:   tc.ID,
				Name: tc.Name,
				Args: tc.Args,
			})
		}
		if mf.Meta != nil {
			rec.Meta = &session.AssistantMeta{
				Model:        mf.Meta.Model,
				Agent:        mf.Meta.Agent,
				InputTokens:  mf.Meta.InputTokens,
				OutputTokens: mf.Meta.OutputTokens,
				FinishReason: mf.Meta.FinishReason,
				Duration:     mf.Meta.Duration,
			}
		}
		s.Messages = append(s.Messages, rec)
	}

	return s, nil
}

// List returns summaries of all saved sessions, sorted by UpdatedAt descending.
func List() ([]SessionSummary, error) {
	dir := SessionsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read sessions dir: %w", err)
	}

	var summaries []SessionSummary
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			// Intentional skip: unreadable session files (permissions, race
			// conditions) are skipped so the rest of the listing still works.
			continue
		}

		var f sessionFile
		if err := json.Unmarshal(data, &f); err != nil {
			// Intentional skip: corrupted session files are skipped rather
			// than failing the entire listing operation.
			continue
		}

		summaries = append(summaries, SessionSummary{
			ID:           id,
			Title:        f.Title,
			WorkDir:      f.WorkDir,
			CreatedAt:    f.CreatedAt,
			UpdatedAt:    f.UpdatedAt,
			MessageCount: len(f.Messages),
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].UpdatedAt.After(summaries[j].UpdatedAt)
	})

	return summaries, nil
}

// Delete removes a saved session by ID.
func Delete(id string) error {
	path := filepath.Join(SessionsDir(), id+".json")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("session %s not found: %w", id, err)
		}
		return fmt.Errorf("delete session %s: %w", id, err)
	}
	return nil
}
