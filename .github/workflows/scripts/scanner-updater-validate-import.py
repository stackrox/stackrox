#!/usr/bin/env python3

"""Validate that every source in an updater bundle was effectively imported."""

import argparse
import json
import re
import subprocess
import sys
import uuid
from collections import defaultdict
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path


COMPLETION_MESSAGE = "update imported"
SUPPORTED_KINDS = frozenset(("vulnerability", "enrichment"))
SOURCE_RE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]*$")
TIMESTAMP_RE = re.compile(
    r"^[0-9]{4}-[0-9]{2}-[0-9]{2}[T ][0-9]{2}:[0-9]{2}:[0-9]{2}"
    r"(?:\.[0-9]{1,6})?(?:Z|[+-][0-9]{2}:[0-9]{2})?$"
)

METADATA_QUERY = r"""
WITH expected AS (
    SELECT item.source, item.ordinality
    FROM jsonb_array_elements_text(:'metadata_sources'::jsonb)
         WITH ORDINALITY AS item(source, ordinality)
)
SELECT COALESCE(
    jsonb_agg(
        jsonb_build_object(
            'source', expected.source,
            'timestamps', COALESCE(
                (
                    SELECT jsonb_agg(metadata.update_timestamp)
                    FROM public.last_vuln_update AS metadata
                    WHERE metadata.key =
                        'bundles/' || expected.source || '.json.zst'
                ),
                '[]'::jsonb
            )
        )
        ORDER BY expected.ordinality
    ),
    '[]'::jsonb
)::text
FROM expected;
"""

OPERATION_QUERY = r"""
WITH expected AS (
    SELECT item.ref::uuid AS ref, item.kind
    FROM jsonb_to_recordset(:'completed_operations'::jsonb)
         AS item(ref text, kind text)
)
SELECT COALESCE(
    jsonb_agg(
        jsonb_build_object(
            'ref', expected.ref::text,
            'operation_kinds', COALESCE(
                (
                    SELECT jsonb_agg(operation.kind)
                    FROM public.update_operation AS operation
                    WHERE operation.ref = expected.ref
                ),
                '[]'::jsonb
            ),
            'links', CASE expected.kind
                WHEN 'vulnerability' THEN (
                    SELECT count(link.vuln)
                    FROM public.update_operation AS operation
                    JOIN public.uo_vuln AS link ON link.uo = operation.id
                    WHERE operation.ref = expected.ref
                )
                WHEN 'enrichment' THEN (
                    SELECT count(link.enrich)
                    FROM public.update_operation AS operation
                    JOIN public.uo_enrich AS link ON link.uo = operation.id
                    WHERE operation.ref = expected.ref
                )
            END
        )
        ORDER BY expected.ref::text
    ),
    '[]'::jsonb
)::text
FROM expected;
"""


class ValidationFailure(Exception):
    """Raised when validation cannot safely continue."""


class DuplicateJSONKey(ValueError):
    """Raised when a JSON object contains an ambiguous duplicate key."""


@dataclass(frozen=True)
class Completion:
    source: str
    bundle: str
    kind: str
    ref: str
    line_number: int


def parse_args():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--log",
        required=True,
        type=Path,
        help="Updater import output containing JSON log records",
    )
    parser.add_argument(
        "--db-conn",
        required=True,
        help="libpq connection string for the matcher database",
    )
    parser.add_argument(
        "--psql",
        default="psql",
        help="psql executable (default: psql)",
    )
    parser.add_argument(
        "sources",
        nargs="+",
        help="Expected source names, as listed in source-config.yaml",
    )
    return parser.parse_args()


