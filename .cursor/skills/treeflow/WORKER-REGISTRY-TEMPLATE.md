# Worker Registry — `registry.json`

> Managed by `tf.py`. Never edit manually. Never sent to workers.

The worker registry is a JSON file at `.beads/context-{plan-name}/registry.json`, created by `tf.py init` and updated atomically by all `tf.py` subcommands.

## Schema

```json
{
  "plan_name": "browser-filesystem",
  "workers": {
    "chrome-api-1": {
      "status": "active|idle|retired|failed",
      "skill": "chrome-api",
      "context_pct": 25,
      "bead": "tabtty-6dx",
      "notification": "pending|received|reconciled",
      "dispatched_at": "2026-04-06T14:32:00Z",
      "idle_since": "2026-04-06T14:45:00Z",
      "summary": "Created namespace fetcher..."
    }
  },
  "routing": {
    "lib/engine/commands/*.ts": {"domain": "commands", "prefix": "commands-"},
    "entrypoints/background.ts": {"domain": "chrome-api", "prefix": "chrome-api-"}
  },
  "phases": {
    "5": {
      "beads": ["tabtty-6dx", "tabtty-cng"],
      "gate_passed": false
    }
  }
}
```

## Worker Status Values

| Status | Meaning | Transition |
|--------|---------|-----------|
| `active` | Currently working on a bead | Set by `tf.py dispatch` |
| `idle` | Stopped, resumable via SendMessage | Set by `tf.py notify` |
| `retired` | Context too full (<40% remaining) | Set by `tf.py retire` |
| `failed` | Errored, needs investigation | Set manually |

## Notification Values

| Value | Meaning |
|-------|---------|
| `pending` | Worker dispatched, no completion notification yet |
| `received` | `<task-notification>` processed via `tf.py notify` |
| `reconciled` | Late notification for already-processed bead |

## Commands

```bash
# Query workers
python3 .beads/tf.py registry                    # All workers (compact)
python3 .beads/tf.py registry --status idle       # Reuse candidates
python3 .beads/tf.py registry --status active     # Currently working
python3 .beads/tf.py registry --skill chrome-api  # Filter by domain

# State transitions
python3 .beads/tf.py dispatch {worker} {bead} --skill {domain}  # → active
python3 .beads/tf.py notify {worker} {bead} --context-pct N     # → idle
python3 .beads/tf.py retire {worker}                             # → retired

# Routing
python3 .beads/tf.py routing                              # Show all routes
python3 .beads/tf.py routing --add "pattern:domain:prefix" # Add route
```

## Reuse Decision Rule

- ≥50% context + same domain → **always reuse** via SendMessage
- 40–50% context + same domain → reuse if task is simple/small
- <40% context → `tf.py retire`, spawn fresh
