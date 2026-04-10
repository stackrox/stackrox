#!/usr/bin/env python3
"""Parse Claude Code session JSONL files and render them in various formats.

Usage:
    claude_session.py <session-id> [flags]
    claude_session.py --list [project-filter]

Modes (mutually exclusive):
    (default)      Full transcript
    --summary      Overview: turns, tokens, tool counts, file ops
    --tools-only   Tool calls and results only
    --compact      One line per event
    --json         Structured JSON (most token-efficient for agents)
    --errors       Only failed tool calls
    --files        File operations only (Read, Write, Edit paths)

Flags (combinable with any mode):
    --redact       Strip potential secrets (tokens, passwords, URLs with creds)
    --thinking     Include thinking blocks in output
    --no-results   Omit tool results (show only tool calls)
    --expand       Resolve persisted tool results (large outputs saved to disk)
    --subagents    Include subagent sessions in output

Searches ~/.claude/projects/*/ for session files.
"""

import json
import re
import sys
import os
import glob
from collections import Counter
from datetime import datetime

CLAUDE_DIR = os.path.expanduser("~/.claude/projects")
TOOL_RESULT_MAX = 3000
TEXT_MAX = 5000
PERSISTED_RE = re.compile(
    r'<persisted-output>\s*Output too large \(([^)]+)\)\. '
    r'Full output saved to: ([^\n]+?)\s*'
    r'Preview \(first [^)]+\):\s*(.*?)\s*(?:</persisted-output>|\.\.\.)',
    re.DOTALL
)

# Patterns for secret redaction
REDACT_PATTERNS = [
    (re.compile(r'(oauth2:)[^\s@]+(@)'), r'\1***\2'),
    (re.compile(r'(token["\s:=]+)["\']?[A-Za-z0-9_\-.]{20,}', re.I), r'\1***'),
    (re.compile(r'(password["\s:=]+)["\']?[^\s"\']+', re.I), r'\1***'),
    (re.compile(r'(secret["\s:=]+)["\']?[^\s"\']+', re.I), r'\1***'),
    (re.compile(r'(api[_-]?key["\s:=]+)["\']?[^\s"\']+', re.I), r'\1***'),
    (re.compile(r'(Bearer\s+)[A-Za-z0-9_\-.]+', re.I), r'\1***'),
    (re.compile(r'(https?://)[^/\s]*:[^/\s]*@'), r'\1***:***@'),
]


def redact(text: str) -> str:
    for pattern, replacement in REDACT_PATTERNS:
        text = pattern.sub(replacement, text)
    return text


def resolve_persisted(text: str, session_dir: str | None) -> str:
    """Replace <persisted-output> references with full file contents."""
    if not session_dir or "<persisted-output>" not in text:
        return text
    def _replace(m):
        size_str, file_path, preview = m.group(1), m.group(2), m.group(3)
        file_path = file_path.strip()
        try:
            with open(file_path) as f:
                return f.read()
        except OSError:
            return f"[persisted: {file_path} ({size_str}, unreadable)]\n{preview}"
    return PERSISTED_RE.sub(_replace, text)


def find_session_dir(session_path: str) -> str | None:
    """Find the companion directory for a session JSONL file."""
    base = session_path.replace(".jsonl", "")
    if os.path.isdir(base):
        return base
    return None


