package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// HistoryEntry is a single prompt history record.
type HistoryEntry struct {
	Prompt string `json:"prompt"`
	Mode   string `json:"mode"` // "normal" or "shell"
}

func historyPath() string {
	return filepath.Join(StateDir(), "prompt-history.jsonl")
}

// LoadHistory reads all prompt history entries from the JSONL file.
// Returns empty slice if the file does not exist.
func LoadHistory() ([]HistoryEntry, error) {
	path := historyPath()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open history: %w", err)
	}
	defer f.Close()

	var entries []HistoryEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry HistoryEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			LogError("history: malformed entry", err)
			continue
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return entries, fmt.Errorf("scan history: %w", err)
	}
	return entries, nil
}

// SaveHistory writes all entries to the JSONL file, replacing any existing content.
func SaveHistory(entries []HistoryEntry) error {
	dir := StateDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	f, err := os.Create(historyPath())
	if err != nil {
		return fmt.Errorf("create history file: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			LogError("history: marshal entry", err)
			continue
		}
		if _, err := w.Write(data); err != nil {
			LogError("history: write entry", err)
			continue
		}
		if err := w.WriteByte('\n'); err != nil {
			LogError("history: write newline", err)
			continue
		}
	}
	return w.Flush()
}

// AppendHistory appends a single entry to the JSONL file.
func AppendHistory(entry HistoryEntry) error {
	dir := StateDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	f, err := os.OpenFile(historyPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open history for append: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal history entry: %w", err)
	}
	data = append(data, '\n')
	n, err := f.Write(data)
	if err != nil {
		return fmt.Errorf("write history entry: %w", err)
	}
	if n != len(data) {
		return fmt.Errorf("partial write to history: wrote %d of %d bytes", n, len(data))
	}
	return nil
}
