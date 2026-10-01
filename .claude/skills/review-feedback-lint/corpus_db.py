#!/usr/bin/env python3
"""SQLite persistence helpers for review-feedback lint corpus scripts."""

from __future__ import annotations

import json
import sqlite3
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, Iterable, List, Optional, Sequence

TRIAGE_STATUSES = ("undecided", "considered", "rejected", "incorporated")


def utc_now() -> str:
    return datetime.now(timezone.utc).isoformat()


def db_path_for_cache_dir(cache_dir: Path) -> Path:
    """Return the shared SQLite corpus DB for a dated cache directory.

    The date range remains part of collection/filter run metadata and the dated
    directory still holds compatibility exports. Keeping one DB at the cache root
    lets triage state survive expanded/overlapping date ranges.
    """
    return cache_dir.parent / "corpus.sqlite3"


def infer_dates_from_cache_dir(cache_dir: Path) -> tuple[str, str]:
    try:
        since, until = cache_dir.name.split("_to_", 1)
    except ValueError as exc:
        raise ValueError(
            f"Cannot infer date range from cache dir {cache_dir}; expected YYYY-MM-DD_to_YYYY-MM-DD"
        ) from exc
    return since, until


def connect(db_path: Path) -> sqlite3.Connection:
    db_path.parent.mkdir(parents=True, exist_ok=True)
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA foreign_keys = ON")
    conn.execute("PRAGMA journal_mode = WAL")
    init_schema(conn)
    return conn