def parse_subagents(session_dir: str) -> list:
    """Parse subagent sessions from the session directory (including nested subdirs)."""
    subagents_dir = os.path.join(session_dir, "subagents")
    if not os.path.isdir(subagents_dir):
        return []

    agents = []
    # Walk recursively to handle nested team/workflow subdirs
    for root, _dirs, files in os.walk(subagents_dir):
        for f in sorted(files):
            if not f.endswith(".meta.json"):
                continue
            agent_id = f.replace(".meta.json", "")
            meta_path = os.path.join(root, f)
            jsonl_path = os.path.join(root, f"{agent_id}.jsonl")

            try:
                with open(meta_path) as mf:
                    meta = json.load(mf)
            except (json.JSONDecodeError, OSError):
                meta = {}

            tool_count = 0
            tool_names = []
            errors = 0
            if os.path.isfile(jsonl_path):
                try:
                    sub_session = parse_session(jsonl_path)
                    sub_calls = extract_tool_calls(sub_session["messages"])
                    tool_count = len(sub_calls)
                    tool_names = [c["name"] for c in sub_calls]
                    errors = sum(1 for c in sub_calls if c["is_error"])
                except Exception:
                    pass

            agents.append({
                "id": agent_id,
                "type": meta.get("agentType", "?"),
                "description": meta.get("description", "?"),
                "worktree": meta.get("worktreePath", ""),
                "tool_count": tool_count,
                "tool_names": tool_names,
                "errors": errors,
                "jsonl_path": jsonl_path if os.path.isfile(jsonl_path) else None,
            })

    # Also check remote-agents
    remote_dir = os.path.join(session_dir, "remote-agents")
    if os.path.isdir(remote_dir):
        for f in sorted(os.listdir(remote_dir)):
            if not f.endswith(".meta.json"):
                continue
            meta_path = os.path.join(remote_dir, f)
            try:
                with open(meta_path) as mf:
                    meta = json.load(mf)
            except (json.JSONDecodeError, OSError):
                meta = {}
            agents.append({
                "id": meta.get("taskId", f.replace(".meta.json", "")),
                "type": f"remote:{meta.get('remoteTaskType', '?')}",
                "description": meta.get("title", meta.get("command", "?")),
                "worktree": "",
                "tool_count": 0,
                "tool_names": [],
                "errors": 0,
                "jsonl_path": None,
            })

    return agents


def find_session_file(session_id: str) -> str | None:
    if os.path.isfile(session_id):
        return session_id
    sid = session_id.replace(".jsonl", "")
    for project_dir in glob.glob(os.path.join(CLAUDE_DIR, "*")):
        candidate = os.path.join(project_dir, f"{sid}.jsonl")
        if os.path.isfile(candidate):
            return candidate
    for project_dir in glob.glob(os.path.join(CLAUDE_DIR, "*")):
        if not os.path.isdir(project_dir):
            continue
        for f in os.listdir(project_dir):
            if f.endswith(".jsonl") and sid in f:
                return os.path.join(project_dir, f)
    return None


def list_sessions(project_filter: str = ""):
    found = []
    for project_dir in sorted(glob.glob(os.path.join(CLAUDE_DIR, "*"))):
        if not os.path.isdir(project_dir):
            continue
        project_name = os.path.basename(project_dir)
        if project_filter and project_filter.lower() not in project_name.lower():
            continue
        for f in sorted(os.listdir(project_dir)):
            if not f.endswith(".jsonl"):
                continue
            path = os.path.join(project_dir, f)
            sid = f.replace(".jsonl", "")
            size = os.path.getsize(path)
            mtime = datetime.fromtimestamp(os.path.getmtime(path))
            label = ""
            try:
                with open(path) as fh:
                    first_msg = ""
                    for line in fh:
                        entry = json.loads(line.strip())
                        etype = entry.get("type", "")
                        # Prefer title over first message
                        if etype in ("ai-title", "custom-title"):
                            label = entry.get("title", "")[:80]
                        elif etype == "user" and not first_msg:
                            inner = entry.get("message", {})
                            content = inner.get("content", "")
                            if isinstance(content, str):
                                first_msg = content.strip()[:80]
                            elif isinstance(content, list):
                                for b in content:
                                    if isinstance(b, dict) and b.get("type") == "text":
                                        first_msg = b.get("text", "").strip()[:80]
                                        break
                    if not label:
                        label = first_msg
            except (json.JSONDecodeError, OSError):
                pass
            found.append((mtime, sid, project_name, size, label))

    found.sort(key=lambda x: x[0], reverse=True)
    print(f"{'DATE':<20} {'SIZE':>8}  {'SESSION ID':<38} {'FIRST MESSAGE'}")
    print("-" * 110)
    for mtime, sid, project, size, msg in found:
        size_str = f"{size // 1024}K" if size >= 1024 else f"{size}B"
        print(f"{mtime.strftime('%Y-%m-%d %H:%M'):<20} {size_str:>8}  {sid:<38} {msg}")


