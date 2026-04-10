# Claude Code Session JSONL Schema

Each line is a JSON object with a `type` field. The file is append-only.

## Common Fields

Every entry includes these session-level fields:

| Field | Type | Description |
|-------|------|-------------|
| `uuid` | UUID | Unique entry identifier |
| `parentUuid` | UUID? | Chain reference to previous entry |
| `isSidechain` | bool | Whether part of a subagent conversation |
| `timestamp` | ISO 8601 | When the entry was written |
| `sessionId` | string | Session UUID |
| `cwd` | string | Working directory |
| `version` | string | Claude Code version |
| `gitBranch` | string | Active git branch |
| `slug` | string | Human-readable session slug |
| `entrypoint` | string | `cli`, `ide`, `sdk-ts`, `sdk-py` |
| `userType` | string | `external` or `internal` |

## Entry Types

### `user` / `assistant`

Conversation messages.

| Field | Type | Description |
|-------|------|-------------|
| `message.content` | string or block[] | Text or content blocks |
| `message.role` | string | `user` or `assistant` |
| `message.usage` | object? | Token usage (assistant only, see below) |
| `permissionMode` | string? | e.g. `bypassPermissions` (first user msg) |
| `agentId` | string? | Set on subagent messages |
| `teamName` | string? | Team name for spawned agents |

#### `message.usage`

| Field | Type | Description |
|-------|------|-------------|
| `input_tokens` | int | Input tokens consumed |
| `output_tokens` | int | Output tokens generated |
| `cache_read_input_tokens` | int | Tokens read from cache |
| `cache_creation_input_tokens` | int | Tokens written to cache |
| `cache_creation.ephemeral_5m_input_tokens` | int | 5-minute cache tokens |
| `cache_creation.ephemeral_1h_input_tokens` | int | 1-hour cache tokens |
| `server_tool_use.web_search_requests` | int | Web search count |
| `server_tool_use.web_fetch_requests` | int | Web fetch count |
| `service_tier` | string? | e.g. `standard` |
| `speed` | string? | e.g. `standard` |

### `system`

System events. Identified by `subtype`:

| Subtype | Key Fields | Description |
|---------|------------|-------------|
| `turn_duration` | `durationMs`, `budgetTokens`, `budgetLimit`, `budgetNudges`, `messageCount` | Turn completion stats |
| `informational` | `text` | User-facing info (budget warnings) |
| `api_error` | | API error details |
| `api_metrics` | | API performance |
| `compact_boundary` | | Context compaction marker |
| `local_command` | | Hook command output |
| `memory_saved` | | Auto-memory save |

### Metadata Entries (last-wins)

Appended at end of session or on change:

| Type | Key Fields | Description |
|------|------------|-------------|
| `last-prompt` | `lastPrompt`, `sessionId` | Session bookmark |
| `ai-title` | `title` | AI-generated session title |
| `custom-title` | `title` | User-set session title |
| `task-summary` | `summary` | Periodic fork-generated summary |
| `pr-link` | `prUrl`, `prNumber`, `prRepository` | GitHub/GitLab PR link |
| `mode` | `mode` | `coordinator` or `normal` |
| `tag` | `tag` | Session tag |

### Skipped Entries

These are stored but not relevant for session viewing:

| Type | Purpose |
|------|---------|
| `file-history-snapshot` | File undo history |
| `attribution-snapshot` | Character contribution tracking |
| `content-replacement` | Prompt cache stability (tool result re-application) |
| `marble-origami-commit` | Context collapse commit |
| `marble-origami-snapshot` | Context collapse queue state |
| `speculation-accept` | Thinking speculation stats |
| `worktree-state` | Worktree session state |
| `agent-name`, `agent-color`, `agent-setting` | Agent appearance/config |

## Content Block Types

Inside `message.content` arrays:

| type | Fields | Description |
|------|--------|-------------|
| `text` | `text` | Plain text |
| `thinking` | `thinking`, `signature` | Extended thinking |
| `tool_use` | `id`, `name`, `input` | Tool invocation |
| `tool_result` | `tool_use_id`, `content`, `is_error` | Tool response |

### Common Tool Names

`Read`, `Write`, `Edit`, `Bash`, `Glob`, `Grep`, `LSP`, `Agent`, `ToolSearch`, `Skill`, `WebFetch`, `WebSearch`, `NotebookEdit`, `TaskCreate`, `TaskUpdate`, `AskUserQuestion`

### Persisted Tool Results

When output exceeds threshold (~50KB default), full content saved to disk:

```
<persisted-output>
Output too large (88.7KB). Full output saved to: <path>
Preview (first 2000 bytes):
...
</persisted-output>
```

Files stored at: `<session-id>/tool-results/<tool-use-id>.txt` (strings) or `.json` (arrays).

Use `--expand` to resolve these inline.

## Companion Directory

```
~/.claude/projects/<project>/
├── <session-id>.jsonl
└── <session-id>/
    ├── tool-results/
    │   ├── <tool-use-id>.txt
    │   └── <tool-use-id>.json
    ├── subagents/
    │   ├── agent-<id>.jsonl
    │   ├── agent-<id>.meta.json    # {agentType, description, worktreePath?}
    │   └── <subdir>/               # Nested team/workflow agents
    │       ├── agent-<id>.jsonl
    │       └── agent-<id>.meta.json
    └── remote-agents/
        └── remote-agent-<taskId>.meta.json  # {taskId, remoteTaskType, title, command, ...}
```

Subagent JSONL files follow the same schema as the main session.