def init_schema(conn: sqlite3.Connection) -> None:
    conn.executescript(
        """
        CREATE TABLE IF NOT EXISTS repositories (
            id INTEGER PRIMARY KEY,
            full_name TEXT NOT NULL UNIQUE
        );

        CREATE TABLE IF NOT EXISTS collection_runs (
            id INTEGER PRIMARY KEY,
            repo_id INTEGER NOT NULL REFERENCES repositories(id),
            since_date TEXT NOT NULL,
            until_date TEXT NOT NULL,
            scope TEXT NOT NULL CHECK (scope IN ('ui', 'go', 'all')),
            started_at TEXT NOT NULL,
            completed_at TEXT,
            pr_count INTEGER NOT NULL DEFAULT 0,
            thread_count INTEGER NOT NULL DEFAULT 0,
            UNIQUE(repo_id, since_date, until_date, scope, started_at)
        );

        CREATE TABLE IF NOT EXISTS pull_requests (
            repo_id INTEGER NOT NULL REFERENCES repositories(id),
            number INTEGER NOT NULL,
            title TEXT NOT NULL,
            url TEXT NOT NULL,
            author TEXT NOT NULL,
            merged_at TEXT NOT NULL,
            base_sha TEXT,
            merge_sha TEXT,
            updated_at TEXT NOT NULL,
            PRIMARY KEY (repo_id, number)
        );

        CREATE TABLE IF NOT EXISTS collection_run_prs (
            run_id INTEGER NOT NULL REFERENCES collection_runs(id) ON DELETE CASCADE,
            repo_id INTEGER NOT NULL,
            pr_number INTEGER NOT NULL,
            PRIMARY KEY (run_id, repo_id, pr_number),
            FOREIGN KEY (repo_id, pr_number) REFERENCES pull_requests(repo_id, number) ON DELETE CASCADE
        );

        CREATE TABLE IF NOT EXISTS pr_files (
            repo_id INTEGER NOT NULL,
            pr_number INTEGER NOT NULL,
            path TEXT NOT NULL,
            PRIMARY KEY (repo_id, pr_number, path),
            FOREIGN KEY (repo_id, pr_number) REFERENCES pull_requests(repo_id, number) ON DELETE CASCADE
        );

        CREATE TABLE IF NOT EXISTS review_threads (
            repo_id INTEGER NOT NULL,
            thread_id TEXT NOT NULL,
            pr_number INTEGER NOT NULL,
            path TEXT NOT NULL,
            line INTEGER,
            resolved INTEGER NOT NULL DEFAULT 0,
            updated_at TEXT NOT NULL,
            PRIMARY KEY (repo_id, thread_id),
            FOREIGN KEY (repo_id, pr_number) REFERENCES pull_requests(repo_id, number) ON DELETE CASCADE
        );

        CREATE TABLE IF NOT EXISTS review_comments (
            repo_id INTEGER NOT NULL,
            comment_id TEXT NOT NULL,
            thread_id TEXT NOT NULL,
            pr_number INTEGER NOT NULL,
            author TEXT NOT NULL,
            body TEXT NOT NULL,
            normalized_body TEXT,
            created_at TEXT NOT NULL,
            diff_hunk TEXT,
            is_initial INTEGER NOT NULL DEFAULT 0,
            filter_reason TEXT,
            triage_status TEXT NOT NULL DEFAULT 'undecided'
                CHECK (triage_status IN ('undecided', 'considered', 'rejected', 'incorporated')),
            triage_reason TEXT,
            triage_updated_at TEXT,
            updated_at TEXT NOT NULL,
            PRIMARY KEY (repo_id, comment_id),
            FOREIGN KEY (repo_id, thread_id) REFERENCES review_threads(repo_id, thread_id) ON DELETE CASCADE,
            FOREIGN KEY (repo_id, pr_number) REFERENCES pull_requests(repo_id, number) ON DELETE CASCADE
        );

        CREATE TABLE IF NOT EXISTS filter_runs (
            id INTEGER PRIMARY KEY,
            repo_id INTEGER NOT NULL REFERENCES repositories(id),
            since_date TEXT NOT NULL,
            until_date TEXT NOT NULL,
            scope TEXT NOT NULL CHECK (scope IN ('ui', 'go', 'all')),
            started_at TEXT NOT NULL,
            completed_at TEXT,
            total_prs INTEGER NOT NULL DEFAULT 0,
            total_threads INTEGER NOT NULL DEFAULT 0,
            filtered_threads INTEGER NOT NULL DEFAULT 0,
            output_threads INTEGER NOT NULL DEFAULT 0
        );

        CREATE TABLE IF NOT EXISTS thread_filter_results (
            filter_run_id INTEGER NOT NULL REFERENCES filter_runs(id) ON DELETE CASCADE,
            repo_id INTEGER NOT NULL,
            thread_id TEXT NOT NULL,
            included INTEGER NOT NULL,
            reason TEXT,
            code_changed_after INTEGER NOT NULL DEFAULT 0,
            compact_json TEXT,
            PRIMARY KEY (filter_run_id, repo_id, thread_id),
            FOREIGN KEY (repo_id, thread_id) REFERENCES review_threads(repo_id, thread_id) ON DELETE CASCADE
        );

        CREATE TABLE IF NOT EXISTS invariants (
            id TEXT PRIMARY KEY,
            created_at TEXT NOT NULL,
            updated_at TEXT NOT NULL,
            status TEXT NOT NULL DEFAULT 'candidate',
            data_json TEXT NOT NULL
        );

        CREATE TABLE IF NOT EXISTS invariant_thread_analyses (
            repo_id INTEGER NOT NULL REFERENCES repositories(id),
            thread_id TEXT NOT NULL,
            scope TEXT NOT NULL CHECK (scope IN ('ui', 'go', 'all')),
            input_hash TEXT NOT NULL,
            model TEXT NOT NULL,
            result_json TEXT NOT NULL,
            analyzed_at TEXT NOT NULL,
            PRIMARY KEY (repo_id, thread_id, scope, input_hash, model),
            FOREIGN KEY (repo_id, thread_id) REFERENCES review_threads(repo_id, thread_id) ON DELETE CASCADE
        );

        CREATE INDEX IF NOT EXISTS idx_pull_requests_merged_at ON pull_requests(repo_id, merged_at);
        CREATE INDEX IF NOT EXISTS idx_pr_files_path ON pr_files(path);
        CREATE INDEX IF NOT EXISTS idx_review_threads_pr ON review_threads(repo_id, pr_number);
        CREATE INDEX IF NOT EXISTS idx_review_comments_thread ON review_comments(repo_id, thread_id, created_at);
        CREATE INDEX IF NOT EXISTS idx_review_comments_triage ON review_comments(triage_status);
        CREATE INDEX IF NOT EXISTS idx_filter_runs_lookup ON filter_runs(repo_id, since_date, until_date, scope, completed_at);
        CREATE INDEX IF NOT EXISTS idx_invariant_thread_analyses_lookup ON invariant_thread_analyses(repo_id, thread_id, scope, input_hash, model);
        """
    )


def get_repo_id(conn: sqlite3.Connection, repo: str) -> int:
    row = conn.execute("SELECT id FROM repositories WHERE full_name = ?", (repo,)).fetchone()
    if row:
        return int(row["id"])
    cur = conn.execute("INSERT INTO repositories(full_name) VALUES (?)", (repo,))
    return int(cur.lastrowid)


def latest_completed_collection_run(
    conn: sqlite3.Connection, repo: str, since: str, until: str, scope: str
) -> Optional[sqlite3.Row]:
    repo_id = get_repo_id(conn, repo)
    return conn.execute(
        """
        SELECT * FROM collection_runs
        WHERE repo_id = ? AND since_date = ? AND until_date = ? AND scope = ? AND completed_at IS NOT NULL
        ORDER BY completed_at DESC
        LIMIT 1
        """,
        (repo_id, since, until, scope),
    ).fetchone()


