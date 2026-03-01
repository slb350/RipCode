# Ripcode ↔ Opencode UI/UX Parity Plan

## Scope

This document tracks **remaining** UI/UX gaps between `ripcode` and `opencode` TUI and defines an implementation plan to reach 1:1 parity (except logo/branding).

In scope:
- TUI layout, interaction model, keybinds, prompt/editor behavior, command/palette UX, session/sidebar/status UX.

Out of scope for this pass:
- Backend architecture changes not directly required by UI/UX parity.
- SQLite/session persistence internals (explicitly deferred; see Deferred section).
- Extensibility framework (custom commands/agents/tools/plugins directories — post-parity).

## Current Parity Snapshot (Already Done)

- Session header/footer chrome exists.
- Wide sidebar exists; narrow overlay sidebar exists; `Ctrl+B` and `/sidebar` toggle.
- `/models` interactive searchable picker with in-memory session cache.
- `/model <id>` runtime switching.
- Command palette exists (`Ctrl+P` and `Ctrl+K`).
- Basic slash handling (`/help`, `/new`, `/clear`, `/models`, `/model`, `/agent`, `/sidebar`, `/exit`).
- Inline autocomplete exists for `/` and `@` (basic).
- Agent cycle exists (`Tab`, `Shift+Tab`).

## Remaining Gap Matrix

Legend:
- Priority: `P0` critical parity blocker, `P1` high, `P2` medium, `P3` polish.

### P0 — Critical Parity Blockers

| Area | Opencode Behavior | Ripcode Current | Gap |
|---|---|---|---|
| Prompt input editing | Readline/Emacs-like shortcuts (`ctrl+a/e/u/k/w`, word move/delete, undo/redo, etc.) | Basic arrows/backspace/newline | Missing advanced editing model |
| Prompt history | Up/down recalls prior prompts; persistent across sessions (50 entries in `~/.opencode/state/prompt-history.jsonl`); mode-aware (normal vs shell); preserves file parts | Not implemented | Missing prompt history UX |
| Shell mode | `!` enters shell prompt mode with explicit mode indicator and exit semantics | Not implemented | Missing shell-mode UX |
| Slash command breadth | `/compact`, `/connect`, `/copy`, `/details`, `/editor`, `/export`, `/fork`, `/rename`, `/sessions`, `/share`, `/skills`, `/status`, `/themes`, `/thinking`, `/timeline`, `/timestamps`, `/undo`, `/redo`, `/unshare` | Minimal subset (`/help`, `/new`, `/clear`, `/models`, `/model`, `/agent`, `/sidebar`, `/exit`) | Missing most built-ins |
| Command palette richness | Categorized + suggested commands + keybind display + global command registration + hidden commands (navigation) + disabled state (conditional) + slash aliases + flat mode when filtering | Flat static list | Missing category/suggested/keybind metadata, hidden/disabled states, and breadth |
| Session undo/redo UX | Message revert markers, restore flow, diff restore semantics, prompt restoration on undo (re-populates input with original prompt text + file parts), conditional redo enable, abort-before-revert for busy sessions | Not implemented | Missing time-travel session UX |
| Session list/switch | Session picker with search, delete (confirm), rename, working status indicator, today/date grouping, time footer | Not implemented | Missing multi-session UI surface |

### P1 — High Priority