def parse_session(path: str, expand: bool = False) -> dict:
    messages = []
    metadata = {}
    turns = []
    usage_totals = {"input_tokens": 0, "output_tokens": 0,
                    "cache_read_tokens": 0, "cache_write_tokens": 0}
    session_dir = find_session_dir(path) if expand else None

    with open(path) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                msg = json.loads(line)
            except json.JSONDecodeError:
                continue

            msg_type = msg.get("type", "")

            # Skip non-conversation entries
            if msg_type in ("file-history-snapshot", "attribution-snapshot",
                            "content-replacement", "marble-origami-commit",
                            "marble-origami-snapshot", "speculation-accept",
                            "worktree-state", "agent-name", "agent-color",
                            "agent-setting", "tag"):
                continue

            if msg_type == "system":
                if msg.get("subtype") == "turn_duration":
                    turns.append({
                        "duration_ms": msg.get("durationMs", 0),
                        "timestamp": msg.get("timestamp", ""),
                        "git_branch": msg.get("gitBranch", ""),
                        "slug": msg.get("slug", ""),
                        "budget_tokens": msg.get("budgetTokens"),
                        "budget_limit": msg.get("budgetLimit"),
                    })
                    metadata["version"] = msg.get("version", "")
                    metadata["cwd"] = msg.get("cwd", "")
                    metadata["session_id"] = msg.get("sessionId", "")
                    metadata["entrypoint"] = msg.get("entrypoint", "")
                continue

            # Session metadata entries (last-wins semantics)
            if msg_type == "last-prompt":
                metadata["last_prompt"] = msg.get("lastPrompt", "")
                metadata["session_id"] = msg.get("sessionId", "")
                continue
            if msg_type == "ai-title":
                metadata["title"] = msg.get("title", "")
                continue
            if msg_type == "custom-title":
                metadata["title"] = msg.get("title", "")
                continue
            if msg_type == "task-summary":
                metadata["task_summary"] = msg.get("summary", "")
                continue
            if msg_type == "pr-link":
                metadata["pr_url"] = msg.get("prUrl", "")
                metadata["pr_number"] = msg.get("prNumber")
                continue
            if msg_type == "mode":
                metadata["mode"] = msg.get("mode", "normal")
                continue

            if msg_type in ("user", "assistant"):
                inner = msg.get("message", {})
                ts = msg.get("timestamp", "")
                role = inner.get("role", msg_type)
                content = inner.get("content", "")

                usage = inner.get("usage", {})
                if usage:
                    usage_totals["input_tokens"] += usage.get("input_tokens", 0)
                    usage_totals["output_tokens"] += usage.get("output_tokens", 0)
                    usage_totals["cache_read_tokens"] += usage.get("cache_read_input_tokens", 0)
                    usage_totals["cache_write_tokens"] += usage.get("cache_creation_input_tokens", 0)

                if msg_type == "user" and "permission_mode" not in metadata:
                    metadata["session_id"] = msg.get("sessionId", "")
                    metadata["version"] = msg.get("version", "")
                    metadata["cwd"] = msg.get("cwd", "")
                    metadata["git_branch"] = msg.get("gitBranch", "")
                    metadata["permission_mode"] = msg.get("permissionMode", "")
                    metadata["entrypoint"] = msg.get("entrypoint", "")

                # Resolve persisted tool results if --expand
                if session_dir and isinstance(content, list):
                    for block in content:
                        if isinstance(block, dict) and block.get("type") == "tool_result":
                            rc = block.get("content", "")
                            if isinstance(rc, list):
                                for sub in rc:
                                    if isinstance(sub, dict) and sub.get("type") == "text":
                                        sub["text"] = resolve_persisted(
                                            sub.get("text", ""), session_dir)
                            elif isinstance(rc, str):
                                block["content"] = resolve_persisted(rc, session_dir)

                messages.append({
                    "role": role,
                    "content": content,
                    "timestamp": ts,
                })

    metadata["turns"] = turns
    metadata["usage"] = usage_totals
    if turns:
        metadata["total_duration_ms"] = sum(t["duration_ms"] for t in turns)
        metadata["turn_count"] = len(turns)

    return {"metadata": metadata, "messages": messages}


def format_timestamp(ts: str) -> str:
    if not ts:
        return ""
    try:
        dt = datetime.fromisoformat(ts.replace("Z", "+00:00"))
        return dt.strftime("%H:%M:%S")
    except (ValueError, TypeError):
        return ts[:19]


def truncate(s: str, max_len: int) -> str:
    if len(s) <= max_len:
        return s
    return s[:max_len] + f"\n... [{len(s)} total chars]"