def begin_collection_run(conn: sqlite3.Connection, repo: str, since: str, until: str, scope: str) -> int:
    repo_id = get_repo_id(conn, repo)
    cur = conn.execute(
        """
        INSERT INTO collection_runs(repo_id, since_date, until_date, scope, started_at)
        VALUES (?, ?, ?, ?, ?)
        """,
        (repo_id, since, until, scope, utc_now()),
    )
    return int(cur.lastrowid)


def complete_collection_run(conn: sqlite3.Connection, run_id: int, pr_count: int, thread_count: int) -> None:
    conn.execute(
        """
        UPDATE collection_runs
        SET completed_at = ?, pr_count = ?, thread_count = ?
        WHERE id = ?
        """,
        (utc_now(), pr_count, thread_count, run_id),
    )


def pr_exists(conn: sqlite3.Connection, repo: str, pr_number: int) -> bool:
    repo_id = get_repo_id(conn, repo)
    row = conn.execute(
        "SELECT 1 FROM pull_requests WHERE repo_id = ? AND number = ?",
        (repo_id, pr_number),
    ).fetchone()
    return row is not None


def link_pr_to_collection_run(conn: sqlite3.Connection, repo: str, pr_number: int, run_id: int) -> None:
    repo_id = get_repo_id(conn, repo)
    conn.execute(
        "INSERT OR IGNORE INTO collection_run_prs(run_id, repo_id, pr_number) VALUES (?, ?, ?)",
        (run_id, repo_id, pr_number),
    )


def load_pr_data(conn: sqlite3.Connection, repo: str, pr_number: int) -> Optional[Dict[str, Any]]:
    repo_id = get_repo_id(conn, repo)
    pr = conn.execute(
        "SELECT * FROM pull_requests WHERE repo_id = ? AND number = ?",
        (repo_id, pr_number),
    ).fetchone()
    if not pr:
        return None
    return _build_pr_data(conn, repo_id, pr)


def upsert_pr_data(conn: sqlite3.Connection, repo: str, pr_data: Dict[str, Any], run_id: int) -> None:
    repo_id = get_repo_id(conn, repo)
    pr = pr_data["pr"]
    now = utc_now()

    conn.execute(
        """
        INSERT INTO pull_requests(repo_id, number, title, url, author, merged_at, base_sha, merge_sha, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(repo_id, number) DO UPDATE SET
            title = excluded.title,
            url = excluded.url,
            author = excluded.author,
            merged_at = excluded.merged_at,
            base_sha = excluded.base_sha,
            merge_sha = excluded.merge_sha,
            updated_at = excluded.updated_at
        """,
        (
            repo_id,
            pr["number"],
            pr["title"],
            pr["url"],
            pr["author"],
            pr["merged_at"],
            pr.get("base_sha"),
            pr.get("merge_sha"),
            now,
        ),
    )
    conn.execute(
        "INSERT OR IGNORE INTO collection_run_prs(run_id, repo_id, pr_number) VALUES (?, ?, ?)",
        (run_id, repo_id, pr["number"]),
    )

    conn.execute("DELETE FROM pr_files WHERE repo_id = ? AND pr_number = ?", (repo_id, pr["number"]))
    conn.executemany(
        "INSERT OR IGNORE INTO pr_files(repo_id, pr_number, path) VALUES (?, ?, ?)",
        [(repo_id, pr["number"], path) for path in pr.get("files_changed", [])],
    )

    for thread in pr_data.get("threads", []):
        conn.execute(
            """
            INSERT INTO review_threads(repo_id, thread_id, pr_number, path, line, resolved, updated_at)
            VALUES (?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT(repo_id, thread_id) DO UPDATE SET
                pr_number = excluded.pr_number,
                path = excluded.path,
                line = excluded.line,
                resolved = excluded.resolved,
                updated_at = excluded.updated_at
            """,
            (
                repo_id,
                str(thread["id"]),
                pr["number"],
                thread.get("path", ""),
                thread.get("line"),
                1 if thread.get("resolved") else 0,
                now,
            ),
        )

        for idx, comment in enumerate(thread.get("comments", [])):
            conn.execute(
                """
                INSERT INTO review_comments(
                    repo_id, comment_id, thread_id, pr_number, author, body, created_at,
                    diff_hunk, is_initial, updated_at
                )
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                ON CONFLICT(repo_id, comment_id) DO UPDATE SET
                    thread_id = excluded.thread_id,
                    pr_number = excluded.pr_number,
                    author = excluded.author,
                    body = excluded.body,
                    created_at = excluded.created_at,
                    diff_hunk = excluded.diff_hunk,
                    is_initial = excluded.is_initial,
                    updated_at = excluded.updated_at
                """,
                (
                    repo_id,
                    str(comment["id"]),
                    str(thread["id"]),
                    pr["number"],
                    comment["author"],
                    comment["body"],
                    comment["created_at"],
                    comment.get("diff_hunk", ""),
                    1 if idx == 0 else 0,
                    now,
                ),
            )


