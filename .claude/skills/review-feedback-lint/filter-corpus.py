#!/usr/bin/env python3
"""
Deterministic Corpus Reduction for Review Feedback Lint Analysis

Filters collected PR review threads to remove:
- Bot comments
- Generated file paths
- Pure acknowledgements
- Non-code discussion
- Formatting-only comments (covered by prettier)
- Paths outside requested scope

Usage:
    ./filter-corpus.py ~/.cache/stackrox/review-lint/2026-08-01_to_2026-09-01

Output:
    {cache_dir}/corpus.sqlite3
    {cache_dir}/filtered-threads.jsonl (compatibility export)
"""

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any, Dict, List

from corpus_db import (
    begin_collection_run,
    begin_filter_run,
    complete_collection_run,
    complete_filter_run,
    connect,
    db_path_for_cache_dir,
    export_filtered_threads_jsonl,
    infer_dates_from_cache_dir,
    latest_completed_collection_run,
    load_pr_data_for_run,
    record_thread_filter_result,
    update_comment_filter_metadata,
    upsert_pr_data,
)


# Bot accounts to filter out
BOT_ACCOUNTS = {
    "dependabot",
    "dependabot[bot]",
    "renovate",
    "renovate[bot]",
    "github-actions",
    "github-actions[bot]",
    "stackrox-bot",
    "codecov",
    "codecov-io",
    "codecov-commenter",
}

# Generated/auto-generated file patterns to skip
GENERATED_FILE_PATTERNS = [
    r"package-lock\.json$",
    r"yarn\.lock$",
    r"pnpm-lock\.yaml$",
    r"/generated/",
    r"\.generated\.",
    r"/mocks/",
    r"\.mock\.",
    r"/node_modules/",
    r"\.min\.js$",
    r"\.min\.css$",
    r"\.bundle\.js$",
    r"\.pb\.go$",  # protobuf generated
    r"_pb2\.py$",  # Python protobuf
    r"\.pb\.h$",
    r"\.pb\.cc$",
]

# Pure acknowledgement patterns (no substance)
ACKNOWLEDGEMENT_PATTERNS = [
    r"^done\.?$",
    r"^fixed\.?$",
    r"^LGTM\.?$",
    r"^thanks\.?$",
    r"^thank you\.?$",
    r"^good catch\.?$",
    r"^nice\.?$",
    r"^ok\.?$",
    r"^okay\.?$",
    r"^agreed\.?$",
    r"^ack\.?$",
    r"^acknowledged\.?$",
    r"^sounds good\.?$",
    r"^makes sense\.?$",
    r"^👍$",
    r"^:\+1:$",
]

# Non-code discussion markers
NON_CODE_MARKERS = [
    "release notes",
    "changelog",
    "PR title",
    "PR description",
    "commit message",
    "squash and merge",
]

# Formatting-only comments (covered by existing tooling)
FORMATTING_ONLY_PATTERNS = [
    r"prettier",
    r"formatting",
    r"whitespace",
    r"indentation",
    r"trailing comma",
    r"semicolon",
    r"line length",
]


def is_bot_comment(author: str) -> bool:
    """Check if comment is from a bot."""
    return author.lower() in BOT_ACCOUNTS or "[bot]" in author.lower()


def is_generated_file(path: str) -> bool:
    """Check if file is auto-generated."""
    return any(re.search(pattern, path, re.IGNORECASE) for pattern in GENERATED_FILE_PATTERNS)


def is_pure_acknowledgement(text: str) -> bool:
    """Check if comment is just an acknowledgement with no substance."""
    text_normalized = text.strip().lower()

    # Remove common punctuation for matching
    text_normalized = re.sub(r'[!.?,;]', '', text_normalized)

    return any(re.match(pattern, text_normalized, re.IGNORECASE) for pattern in ACKNOWLEDGEMENT_PATTERNS)


def is_non_code_discussion(text: str) -> bool:
    """Check if comment is about non-code topics."""
    text_lower = text.lower()
    return any(marker in text_lower for marker in NON_CODE_MARKERS)


def is_formatting_only(text: str) -> bool:
    """Check if comment is only about formatting covered by existing tools."""
    text_lower = text.lower()

    # Must mention formatting
    has_formatting_mention = any(
        re.search(pattern, text_lower) for pattern in FORMATTING_ONLY_PATTERNS
    )

    if not has_formatting_mention:
        return False

    # And must be short (< 100 chars) without code suggestions
    if len(text) < 100 and "```" not in text:
        return True

    return False


def extract_coderabbitai_actionable_feedback(text: str) -> str:
    """
    Extract actionable feedback from coderabbitai comment, removing verbose sections.

    CodeRabbitAI comments have structure:
    - Category tags (keep)
    - <details> sections with script output (remove)
    - Actionable feedback in bold or plain text (keep)
    - HTML comments (remove)
    """
    # Remove <details>...</details> blocks (including nested content)
    text = re.sub(r'<details>.*?</details>', '', text, flags=re.DOTALL | re.IGNORECASE)

    # Remove HTML comments
    text = re.sub(r'<!--.*?-->', '', text, flags=re.DOTALL)

    # Remove extra whitespace
    text = re.sub(r'\n\s*\n\s*\n+', '\n\n', text)

    return text.strip()


