package tool

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCtx(t *testing.T) Context {
	t.Helper()
	return Context{
		SessionID: "test",
		WorkDir:   t.TempDir(),
		Abort:     context.Background(),
	}
}

func TestBash_ImplementsTool(t *testing.T) {
	var _ Tool = &BashTool{}
}

func TestBash_ID(t *testing.T) {
	b := NewBashTool()
	assert.Equal(t, "bash", b.ID())
}

func TestBash_SimpleCommand(t *testing.T) {
	b := NewBashTool()
	ctx := newTestCtx(t)

	result := b.Execute(ctx, `{"command":"echo hello"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, "hello")
}

func TestBash_WorkDir(t *testing.T) {
	b := NewBashTool()
	ctx := newTestCtx(t)

	result := b.Execute(ctx, `{"command":"pwd"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, ctx.WorkDir)
}

func TestBash_CustomWorkDir(t *testing.T) {
	b := NewBashTool()
	ctx := newTestCtx(t)

	// Subdirectory within workdir.
	sub := filepath.Join(ctx.WorkDir, "subdir")
	require.NoError(t, os.MkdirAll(sub, 0o755))

	result := b.Execute(ctx, `{"command":"pwd","workdir":"`+sub+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, sub)
}

func TestBash_WorkDirTraversalBlocked(t *testing.T) {
	b := NewBashTool()
	ctx := newTestCtx(t)

	// Absolute path outside workdir.
	result := b.Execute(ctx, `{"command":"pwd","workdir":"/etc"}`)
	require.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "workdir")

	// Relative traversal outside workdir.
	result = b.Execute(ctx, `{"command":"pwd","workdir":"../../etc"}`)
	require.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "workdir")
}

func TestBash_WorkDirSubdirAllowed(t *testing.T) {
	b := NewBashTool()
	ctx := newTestCtx(t)

	// Subdirectory within workdir should be allowed.
	result := b.Execute(ctx, `{"command":"pwd","workdir":"`+ctx.WorkDir+`"}`)
	require.NoError(t, result.Error)
	assert.Contains(t, result.Output, ctx.WorkDir)
}

func TestBash_CommandFailure(t *testing.T) {
	b := NewBashTool()
	ctx := newTestCtx(t)

	result := b.Execute(ctx, `{"command":"exit 1"}`)
	assert.Error(t, result.Error)
}

func TestBash_BlockedCommands(t *testing.T) {
	b := NewBashTool()
	ctx := newTestCtx(t)

	blocked := []string{
		`{"command":"rm -rf /"}`,
		`{"command":"sudo rm something"}`,
		`{"command":"mkfs.ext4 /dev/sda"}`,
		`{"command":"dd if=/dev/zero of=/dev/sda"}`,
	}

	for _, args := range blocked {
		result := b.Execute(ctx, args)
		assert.Error(t, result.Error, "should block: %s", args)
		assert.Contains(t, result.Error.Error(), "blocked")
	}
}

func TestBash_BlockedCommandVariants(t *testing.T) {
	b := NewBashTool()
	ctx := newTestCtx(t)

	variants := []string{
		// Flag reordering
		`{"command":"rm -f -r /"}`,
		`{"command":"rm -r -f /"}`,
		// Extra whitespace
		`{"command":"rm  -rf  /"}`,
		// Backslash escapes in command
		`{"command":"r\\m -rf /"}`,
		// sudo with extra spaces
		`{"command":"sudo   rm something"}`,
		// Tab characters
		`{"command":"sudo\trm something"}`,
		// Fork bomb
		`{"command":":(){ :|:& };:"}`,
		// chmod 777 root
		`{"command":"chmod -R 777 /"}`,
		// rm root with --no-preserve-root
		`{"command":"rm -rf / --no-preserve-root"}`,
	}

	for _, args := range variants {
		result := b.Execute(ctx, args)
		assert.Error(t, result.Error, "should block: %s", args)
		assert.Contains(t, result.Error.Error(), "blocked", "should block: %s", args)
	}
}

func TestBash_AllowedCommands(t *testing.T) {
	b := NewBashTool()
	ctx := newTestCtx(t)

	// Commands that look similar but are safe
	allowed := []string{
		`{"command":"ls /"}`,
		`{"command":"rm file.txt"}`,
		`{"command":"rm -rf /tmp/build-cache"}`,
		`{"command":"rm -rf /var/log/old"}`,
	}

	for _, args := range allowed {
		result := b.Execute(ctx, args)
		// These should not be blocked (may error for other reasons but not "blocked")
		if result.Error != nil {
			assert.NotContains(t, result.Error.Error(), "blocked", "should not block: %s", args)
		}
	}
}

func TestBash_InvalidJSON(t *testing.T) {
	b := NewBashTool()
	ctx := newTestCtx(t)

	result := b.Execute(ctx, `{bad json}`)
	assert.Error(t, result.Error)
}

func TestBash_MissingCommand(t *testing.T) {
	b := NewBashTool()
	ctx := newTestCtx(t)

	result := b.Execute(ctx, `{}`)
	assert.Error(t, result.Error)
}

func TestBash_OutputTruncation(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("requires unix")
	}

	b := NewBashTool()
	ctx := newTestCtx(t)

	// Generate output larger than MaxOutputBytes
	result := b.Execute(ctx, `{"command":"yes | head -100000"}`)
	require.NoError(t, result.Error)

	// Output should be truncated and include a truncation notice
	if len(result.Output) > MaxOutputBytes+500 {
		t.Errorf("output not truncated: got %d bytes", len(result.Output))
	}
	if len(result.Output) > MaxOutputBytes {
		assert.Contains(t, result.Output, "truncated")
	}
}

func TestBash_Timeout(t *testing.T) {
	b := NewBashTool()
	ctx := newTestCtx(t)

	// Use a very short timeout
	result := b.Execute(ctx, `{"command":"sleep 10","timeout":100}`)
	assert.Error(t, result.Error)
	assert.True(t,
		strings.Contains(result.Error.Error(), "killed") ||
			strings.Contains(result.Error.Error(), "signal"),
		"expected timeout error, got: %s", result.Error)
}

func TestBash_ContextCancellation(t *testing.T) {
	b := NewBashTool()
	abortCtx, cancel := context.WithCancel(context.Background())
	ctx := Context{
		SessionID: "test",
		WorkDir:   t.TempDir(),
		Abort:     abortCtx,
	}

	// Cancel immediately
	cancel()

	result := b.Execute(ctx, `{"command":"sleep 10"}`)
	assert.Error(t, result.Error)
}

func TestBash_MidExecutionCancellation(t *testing.T) {
	b := NewBashTool()
	abortCtx, cancel := context.WithCancel(context.Background())
	ctx := Context{
		SessionID: "test",
		WorkDir:   t.TempDir(),
		Abort:     abortCtx,
	}

	done := make(chan Result, 1)
	go func() {
		done <- b.Execute(ctx, `{"command":"sleep 30"}`)
	}()

	// Cancel mid-execution.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case result := <-done:
		assert.Error(t, result.Error)
	case <-time.After(5 * time.Second):
		t.Fatal("command did not terminate after cancellation")
	}
}

func TestBash_Parameters(t *testing.T) {
	b := NewBashTool()
	params := b.Parameters()

	props := params["properties"].(map[string]any)
	assert.Contains(t, props, "command")
	assert.Contains(t, props, "timeout")
	assert.Contains(t, props, "workdir")

	required := params["required"].([]string)
	assert.Contains(t, required, "command")
}
