---
name: session-viewer
description: Parses and displays Claude Code session JSONL files as readable transcripts, structured JSON, compact views, error reports, or file operation summaries. Use when the user wants to view, inspect, analyze, debug, or compare Claude Code sessions by ID or file path. Also lists available sessions.
---

# Session Viewer

Parses Claude Code session JSONL files from `~/.claude/projects/` and renders them in agent-optimized formats.

## Usage

```bash
python session-viewer/claude_session.py <session-id> [flags]
python session-viewer/claude_session.py --list [project-filter]
```

## Modes

Choose the most token-efficient mode for the task:

| Flag | Output | When to use |
|------|--------|-------------|
| `--json` | Structured JSON with all metadata | **Agent default** — parse selectively |
| `--compact` | One line per event | Quick scan of session flow |
| `--summary` | Turns, tokens, tool counts, file ops | Overview before deeper analysis |
| `--errors` | Failed tool calls only | Debugging failures |
| `--files` | File operations (Read/Write/Edit) | Understanding what changed |
| `--tools-only` | Tool calls and results | Reviewing tool usage patterns |
| *(none)* | Full transcript | Complete end-to-end review |

## Flags

Combinable with any mode:

| Flag | Effect |
|------|--------|
| `--redact` | Strip secrets (OAuth tokens, passwords, API keys, bearer tokens, credential URLs) |
| `--thinking` | Include thinking blocks |
| `--no-results` | Omit tool results (calls only) |
| `--expand` | Resolve persisted tool results (large outputs saved to `<session-id>/tool-results/`) |
| `--subagents` | Include subagent sessions (spawned Agent calls with their own JSONL) |

## Workflow

1. **Start with `--json` or `--summary`** to understand scope
2. **Drill into `--errors`** if failures occurred
3. **Use `--files`** to see what was modified
4. **Use `--compact`** to trace the full conversation flow
5. **Fall back to full transcript** only for specific sections

## Session JSONL Schema

See [SCHEMA.md](SCHEMA.md) for the Claude Code session file format reference.