def load_pr_data_for_run(conn: sqlite3.Connection, run_id: int) -> List[Dict[str, Any]]:
    run = conn.execute("SELECT repo_id FROM collection_runs WHERE id = ?", (run_id,)).fetchone()
    if not run:
        return []
    repo_id = int(run["repo_id"])
    prs = conn.execute(
        """
        SELECT pr.* FROM pull_requests pr
        JOIN collection_run_prs crp ON crp.repo_id = pr.repo_id AND crp.pr_number = pr.number
        WHERE crp.run_id = ?
        ORDER BY pr.number
        """,
        (run_id,),
    ).fetchall()
    return [_build_pr_data(conn, repo_id, pr) for pr in prs]


def _build_pr_data(conn: sqlite3.Connection, repo_id: int, pr: sqlite3.Row) -> Dict[str, Any]:
    files = [
        row["path"]
        for row in conn.execute(
            "SELECT path FROM pr_files WHERE repo_id = ? AND pr_number = ? ORDER BY path",
            (repo_id, pr["number"]),
        )
    ]
    threads = []
    for thread in conn.execute(
        """
        SELECT * FROM review_threads
        WHERE repo_id = ? AND pr_number = ?
        ORDER BY CAST(thread_id AS INTEGER), thread_id
        """,
        (repo_id, pr["number"]),
    ):
        comments = []
        for comment in conn.execute(
            """
            SELECT * FROM review_comments
            WHERE repo_id = ? AND thread_id = ?
            ORDER BY created_at, CAST(comment_id AS INTEGER), comment_id
            """,
            (repo_id, thread["thread_id"]),
        ):
            comments.append(
                {
                    "id": comment["comment_id"],
                    "author": comment["author"],
                    "body": comment["body"],
                    "normalized_body": comment["normalized_body"],
                    "created_at": comment["created_at"],
                    "diff_hunk": comment["diff_hunk"] or "",
                    "triage_status": comment["triage_status"],
                    "filter_reason": comment["filter_reason"],
                }
            )
        threads.append(
            {
                "id": thread["thread_id"],
                "path": thread["path"],
                "line": thread["line"],
                "resolved": bool(thread["resolved"]),
                "comments": comments,
            }
        )
    return {
        "pr": {
            "number": pr["number"],
            "title": pr["title"],
            "url": pr["url"],
            "author": pr["author"],
            "merged_at": pr["merged_at"],
            "base_sha": pr["base_sha"],
            "merge_sha": pr["merge_sha"],
            "files_changed": files,
        },
        "threads": threads,
    }


def begin_filter_run(conn: sqlite3.Connection, repo: str, since: str, until: str, scope: str) -> int:
    repo_id = get_repo_id(conn, repo)
    cur = conn.execute(
        """
        INSERT INTO filter_runs(repo_id, since_date, until_date, scope, started_at)
        VALUES (?, ?, ?, ?, ?)
        """,
        (repo_id, since, until, scope, utc_now()),
    )
    return int(cur.lastrowid)


def complete_filter_run(conn: sqlite3.Connection, run_id: int, stats: Dict[str, Any]) -> None:
    conn.execute(
        """
        UPDATE filter_runs
        SET completed_at = ?, total_prs = ?, total_threads = ?, filtered_threads = ?, output_threads = ?
        WHERE id = ?
        """,
        (
            utc_now(),
            stats.get("total_prs", 0),
            stats.get("total_threads", 0),
            stats.get("filtered_threads", 0),
            stats.get("output_threads", 0),
            run_id,
        ),
    )


