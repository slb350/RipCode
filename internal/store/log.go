package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogError appends a timestamped error line to the error log.
// Best-effort: silently drops if the log file cannot be opened.
func LogError(msg string, err error) {
	path := filepath.Join(StateDir(), "errors.log")
	os.MkdirAll(filepath.Dir(path), 0o755)
	f, ferr := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if ferr != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s: %v\n", time.Now().Format(time.RFC3339), msg, err)
}
