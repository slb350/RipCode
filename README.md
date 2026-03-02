# ripcode (archived)

> **This project is no longer maintained.** It was a learning exercise for building terminal UIs with [Bubble Tea v2](https://charm.land/bubbletea/v2) and [Lip Gloss v2](https://charm.land/lipgloss/v2). For a production-quality agentic coding assistant in the terminal, use [Crush](https://github.com/charmbracelet/crush) by Charmbracelet — built by the same team that makes Bubble Tea, with active development and community support.

---

An agentic AI coding assistant for the terminal, built in Go with Bubble Tea v2.

## What was built

Over 6 development phases and 1200+ tests, ripcode implemented:

- **Agentic loop** — prompt → stream → tool calls → execute → repeat
- **8 built-in tools** — bash, read, write, edit, glob, grep, ls, todo
- **Full TUI** — two-screen state machine, command palette, 15 dialogs, sidebar, toast notifications
- **Session management** — persistence, undo/redo, timeline, fork, export
- **Streaming** — real-time SSE with reasoning/thinking display, part-based messages
- **Provider-agnostic** — OpenRouter gateway with 400+ models, fuzzy model search, thinking budget variants
- **Rich input** — readline keybindings, prompt history/stash, shell mode, `@file` autocomplete with frecency ranking
- **26 slash commands** across 4 categories with inline autocomplete and keybind hints
- **Message actions** — click-to-copy/revert/fork with chat-to-session index mapping
- **Diff rendering** — before/after capture in edit/write tools, unified diff display in chat
- **Security hardening** — symlink rejection, session ID sanitization, bash blocklist, atomic writes

## What was learned

The patterns and lessons from this project are captured in a reusable workflow document:
[Bubble Tea v2 TUI Development Workflow](https://github.com/slb350/ripcode/blob/main/docs/bubbletea-tui-workflow.md) (also at `~/Dev/workflows/bubbletea-tui-workflow.md`)

Key takeaways:
- Value receivers on the main App model with Bubble Tea's functional update loop
- Dialog pattern: state struct + key handler + renderer, one file per dialog
- Component isolation: components return `string`, only the top-level App returns `tea.View`
- Test everything via simulated key/mouse messages — 1200+ tests with race detection
- `WindowSizeMsg` must arrive after state is set, or layout computes wrong dimensions

## Tech Stack

| Component   | Choice                                                    |
| ----------- | --------------------------------------------------------- |
| Language    | Go 1.25                                                   |
| TUI         | [Bubble Tea v2](https://charm.land/bubbletea/v2)         |
| Styling     | [Lip Gloss v2](https://charm.land/lipgloss/v2)           |
| Testing     | [testify](https://github.com/stretchr/testify)           |
| LLM Gateway | [OpenRouter](https://openrouter.ai)                      |

## License

MIT