| Area | Opencode Behavior | Ripcode Current | Gap |
|---|---|---|---|
| Toast notification system | Positioned overlay for success/error/warning/info feedback with auto-dismiss; used by every command for user feedback | Not implemented | Missing toast feedback UX (silent commands without it) |
| Prompt external editor | `/editor` or `<leader>e` opens `$EDITOR` and rehydrates prompt parts | Not implemented | Missing compose-in-editor UX |
| Prompt stash | Save/restore/list draft prompts (`prompt.stash`, `prompt.stash.pop`, `prompt.stash.list`); stash picker dialog; persisted | Not implemented | Missing draft save/restore UX |
| Export transcript | `/export` with rich options dialog: thinking, tool details, assistant metadata toggles, custom filename, open-in-editor-without-saving | Not implemented | Missing transcript export UX (both command and options dialog) |
| Connect provider flow | `/connect` onboarding dialog + provider key UX + OAuth flow (auto, code methods) + popular providers prioritized | Not implemented | Missing provider onboarding parity |
| Models dialog fidelity | Favorites/recents/provider grouping/free badges/fuzzy ranking + `ctrl+f` toggle favorite + `ctrl+a` provider list within picker + `F2`/`shift+F2` cycle recent models | Basic filtered list | Missing opencode model selection ergonomics |
| Model variants | Variant display + `ctrl+t` cycle + variant-specific status + variant badge in prompt footer | Not implemented | Missing variant UX |
| Thinking visibility | Toggle reasoning blocks display (`/thinking`) with dimmed syntax rendering + `thinkingOpacity` theme variable | Not implemented | Missing reasoning visibility controls |
| Tool details toggle | `/details` toggle and per-part detail controls + generic tool output toggle | Not implemented | Missing tool detail visibility controls |
| Sidebar sections | MCP/LSP/Todo/Modified Files with expandable/collapsible groups + getting started card (dismissible) | Compact synthetic summary only | Missing real structured side-panel sections |
| Sidebar data fidelity | Live MCP/LSP/todo/diff counts and health states | Partial token/tool stats | Missing real data-backed panels |
| Footer status richness | Permissions count (`△ N`), MCP status (`⊙ N`), LSP status (`• N`), `/status` hint, welcome message for new users | Basic running/help/connect hints | Missing status instrumentation |
| Message rendering engine | Rich part-based timeline (text/tool/reasoning) + interrupted badge + queued badge + compaction separator + revert visual markers + file attachment badges (`txt`/`img`/`pdf`/`dir`) + message click actions dialog (revert, copy, fork) | Simplified role-based rendering | Missing part-level renderer parity |
| Help dialog | Structured `/help` dialog showing keybinds, commands, version info | Text output to chat | Missing proper help dialog |
| Status dialog | `/status` (`<leader>s`) shows MCP servers, LSP clients, formatters, plugins with health indicators | Not implemented | Missing system observability dialog |
| Agent list dialog | `<leader>a` opens searchable agent picker with native/description indicators | Tab/Shift+Tab cycling only | Missing full agent picker dialog |

### P2 — Medium Priority

| Area | Opencode Behavior | Ripcode Current | Gap |
|---|---|---|---|
| Timestamps + metadata toggles | `/timestamps` toggle + assistant metadata toggle (model/cost/duration) + header visibility toggle | Fixed display model | Missing view toggles |
| Copy actions | `/copy` copies transcript to clipboard; `<leader>y` copies last assistant message to clipboard; separate from `/export` file export | Not implemented | Missing clipboard UX |
| Message navigation | next/prev user message jumps (`messages_next`/`messages_previous`), page/half-page/line movement, `ctrl+g`/`Home` first message, `ctrl+alt+g`/`End` last message, `messages_last_user` jump | Basic scroll wheel only | Missing timeline navigation keybinds |
| Scrollbar controls | Optional session scrollbar toggle (`scrollbar_toggle` keybind) | Not implemented | Missing scrollbar UX parity |
| Sidebar behavior | Auto+overlay behavior + getting started card (dismissible) | Implemented but no mouse hover affordances or getting started card | Missing interaction polish |
| Input attachments | Image/file paste handling with summarization and virtual markers + image drag-and-drop + `ctrl+v` image paste | Not implemented | Missing attachment UX parity |
| `@` autocomplete fidelity | Frecency-based ranking, line-range support (`@file.ts#10-20`), MCP resources, subagent mentions, directory expansion with Tab | Basic local file contains-match | Missing rich mention UX |
| `/` autocomplete fidelity | Aliases/description-rich slash entries from all command sources + dynamic registration | Static subset | Missing dynamic slash registry parity |
| Keybind system | 80+ individually configurable keybinds via `tui.json`, leader key (`ctrl+x`) prefix system, `none` to disable, compound chords, keybind display in all dialogs | Hard-coded keys | Missing keybind config system |
| Theme system | 34 built-in themes + custom theme directory (`.opencode/themes/`) + system theme (auto-detect terminal) + dark/light mode toggle + 40+ color variables (syntax, diff, markdown colors) + live preview on hover in theme dialog | Static single theme | Missing theme UX parity |
| Session hierarchy UX | Parent/child session navigation (`up`/`down`/`left`/`right` keys), subagent session indicators, session tree traversal | Not implemented | Missing subagent navigation UX |
| Permission/question prompts | Inline permission UI with allow once/always/reject + pattern confirmation + diff view for edits (split/unified) + fullscreen toggle; question UI with multi-question tabs, single/multi-select, custom input, number keys for quick select | Not implemented | Missing control-loop intervention UX |
| Session rename | `/rename` (`ctrl+r`) opens text input dialog to rename session | Not implemented | Missing session rename UX |
| Session fork | `/fork` creates child session from a message with initial prompt | Not implemented | Missing session forking UX |
| Timeline jump | `/timeline` (`<leader>g`) opens message picker to jump to any message | Not implemented | Missing timeline jump dialog |
| MCP enable/disable dialog | Toggle MCP servers on/off with space key, status indicators | Not implemented | Missing MCP runtime control dialog |
| Diff rendering options | Split view (>120 chars) or unified, syntax highlighting, line numbers, word wrap mode, stacked option | Not implemented | Missing diff view configuration |
| Code concealment toggle | `<leader>h` / `messages_toggle_conceal` toggles code blocks in messages | Not implemented | Missing code block visibility control |

