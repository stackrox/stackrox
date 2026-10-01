#!/usr/bin/env python3
"""
GitHub PR Corpus Collector for Review Feedback Lint Analysis

Collects merged PRs from stackrox/stackrox within a date range,
filtering to only PRs that touch specified scope (ui/, Go/backend files, or all).

Usage:
    ./collect-prs.py --since 2026-08-01 --until 2026-09-01 --scope ui

Output:
    ~/.cache/stackrox/review-lint/corpus.sqlite3
    ~/.cache/stackrox/review-lint/{date-range}/raw-prs.jsonl (compatibility export)
"""

import argparse
import json
import os
import subprocess
import sys
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List

PR_SEARCH_LIMIT = 1000

from corpus_db import (
    begin_collection_run,
    complete_collection_run,
    connect,
    db_path_for_cache_dir,
    export_raw_prs_jsonl,
    latest_completed_collection_run,
    upsert_pr_data,
)


def run_gh_command(args: List[str]) -> str:
    """Run a gh CLI command and return output."""
    try:
        result = subprocess.run(
            ["gh"] + args,
            capture_output=True,
            text=True,
            check=True
        )
        return result.stdout
    except subprocess.CalledProcessError as e:
        print(f"Error running gh command: {e}", file=sys.stderr)
        print(f"stderr: {e.stderr}", file=sys.stderr)
        sys.exit(1)
    except FileNotFoundError:
        print("Error: 'gh' CLI not found. Please install: https://cli.github.com/", file=sys.stderr)
        sys.exit(1)


def validate_date(date_str: str) -> str:
    """Validate date format YYYY-MM-DD."""
    try:
        datetime.strptime(date_str, "%Y-%m-%d")
        return date_str
    except ValueError:
        raise argparse.ArgumentTypeError(f"Invalid date format: {date_str}. Use YYYY-MM-DD")


def get_cache_dir(since: str, until: str) -> Path:
    """Get cache directory for this date range."""
    cache_root = Path(
        os.environ.get(
            "REVIEW_LINT_CACHE_ROOT",
            str(Path.home() / ".cache" / "stackrox" / "review-lint"),
        )
    )
    cache_dir = cache_root / f"{since}_to_{until}"
    cache_dir.mkdir(parents=True, exist_ok=True)
    return cache_dir


GO_REVIEW_FILE_EXTENSIONS = (".go", ".proto")
GO_REVIEW_FILENAMES = {"go.mod", "go.sum"}


def is_go_review_path(path: str) -> bool:
    """Return whether a path is relevant to Go/backend review mining."""
    name = Path(path).name
    return name in GO_REVIEW_FILENAMES or path.endswith(GO_REVIEW_FILE_EXTENSIONS)


def scope_matches_files(files: List[str], scope: str) -> bool:
    """Check if any file matches the requested scope."""
    if scope == "all":
        return True
    if scope == "go":
        return any(is_go_review_path(f) for f in files)

    scope_prefix = f"{scope}/"
    return any(f.startswith(scope_prefix) for f in files)


def fetch_merged_prs(since: str, until: str, repo: str) -> List[Dict[str, Any]]:
    """Fetch all merged PRs in the date range."""
    print(f"Fetching merged PRs from {since} to {until}...", file=sys.stderr)

    # Build search query
    search_query = f"repo:{repo} is:pr is:merged merged:{since}..{until}"

    # Fetch PRs with all needed fields
    gh_args = [
        "pr", "list",
        "--repo", repo,
        "--search", search_query,
        "--state", "merged",
        "--limit", str(PR_SEARCH_LIMIT),
        "--json", "number,title,url,author,mergedAt,baseRefOid,mergeCommit,files"
    ]

    output = run_gh_command(gh_args)
    prs = json.loads(output)

    print(f"Found {len(prs)} merged PRs", file=sys.stderr)
    if len(prs) >= PR_SEARCH_LIMIT:
        print(
            f"Error: GitHub returned {len(prs)} PRs, which reaches the collection limit. "
            "Split the date range to avoid silently missing PRs.",
            file=sys.stderr,
        )
        sys.exit(1)
    return prs


def flatten_paginated_json(output: str) -> List[Dict[str, Any]]:
    """Parse `gh api --paginate --slurp` output into a flat list."""
    if not output.strip():
        return []

    data = json.loads(output)
    if not isinstance(data, list):
        raise ValueError(f"Expected paginated JSON array, got {type(data).__name__}")

    # With --slurp, gh returns a list of pages. For array endpoints each page is
    # itself an array. Be tolerant of already-flat arrays for older gh versions.
    if all(isinstance(page, list) for page in data):
        return [item for page in data for item in page]
    if all(isinstance(item, dict) for item in data):
        return data

    raise ValueError("Unexpected paginated JSON shape")


def fetch_pr_review_comments(repo: str, pr_number: int) -> List[Dict[str, Any]]:
    """Fetch review comments for a PR using GitHub API."""
    # Use gh api to get review comments (inline code review threads).
    # --slurp keeps multi-page output valid JSON instead of concatenated arrays.
    api_path = f"repos/{repo}/pulls/{pr_number}/comments"

    try:
        output = run_gh_command(["api", api_path, "--paginate", "--slurp"])
        return flatten_paginated_json(output)
    except json.JSONDecodeError as e:
        print(f"Error: Failed to parse comments for PR #{pr_number}: {e}", file=sys.stderr)
        sys.exit(1)
    except ValueError as e:
        print(f"Error: Failed to fetch comments for PR #{pr_number}: {e}", file=sys.stderr)
        sys.exit(1)


