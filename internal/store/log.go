package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogError appends a timestamped error line to the error log.
// Falls back to stderr if the log file cannot be opened.
func LogError(msg string, err error) {
	path := filepath.Join(StateDir(), "errors.log")
	if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
		fmt.Fprintf(os.Stderr, "[ripcode] (log file unavailable) %s: %v\n", msg, err)
		return
	}
	f, ferr := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if ferr != nil {
		fmt.Fprintf(os.Stderr, "[ripcode] (log file unavailable) %s: %v\n", msg, err)
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s: %v\n", time.Now().Format(time.RFC3339), msg, err)
}