def format_tokens(n: int) -> str:
    if n >= 1_000_000:
        return f"{n / 1_000_000:.1f}M"
    if n >= 1_000:
        return f"{n / 1_000:.1f}K"
    return str(n)


def print_metadata(meta: dict):
    print("=" * 70)
    title = meta.get("title", "")
    if title:
        print(f"Title:   {title}")
    print(f"Session: {meta.get('session_id', 'unknown')}")
    print(f"Branch:  {meta.get('git_branch', '?')}  |  CWD: {meta.get('cwd', '?')}")
    mode = meta.get("mode", "")
    mode_str = f"  |  Mode: {mode}" if mode and mode != "normal" else ""
    print(f"Version: {meta.get('version', '?')}  |  Permissions: {meta.get('permission_mode', '?')}{mode_str}")
    tc = meta.get("turn_count", 0)
    if tc:
        total_s = meta.get("total_duration_ms", 0) / 1000
        print(f"Turns:   {tc}  |  Total duration: {total_s:.1f}s")
    usage = meta.get("usage", {})
    if any(usage.values()):
        inp = format_tokens(usage["input_tokens"])
        out = format_tokens(usage["output_tokens"])
        cr = format_tokens(usage["cache_read_tokens"])
        cw = format_tokens(usage["cache_write_tokens"])
        print(f"Tokens:  in={inp}  out={out}  cache_read={cr}  cache_write={cw}")
    pr = meta.get("pr_url", "")
    if pr:
        print(f"PR:      {pr}")
    print("=" * 70)


# -- Extractors ----------------------------------------------------------------

def extract_tool_calls(messages: list) -> list:
    pending = {}
    results = []

    for msg in messages:
        content = msg["content"]
        if not isinstance(content, list):
            continue
        for block in content:
            if not isinstance(block, dict):
                continue
            bt = block.get("type", "")
            if bt == "tool_use":
                call = {
                    "name": block.get("name", "?"),
                    "id": block.get("id", ""),
                    "input": block.get("input", {}),
                    "timestamp": msg.get("timestamp", ""),
                    "result_text": "",
                    "is_error": False,
                }
                pending[call["id"]] = call
                results.append(call)
            elif bt == "tool_result":
                tool_id = block.get("tool_use_id", "")
                result_text = _extract_result_text(block.get("content", ""))
                if tool_id in pending:
                    pending[tool_id]["result_text"] = result_text
                    pending[tool_id]["is_error"] = block.get("is_error", False)

    return results


def extract_file_ops(tool_calls: list) -> list:
    ops = []
    for call in tool_calls:
        name = call["name"]
        inp = call["input"]
        ts = format_timestamp(call["timestamp"])

        if name == "Read":
            ops.append({"op": "read", "path": inp.get("file_path", "?"), "timestamp": ts, "error": call["is_error"]})
        elif name == "Write":
            ops.append({"op": "write", "path": inp.get("file_path", "?"), "timestamp": ts, "error": call["is_error"]})
        elif name == "Edit":
            ops.append({"op": "edit", "path": inp.get("file_path", "?"), "timestamp": ts, "error": call["is_error"]})
        elif name == "Glob":
            pattern = inp.get("pattern", "?")
            ops.append({"op": "glob", "path": pattern, "timestamp": ts, "error": call["is_error"]})
        elif name == "Grep":
            pattern = inp.get("pattern", "?")
            ops.append({"op": "grep", "path": pattern, "timestamp": ts, "error": call["is_error"]})

    return ops


def extract_user_messages(messages: list) -> list:
    user_msgs = []
    for msg in messages:
        if msg["role"] != "user":
            continue
        content = msg["content"]
        if isinstance(content, str) and content.strip():
            user_msgs.append({"text": content.strip(), "timestamp": msg.get("timestamp", "")})
        elif isinstance(content, list):
            for block in content:
                if isinstance(block, dict) and block.get("type") == "text" and block.get("text", "").strip():
                    user_msgs.append({"text": block["text"].strip(), "timestamp": msg.get("timestamp", "")})
    return user_msgs


def _extract_result_text(result_content) -> str:
    if isinstance(result_content, list):
        return "".join(
            sub.get("text", "") for sub in result_content
            if isinstance(sub, dict) and sub.get("type") == "text"
        )
    if isinstance(result_content, str):
        return result_content
    return ""