GO_REVIEW_FILE_EXTENSIONS = (".go", ".proto")
GO_REVIEW_FILENAMES = {"go.mod", "go.sum"}


def is_go_review_path(path: str) -> bool:
    """Return whether a path is relevant to Go/backend review mining."""
    name = Path(path).name
    return name in GO_REVIEW_FILENAMES or path.endswith(GO_REVIEW_FILE_EXTENSIONS)


def is_path_in_scope(path: str, scope: str) -> bool:
    """Check if file path matches the requested scope."""
    if scope == "all":
        return True
    if scope == "go":
        return is_go_review_path(path)
    return path.startswith(f"{scope}/")


def normalized_comment_body(comment: Dict[str, Any]) -> str:
    """Return the body used for filtering and downstream analysis."""
    body = comment.get("body", "")
    if comment.get("author") == "coderabbitai[bot]":
        return extract_coderabbitai_actionable_feedback(body)
    return body


def should_filter_comment(comment: Dict[str, Any]) -> tuple[bool, str]:
    """
    Determine if a comment should be filtered out.
    Returns (should_filter, reason).
    """
    author = comment.get("author", "")
    body = normalized_comment_body(comment)

    # Keep CodeRabbit comments if their actionable portion survives the normal
    # filters. Other bot comments are still treated as noise.
    if is_bot_comment(author) and author != "coderabbitai[bot]":
        return True, "bot_author"

    if is_pure_acknowledgement(body):
        return True, "pure_acknowledgement"

    if is_non_code_discussion(body):
        return True, "non_code_discussion"

    if is_formatting_only(body):
        return True, "formatting_only"

    # Very short comments with no code/substance
    if len(body.strip()) < 10 and "```" not in body:
        return True, "too_short"

    return False, ""


def code_likely_changed_after_comment(thread: Dict[str, Any], pr_data: Dict[str, Any]) -> bool:
    """
    Heuristic to determine if code was likely changed after the initial review comment.

    This is a simple heuristic based on:
    - Presence of replies from the PR author
    - Timing of comments (author reply after reviewer comment)
    """
    if not thread.get("comments"):
        return False

    initial_comment = thread["comments"][0]
    pr_author = pr_data["pr"]["author"]
    reviewer = initial_comment["author"]

    # Check if author replied after the review
    for comment in thread["comments"][1:]:
        if comment["author"] == pr_author:
            # Author replied - possible acknowledgement of change
            return True

    # If there are multiple comments in the thread, likely some discussion/changes
    return len(thread["comments"]) > 2


def filter_thread(thread: Dict[str, Any], pr_data: Dict[str, Any], scope: str) -> tuple[bool, str]:
    """
    Determine if an entire thread should be filtered out.
    Returns (should_filter, reason).
    """
    path = thread.get("path", "")

    # Filter by path scope
    if not is_path_in_scope(path, scope):
        return True, "out_of_scope"

    # Filter generated files
    if is_generated_file(path):
        return True, "generated_file"

    # Filter if no comments
    if not thread.get("comments"):
        return True, "empty_thread"

    # Filter if all comments should be filtered
    comments = thread["comments"]
    all_filtered = all(should_filter_comment(c)[0] for c in comments)
    if all_filtered:
        return True, "all_comments_filtered"

    return False, ""


def create_compact_record(thread: Dict[str, Any], pr_data: Dict[str, Any]) -> Dict[str, Any]:
    """Create compact filtered thread record."""
    initial_comment = thread["comments"][0]

    initial_body = normalized_comment_body(initial_comment)

    # Get substantive replies (filter out bots and acknowledgements)
    substantive_replies = []
    for comment in thread["comments"][1:]:
        if not should_filter_comment(comment)[0]:
            reply_body = normalized_comment_body(comment)

            substantive_replies.append({
                "id": comment["id"],
                "author": comment["author"],
                "body": reply_body,
                "created_at": comment["created_at"],
                "triage_status": comment.get("triage_status", "undecided")
            })

    return {
        "pr_number": pr_data["pr"]["number"],
        "pr_url": pr_data["pr"]["url"],
        "pr_author": pr_data["pr"]["author"],
        "thread_id": thread["id"],
        "file_path": thread["path"],
        "line": thread["line"],
        "initial_comment": {
            "id": initial_comment["id"],
            "author": initial_comment["author"],
            "body": initial_body,
            "created_at": initial_comment["created_at"],
            "diff_hunk": initial_comment.get("diff_hunk", ""),
            "triage_status": initial_comment.get("triage_status", "undecided")
        },
        "replies": substantive_replies,
        "code_changed_after": code_likely_changed_after_comment(thread, pr_data),
        "resolution_status": "resolved" if thread.get("resolved") else "unknown"
    }


