package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// MaxOutputBytes is the limit beyond which tool output is truncated.
const MaxOutputBytes = 50 * 1024 // 50 KB

const defaultTimeout = 2 * time.Minute

// blockedPatterns contains regex patterns for commands that are never allowed.
// These are defense-in-depth — not a security sandbox.
var blockedPatterns = []*regexp.Regexp{
	// rm with -r and -f flags in any order, targeting root
	regexp.MustCompile(`\brm\s+(-[a-z]*r[a-z]*\s+)*-[a-z]*f[a-z]*\s+/(\s|$)`),
	regexp.MustCompile(`\brm\s+(-[a-z]*f[a-z]*\s+)*-[a-z]*r[a-z]*\s+/(\s|$)`),
	regexp.MustCompile(`\brm\s+-rf\s+/(\s|$)`),
	// sudo rm / sudo dd
	regexp.MustCompile(`\bsudo\s+rm\b`),
	regexp.MustCompile(`\bsudo\s+dd\b`),
	// mkfs variants
	regexp.MustCompile(`\bmkfs\b`),
	// dd to block devices
	regexp.MustCompile(`\bdd\s+.*of=/dev/(sd|nvm)`),
	// Fork bomb
	regexp.MustCompile(`:\(\)\s*\{.*\|.*&.*\}\s*;`),
	// Write to block devices
	regexp.MustCompile(`>\s*/dev/sd`),
	// chmod 777 root
	regexp.MustCompile(`\bchmod\s+(-[a-zA-Z]*\s+)*777\s+/(\s|$)`),
}

// BashTool executes shell commands.
type BashTool struct{}

// NewBashTool creates a new bash tool instance.
func NewBashTool() *BashTool {
	return &BashTool{}
}

func (b *BashTool) ID() string { return "bash" }

func (b *BashTool) Description() string {
	return "Execute a shell command and return its output."
}

func (b *BashTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The shell command to execute",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Timeout in milliseconds (default: 120000)",
			},
			"workdir": map[string]any{
				"type":        "string",
				"description": "Working directory for the command",
			},
		},
		"required": []string{"command"},
	}
}

type bashArgs struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
	WorkDir string `json:"workdir,omitempty"`
}

func (b *BashTool) Execute(ctx Context, argsJSON string) Result {
	var args bashArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Result{Error: fmt.Errorf("parse args: %w", err)}
	}

	if args.Command == "" {
		return Result{Error: fmt.Errorf("command is required")}
	}

	if err := checkBlocked(args.Command); err != nil {
		return Result{Error: err}
	}

	timeout := defaultTimeout
	if args.Timeout > 0 {
		timeout = time.Duration(args.Timeout) * time.Millisecond
	}

	cmdCtx, cancel := context.WithTimeout(ctx.Abort, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "sh", "-c", args.Command)
	cmd.Dir = args.WorkDir
	if cmd.Dir == "" {
		cmd.Dir = ctx.WorkDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	output = truncate(output, MaxOutputBytes)

	if err != nil {
		return Result{
			Output: output,
			Title:  args.Command,
			Error:  fmt.Errorf("command failed: %w", err),
		}
	}

	return Result{
		Output: output,
		Title:  args.Command,
	}
}

// checkBlocked returns an error if the command matches a blocked pattern.
func checkBlocked(cmd string) error {
	normalized := normalizeCommand(cmd)
	for _, pattern := range blockedPatterns {
		if pattern.MatchString(normalized) {
			return fmt.Errorf("blocked: command matches dangerous pattern %q", pattern.String())
		}
	}
	return nil
}

// normalizeCommand strips backslash escapes, collapses whitespace, and
// lowercases for blocklist matching.
func normalizeCommand(cmd string) string {
	// Remove backslash escapes (e.g. r\m → rm)
	cmd = strings.ReplaceAll(cmd, "\\", "")
	// Replace tabs with spaces
	cmd = strings.ReplaceAll(cmd, "\t", " ")
	// Collapse multiple spaces
	parts := strings.Fields(cmd)
	cmd = strings.Join(parts, " ")
	return strings.ToLower(strings.TrimSpace(cmd))
}

// truncate shortens output to maxBytes, appending a notice if truncated.
func truncate(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes] + "\n\n[output truncated — exceeded " + fmt.Sprintf("%d", maxBytes) + " bytes]"
}