def _tool_input_summary(name: str, inp: dict) -> str:
    if name == "Read":
        return inp.get("file_path", "?")
    if name == "Write":
        return inp.get("file_path", "?")
    if name == "Edit":
        return inp.get("file_path", "?")
    if name == "Glob":
        p = inp.get("pattern", "?")
        d = inp.get("path", "")
        return f"{p} in {d}" if d else p
    if name == "Grep":
        return f"/{inp.get('pattern', '?')}/ in {inp.get('path', '.')}"
    if name == "Bash":
        return inp.get("command", "?").replace("\n", " ")[:100]
    if name == "LSP":
        return f"{inp.get('command', '?')} {inp.get('file_path', '')}"
    if name == "Agent":
        return inp.get("description", inp.get("prompt", "?"))[:100]
    if name == "ToolSearch":
        return inp.get("query", "?")
    if name == "Skill":
        return inp.get("name", "?")
    s = json.dumps(inp)
    return s[:100] + "..." if len(s) > 100 else s


# -- Output modes --------------------------------------------------------------

def print_transcript(session: dict, tools_only: bool = False,
                     show_thinking: bool = False, no_results: bool = False,
                     do_redact: bool = False):
    meta = session["metadata"]
    print_metadata(meta)
    print()

    tool_calls = {}
    turn = 0

    for msg in session["messages"]:
        role = msg["role"]
        content = msg["content"]
        ts = format_timestamp(msg.get("timestamp", ""))
        ts_prefix = f"[{ts}] " if ts else ""

        if isinstance(content, str):
            if content.strip():
                if tools_only and role == "assistant":
                    continue
                text = truncate(content, TEXT_MAX)
                if do_redact:
                    text = redact(text)
                print(f"{ts_prefix}{role.upper()}: {text}")
                print()
            continue

        if not isinstance(content, list):
            continue

        for block in content:
            if not isinstance(block, dict):
                continue
            bt = block.get("type", "")

            if bt == "text":
                text = block.get("text", "")
                if text.strip():
                    if tools_only and role == "assistant":
                        continue
                    text = truncate(text, TEXT_MAX)
                    if do_redact:
                        text = redact(text)
                    print(f"{ts_prefix}{role.upper()}: {text}")
                    print()

            elif bt == "thinking" and show_thinking:
                thinking = block.get("thinking", "")
                if thinking.strip():
                    print(f"{ts_prefix}THINKING: {truncate(thinking, TEXT_MAX)}")
                    print()

            elif bt == "tool_use":
                name = block.get("name", "?")
                tool_id = block.get("id", "")
                inp = block.get("input", {})
                tool_calls[tool_id] = name
                turn += 1

                inp_str = json.dumps(inp, indent=2) if inp else "{}"
                if len(inp_str) > 1000:
                    inp_str = inp_str[:1000] + "\n  ... [truncated]"
                if do_redact:
                    inp_str = redact(inp_str)

                print(f"{ts_prefix}TOOL [{turn}]: {name}")
                print(f"  Input: {inp_str}")
                print()

            elif bt == "tool_result" and not no_results:
                tool_id = block.get("tool_use_id", "")
                tool_name = tool_calls.get(tool_id, "?")
                is_error = block.get("is_error", False)
                result_text = _extract_result_text(block.get("content", ""))
                if do_redact:
                    result_text = redact(result_text)
                status = "ERROR" if is_error else "OK"
                print(f"  Result ({tool_name}) [{status}]: {truncate(result_text, TOOL_RESULT_MAX)}")
                print()


def print_compact(session: dict, do_redact: bool = False):
    meta = session["metadata"]
    print_metadata(meta)
    print()

    turn = 0
    for msg in session["messages"]:
        role = msg["role"]
        content = msg["content"]
        ts = format_timestamp(msg.get("timestamp", ""))

        if isinstance(content, str) and content.strip():
            line = content.strip().replace("\n", " ")[:120]
            if do_redact:
                line = redact(line)
            print(f"[{ts}] {role.upper()}: {line}")
            continue

        if not isinstance(content, list):
            continue

        for block in content:
            if not isinstance(block, dict):
                continue
            bt = block.get("type", "")

            if bt == "text":
                text = block.get("text", "").strip().replace("\n", " ")[:120]
                if text:
                    if do_redact:
                        text = redact(text)
                    print(f"[{ts}] {role.upper()}: {text}")

            elif bt == "tool_use":
                turn += 1
                name = block.get("name", "?")
                summary = _tool_input_summary(name, block.get("input", {}))
                if do_redact:
                    summary = redact(summary)
                print(f"[{ts}] TOOL[{turn}] {name}: {summary}")

            elif bt == "tool_result":
                if block.get("is_error", False):
                    result_text = _extract_result_text(block.get("content", ""))[:100]
                    if do_redact:
                        result_text = redact(result_text)
                    print(f"         ERROR: {result_text}")