### P3 — Polish

| Area | Opencode Behavior | Ripcode Current | Gap |
|---|---|---|---|
| Mobile/narrow ergonomics | Overlay interactions mostly done | Overlay lacks pointer affordance polish and hit-target hints | Minor polish |
| Mouse interactions | Rich mouse hover/click actions in timeline/sidebar + message click to open actions dialog | Limited click handling | Missing mouse ergonomics |
| Empty/loading states | Polished loading and no-result states across dialogs | Mixed/basic text states | Missing consistency |
| Terminal title management | Updates terminal title bar with session/model info | Not implemented | Missing terminal title UX |
| Terminal suspend | `ctrl+z` suspends to shell, resume later | Not implemented | Missing suspend/resume |
| Animated spinner | Animated block spinner with agent color, toggleable animation | Static indicator | Missing animation parity |
| Scroll acceleration | Configurable smooth macOS-style scrolling in `tui.json` | Fixed scroll speed | Missing scroll config |

## Implementation Plan (Phased)

## Phase 1: Core Prompt and Command Parity (P0) ✅ COMPLETE

Goal: Make daily coding loop feel equivalent.

Deliverables:
- Advanced input keybindings (`ctrl+a/e/u/k/w`, word motions/deletes, undo/redo).
- Prompt history traversal with safe cursor behavior (persistent, mode-aware).
- Shell mode (`!`) with explicit mode badge and exit semantics.
- Expand slash command set: `/compact`, `/connect`, `/copy`, `/details`, `/editor`, `/export`, `/fork`, `/rename`, `/sessions`, `/skills`, `/status`, `/themes`, `/thinking`, `/timeline`, `/timestamps`, `/undo`, `/redo`.
- Command palette: categories, suggested section, keybind display, hidden/disabled states, slash aliases.
- Toast notification system (required for command feedback across all phases).

Acceptance criteria:
- User can execute full coding loop without leaving keyboard.
- Slash and palette surfaces expose same core actions as opencode.
- All commands provide visible feedback via toast or dialog.
- Unit tests for each command + keybind route.

## Phase 2: Session Lifecycle and Timeline UX (P0/P1) ✅ COMPLETE

Goal: Match opencode's session/time-travel usability.

Deliverables:
- Session list/resume dialog with search, delete (confirm), rename, date grouping.
- Session rename dialog (`/rename`, `ctrl+r`).
- Undo/redo UI with visible revert marker, prompt restoration on undo (re-populate input with original prompt text + file parts), conditional redo enable, abort-before-revert for busy sessions.
- Session fork dialog (`/fork`, fork from any message into child session).
- Timeline jump dialog (`/timeline`, `<leader>g`, jump to any message).
- Transcript copy (`/copy` to clipboard) and export (`/export` with options dialog: thinking, details, metadata, filename, open-in-editor).
- Message navigation commands (next/prev/page/half-page/line/home/end/last-user).
- Help dialog (structured, not text-to-chat).
- Status dialog (`/status`, `<leader>s`, MCP/LSP/formatter/plugin health).
- Prompt stash (save/restore/list draft prompts + stash picker dialog).