def organize_comments_into_threads(comments: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    """Organize flat list of comments into threaded conversations."""
    # Group comments by thread (in_reply_to_id or original comment id)
    threads: Dict[int, List[Dict[str, Any]]] = {}

    for comment in comments:
        # Original comments have no in_reply_to_id
        thread_root_id = comment.get("in_reply_to_id") or comment["id"]

        if thread_root_id not in threads:
            threads[thread_root_id] = []
        threads[thread_root_id].append(comment)

    # Convert to structured thread format
    structured_threads = []
    for thread_id, thread_comments in threads.items():
        # Sort by created_at
        thread_comments.sort(key=lambda c: c["created_at"])

        # First comment is the root
        root = thread_comments[0]

        structured_threads.append({
            "id": str(thread_id),
            "path": root.get("path", ""),
            "line": root.get("original_line") or root.get("line"),
            "resolved": False,  # GitHub API doesn't provide resolution status in comments
            "comments": [
                {
                    "id": str(c["id"]),
                    "author": c["user"]["login"],
                    "body": c["body"],
                    "created_at": c["created_at"],
                    "diff_hunk": c.get("diff_hunk", "")
                }
                for c in thread_comments
            ]
        })

    return structured_threads


def collect_pr_data(pr: Dict[str, Any], repo: str) -> Dict[str, Any]:
    """Collect all data for a single PR."""
    pr_number = pr["number"]

    # Extract file paths
    files_changed = [f["path"] for f in pr.get("files", [])]

    # Fetch review comments
    comments = fetch_pr_review_comments(repo, pr_number)
    threads = organize_comments_into_threads(comments)

    return {
        "pr": {
            "number": pr_number,
            "title": pr["title"],
            "url": pr["url"],
            "author": pr["author"]["login"],
            "merged_at": pr["mergedAt"],
            "base_sha": pr["baseRefOid"],
            "merge_sha": pr["mergeCommit"]["oid"] if pr.get("mergeCommit") else None,
            "files_changed": files_changed
        },
        "threads": threads
    }


def main():
    parser = argparse.ArgumentParser(
        description="Collect GitHub PR review corpus for lint rule mining"
    )
    parser.add_argument(
        "--since",
        required=True,
        type=validate_date,
        help="Start date (YYYY-MM-DD)"
    )
    parser.add_argument(
        "--until",
        required=True,
        type=validate_date,
        help="End date (YYYY-MM-DD)"
    )
    parser.add_argument(
        "--scope",
        choices=["ui", "go", "all"],
        default="all",
        help="Filter to PRs touching specific scope"
    )
    parser.add_argument(
        "--repo",
        default="stackrox/stackrox",
        help="GitHub repository (default: stackrox/stackrox)"
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Force re-fetch even if cache exists"
    )

    args = parser.parse_args()

    # Get cache directory and SQLite corpus database.
    cache_dir = get_cache_dir(args.since, args.until)
    db_file = db_path_for_cache_dir(cache_dir)
    output_file = cache_dir / "raw-prs.jsonl"

    conn = connect(db_file)

    # Check if already cached. Re-fetching upserts GitHub data but preserves triage status.
    previous_run = latest_completed_collection_run(conn, args.repo, args.since, args.until, args.scope)
    if previous_run and not args.force:
        print(f"SQLite corpus already has a completed collection run at {db_file}", file=sys.stderr)
        print(f"Use --force to re-fetch from GitHub", file=sys.stderr)
        export_raw_prs_jsonl(conn, int(previous_run["id"]), output_file)
        print(f"Compatibility export refreshed at {output_file}", file=sys.stderr)
        sys.exit(0)

    # Fetch PRs
    prs = fetch_merged_prs(args.since, args.until, args.repo)

    # Filter by scope
    if args.scope != "all":
        original_count = len(prs)
        prs = [pr for pr in prs if scope_matches_files([f["path"] for f in pr.get("files", [])], args.scope)]
        print(f"Filtered to {len(prs)} PRs matching scope {args.scope} (from {original_count} total)", file=sys.stderr)

    # Collect detailed data for each PR and persist it as we go so interrupted
    # runs still retain fetched data. Comment triage status is preserved by upserts.
    print(f"Fetching review comments for {len(prs)} PRs...", file=sys.stderr)
    run_id = begin_collection_run(conn, args.repo, args.since, args.until, args.scope)
    pr_data = []
    try:
        for i, pr in enumerate(prs, 1):
            if i % 10 == 0:
                print(f"  Progress: {i}/{len(prs)}", file=sys.stderr)

            data = collect_pr_data(pr, args.repo)
            upsert_pr_data(conn, args.repo, data, run_id)
            conn.commit()
            pr_data.append(data)

        total_threads = sum(len(data["threads"]) for data in pr_data)
        complete_collection_run(conn, run_id, len(pr_data), total_threads)
        conn.commit()
    except Exception:
        conn.rollback()
        raise

    # Compatibility export for existing tools and easy inspection.
    export_raw_prs_jsonl(conn, run_id, output_file)

    # Print summary
    print(f"\n✓ Collection complete", file=sys.stderr)
    print(f"  PRs: {len(pr_data)}", file=sys.stderr)
    print(f"  Review threads: {total_threads}", file=sys.stderr)
    print(f"  SQLite corpus: {db_file}", file=sys.stderr)
    print(f"  JSONL export: {output_file}", file=sys.stderr)


if __name__ == "__main__":
    main()