def filter_corpus(cache_dir: Path, scope: str, repo: str) -> None:
    """Filter raw PR corpus and produce compact records."""
    db_file = db_path_for_cache_dir(cache_dir)
    output_file = cache_dir / "filtered-threads.jsonl"
    since, until = infer_dates_from_cache_dir(cache_dir)

    conn = connect(db_file)
    collection_run = latest_completed_collection_run(conn, repo, since, until, scope)
    if not collection_run:
        legacy_input = cache_dir / "raw-prs.jsonl"
        if legacy_input.exists():
            print(f"Importing legacy JSONL corpus into SQLite: {legacy_input}", file=sys.stderr)
            import_run_id = begin_collection_run(conn, repo, since, until, scope)
            pr_count = 0
            thread_count = 0
            with legacy_input.open("r") as f:
                for line in f:
                    pr_data = json.loads(line)
                    upsert_pr_data(conn, repo, pr_data, import_run_id)
                    pr_count += 1
                    thread_count += len(pr_data.get("threads", []))
            complete_collection_run(conn, import_run_id, pr_count, thread_count)
            conn.commit()
            collection_run = latest_completed_collection_run(conn, repo, since, until, scope)

    if not collection_run:
        print(
            f"Error: No completed collection run for repo={repo} range={since}..{until} scope={scope}",
            file=sys.stderr,
        )
        sys.exit(1)

    filter_run_id = begin_filter_run(conn, repo, since, until, scope)

    # Statistics
    stats = {
        "total_prs": 0,
        "total_threads": 0,
        "filtered_threads": 0,
        "output_threads": 0,
        "filter_reasons": {},
    }

    filtered_threads = []

    # Process each PR from SQLite. Comment triage status is preserved and can be
    # used by later phases to decide whether a comment should be reconsidered.
    try:
        for pr_data in load_pr_data_for_run(conn, int(collection_run["id"])):
            stats["total_prs"] += 1

            for thread in pr_data.get("threads", []):
                stats["total_threads"] += 1

                # Store per-comment normalization/filter metadata for ad-hoc SQL
                # inspection and joins with future analysis databases.
                for comment in thread.get("comments", []):
                    comment_filtered, comment_reason = should_filter_comment(comment)
                    update_comment_filter_metadata(
                        conn,
                        repo,
                        comment["id"],
                        normalized_comment_body(comment),
                        comment_reason if comment_filtered else None,
                    )

                should_filter, reason = filter_thread(thread, pr_data, scope)
                if should_filter:
                    stats["filtered_threads"] += 1
                    stats["filter_reasons"][reason] = stats["filter_reasons"].get(reason, 0) + 1
                    record_thread_filter_result(conn, filter_run_id, repo, thread["id"], False, reason)
                    continue

                # Create compact record
                compact = create_compact_record(thread, pr_data)
                filtered_threads.append(compact)
                stats["output_threads"] += 1
                record_thread_filter_result(
                    conn,
                    filter_run_id,
                    repo,
                    thread["id"],
                    True,
                    "",
                    compact["code_changed_after"],
                    compact,
                )

        complete_filter_run(conn, filter_run_id, stats)
        conn.commit()
    except Exception:
        conn.rollback()
        raise

    # Compatibility export for existing tools and easy inspection.
    export_filtered_threads_jsonl(filtered_threads, output_file)

    # Print summary
    print(f"\n✓ Filtering complete", file=sys.stderr)
    print(f"  Input PRs: {stats['total_prs']}", file=sys.stderr)
    print(f"  Total threads: {stats['total_threads']}", file=sys.stderr)
    print(f"  Filtered out: {stats['filtered_threads']}", file=sys.stderr)
    print(f"  Output threads: {stats['output_threads']}", file=sys.stderr)
    print(f"\nFilter reasons:", file=sys.stderr)
    for reason, count in sorted(stats["filter_reasons"].items(), key=lambda x: -x[1]):
        print(f"  {reason}: {count}", file=sys.stderr)
    print(f"\nSQLite corpus: {db_file}", file=sys.stderr)
    print(f"JSONL export: {output_file}", file=sys.stderr)


def main():
    parser = argparse.ArgumentParser(
        description="Filter PR review corpus to remove noise"
    )
    parser.add_argument(
        "cache_dir",
        type=Path,
        help="Cache directory containing raw-prs.jsonl"
    )
    parser.add_argument(
        "--scope",
        choices=["ui", "go", "all"],
        default="all",
        help="Scope to filter (default: all)"
    )
    parser.add_argument(
        "--repo",
        default="stackrox/stackrox",
        help="GitHub repository (default: stackrox/stackrox)"
    )

    args = parser.parse_args()

    if not args.cache_dir.is_dir():
        print(f"Error: Cache directory not found: {args.cache_dir}", file=sys.stderr)
        sys.exit(1)

    filter_corpus(args.cache_dir, args.scope, args.repo)


if __name__ == "__main__":
    main()
