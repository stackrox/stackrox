#!/usr/bin/env python3
"""Inspect and update review comment triage status in the SQLite corpus."""

from __future__ import annotations

import argparse
import sqlite3
import sys
from pathlib import Path

from corpus_db import TRIAGE_STATUSES, connect, db_path_for_cache_dir, get_repo_id, utc_now


def summarize(conn: sqlite3.Connection, repo: str) -> None:
    repo_id = get_repo_id(conn, repo)
    rows = conn.execute(
        """
        SELECT triage_status, COUNT(*) AS count
        FROM review_comments
        WHERE repo_id = ?
        GROUP BY triage_status
        ORDER BY triage_status
        """,
        (repo_id,),
    ).fetchall()
    for row in rows:
        print(f"{row['triage_status']}: {row['count']}")


def list_comments(conn: sqlite3.Connection, repo: str, status: str, limit: int) -> None:
    repo_id = get_repo_id(conn, repo)
    rows = conn.execute(
        """
        SELECT rc.comment_id, rc.triage_status, rc.author, rc.normalized_body, rc.body,
               rt.path, rt.line, pr.number AS pr_number, pr.url AS pr_url
        FROM review_comments rc
        JOIN review_threads rt ON rt.repo_id = rc.repo_id AND rt.thread_id = rc.thread_id
        JOIN pull_requests pr ON pr.repo_id = rc.repo_id AND pr.number = rc.pr_number
        WHERE rc.repo_id = ? AND rc.is_initial = 1 AND rc.triage_status = ?
        ORDER BY pr.number, CAST(rc.comment_id AS INTEGER), rc.comment_id
        LIMIT ?
        """,
        (repo_id, status, limit),
    ).fetchall()
    for row in rows:
        body = (row["normalized_body"] or row["body"] or "").replace("\n", " ")
        if len(body) > 160:
            body = body[:157] + "..."
        print(
            f"{row['comment_id']} | {row['triage_status']} | PR #{row['pr_number']} | "
            f"{row['path']}:{row['line']} | {row['author']} | {body}"
        )


def mark_comments(conn: sqlite3.Connection, repo: str, comment_ids: list[str], status: str, reason: str | None) -> None:
    repo_id = get_repo_id(conn, repo)
    updated = 0
    for comment_id in comment_ids:
        cur = conn.execute(
            """
            UPDATE review_comments
            SET triage_status = ?, triage_reason = ?, triage_updated_at = ?, updated_at = ?
            WHERE repo_id = ? AND comment_id = ?
            """,
            (status, reason, utc_now(), utc_now(), repo_id, comment_id),
        )
        updated += cur.rowcount
    conn.commit()
    print(f"Updated {updated} comment(s) to {status}")


def main() -> None:
    parser = argparse.ArgumentParser(description="Inspect or update corpus comment triage status")
    parser.add_argument("cache_dir", type=Path, help="Cache directory containing corpus.sqlite3")
    parser.add_argument("--repo", default="stackrox/stackrox", help="GitHub repository")

    subparsers = parser.add_subparsers(dest="command", required=True)

    subparsers.add_parser("summary", help="Show comment counts by triage status")

    list_parser = subparsers.add_parser("list", help="List initial comments by triage status")
    list_parser.add_argument("--status", choices=TRIAGE_STATUSES, default="undecided")
    list_parser.add_argument("--limit", type=int, default=50)

    mark_parser = subparsers.add_parser("mark", help="Set triage status for one or more comment IDs")
    mark_parser.add_argument("status", choices=TRIAGE_STATUSES)
    mark_parser.add_argument("comment_ids", nargs="+")
    mark_parser.add_argument("--reason", help="Optional reason/provenance for the status change")

    args = parser.parse_args()
    db_file = db_path_for_cache_dir(args.cache_dir)
    if not db_file.exists():
        print(f"Error: SQLite corpus not found: {db_file}", file=sys.stderr)
        sys.exit(1)

    conn = connect(db_file)
    if args.command == "summary":
        summarize(conn, args.repo)
    elif args.command == "list":
        list_comments(conn, args.repo, args.status, args.limit)
    elif args.command == "mark":
        mark_comments(conn, args.repo, args.comment_ids, args.status, args.reason)


if __name__ == "__main__":
    main()