def record_thread_filter_result(
    conn: sqlite3.Connection,
    filter_run_id: int,
    repo: str,
    thread_id: str,
    included: bool,
    reason: str,
    code_changed_after: bool = False,
    compact_record: Optional[Dict[str, Any]] = None,
) -> None:
    repo_id = get_repo_id(conn, repo)
    conn.execute(
        """
        INSERT INTO thread_filter_results(
            filter_run_id, repo_id, thread_id, included, reason, code_changed_after, compact_json
        )
        VALUES (?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(filter_run_id, repo_id, thread_id) DO UPDATE SET
            included = excluded.included,
            reason = excluded.reason,
            code_changed_after = excluded.code_changed_after,
            compact_json = excluded.compact_json
        """,
        (
            filter_run_id,
            repo_id,
            str(thread_id),
            1 if included else 0,
            reason or None,
            1 if code_changed_after else 0,
            json.dumps(compact_record) if compact_record is not None else None,
        ),
    )


def update_comment_filter_metadata(
    conn: sqlite3.Connection,
    repo: str,
    comment_id: str,
    normalized_body: Optional[str],
    filter_reason: Optional[str],
) -> None:
    repo_id = get_repo_id(conn, repo)
    conn.execute(
        """
        UPDATE review_comments
        SET normalized_body = ?, filter_reason = ?, updated_at = ?
        WHERE repo_id = ? AND comment_id = ?
        """,
        (normalized_body, filter_reason, utc_now(), repo_id, str(comment_id)),
    )


def latest_completed_filter_run(
    conn: sqlite3.Connection, repo: str, since: str, until: str, scope: str
) -> Optional[sqlite3.Row]:
    repo_id = get_repo_id(conn, repo)
    return conn.execute(
        """
        SELECT * FROM filter_runs
        WHERE repo_id = ? AND since_date = ? AND until_date = ? AND scope = ? AND completed_at IS NOT NULL
        ORDER BY completed_at DESC
        LIMIT 1
        """,
        (repo_id, since, until, scope),
    ).fetchone()


def cached_invariant_thread_analysis(
    conn: sqlite3.Connection,
    repo: str,
    thread_id: str,
    scope: str,
    input_hash: str,
    model: str,
) -> Optional[Dict[str, Any]]:
    repo_id = get_repo_id(conn, repo)
    row = conn.execute(
        """
        SELECT result_json FROM invariant_thread_analyses
        WHERE repo_id = ? AND thread_id = ? AND scope = ? AND input_hash = ? AND model = ?
        """,
        (repo_id, str(thread_id), scope, input_hash, model),
    ).fetchone()
    if not row:
        return None
    return json.loads(row["result_json"])


def record_invariant_thread_analysis(
    conn: sqlite3.Connection,
    repo: str,
    thread_id: str,
    scope: str,
    input_hash: str,
    model: str,
    result: Dict[str, Any],
) -> None:
    repo_id = get_repo_id(conn, repo)
    conn.execute(
        """
        INSERT INTO invariant_thread_analyses(repo_id, thread_id, scope, input_hash, model, result_json, analyzed_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(repo_id, thread_id, scope, input_hash, model) DO UPDATE SET
            result_json = excluded.result_json,
            analyzed_at = excluded.analyzed_at
        """,
        (repo_id, str(thread_id), scope, input_hash, model, json.dumps(result), utc_now()),
    )


def load_filtered_threads(
    conn: sqlite3.Connection,
    filter_run_id: int,
    statuses: Optional[Sequence[str]] = None,
) -> List[Dict[str, Any]]:
    params: List[Any] = [filter_run_id]
    status_clause = ""
    if statuses:
        invalid = set(statuses) - set(TRIAGE_STATUSES)
        if invalid:
            raise ValueError(f"Invalid triage statuses: {', '.join(sorted(invalid))}")
        placeholders = ", ".join("?" for _ in statuses)
        status_clause = f"""
            AND EXISTS (
                SELECT 1 FROM review_comments rc
                WHERE rc.repo_id = tfr.repo_id
                  AND rc.thread_id = tfr.thread_id
                  AND rc.is_initial = 1
                  AND rc.triage_status IN ({placeholders})
            )
        """
        params.extend(statuses)

    rows = conn.execute(
        f"""
        SELECT tfr.compact_json
        FROM thread_filter_results tfr
        WHERE tfr.filter_run_id = ? AND tfr.included = 1 AND tfr.compact_json IS NOT NULL
        {status_clause}
        ORDER BY tfr.rowid
        """,
        params,
    ).fetchall()
    return [json.loads(row["compact_json"]) for row in rows]


def export_raw_prs_jsonl(conn: sqlite3.Connection, run_id: int, output_file: Path) -> None:
    with output_file.open("w") as f:
        for pr_data in load_pr_data_for_run(conn, run_id):
            f.write(json.dumps(pr_data) + "\n")


def export_filtered_threads_jsonl(threads: Iterable[Dict[str, Any]], output_file: Path) -> None:
    with output_file.open("w") as f:
        for thread in threads:
            f.write(json.dumps(thread) + "\n")
