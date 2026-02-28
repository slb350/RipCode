# ripcode

An agentic AI coding assistant for the terminal, built in Go with [Bubble Tea v2](https://charm.land/bubbletea/v2).

Inspired by [opencode](https://github.com/anomalyco/opencode) — reimagined as a single Go binary with no runtime dependencies.

## Architecture

```
cmd/ripcode/          Entry point
internal/
  agent/              Agent modes (build, plan) and agentic loop
  client/             LLM provider interface and streaming
  config/             Configuration loading (.env, flags, config file)
  provider/           Provider adapters (OpenRouter, Anthropic, OpenAI)
  session/            Conversation session management and persistence
  tool/               Tool definitions and executors
  tui/                Bubble Tea v2 terminal UI
    components/       Reusable TUI components (chat, sidebar, input, etc.)
    styles/           lipgloss v2 theme and style definitions
reference/            Cloned repos for study (gitignored)
```

## Tech Stack

| Component | Choice |
|-----------|--------|
| Language | Go 1.26 |
| TUI | [Bubble Tea v2](https://charm.land/bubbletea/v2) + [Bubbles v2](https://charm.land/bubbles/v2) |
| Styling | [Lip Gloss v2](https://charm.land/lipgloss/v2) |
| Testing | [testify](https://github.com/stretchr/testify) |
| LLM Gateway | OpenRouter (multi-provider) |

## Development

```bash
go build -o bin/ripcode ./cmd/ripcode    # Build
go test ./... -v -race                    # Test
go vet ./...                              # Lint
bin/ripcode                               # Run
```

## Status

Early development. See CLAUDE.md for coding conventions and project decisions.