Acceptance criteria:
- User can safely traverse and restore session state from TUI only.
- Timeline behavior matches opencode interaction patterns.
- Every lifecycle action has toast feedback.

## Phase 3: Model/Provider UX Parity (P1) ✅ COMPLETE

Goal: Match opencode's model/provider ergonomics.

Deliverables:
- `/connect` provider onboarding dialog (OAuth flow, API key entry, popular providers).
- Model picker enhancements: favorites (`ctrl+f`), recents (`F2`/`shift+F2` cycle), provider sections, fuzzy sort, free model badges, `ctrl+a` provider list within picker.
- Variant cycle (`ctrl+t`) and variant indicator badge in prompt footer.
- Better model metadata in status/sidebar.
- Agent list dialog (`<leader>a`, searchable picker with native/description indicators).

Acceptance criteria:
- Switching model/provider is as fast and discoverable as opencode.
- Model picker supports high-volume model lists without friction.
- Agent selection available via both Tab cycling and dialog.

## Phase 4: Sidebar + Status Depth (P1/P2)

Goal: Match side-panel utility and session observability.

Deliverables:
- Data-backed MCP/LSP/Todo/Modified Files sections with expand/collapse groups and health indicators.
- MCP enable/disable dialog (toggle servers on/off at runtime).
- Getting started card (dismissible, for new users).
- Footer status refinements (permissions count, MCP count, LSP count, retries).
- Overlay polish (hints, focus cues, optional backdrop click animations).

Acceptance criteria:
- Sidebar shows operational state useful for coding decisions.
- Narrow and wide layouts feel equivalent functionally.
- MCP servers can be toggled without config file edits.

## Phase 5: Prompt Part and Rendering Fidelity (P1/P2)

Goal: Match opencode's rich timeline and prompt-part behavior.

Deliverables:
- Part-based assistant rendering (text/tool/reasoning).
- Thinking visibility toggle with dimmed syntax rendering.
- Tool details toggle and structured tool output states.
- Code block concealment toggle (`<leader>h`).
- Diff rendering: split (>120 chars) or unified view, syntax highlighting, line numbers, word wrap mode.
- Interrupted badge, queued badge, compaction separator, revert visual markers in timeline.
- File attachment badges (`txt`/`img`/`pdf`/`dir`) on user messages.
- Message click actions dialog (revert, copy, fork).
- Attachment pipeline for file/image paste with virtual markers + image drag-and-drop.
- Rich `@` autocomplete (frecency ranking, line ranges, directory expansion with Tab, MCP resources, agent mentions).
- Rich `/` autocomplete (aliases, descriptions, dynamic registration from all command sources).

Acceptance criteria:
- Message/prompt rendering behavior aligns with opencode expectations.
- Advanced prompt composition works without manual workarounds.
- Diff views are readable at all terminal widths.

## Phase 6: Keybind/Theming/Config Parity (P2)

Goal: Match customization surface.

Deliverables:
- Configurable keymap schema (`tui.json` equivalent) with 80+ individually overridable keybinds.
- Leader key (`ctrl+x`) prefix system and compound chord support.
- `none` value to disable any keybind.
- Keybind display in all dialogs.
- Theme system: 34+ built-in themes, custom theme directory support, system theme (auto-detect terminal palette), dark/light mode toggle.
- Theme dialog with live preview on hover.
- Theme color schema: 40+ variables including syntax, diff, and markdown colors.
- Visibility toggles: timestamps, header, scrollbar, tool details, generic tool output, assistant metadata, code concealment, tips.
- Terminal title management (session/model info in title bar).
- Terminal suspend (`ctrl+z`).
- Scroll acceleration configuration.
- Animated spinner (toggleable).

Acceptance criteria:
- Power users can remap and tune UI behavior without code changes.
- Theme switching is instant with preview.
- All visibility preferences persist across restarts.

## Data Prerequisites (UI/UX Parity Blockers)