def print_errors(session: dict, do_redact: bool = False):
    meta = session["metadata"]
    print_metadata(meta)
    print()

    calls = extract_tool_calls(session["messages"])
    errors = [c for c in calls if c["is_error"]]

    if not errors:
        print("No errors found.")
        return

    print(f"ERRORS: {len(errors)}")
    print()
    for i, call in enumerate(errors, 1):
        ts = format_timestamp(call["timestamp"])
        result = call["result_text"][:500]
        inp_str = json.dumps(call["input"], indent=2)[:500]
        if do_redact:
            result = redact(result)
            inp_str = redact(inp_str)
        print(f"[{ts}] ERROR {i}: {call['name']}")
        print(f"  Input: {inp_str}")
        print(f"  Result: {result}")
        print()


def print_files(session: dict, do_redact: bool = False):
    meta = session["metadata"]
    print_metadata(meta)
    print()

    calls = extract_tool_calls(session["messages"])
    ops = extract_file_ops(calls)

    if not ops:
        print("No file operations found.")
        return

    by_op = {}
    for op in ops:
        by_op.setdefault(op["op"], []).append(op)

    for op_type in ["write", "edit", "read", "glob", "grep"]:
        items = by_op.get(op_type, [])
        if not items:
            continue
        print(f"{op_type.upper()} ({len(items)}):")
        seen = set()
        for item in items:
            path = item["path"]
            if do_redact:
                path = redact(path)
            err = " [ERROR]" if item["error"] else ""
            key = (path, err)
            if key in seen and op_type == "read":
                continue
            seen.add(key)
            print(f"  [{item['timestamp']}] {path}{err}")
        print()


def print_summary(session: dict, do_redact: bool = False, subagents: list | None = None):
    meta = session["metadata"]
    print_metadata(meta)
    print()

    calls = extract_tool_calls(session["messages"])
    user_msgs = extract_user_messages(session["messages"])
    file_ops = extract_file_ops(calls)
    errors = [c for c in calls if c["is_error"]]

    print("USER MESSAGES:")
    for i, m in enumerate(user_msgs, 1):
        text = m["text"]
        if do_redact:
            text = redact(text)
        print(f"  {i}. {truncate(text, 200)}")
    print()

    tool_names = [c["name"] for c in calls]
    print(f"TOOLS CALLED: {len(tool_names)}")
    for name, count in Counter(tool_names).most_common():
        print(f"  {name}: {count}x")
    if errors:
        print(f"  ERRORS: {len(errors)}")
    print()

    writes = [o for o in file_ops if o["op"] in ("write", "edit")]
    reads = [o for o in file_ops if o["op"] == "read"]
    if writes or reads:
        print("FILE OPERATIONS:")
        if writes:
            unique_writes = sorted(set(o["path"] for o in writes))
            print(f"  Modified ({len(unique_writes)}):")
            for p in unique_writes:
                if do_redact:
                    p = redact(p)
                print(f"    {p}")
        if reads:
            unique_reads = sorted(set(o["path"] for o in reads))
            print(f"  Read ({len(unique_reads)}):")
            for p in unique_reads:
                if do_redact:
                    p = redact(p)
                print(f"    {p}")
        print()

    turns = meta.get("turns", [])
    if turns:
        print(f"TURNS: {len(turns)}")
        for i, t in enumerate(turns, 1):
            secs = t["duration_ms"] / 1000
            branch = t.get("git_branch", "")
            ts = format_timestamp(t.get("timestamp", ""))
            print(f"  {i}. [{ts}] {secs:.1f}s  branch={branch}")
        print()

    assistant_text_len = 0
    for msg in session["messages"]:
        if msg["role"] != "assistant":
            continue
        content = msg["content"]
        if isinstance(content, str):
            assistant_text_len += len(content)
        elif isinstance(content, list):
            for block in content:
                if isinstance(block, dict) and block.get("type") == "text":
                    assistant_text_len += len(block.get("text", ""))
    print(f"ASSISTANT OUTPUT: {assistant_text_len} chars")
    print()

    if subagents:
        print(f"SUBAGENTS: {len(subagents)}")
        for sa in subagents:
            tools = f"{sa['tool_count']} tools"
            errs = f", {sa['errors']} errors" if sa["errors"] else ""
            print(f"  {sa['type']}: {sa['description']} ({tools}{errs})")
        print()


