package store

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const historyScanMaxLineBytes = 1024 * 1024 // 1 MiB

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
// The second return value is the number of malformed entries that were skipped.
func LoadHistory() (entries []HistoryEntry, skipped int, err error) {
	path := historyPath()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("open history: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), historyScanMaxLineBytes)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry HistoryEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			LogError("history: malformed entry", err)
			skipped++
			continue
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return entries, skipped, fmt.Errorf("scan history: %w", err)
	}
	return entries, skipped, nil
}

// SaveHistory writes all entries to the JSONL file atomically, replacing any
// existing content. Fails fast if any entry cannot be marshaled — the file
// is only written when all entries serialize successfully.
func SaveHistory(entries []HistoryEntry) error {
	dir := StateDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	var buf bytes.Buffer
	for i, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("marshal history entry %d: %w", i, err)
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}
	if err := atomicWrite(historyPath(), buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write history: %w", err)
	}
	return nil
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