The remaining parity work is blocked less by rendering and more by missing data planes.
Below is the minimum data contract needed to make ripcode's UI behave like opencode's UI.

| Data Domain | Opencode Source(s) | Required Fields / Semantics | Ripcode Current | Gap / Required Work | Blocks |
|---|---|---|---|---|---|
| Session registry | `/session`, events `session.created|updated|deleted` | `id`, `title`, `parentID`, `time`, `share`, `revert`, `permission` | Single in-memory session object | Add multi-session index + active session pointer + event-driven updates | `/sessions`, resume, child-session navigation, share/revert badges |
| Session status | `/session/status`, event `session.status` | Per-session `idle|busy|retry` with retry metadata (`attempt`, `next`, `message`) | Local boolean `streaming` only | Add per-session status store and bind footer/statusbar | Accurate footer status, retry indicators, safe interrupt UX |
| Message timeline (message info) | `/session/{id}/message`, events `message.updated|removed` | Full `Message` metadata: role, timestamps, model/provider/variant, cost/tokens, errors | Flat chat entries with role + text | Add canonical message store keyed by `sessionID` + sorted insert/update/remove | Timeline fidelity, metadata toggles, model/cost/context indicators |
| Message parts (rich rendering) | `/session/{id}/message` parts + events `message.part.updated|delta|removed` | Union parts (`text`, `reasoning`, `tool`, `file`, `snapshot`, `patch`, `retry`, etc.) and incremental deltas | No part-level store | Add part map keyed by `messageID`, delta patching, part lifecycle | Thinking/tool-details toggles, structured tool output, patch/snapshot rendering |
| Permission requests | `/permission` + events `permission.asked|replied` + `/permission/{requestID}/reply` | Pending queue by session, `id`, `permission`, `patterns`, tool context, reply action | Not implemented | Add permission request queue + inline UI + reply actions | Permission prompt parity, controlled tool execution loops |
| Question requests | `/question` + events `question.asked|replied|rejected` + `/question/{requestID}/reply|reject` | Multi-question payloads with options, custom input, multi-select | Not implemented | Add question queue + response/reject UX | Interactive agent questions and branch decisions |
| Todo state | `/session/{id}/todo`, event `todo.updated` | Ordered task list with `content`, `status`, `priority` | Only local `todo` tool outputs in chat | Add canonical todo store by session + sidebar renderer | Sidebar Todo section parity |
| Modified files / diff | `/session/{id}/diff`, event `session.diff` | File list + additions/deletions/status | Not implemented | Add session diff store + collapsible sidebar section | Sidebar Modified Files parity |
| MCP status/resources | `/mcp/status`, `/experimental/resource`, relevant MCP events | MCP server statuses (`connected`, `failed`, `needs_auth`, etc.) + resource catalog | Not implemented | Add MCP store + health counters + error states | Sidebar MCP + footer MCP parity, `@` resource mention fidelity, MCP dialog |
| LSP status | `/lsp/status`, event `lsp.updated` | Connected/error LSP clients and roots | Not implemented | Add LSP state store and event-triggered refresh | Sidebar LSP section + footer LSP count |
| Provider/model catalog | `/config/providers`, `/provider`, `/provider/auth` | Provider defaults/connected/auth methods + rich model metadata (limits, cost, status, variants, capabilities) | `/models` via OpenRouter list (`id`,`name`) + session-memory cache | Adopt richer model/provider schema from opencode SDK surface | Model picker parity, variant cycle, `/connect`, accurate context/cost UI |
| Command registry | `/command` + in-app command registration metadata | Command metadata: category, suggested, slash aliases, keybind label, enabled/hidden/disabled | Static list in `app.go` | Build dynamic command registry and slash registry | Command palette richness + slash autocomplete fidelity |
| Agent catalog | `/app/agents` + last user message metadata | Agent list (primary/subagent), permissions, defaults, color, steps, model/variant | Hard-coded `build`/`plan` | Add agent registry + active agent sync with session history | Full agent UX parity and session-aware agent state |
| Workspace/path/vcs info | `/path/get`, `/vcs/get`, `/formatter/status` | Directory/path segments, branch, formatter availability | Workdir only from local session | Add workspace metadata store | Footer/status contextual parity |
| Prompt/attachment parts | Prompt part model (`text`,`file`,`agent`,`subtask`) + paste/image pipeline | Virtual extmark-backed prompt parts, summarized paste handling, editor round-trip | Plain text input only, no part graph | Add prompt-part state model + attachment ingestion pipeline | `@`/file paste/editor parity and reliable prompt composition |
| Prompt history | Persistent JSONL file (`~/.opencode/state/prompt-history.jsonl`) | Last 50 prompts with mode (normal/shell), file parts | Not implemented | Add persistent prompt history store + mode tracking | Prompt history recall, shell mode history |
| Prompt stash | In-memory stash stack with KV persistence | Stash entries with input text + file parts | Not implemented | Add stash store + stash dialog + keybinds | Draft save/restore UX |
| Session control APIs | `/session/{id}/revert|unrevert|fork|share|unshare|shell|command` | Revert/fork/share/shell state transitions and responses | Not implemented | Integrate control actions + optimistic UI + event reconciliation | `/undo`,`/redo`,`/fork`,`/share`,`!` shell mode parity |
| Local UI preferences | Local KV in opencode | `theme`, `theme_mode`, `sidebar`, `thinking_visibility`, `tool_details_visibility`, `scrollbar_visible`, `header_visible`, `timestamps`, `diff_wrap_mode`, `animations_enabled`, `generic_tool_output_visibility`, `assistant_metadata_visibility`, `dismissed_getting_started`, `tips_hidden` | Mostly transient UI flags | Add local persisted UI preference store (15+ keys) | Persistent UI behavior parity |

