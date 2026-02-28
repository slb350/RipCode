# ripcode

An agentic AI coding assistant for the terminal. Go + Bubble Tea v2.

## Project Layout

```
cmd/ripcode/              Entry point — wires config, providers, TUI
internal/
  agent/                  Agent modes and agentic tool-call loop
    agent.go              Agent interface, mode definitions (build/plan)
    loop.go               Core agentic loop: prompt → tool calls → execute → repeat
  client/                 LLM provider abstraction
    client.go             ChatClient interface, Message, StreamEvent types
    openrouter.go         OpenRouter provider implementation
  config/                 Configuration
    config.go             Load from env, .env, config file, CLI flags
  provider/               Provider registry and model catalog
    provider.go           Provider interface and registry
    models.go             Model definitions and capabilities
  session/                Conversation management
    session.go            Session lifecycle, message history
    storage.go            SQLite persistence (future)
  tool/                   Tool system
    tool.go               Tool interface, registry
    bash.go               Shell command execution with safety
    read.go               File reading
    write.go              File writing
    edit.go               File editing (line-range precision)
    glob.go               Pattern-based file discovery
    grep.go               Content search
    ls.go                 Directory listing
    todo.go               Session-scoped task tracking
  tui/                    Terminal UI (Bubble Tea v2)
    app.go                Top-level model, routing, focus management
    components/           Reusable components
      chat.go             Chat viewport with markdown rendering
      input.go            Multi-line input with vim bindings
      sidebar.go          Session list / file tree
      statusbar.go        Model info, token count, mode indicator
      toolbar.go          Tool execution feedback
    styles/               Theme and style definitions
      theme.go            Color palette, adaptive light/dark
tests/                    Integration tests
reference/                Cloned repos for study (gitignored)
```

## Environment

- **Runtime:** Go 1.26
- **TUI:** charm.land/bubbletea/v2, charm.land/bubbles/v2, charm.land/lipgloss/v2
- **Testing:** github.com/stretchr/testify
- **Module:** github.com/stephenbrandon/ripcode

## How to Run

```bash
go build -o bin/ripcode ./cmd/ripcode    # Build
go test ./... -v -race                    # Test with race detection
go vet ./...                              # Static analysis
bin/ripcode                               # Run
```

## Coding Conventions

### Go Style
- **Strict typing.** No `any` unless interfacing with JSON. Use `unknown` patterns with type assertions.
- **Functional style preferred.** Pure functions, immutable data, composition over inheritance.
- **No classes unless the problem calls for it.** Prefer interfaces + functions.
- **Error handling.** Return errors, don't panic. Wrap with `fmt.Errorf("context: %w", err)`.
- **Naming.** Short, clear names. Single-word where unambiguous. `msg` not `message`, `cfg` not `configuration`.

### Bubble Tea v2 Conventions
- `View()` returns `tea.View` (not string). Use `tea.NewView(content)` and set struct fields declaratively.
- Key events are `tea.KeyPressMsg` / `tea.KeyReleaseMsg` (not `tea.KeyMsg`).
- Mouse events split into `tea.MouseClickMsg`, `tea.MouseWheelMsg`, etc.
- Lip Gloss is pure — no direct I/O calls from lipgloss. Bubble Tea manages terminal.

### Testing
- **Always TDD.** Write failing tests first, implement to pass, refactor.
- **Tests in same package** (`_test` suffix) for unit tests, `tests/` for integration.
- **No mocks when avoidable.** Use interfaces and test doubles.
- **Race detection always.** `go test -race` on every run.

### Architecture Principles
- **Client/server separation.** Agent logic is independent of TUI. The TUI is a consumer.
- **Provider-agnostic.** LLM calls go through an interface. OpenRouter is the default but not the only option.
- **Tool registry pattern.** Tools implement a common interface. New tools are registered, not hardcoded.
- **Session-based.** Conversations are sessions with persistent history.
- **Safety-first tools.** Destructive commands are blocked. File writes are logged.

### Commit Messages
```
type(scope): Brief description

Detailed explanation:
- What changed and why
- Testing notes
```
Types: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Single binary | Yes | No Node.js/Bun runtime dependency |
| Bubble Tea v2 | Over v1 | Declarative views, better keyboard handling, new renderer |
| OpenRouter first | Provider-agnostic interface | 400+ models via one gateway, add direct providers later |
| SQLite for sessions | Future | Lightweight, embedded, no server process |
| No client/server split initially | Monolithic | Simpler. Add HTTP API + SSE later if needed |

## Reference Material

`reference/opencode/` contains a shallow clone of [anomalyco/opencode](https://github.com/anomalyco/opencode) for architectural reference. Key directories to study:

| Path | What to learn |
|------|---------------|
| `packages/opencode/src/agent/` | Agent modes, system prompts |
| `packages/opencode/src/tool/` | Tool definitions (20+ tools with `.txt` prompt files) |
| `packages/opencode/src/session/` | Session management, message handling, compaction |
| `packages/opencode/src/provider/` | Multi-provider abstraction |
| `packages/opencode/src/server/` | HTTP API structure, SSE events |
| `packages/opencode/src/config/` | Config loading patterns |

## Important Rules

- **Never take shortcuts.** Write correct, clean code.
- **TDD always.** Failing test first. Green implementation second. Refactor third.
- **Run tests constantly.** After every meaningful code change.
- **Type check regularly.** `go vet ./...` after completing a logical chunk.
- **Reference opencode for patterns, not copy/paste.** We're building in Go idioms.
- **No force pushes.** No committing .env files.
