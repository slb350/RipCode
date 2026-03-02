# ripcode

An agentic AI coding assistant for the terminal, built in Go with [Bubble Tea v2](https://charm.land/bubbletea/v2).

Inspired by [opencode](https://github.com/anomalyco/opencode) — reimagined as a single Go binary with no runtime dependencies.

## Features

### Core

- **Agentic loop** — prompt → stream → tool calls → execute → repeat
- **8 built-in tools** — bash, read, write, edit, glob, grep, ls, todo
- **Streaming responses** — real-time token-by-token output
- **Agent modes** — build (full access) and plan (read-only), switchable with `Tab`
- **Safety blocklist** — dangerous shell commands are rejected
- **Provider-agnostic** — OpenRouter gateway with 400+ models

### Input & Editing

- **Readline keybindings** — `Ctrl+A/E` (home/end), `Ctrl+U/K` (kill line), `Ctrl+W` (kill word), `Alt+B/F` (word motions), `Alt+D` (delete word forward), `Alt+Backspace` (delete word back)
- **Input undo/redo** — `Ctrl+-` to undo, `Ctrl+.` to redo edits within the input
- **Prompt history** — `Up/Down` recalls prior prompts with safe cursor behavior
- **Shell mode** — prefix with `!` to run shell commands directly, with mode badge indicator
- **Prompt stash** — `/stash` saves drafts, `/stash pop` restores, `/stash list` browses saved drafts
- **Contextual Ctrl+C** — clears input when text present, exits when empty

### Commands & Palette

- **26 slash commands** across 4 categories (Session, View, Agent, System)
- **Command palette** — `Ctrl+P` / `Ctrl+K` with categorized display, keybind hints, suggested section, and type-to-filter
- **Inline autocomplete** — `/` for commands, `@` for file mentions
- **Toast notifications** — success/error/warning/info feedback with auto-dismiss

### Session Management

- **Session persistence** — auto-saves to `~/.ripcode/sessions/` as JSON
- **Session list/resume** — `/sessions` dialog with search, date grouping, delete confirmation
- **Session rename** — `/rename` or `Ctrl+R` to rename the current session
- **Undo/redo** — `/undo` reverts last exchange and restores prompt to input, `/redo` re-applies
- **Timeline jump** — `/timeline` dialog to jump to any user message
- **Copy to clipboard** — `/copy` copies last assistant response
- **Export transcript** — `/export` with toggles for tool calls and metadata, custom filename

### Message Navigation

- **Page scroll** — `PageUp/PageDown`, `Ctrl+Alt+U/D` (half-page), `Ctrl+Alt+Y/E` (line)
- **Jump to edges** — `Ctrl+G` (top), `Ctrl+Alt+G` (bottom)

### Dialogs

- **Help dialog** — `/help` with tabbed Commands/Keybinds sections and type-to-filter
- **Status dialog** — `/status` showing model, agent, tokens, working directory
- **Model picker** — `/models` with searchable list and in-memory cache
- **Session picker** — `/sessions` with date grouping and inline delete
- **Stash picker** — `/stash list` to browse and restore saved drafts

### Layout

- **Two-screen state machine** — Home screen with logo/tips, Session screen for conversation
- **Sidebar** — `Ctrl+B` toggles wide sidebar; narrow overlay on small terminals
- **Status bar** — model name, agent mode, token counts, activity indicator
- **Inline tool calls** — tool execution displayed inline with status icons

## Architecture

```text
cmd/ripcode/              Entry point
internal/
  agent/                  Agent modes (build/plan) and agentic loop
  client/                 OpenRouter SSE streaming client
  config/                 Configuration loading (env + .env)
  provider/               Provider interface, types, thinking variants
  session/                Session lifecycle, message records, undo/redo
  store/                  JSON persistence (sessions, model prefs, MCP/LSP, UI)
  tool/                   Tool interface, registry, 8 implementations
  tui/                    Bubble Tea v2 terminal UI
    app.go                Top-level model, Update, View, state machine
    commands.go           Command registry and slash command handlers
    dialog_*.go           14 dialog modules (model, sessions, help, export, ...)
    keys.go               Key routing and leader key dispatch
    sidebar.go            Sidebar rendering and section toggling
    submit.go             Input submission and agent dispatch
    ...                   events, helpers, inline, models, palette, persist
    components/           Chat, input, status bar, home, prompt stash, toast
    styles/               Lip Gloss v2 theme
tests/                    Integration tests
```

## Tech Stack

| Component   | Choice                                                    |
| ----------- | --------------------------------------------------------- |
| Language    | Go 1.25                                                   |
| TUI         | [Bubble Tea v2](https://charm.land/bubbletea/v2)         |
| Styling     | [Lip Gloss v2](https://charm.land/lipgloss/v2)           |
| Testing     | [testify](https://github.com/stretchr/testify)           |
| Glob        | [doublestar](https://github.com/bmatcuk/doublestar)      |
| Clipboard   | [atotto/clipboard](https://github.com/atotto/clipboard)  |
| LLM Gateway | [OpenRouter](https://openrouter.ai)                      |

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

| Variable             | Default      | Description          |
| -------------------- | ------------ | -------------------- |
| `OPENROUTER_API_KEY` | (required)   | OpenRouter API key   |
| `RIPCODE_MODEL`      | `z-ai/glm-5` | Model to use        |
| `RIPCODE_MAX_STEPS`  | `100`        | Max loop iterations  |

## Key Bindings

### Input Editing

| Key                        | Action              |
| -------------------------- | ------------------- |
| `Enter`                    | Submit prompt       |
| `Shift+Enter`              | New line            |
| `Ctrl+A`                   | Move to line start  |
| `Ctrl+E`                   | Move to line end    |
| `Ctrl+U`                   | Delete to line start |
| `Ctrl+K`                   | Delete to line end  |
| `Ctrl+W`                   | Delete word back    |
| `Alt+D`                    | Delete word forward |
| `Alt+B` / `Ctrl+Left`     | Move word left      |
| `Alt+F` / `Ctrl+Right`    | Move word right     |
| `Ctrl+-`                   | Undo input edit     |
| `Ctrl+.`                   | Redo input edit     |
| `Up` / `Down`              | History recall      |

### Chat Navigation

| Key                              | Action          |
| -------------------------------- | --------------- |
| `PageUp` / `PageDown`           | Scroll page     |
| `Ctrl+Alt+U` / `Ctrl+Alt+D`    | Scroll half-page |
| `Ctrl+Alt+Y` / `Ctrl+Alt+E`    | Scroll line     |
| `Ctrl+G`                         | Scroll to top   |
| `Ctrl+Alt+G`                     | Scroll to bottom |

### Application

| Key                      | Action                       |
| ------------------------ | ---------------------------- |
| `Ctrl+P` / `Ctrl+K`    | Command palette              |
| `Ctrl+B`                | Toggle sidebar               |
| `Ctrl+R`                | Rename session               |
| `Ctrl+L`                | Clear chat                   |
| `Tab` / `Shift+Tab`     | Cycle agent mode             |
| `Esc`                    | Cancel streaming / close dialog |
| `Ctrl+C`                 | Clear input / exit           |

## Slash Commands

| Command       | Aliases          | Category | Description                   |
| ------------- | ---------------- | -------- | ----------------------------- |
| `/help`       | `/commands`      | System   | Show available commands       |
| `/status`     |                  | System   | View system status            |
| `/exit`       | `/quit`, `/q`    | System   | Quit ripcode                  |
| `/new`        | `/clear`         | Session  | Start new session             |
| `/rename`     |                  | Session  | Rename current session        |
| `/sessions`   |                  | Session  | Browse and resume sessions    |
| `/undo`       |                  | Session  | Revert last exchange          |
| `/redo`       |                  | Session  | Restore reverted exchange     |
| `/timeline`   |                  | Session  | Jump to any message           |
| `/copy`       |                  | Session  | Copy last response            |
| `/export`     |                  | Session  | Export transcript to file     |
| `/stash`      |                  | Session  | Save input draft              |
| `/stash-pop`  |                  | Session  | Restore last draft            |
| `/stash-list` |                  | Session  | Browse saved drafts           |
| `/models`     |                  | Agent    | Search and switch models      |
| `/model`      |                  | Agent    | Set model by ID               |
| `/agent`      |                  | Agent    | Toggle build/plan agent       |
| `/sidebar`    |                  | View     | Toggle sidebar                |

## Development

```bash
go test ./... -v -race    # 670+ tests with race detection
go vet ./...              # Static analysis
```

## License

MIT