### Bootstrap + Sync Requirements

To avoid partial/inconsistent UI, ripcode needs an explicit two-stage data bootstrap similar to opencode:

1. Blocking bootstrap (must resolve before session screen is considered ready)
 - providers (`/config/providers`)
 - provider list (`/provider`)
 - agents (`/app/agents`)
 - config (`/config`)
 - session list (`/session`) when continuing/resuming

2. Non-blocking hydration (populate progressively)
 - commands (`/command`)
 - LSP status (`/lsp/status`)
 - MCP status/resources (`/mcp/status`, `/experimental/resource`)
 - formatter status (`/formatter/status`)
 - session statuses (`/session/status`)
 - provider auth methods (`/provider/auth`)
 - VCS/path info (`/vcs/get`, `/path/get`)

3. Per-session hydrate on open/switch
 - `/session/{id}`
 - `/session/{id}/message?limit=...`
 - `/session/{id}/todo`
 - `/session/{id}/diff`

4. Event-driven reconciliation loop
 - subscribe to event stream and patch store incrementally for message/part/status/permission/question/todo/diff/lsp/mcp/session updates.

### Data Model Work Needed in Ripcode

Introduce a central TUI sync store (not spread across components) with session-scoped indexes:

- `sessionsByID`, `sessionOrder`, `activeSessionID`
- `messagesBySessionID`, `partsByMessageID`
- `sessionStatusByID`, `sessionDiffByID`, `todosBySessionID`
- `permissionsBySessionID`, `questionsBySessionID`
- `mcpByName`, `mcpResources`, `lspList`
- `providers`, `providerDefaults`, `providerAuth`, `agents`, `commands`
- `pathInfo`, `vcsInfo`, `formatterStatus`
- `uiPrefs` (persisted locally)
- `promptHistory` (persistent JSONL)
- `promptStash` (in-memory + KV persistence)

Without this store, UI layout parity can be approximated visually, but behavior parity will continue to drift.

### Immediate Data Prereqs For Layout-First Parity

If we continue with layout parity first, these are the minimum required feeds to avoid fake/stubbed sections:

1. `session`, `session.status`, `session.diff`, `todo`, `message`, `part`
2. `mcp.status`, `lsp.status`
3. `providers + models` (with limits/cost/status/variants, not just id/name)
4. `command` list metadata

Everything else can follow, but these four feeds are the floor for believable 1:1 layout parity.

## Deferred (Post-Parity)

