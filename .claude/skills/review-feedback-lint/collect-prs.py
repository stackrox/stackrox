#!/usr/bin/env python3
"""
GitHub PR Corpus Collector for Review Feedback Lint Analysis

Collects merged PRs from stackrox/stackrox within a date range,
filtering to only PRs that touch specified scope (ui/, Go/backend files, or all).
Collection is conservative for historical mining: PR candidates and comment pages
are fetched in bounded batches, and PRs already present in SQLite are linked into
new overlapping runs without re-fetching their review comments unless --force is used.

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
import time
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Optional

PR_SEARCH_LIMIT = 1000
DEFAULT_PR_BATCH_SIZE = 50
DEFAULT_FILE_PAGE_SIZE = 100
DEFAULT_COMMENT_PAGE_SIZE = 100
DEFAULT_API_DELAY_SECONDS = 0.25

from corpus_db import (
    begin_collection_run,
    complete_collection_run,
    connect,
    db_path_for_cache_dir,
    export_raw_prs_jsonl,
    latest_completed_collection_run,
    link_pr_to_collection_run,
    load_pr_data,
    pr_exists,
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


def fetch_merged_pr_candidates(
    since: str, until: str, repo: str, batch_size: int, api_delay_seconds: float
) -> List[Dict[str, Any]]:
    """Fetch merged PR search results for the date range in explicit batches.

    Search results identify PR numbers only. Full PR metadata is fetched per PR so
    the collector can skip PRs already present in SQLite and avoid re-fetching
    review comments that were collected in an earlier overlapping run.
    """
    print(f"Fetching merged PR candidates from {since} to {until}...", file=sys.stderr)

    search_query = f"repo:{repo} is:pr is:merged merged:{since}..{until}"
    candidates: List[Dict[str, Any]] = []
    page = 1
    total_count: Optional[int] = None

    while True:
        output = run_gh_command([
            "api",
            "--method", "GET",
            "search/issues",
            "-f", f"q={search_query}",
            "-f", "sort=updated",
            "-f", "order=desc",
            "-f", f"per_page={batch_size}",
            "-f", f"page={page}",
        ])
        payload = json.loads(output)
        if total_count is None:
            total_count = int(payload.get("total_count", 0))
            if total_count >= PR_SEARCH_LIMIT:
                print(
                    f"Error: GitHub search reports {total_count} PRs, which reaches the {PR_SEARCH_LIMIT}-result search cap. "
                    "Split the date range to avoid silently missing PRs.",
                    file=sys.stderr,
                )
                sys.exit(1)

        items = payload.get("items", [])
        if not items:
            break

        candidates.extend(items)
        print(f"  Candidate PRs: {len(candidates)}/{total_count}", file=sys.stderr)
        if len(candidates) >= total_count:
            break
        if len(candidates) >= PR_SEARCH_LIMIT:
            print(
                f"Error: reached GitHub's {PR_SEARCH_LIMIT}-result search cap. Split the date range.",
                file=sys.stderr,
            )
            sys.exit(1)

        page += 1
        sleep_between_requests(api_delay_seconds)

    print(f"Found {len(candidates)} merged PR candidates", file=sys.stderr)
    return candidates


def sleep_between_requests(seconds: float) -> None:
    if seconds > 0:
        time.sleep(seconds)


def fetch_pr_details(repo: str, pr_number: int) -> Dict[str, Any]:
    """Fetch PR metadata once. Changed files are fetched separately with pagination."""
    output = run_gh_command([
        "pr", "view", str(pr_number),
        "--repo", repo,
        "--json", "number,title,url,author,mergedAt,baseRefOid,mergeCommit",
    ])
    return json.loads(output)


def fetch_pr_files(repo: str, pr_number: int, page_size: int, api_delay_seconds: float) -> List[str]:
    """Fetch the complete changed-file list for a PR page by page."""
    api_path = f"repos/{repo}/pulls/{pr_number}/files"
    files: List[str] = []
    page = 1

    while True:
        try:
            output = run_gh_command([
                "api",
                "--method", "GET",
                api_path,
                "-f", f"per_page={page_size}",
                "-f", f"page={page}",
            ])
            page_files = json.loads(output)
        except json.JSONDecodeError as e:
            print(f"Error: Failed to parse files for PR #{pr_number}: {e}", file=sys.stderr)
            sys.exit(1)

        if not isinstance(page_files, list):
            print(
                f"Error: Expected a file list for PR #{pr_number}, got {type(page_files).__name__}",
                file=sys.stderr,
            )
            sys.exit(1)

        files.extend(file_info["filename"] for file_info in page_files)
        if len(page_files) < page_size:
            break

        page += 1
        sleep_between_requests(api_delay_seconds)

    return files


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


def fetch_pr_review_comments(
    repo: str,
    pr_number: int,
    page_size: int,
    api_delay_seconds: float,
) -> List[Dict[str, Any]]:
    """Fetch all inline review comments for a PR page by page."""
    api_path = f"repos/{repo}/pulls/{pr_number}/comments"
    comments: List[Dict[str, Any]] = []
    page = 1

    while True:
        try:
            output = run_gh_command([
                "api",
                "--method", "GET",
                api_path,
                "-f", f"per_page={page_size}",
                "-f", f"page={page}",
            ])
            page_comments = json.loads(output)
        except json.JSONDecodeError as e:
            print(f"Error: Failed to parse comments for PR #{pr_number}: {e}", file=sys.stderr)
            sys.exit(1)

        if not isinstance(page_comments, list):
            print(
                f"Error: Expected a comment list for PR #{pr_number}, got {type(page_comments).__name__}",
                file=sys.stderr,
            )
            sys.exit(1)

        comments.extend(page_comments)
        if len(page_comments) < page_size:
            break

        page += 1
        sleep_between_requests(api_delay_seconds)

    return comments


def github_login(user: Optional[Dict[str, Any]]) -> str:
    """Return a stable author login for GitHub records with nullable users."""
    return (user or {}).get("login") or "unknown"


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
                    "author": github_login(c.get("user")),
                    "body": c["body"],
                    "created_at": c["created_at"],
                    "diff_hunk": c.get("diff_hunk", "")
                }
                for c in thread_comments
            ]
        })

    return structured_threads


def collect_pr_data(
    pr: Dict[str, Any],
    repo: str,
    files_changed: List[str],
    comment_page_size: int,
    api_delay_seconds: float,
) -> Dict[str, Any]:
    """Collect all data for a single PR."""
    pr_number = pr["number"]

    # Fetch review comments
    comments = fetch_pr_review_comments(repo, pr_number, comment_page_size, api_delay_seconds)
    threads = organize_comments_into_threads(comments)

    return {
        "pr": {
            "number": pr_number,
            "title": pr["title"],
            "url": pr["url"],
            "author": github_login(pr.get("author")),
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
    parser.add_argument(
        "--pr-batch-size",
        type=int,
        default=DEFAULT_PR_BATCH_SIZE,
        help=f"Number of PR search results to request per API call (default: {DEFAULT_PR_BATCH_SIZE})"
    )
    parser.add_argument(
        "--file-page-size",
        type=int,
        default=DEFAULT_FILE_PAGE_SIZE,
        help=f"Number of changed files to request per PR file page (default: {DEFAULT_FILE_PAGE_SIZE})"
    )
    parser.add_argument(
        "--comment-page-size",
        type=int,
        default=DEFAULT_COMMENT_PAGE_SIZE,
        help=f"Number of review comments to request per PR comment page (default: {DEFAULT_COMMENT_PAGE_SIZE})"
    )
    parser.add_argument(
        "--api-delay-seconds",
        type=float,
        default=DEFAULT_API_DELAY_SECONDS,
        help=f"Delay between paginated API calls (default: {DEFAULT_API_DELAY_SECONDS})"
    )

    args = parser.parse_args()
    if args.pr_batch_size <= 0 or args.file_page_size <= 0 or args.comment_page_size <= 0:
        print("Error: --pr-batch-size, --file-page-size, and --comment-page-size must be positive", file=sys.stderr)
        sys.exit(1)
    if args.api_delay_seconds < 0:
        print("Error: --api-delay-seconds must be non-negative", file=sys.stderr)
        sys.exit(1)

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

    candidates = fetch_merged_pr_candidates(
        args.since,
        args.until,
        args.repo,
        args.pr_batch_size,
        args.api_delay_seconds,
    )

    # Collect detailed data for each PR and persist it as we go so interrupted
    # runs still retain fetched data. PRs already present in SQLite are linked to
    # this collection run without re-fetching their comments.
    print(f"Collecting scoped PR data for {len(candidates)} candidates...", file=sys.stderr)
    run_id = begin_collection_run(conn, args.repo, args.since, args.until, args.scope)
    pr_data = []
    try:
        for i, candidate in enumerate(candidates, 1):
            pr_number = int(candidate["number"])
            if i % 10 == 0:
                print(f"  Progress: {i}/{len(candidates)}", file=sys.stderr)

            if pr_exists(conn, args.repo, pr_number) and not args.force:
                existing = load_pr_data(conn, args.repo, pr_number)
                if not existing or not scope_matches_files(existing["pr"].get("files_changed", []), args.scope):
                    continue
                link_pr_to_collection_run(conn, args.repo, pr_number, run_id)
                conn.commit()
                pr_data.append(existing)
                continue

            pr = fetch_pr_details(args.repo, pr_number)
            files_changed = fetch_pr_files(
                args.repo,
                pr_number,
                args.file_page_size,
                args.api_delay_seconds,
            )
            if not scope_matches_files(files_changed, args.scope):
                conn.commit()
                sleep_between_requests(args.api_delay_seconds)
                continue

            data = collect_pr_data(
                pr,
                args.repo,
                files_changed,
                args.comment_page_size,
                args.api_delay_seconds,
            )
            upsert_pr_data(conn, args.repo, data, run_id)
            conn.commit()
            pr_data.append(data)
            sleep_between_requests(args.api_delay_seconds)

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
