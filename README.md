# ripcode

An agentic AI coding assistant for the terminal, built in Go with [Bubble Tea v2](https://charm.land/bubbletea/v2).

Inspired by [opencode](https://github.com/anomalyco/opencode) — reimagined as a single Go binary with no runtime dependencies.

## Features

- **Agentic loop** — prompt → stream → tool calls → execute → repeat
- **8 built-in tools** — bash, read, write, edit, glob, grep, ls, todo
- **Streaming responses** — real-time token-by-token output
- **Agent modes** — build (full access) and plan (read-only)
- **Safety blocklist** — dangerous shell commands are rejected
- **Provider-agnostic** — OpenRouter gateway with 400+ models

## Architecture

```
cmd/ripcode/          Entry point — wires everything together
internal/
  agent/              Agent modes (build/plan) and agentic loop
  client/             OpenRouter SSE streaming client
  config/             Configuration loading (env + .env)
  provider/           Provider interface and types
  session/            Conversation session and token tracking
  tool/               Tool interface, registry, and 8 implementations
  tui/                Bubble Tea v2 terminal UI
    components/       Chat viewport, input, status bar, tool panel
    styles/           Lip Gloss v2 theme
tests/                Integration tests
```

## Tech Stack

| Component | Choice |
|-----------|--------|
| Language | Go 1.25 |
| TUI | [Bubble Tea v2](https://charm.land/bubbletea/v2) |
| Styling | [Lip Gloss v2](https://charm.land/lipgloss/v2) |
| Testing | [testify](https://github.com/stretchr/testify) |
| Glob | [doublestar](https://github.com/bmatcuk/doublestar) |
| LLM Gateway | [OpenRouter](https://openrouter.ai) |

## Quick Start

```bash
# Build
go build -o bin/ripcode ./cmd/ripcode

# Run (requires OpenRouter API key)
OPENROUTER_API_KEY=your-key bin/ripcode

# Or use a .env file
echo "OPENROUTER_API_KEY=your-key" > .env
bin/ripcode
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENROUTER_API_KEY` | (required) | OpenRouter API key |
| `RIPCODE_MODEL` | `z-ai/glm-5` | Model to use |
| `RIPCODE_MAX_STEPS` | `100` | Max agentic loop iterations |

## Key Bindings

| Key | Action |
|-----|--------|
| `Enter` | Submit input |
| `Shift+Enter` | New line in input |
| `Esc` | Cancel streaming / quit |
| `Ctrl+C` | Force quit |
| `Ctrl+L` | Clear chat |

## Development

```bash
go test ./... -v -race    # 147 tests with race detection
go vet ./...              # Static analysis
```

## License

MIT