- SQLite session persistence and restore internals.
- Full sharing/unsharing remote workflows (UI commands `/share`/`/unshare` should exist but can be disabled until backend is ready).
- Desktop-specific prompts and integrations.
- Extensibility framework:
  - Custom commands directory (`.opencode/command/` — user-defined slash commands with `$ARGUMENTS`, `$1`, `$2`).
  - Custom agents directory (`.opencode/agent/` — user-defined agent definitions).
  - Custom tools directory (`.opencode/tools/` — user-defined LLM tools).
  - Plugin system (`.opencode/plugin/` — event hooks and extensions).

## Cross-Cutting Requirements

- Keep behavior covered by tests:
  - command routing tests
  - keybind tests
  - layout width/overlay tests
  - model/provider dialog tests
  - toast notification tests
  - dialog lifecycle tests (open/close/escape/select)
- Add snapshot-style TUI regression tests for critical views:
  - wide session + sidebar
  - narrow session + overlay
  - model picker
  - command palette
  - permission prompt
  - export options dialog
- Preserve current in-memory model caching contract:
  - load once per session
  - refetch on app restart

## Opencode Complete Command Reference

Full slash command inventory from opencode source (for tracking completeness):

| Command | Aliases | Category | Keybind | Phase |
|---|---|---|---|---|
| `/compact` | `/summarize` | Session | `<leader>c` | 1 |
| `/connect` | — | Provider | — | 1 |
| `/copy` | — | Session | — | 1 |
| `/details` | — | Session | `tool_details` | 1 |
| `/editor` | — | Session | `<leader>e` | 1 |
| `/exit` | `/quit`, `/q` | System | `app_exit` | Done |
| `/export` | — | Session | `<leader>x` | 1 |
| `/fork` | — | Session | `session_fork` | 2 |
| `/help` | — | System | — | 2 |
| `/models` | — | Model | `<leader>m` | Done |
| `/new` | — | Session | `<leader>n` | Done |
| `/rename` | — | Session | `ctrl+r` | 2 |
| `/redo` | — | Session | `<leader>r` | 1 |
| `/sessions` | — | Session | `<leader>l` | 1 |
| `/share` | — | Session | `session_share` | Deferred |
| `/skills` | — | Session | — | 2 |
| `/status` | — | System | `<leader>s` | 2 |
| `/themes` | — | System | `<leader>t` | 1 |
| `/thinking` | `/toggle-thinking` | Session | `display_thinking` | 1 |
| `/timeline` | — | Session | `<leader>g` | 2 |
| `/timestamps` | `/toggle-timestamps` | Session | — | 1 |
| `/undo` | — | Session | `<leader>u` | 1 |
| `/unshare` | — | Session | `session_unshare` | Deferred |

## Opencode Complete Keybind Reference

Full keybind inventory from opencode config (80+ bindings):

### Application
| Keybind | Default | Phase |
|---|---|---|
| `leader` | `ctrl+x` | 6 |
| `app_exit` | `ctrl+c,ctrl+d,<leader>q` | Done |
| `terminal_suspend` | `ctrl+z` | 6 |

### Session
| Keybind | Default | Phase |
|---|---|---|
| `session_new` | `<leader>n` | Done |
| `session_list` | `<leader>l` | 1 |
| `session_timeline` | `<leader>g` | 2 |
| `session_fork` | `none` | 2 |
| `session_rename` | `ctrl+r` | 2 |
| `session_delete` | `ctrl+d` | 2 |
| `session_share` | `none` | Deferred |
| `session_unshare` | `none` | Deferred |
| `session_interrupt` | `escape` | Done |
| `session_compact` | `<leader>c` | 1 |
| `session_export` | `<leader>x` | 1 |
| `session_child_first` | `<leader>down` | 2 |
| `session_child_cycle` | `right` | 2 |
| `session_child_cycle_reverse` | `left` | 2 |
| `session_parent` | `up` | 2 |