def reject_duplicate_json_keys(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise DuplicateJSONKey(f"duplicate JSON object key {key!r}")
        result[key] = value
    return result


def strict_json_loads(value):
    return json.loads(value, object_pairs_hook=reject_duplicate_json_keys)


def validate_expected_sources(sources):
    seen = set()
    for source in sources:
        if not SOURCE_RE.fullmatch(source):
            raise ValidationFailure(
                f"invalid expected source name {source!r}; refusing to query the database"
            )
        if source in seen:
            raise ValidationFailure(f"duplicate expected source name: {source}")
        seen.add(source)

        key = f"bundles/{source}.json.zst"
        if len(key.encode("ascii")) > 128:
            raise ValidationFailure(
                f"{source}: metadata key exceeds last_vuln_update.key capacity"
            )


def completion_message(record, line_number):
    fields = []
    for field_name in ("msg", "message"):
        if field_name not in record:
            continue
        value = record[field_name]
        if not isinstance(value, str):
            raise ValidationFailure(
                f"updater log line {line_number}: {field_name} must be a string"
            )
        fields.append((field_name, value))

    if not fields:
        return None

    values = {value for _, value in fields}
    if len(values) != 1:
        raise ValidationFailure(
            f"updater log line {line_number}: conflicting msg/message fields"
        )
    return fields[0][1]


def parse_completion_events(log_path, sources):
    errors = []
    completions = []
    expected_bundles = {
        f"bundles/{source}.json.zst": source for source in sources
    }
    seen_refs = {}

    try:
        log_text = log_path.read_text(encoding="utf-8")
    except (OSError, UnicodeError) as err:
        return [], [f"cannot read updater log {log_path}: {err}"]

    for line_number, raw_line in enumerate(log_text.splitlines(), start=1):
        line = raw_line.strip()
        if not line:
            continue

        try:
            record = strict_json_loads(line)
        except json.JSONDecodeError as err:
            errors.append(
                f"updater log line {line_number}: malformed JSON: {err.msg}"
            )
            continue
        except DuplicateJSONKey as err:
            errors.append(f"updater log line {line_number}: {err}")
            continue

        if not isinstance(record, dict):
            errors.append(
                f"updater log line {line_number}: expected a JSON object"
            )
            continue

        try:
            message = completion_message(record, line_number)
        except ValidationFailure as err:
            errors.append(str(err))
            continue
        if message != COMPLETION_MESSAGE:
            continue

        bundle = record.get("bundle")
        kind = record.get("kind")
        ref = record.get("ref")

        if not isinstance(bundle, str) or not bundle:
            errors.append(
                f"updater log line {line_number}: completion event has no valid bundle"
            )
            continue
        source = expected_bundles.get(bundle)
        if source is None:
            errors.append(
                f"updater log line {line_number}: completion event has unexpected "
                f"bundle {bundle!r}"
            )
            continue

        if not isinstance(kind, str) or kind not in SUPPORTED_KINDS:
            errors.append(
                f"{source}: completion event on log line {line_number} has unknown "
                f"kind {kind!r}"
            )
            continue

        if not isinstance(ref, str):
            errors.append(
                f"{source}: completion event on log line {line_number} has no valid ref"
            )
            continue
        try:
            parsed_ref = uuid.UUID(ref)
        except ValueError:
            errors.append(
                f"{source}: completion event on log line {line_number} has invalid "
                f"UUID ref {ref!r}"
            )
            continue
        canonical_ref = str(parsed_ref)
        if canonical_ref != ref:
            errors.append(
                f"{source}: completion event on log line {line_number} has "
                f"non-canonical UUID ref {ref!r}"
            )
            continue

        completion = Completion(source, bundle, kind, canonical_ref, line_number)
        previous = seen_refs.get(canonical_ref)
        if previous is not None:
            if (
                previous.source == completion.source
                and previous.bundle == completion.bundle
                and previous.kind == completion.kind
            ):
                errors.append(
                    f"{source}: duplicate completion event for ref {canonical_ref} "
                    f"on log lines {previous.line_number} and {line_number}"
                )
            else:
                errors.append(
                    f"conflicting completion events for ref {canonical_ref} on log "
                    f"lines {previous.line_number} and {line_number}"
                )
            continue

        seen_refs[canonical_ref] = completion
        completions.append(completion)

    completed_sources = {completion.source for completion in completions}
    for source in sources:
        if source not in completed_sources:
            errors.append(f"{source}: missing update imported completion event")

    return completions, errors


def run_json_query(psql, db_conn, variable_name, variable_value, query, purpose):
    encoded_value = json.dumps(
        variable_value, separators=(",", ":"), sort_keys=True
    )
    command = [
        psql,
        "--no-psqlrc",
        "--quiet",
        "--tuples-only",
        "--no-align",
        "--set=ON_ERROR_STOP=1",
        f"--set={variable_name}={encoded_value}",
        f"--dbname={db_conn}",
    ]

    try:
        result = subprocess.run(
            command,
            input=query,
            capture_output=True,
            text=True,
            check=False,
        )
    except OSError as err:
        raise ValidationFailure(f"cannot run psql for {purpose}: {err}") from err

    if result.returncode != 0:
        details = " ".join(result.stderr.strip().splitlines())
        if not details:
            details = "psql produced no error details"
        raise ValidationFailure(
            f"database {purpose} query failed with exit status "
            f"{result.returncode}: {details}"
        )

    lines = result.stdout.splitlines()
    if len(lines) != 1 or not lines[0].strip():
        raise ValidationFailure(
            f"database {purpose} query returned ambiguous output: expected exactly "
            f"one non-empty JSON line, got {len(lines)}"
        )

    try:
        return strict_json_loads(lines[0])
    except json.JSONDecodeError as err:
        raise ValidationFailure(
            f"database {purpose} query returned malformed JSON: {err.msg}"
        ) from err
    except DuplicateJSONKey as err:
        raise ValidationFailure(
            f"database {purpose} query returned ambiguous JSON: {err}"
        ) from err


def parse_timestamp(value):
    if not isinstance(value, str) or not TIMESTAMP_RE.fullmatch(value):
        raise ValueError("expected a complete ISO-8601 date and time")

    normalized = value[:-1] + "+00:00" if value.endswith("Z") else value
    timestamp = datetime.fromisoformat(normalized)
    if timestamp.tzinfo is None:
        epoch = datetime(1970, 1, 1)
    else:
        timestamp = timestamp.astimezone(timezone.utc)
        epoch = datetime(1970, 1, 1, tzinfo=timezone.utc)
    return timestamp, epoch


def validate_metadata_results(results, sources):
    errors = []
    timestamps = {}
    expected = set(sources)
    seen = set()

    if not isinstance(results, list):
        return {}, ["database metadata query result must be a JSON array"]

    for index, result in enumerate(results):
        if not isinstance(result, dict):
            errors.append(f"database metadata result {index} is not an object")
            continue

        source = result.get("source")
        rows = result.get("timestamps")
        if not isinstance(source, str) or source not in expected:
            errors.append(
                f"database metadata result {index} has unexpected source {source!r}"
            )
            continue
        if source in seen:
            errors.append(f"{source}: duplicate database metadata result")
            continue
        seen.add(source)

        if not isinstance(rows, list):
            errors.append(f"{source}: metadata timestamps result is not an array")
            continue
        if not rows:
            errors.append(f"{source}: missing row in last_vuln_update")
            continue
        if len(rows) != 1:
            errors.append(
                f"{source}: ambiguous last_vuln_update result; expected one row, "
                f"got {len(rows)}"
            )
            continue

        value = rows[0]
        if value is None:
            errors.append(f"{source}: NULL update_timestamp in last_vuln_update")
            continue
        try:
            timestamp, epoch = parse_timestamp(value)
        except (TypeError, ValueError) as err:
            errors.append(
                f"{source}: malformed update_timestamp {value!r}: {err}"
            )
            continue
        if timestamp <= epoch:
            errors.append(
                f"{source}: update_timestamp {value!r} is not later than Unix epoch"
            )
            continue
        timestamps[source] = value

    for source in sources:
        if source not in seen:
            errors.append(f"{source}: database metadata query omitted the source")

    return timestamps, errors


def validate_operation_results(results, completions, sources):
    errors = []
    expected = {completion.ref: completion for completion in completions}
    seen = set()
    invalid_sources = set()
    links_by_source = defaultdict(int)

    if not isinstance(results, list):
        return {}, ["database operation query result must be a JSON array"]

    for index, result in enumerate(results):
        if not isinstance(result, dict):
            errors.append(f"database operation result {index} is not an object")
            continue

        ref = result.get("ref")
        completion = expected.get(ref)
        if completion is None:
            errors.append(
                f"database operation result {index} has unexpected ref {ref!r}"
            )
            continue
        if ref in seen:
            errors.append(f"{completion.source}: duplicate database result for ref {ref}")
            invalid_sources.add(completion.source)
            continue
        seen.add(ref)

        operation_kinds = result.get("operation_kinds")
        links = result.get("links")
        valid = True

        if not isinstance(operation_kinds, list):
            errors.append(
                f"{completion.source}: database kinds for ref {ref} are malformed"
            )
            valid = False
        elif not operation_kinds:
            errors.append(
                f"{completion.source}: update operation ref {ref} is missing from "
                "the database"
            )
            valid = False
        elif len(operation_kinds) != 1:
            errors.append(
                f"{completion.source}: update operation ref {ref} is ambiguous in "
                "the database"
            )
            valid = False
        elif operation_kinds[0] != completion.kind:
            errors.append(
                f"{completion.source}: update operation ref {ref} has database kind "
                f"{operation_kinds[0]!r}, expected {completion.kind!r}"
            )
            valid = False

        if type(links) is not int or links < 0:
            errors.append(
                f"{completion.source}: relation count for ref {ref} is malformed"
            )
            valid = False

        if valid:
            links_by_source[completion.source] += links
        else:
            invalid_sources.add(completion.source)

    for ref, completion in expected.items():
        if ref not in seen:
            errors.append(
                f"{completion.source}: database operation query omitted ref {ref}"
            )
            invalid_sources.add(completion.source)

    completions_by_source = defaultdict(list)
    for completion in completions:
        completions_by_source[completion.source].append(completion)

    for source in sources:
        source_completions = completions_by_source[source]
        if not source_completions or source in invalid_sources:
            continue
        if links_by_source[source] == 0:
            kinds = "/".join(
                sorted({completion.kind for completion in source_completions})
            )
            errors.append(
                f"{source}: zero effective {kinds} relations across "
                f"{len(source_completions)} completed operation(s)"
            )

    return dict(links_by_source), errors


def github_error(message):
    escaped = message.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")
    print(f"::error::{escaped}")


def main():
    args = parse_args()
    errors = []

    try:
        validate_expected_sources(args.sources)
    except ValidationFailure as err:
        github_error(str(err))
        return 1

    completions, log_errors = parse_completion_events(args.log, args.sources)
    errors.extend(log_errors)

    timestamps = {}
    try:
        metadata_results = run_json_query(
            args.psql,
            args.db_conn,
            "metadata_sources",
            args.sources,
            METADATA_QUERY,
            "metadata",
        )
    except ValidationFailure as err:
        errors.append(str(err))
    else:
        timestamps, metadata_errors = validate_metadata_results(
            metadata_results, args.sources
        )
        errors.extend(metadata_errors)

    links_by_source = {}
    if completions:
        operation_parameters = [
            {"ref": completion.ref, "kind": completion.kind}
            for completion in completions
        ]
        try:
            operation_results = run_json_query(
                args.psql,
                args.db_conn,
                "completed_operations",
                operation_parameters,
                OPERATION_QUERY,
                "operation",
            )
        except ValidationFailure as err:
            errors.append(str(err))
        else:
            links_by_source, operation_errors = validate_operation_results(
                operation_results, completions, args.sources
            )
            errors.extend(operation_errors)

    if errors:
        for error in errors:
            github_error(error)
        return 1

    operation_counts = defaultdict(int)
    for completion in completions:
        operation_counts[completion.source] += 1

    for source in args.sources:
        print(
            f"source '{source}': imported "
            f"(update_timestamp={timestamps[source]}, "
            f"effective_links={links_by_source[source]}, "
            f"operations={operation_counts[source]})"
        )
    print(f"Bundle import validation passed for {len(args.sources)} source(s)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
