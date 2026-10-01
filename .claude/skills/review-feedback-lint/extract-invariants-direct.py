#!/usr/bin/env python3
"""
Direct Invariant Extraction Helper

Reads filtered threads from the SQLite corpus and outputs them in a format for Claude to analyze.
Claude will analyze batches and produce invariant records.

Usage:
    ./extract-invariants-direct.py ~/.cache/stackrox/review-lint/2026-08-11_to_2026-09-11 --batch-size 10 --batch 1
"""

import argparse
import json
from pathlib import Path
import sys
from typing import List, Dict, Any

from corpus_db import (
    TRIAGE_STATUSES,
    connect,
    db_path_for_cache_dir,
    infer_dates_from_cache_dir,
    latest_completed_filter_run,
    load_filtered_threads,
)


def load_threads(cache_dir: Path, repo: str, scope: str, statuses: List[str]) -> List[Dict[str, Any]]:
    """Load filtered threads from SQLite."""
    db_file = db_path_for_cache_dir(cache_dir)
    if not db_file.exists():
        print(f"Error: SQLite corpus not found: {db_file}", file=sys.stderr)
        sys.exit(1)

    since, until = infer_dates_from_cache_dir(cache_dir)
    conn = connect(db_file)
    filter_run = latest_completed_filter_run(conn, repo, since, until, scope)
    if not filter_run:
        print(
            f"Error: No completed filter run for repo={repo} range={since}..{until} scope={scope}",
            file=sys.stderr,
        )
        sys.exit(1)

    return load_filtered_threads(conn, int(filter_run["id"]), statuses)


def format_thread_for_analysis(thread: Dict[str, Any]) -> str:
    """Format a single thread for analysis."""
    replies_text = ""
    if thread.get("replies"):
        replies_text = "\nReplies:\n" + "\n".join([
            f"  - {r['author']}: {r['body'][:150]}..."
            for r in thread["replies"]
        ])

    return f"""
---
PR #{thread['pr_number']} | Thread {thread['thread_id']} | {Path(thread['file_path']).name}
Reviewer: {thread['initial_comment']['author']}

{thread['initial_comment']['body'][:500]}{'...' if len(thread['initial_comment']['body']) > 500 else ''}{replies_text}
---
"""


def main():
    parser = argparse.ArgumentParser(
        description="Prepare threads for invariant extraction"
    )
    parser.add_argument(
        "cache_dir",
        type=Path,
        help="Cache directory containing filtered-threads.jsonl"
    )
    parser.add_argument(
        "--batch-size",
        type=int,
        default=10,
        help="Number of threads per batch"
    )
    parser.add_argument(
        "--batch",
        type=int,
        default=1,
        help="Which batch to output (1-indexed)"
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output raw JSON instead of formatted text"
    )
    parser.add_argument(
        "--scope",
        choices=["ui", "go", "all"],
        default="all",
        help="Filter run scope to read (default: all)"
    )
    parser.add_argument(
        "--repo",
        default="stackrox/stackrox",
        help="GitHub repository (default: stackrox/stackrox)"
    )
    parser.add_argument(
        "--status",
        action="append",
        choices=TRIAGE_STATUSES,
        default=None,
        help="Initial-comment triage status to include; repeatable. Defaults to undecided."
    )

    args = parser.parse_args()

    if not args.cache_dir.is_dir():
        print(f"Error: Cache directory not found: {args.cache_dir}", file=sys.stderr)
        sys.exit(1)

    # Load threads. By default, only undecided comments are emitted so follow-up
    # runs can focus on feedback that has not yet been accepted/rejected/incorporated.
    threads = load_threads(args.cache_dir, args.repo, args.scope, args.status or ["undecided"])

    # Calculate batch boundaries
    start_idx = (args.batch - 1) * args.batch_size
    end_idx = min(start_idx + args.batch_size, len(threads))

    batch_threads = threads[start_idx:end_idx]

    print(f"Total threads: {len(threads)}", file=sys.stderr)
    print(f"Batch {args.batch}: threads {start_idx+1}-{end_idx}", file=sys.stderr)
    print("", file=sys.stderr)

    if args.json:
        # Output raw JSON for programmatic processing
        for thread in batch_threads:
            print(json.dumps(thread))
    else:
        # Output formatted text for Claude analysis
        print(f"Batch {args.batch}/{(len(threads) + args.batch_size - 1) // args.batch_size}")
        print(f"Analyzing threads {start_idx+1}-{end_idx} of {len(threads)}")
        print("=" * 80)

        for thread in batch_threads:
            print(format_thread_for_analysis(thread))


if __name__ == "__main__":
    main()