### Messages
| Keybind | Default | Phase |
|---|---|---|
| `messages_page_up` | `pageup,ctrl+alt+b` | 2 |
| `messages_page_down` | `pagedown,ctrl+alt+f` | 2 |
| `messages_line_up` | `ctrl+alt+y` | 2 |
| `messages_line_down` | `ctrl+alt+e` | 2 |
| `messages_half_page_up` | `ctrl+alt+u` | 2 |
| `messages_half_page_down` | `ctrl+alt+d` | 2 |
| `messages_first` | `ctrl+g,home` | 2 |
| `messages_last` | `ctrl+alt+g,end` | 2 |
| `messages_next` | `none` | 2 |
| `messages_previous` | `none` | 2 |
| `messages_last_user` | `none` | 2 |
| `messages_copy` | `<leader>y` | 1 |
| `messages_undo` | `<leader>u` | 1 |
| `messages_redo` | `<leader>r` | 1 |
| `messages_toggle_conceal` | `<leader>h` | 5 |

### UI Toggles
| Keybind | Default | Phase |
|---|---|---|
| `sidebar_toggle` | `<leader>b` | Done |
| `scrollbar_toggle` | `none` | 6 |
| `theme_list` | `<leader>t` | 1 |
| `status_view` | `<leader>s` | 2 |
| `tool_details` | `none` | 1 |
| `display_thinking` | `none` | 1 |
| `tips_toggle` | `<leader>h` | 6 |
| `terminal_title_toggle` | `none` | 6 |
| `username_toggle` | `none` | 6 |

### Model/Agent
| Keybind | Default | Phase |
|---|---|---|
| `model_list` | `<leader>m` | Done |
| `model_cycle_recent` | `f2` | 3 |
| `model_cycle_recent_reverse` | `shift+f2` | 3 |
| `model_cycle_favorite` | `none` | 3 |
| `model_cycle_favorite_reverse` | `none` | 3 |
| `model_provider_list` | `ctrl+a` | 3 |
| `model_favorite_toggle` | `ctrl+f` | 3 |
| `agent_list` | `<leader>a` | 3 |
| `agent_cycle` | `tab` | Done |
| `agent_cycle_reverse` | `shift+tab` | Done |
| `variant_cycle` | `ctrl+t` | 3 |
| `command_list` | `ctrl+p` | Done |
| `editor_open` | `<leader>e` | 1 |

### Input/Textarea (40+ bindings)
| Keybind | Default | Phase |
|---|---|---|
| `input_clear` | `ctrl+c` | 1 |
| `input_paste` | `ctrl+v` | 5 |
| `input_submit` | `return` | Done |
| `input_newline` | `shift+return,ctrl+return,alt+return,ctrl+j` | Done |
| `input_move_left` | `left` | Done |
| `input_move_right` | `right` | Done |
| `input_move_up` | `up` | Done |
| `input_move_down` | `down` | Done |
| `input_move_line_start` | `home,ctrl+a` | 1 |
| `input_move_line_end` | `end,ctrl+e` | 1 |
| `input_move_word_left` | `ctrl+left,alt+b` | 1 |
| `input_move_word_right` | `ctrl+right,alt+f` | 1 |
| `input_move_start` | `ctrl+home` | 1 |
| `input_move_end` | `ctrl+end` | 1 |
| `input_delete_char_left` | `backspace` | Done |
| `input_delete_char_right` | `delete,ctrl+d` | 1 |
| `input_delete_word_left` | `ctrl+w,ctrl+backspace,alt+backspace` | 1 |
| `input_delete_word_right` | `alt+d,ctrl+delete` | 1 |
| `input_delete_to_start` | `ctrl+u` | 1 |
| `input_delete_to_end` | `ctrl+k` | 1 |
| `input_undo` | `ctrl+-,super+z` | 1 |
| `input_redo` | `ctrl+.,super+shift+z` | 1 |
| `input_select_all` | `ctrl+a` (context-dependent) | 1 |
| `history_previous` | `up` (context-dependent) | 1 |
| `history_next` | `down` (context-dependent) | 1 |
| `stash_delete` | `ctrl+d` | 2 |

## Execution Order Recommendation

1. Phase 1
2. Phase 2
3. Phase 3
4. Phase 4
5. Phase 5
6. Phase 6

Reason:
- Prompt + command ergonomics are the highest daily friction and unblock all other parity work.
- Toast notification system is foundational — needed for feedback in every subsequent phase.
- Session/time-travel UX is the next most visible behavior gap.
- Model/provider and sidebar/rendering depth can iterate without destabilizing core input flow.
- Keybind/theme config is last because it's customization of surfaces that must exist first.