def print_json(session: dict, do_redact: bool = False, subagents: list | None = None):
    meta = session["metadata"]
    calls = extract_tool_calls(session["messages"])
    user_msgs = extract_user_messages(session["messages"])
    file_ops = extract_file_ops(calls)
    errors = [c for c in calls if c["is_error"]]

    output = {
        "session_id": meta.get("session_id", ""),
        "title": meta.get("title", ""),
        "cwd": meta.get("cwd", ""),
        "git_branch": meta.get("git_branch", ""),
        "version": meta.get("version", ""),
        "permission_mode": meta.get("permission_mode", ""),
        "entrypoint": meta.get("entrypoint", ""),
        "mode": meta.get("mode", "normal"),
        "pr_url": meta.get("pr_url", ""),
        "turn_count": meta.get("turn_count", 0),
        "total_duration_ms": meta.get("total_duration_ms", 0),
        "usage": meta.get("usage", {}),
        "turns": meta.get("turns", []),
        "user_messages": [m["text"][:500] for m in user_msgs],
        "tool_counts": dict(Counter(c["name"] for c in calls).most_common()),
        "tool_calls": [
            {
                "name": c["name"],
                "input_summary": _tool_input_summary(c["name"], c["input"]),
                "is_error": c["is_error"],
                "timestamp": format_timestamp(c["timestamp"]),
            }
            for c in calls
        ],
        "errors": [
            {
                "name": c["name"],
                "input": c["input"],
                "result": c["result_text"][:500],
                "timestamp": format_timestamp(c["timestamp"]),
            }
            for c in errors
        ],
        "files_modified": sorted(set(
            o["path"] for o in file_ops if o["op"] in ("write", "edit")
        )),
        "files_read": sorted(set(
            o["path"] for o in file_ops if o["op"] == "read"
        )),
        "subagents": [
            {
                "type": sa["type"],
                "description": sa["description"],
                "tool_count": sa["tool_count"],
                "errors": sa["errors"],
            }
            for sa in (subagents or [])
        ],
    }

    text = json.dumps(output, indent=2)
    if do_redact:
        text = redact(text)
    print(text)


def main():
    args = sys.argv[1:]

    if not args or args[0] in ("-h", "--help"):
        print(__doc__)
        sys.exit(0)

    if args[0] == "--list":
        project_filter = args[1] if len(args) > 1 else ""
        list_sessions(project_filter)
        sys.exit(0)

    session_id = args[0]
    flags = set(args[1:])

    path = find_session_file(session_id)
    if not path:
        print(f"Session not found: {session_id}")
        print(f"Searched in: {CLAUDE_DIR}/*/")
        sys.exit(1)

    do_redact = "--redact" in flags
    show_thinking = "--thinking" in flags
    no_results = "--no-results" in flags
    expand = "--expand" in flags
    show_subagents = "--subagents" in flags

    session = parse_session(path, expand=expand)

    subagents = None
    if show_subagents:
        session_dir = find_session_dir(path)
        if session_dir:
            subagents = parse_subagents(session_dir)

    if "--json" in flags:
        print_json(session, do_redact, subagents)
    elif "--summary" in flags:
        print_summary(session, do_redact, subagents)
    elif "--errors" in flags:
        print_errors(session, do_redact)
    elif "--files" in flags:
        print_files(session, do_redact)
    elif "--compact" in flags:
        print_compact(session, do_redact)
    else:
        print_transcript(session, tools_only="--tools-only" in flags,
                         show_thinking=show_thinking, no_results=no_results,
                         do_redact=do_redact)


if __name__ == "__main__":
    main()
