#!/usr/bin/env python3
"""Inspect captured DNF history with the queries used by the pinned ClairCore."""

import argparse
import json
from pathlib import Path
import sqlite3


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("database", type=Path)
    parser.add_argument("--table", action="append", default=[], help="Dump a raw table; repeat as needed")
    parser.add_argument("--package", nargs=5, metavar=("NAME", "EPOCH", "VERSION", "RELEASE", "ARCH"))
    args = parser.parse_args()

    # mode=ro preserves the captured database, including when its schema is unexpected.
    with sqlite3.connect(args.database.resolve().as_uri() + "?mode=ro", uri=True) as db:
        db.row_factory = sqlite3.Row

        def rows(query, parameters=()):
            return [dict(row) for row in db.execute(query, parameters)]

        schema = rows("SELECT name, sql FROM sqlite_schema WHERE type = 'table' AND name NOT LIKE 'sqlite_%'")
        tables = {row["name"] for row in schema}
        dnf5 = {"config", "trans_item_action"}.issubset(tables)
        if dnf5:
            latest = """
                SELECT max(i.id) AS uniq FROM trans_item AS i
                JOIN trans_item_action AS a ON (i.action_id = a.id)
                WHERE a.Name <> 'Remove' GROUP BY i.item_id
            """
            package_items = """
                SELECT item_id FROM rpm AS r
                JOIN pkg_name AS n ON (r.name_id = n.id)
                JOIN arch AS a ON (r.arch_id = a.id)
                WHERE n.name = ? AND r.epoch = ? AND r.version = ?
                  AND r.release = ? AND a.name = ?
            """
        else:
            latest = """
                SELECT max(id) AS uniq FROM trans_item
                WHERE action <> 8 GROUP BY item_id
            """
            package_items = """
                SELECT item_id FROM rpm
                WHERE name = ? AND epoch = ? AND version = ? AND release = ? AND arch = ?
            """
        base = f"""
            FROM trans_item JOIN ({latest}) ON (uniq = trans_item.id)
            JOIN repo ON (repo.id = repo_id)
        """
        repo_query = "SELECT DISTINCT repoid " + base
        result = {
            "database": str(args.database),
            "schema_selected_by_claircore": "dnf5" if dnf5 else "dnf4",
            "schema": schema,
            "raw_repositories": rows("SELECT * FROM repo"),
            "all_repoids_query": repo_query,
            "repoids_used_by_scanner": rows(repo_query),
        }
        if args.package:
            query = "SELECT repoid " + base + f" JOIN ({package_items}) USING (item_id)"
            result["package_lookup"] = {
                "parameters": args.package,
                "query": query,
                "result": rows(query, args.package),
                "all_transaction_rows_for_package": rows(
                    f"SELECT trans_item.* FROM trans_item WHERE item_id IN ({package_items})",
                    args.package,
                ),
            }
        result["raw_tables"] = {}
        for table in args.table:
            if table not in tables:
                parser.error(f"Unknown table {table!r}; available tables: {sorted(tables)}")
            quoted = '"' + table.replace('"', '""') + '"'
            result["raw_tables"][table] = rows(f"SELECT * FROM {quoted}")
        print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
