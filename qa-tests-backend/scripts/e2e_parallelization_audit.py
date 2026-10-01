#!/usr/bin/env python3
"""Summarize observer-only Groovy spec footprints from a GHA job log."""

import argparse
import json
import sys
from collections import defaultdict
from datetime import datetime
from pathlib import Path

PREFIX = "e2e_parallel_audit "
READ_METHODS = {"GET", "HEAD"}
READ_RPC_PREFIXES = ("get", "list", "search", "count", "find", "lookup")
WRITE_RPC_PREFIXES = (
    "create", "update", "delete", "remove", "upsert", "set", "add", "generate",
    "revoke", "grant", "import", "register", "enable", "disable", "approve", "cancel",
)
LIMITATIONS = [
    "Findings are shared-write candidates, not proof that operations raced or tests are unsafe.",
    "Only Fabric8 Kubernetes requests and Central gRPC calls are observed; direct REST, external systems, and controller side effects are not covered.",
    "Async calls that lose test-thread MDC and specs missing lifecycle markers may be unattributed or temporally unknown.",
    "Central conflicts are service-level because request payloads are intentionally not captured.",
]


def parse_events(lines):
    events, malformed = [], 0
    for line in lines:
        marker = line.find(PREFIX)
        if marker < 0:
            continue
        try:
            event = json.loads(line[marker + len(PREFIX) :])
        except json.JSONDecodeError:
            malformed += 1
            continue
        if isinstance(event, dict) and event.get("schema_version") == 1:
            events.append(event)
        else:
            malformed += 1
    return events, malformed


def _timestamp(value):
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00"))
    except (AttributeError, TypeError, ValueError):
        return None


def _access(operation):
    attrs = operation.get("attributes", operation)
    if attrs.get("system") == "kubernetes":
        return "read" if attrs.get("method", "").upper() in READ_METHODS else "write"
    if attrs.get("system") == "central-grpc":
        method = attrs.get("method", "").lower()
        if method.startswith(READ_RPC_PREFIXES):
            return "read"
        if method.startswith(WRITE_RPC_PREFIXES):
            return "write"
    return "write"  # Unknown operations are conservatively treated as mutations.


def _path_overlap(first, second):
    first, second = (first or "").rstrip("/"), (second or "").rstrip("/")
    return bool(first and second) and (
        first == second or first.startswith(second + "/") or second.startswith(first + "/")
    )


def _shared_write(first, second):
    a, b = first.get("attributes", first), second.get("attributes", second)
    system_a, system_b = a.get("system"), b.get("system")
    if system_a == system_b == "kubernetes" and _path_overlap(a.get("path"), b.get("path")):
        return "kubernetes", min(a["path"], b["path"], key=len)
    if system_a == system_b == "central-grpc" and a.get("service") == b.get("service"):
        return "central-grpc", a.get("service", "unknown-service")
    if system_a == system_b == "process-global" and a.get("key") == b.get("key"):
        return "process-global", a.get("key", "unknown-global-state")
    if system_a == "process-global" and system_b == "central-grpc":
        return "process-global", a.get("key", "unknown-global-state")
    if system_b == "process-global" and system_a == "central-grpc":
        return "process-global", b.get("key", "unknown-global-state")
    return None


def analyze(events, malformed=0):
    specs, unassigned = {}, 0
    for event in events:
        execution_id = event.get("execution_id")
        if not execution_id:
            unassigned += event.get("event_type") == "operation"
            continue
        key = (event.get("run_id", "local"), event.get("lane_id", "unknown"), execution_id)
        spec = specs.setdefault(
            key,
            {"run_id": key[0], "lane_id": key[1], "execution_id": execution_id,
             "name": event.get("specification", "unknown-spec"), "start": None, "end": None, "operations": []},
        )
        when = _timestamp(event.get("timestamp"))
        if event.get("event_type") == "spec_start":
            spec["start"] = when
        elif event.get("event_type") == "spec_end":
            spec["end"] = when
        elif event.get("event_type") == "operation":
            spec["operations"].append({
                "test_name": event.get("test_name"),
                **(event.get("attributes") or {}),
            })

    by_lane, findings = defaultdict(list), {}
    for spec in specs.values():
        by_lane[(spec["run_id"], spec["lane_id"])].append(spec)

    concurrent_pairs = pair_count = 0
    for lane_specs in by_lane.values():
        for index, first in enumerate(lane_specs):
            for second in lane_specs[index + 1 :]:
                pair_count += 1
                complete = all(s["start"] and s["end"] for s in (first, second))
                overlap = complete and first["start"] < second["end"] and second["start"] < first["end"]
                relation = "overlapping" if overlap else "sequential" if complete else "unknown"
                concurrent_pairs += overlap
                for op_a in first["operations"]:
                    for op_b in second["operations"]:
                        shared = _shared_write(op_a, op_b)
                        if not shared or (_access(op_a) == _access(op_b) == "read"):
                            continue
                        system, resource = shared
                        key = (tuple(sorted((first["execution_id"], second["execution_id"]))), system, resource)
                        finding = findings.setdefault(key, {
                            "system": system, "resource": resource, "temporal_relation": relation,
                            "specifications": {},
                        })
                        for spec, operation in ((first, op_a), (second, op_b)):
                            evidence = {k: operation[k] for k in ("test_name", "method") if operation.get(k)}
                            samples = finding["specifications"].setdefault(spec["name"], [])
                            if evidence not in samples:
                                samples.append(evidence)

    report_specs = [
        {
            "run_id": spec["run_id"], "lane_id": spec["lane_id"],
            "execution_id": spec["execution_id"], "specification": spec["name"],
            "start": spec["start"].isoformat() if spec["start"] else None,
            "end": spec["end"].isoformat() if spec["end"] else None,
            "lifecycle_complete": bool(spec["start"] and spec["end"]),
            "operations": spec["operations"],
        }
        for spec in specs.values()
    ]
    return {
        "schema_version": 1, "audit_events": len(events), "malformed_events": malformed,
        "specifications": report_specs, "specification_pairs_compared": pair_count,
        "overlapping_specification_pairs": concurrent_pairs, "unattributed_operations": unassigned,
        "shared_write_candidates": list(findings.values()), "limitations": LIMITATIONS,
    }


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("log", nargs="?", default="-", help="GHA job log file, or -/stdin")
    parser.add_argument("--format", choices=("text", "json"), default="text")
    args = parser.parse_args(argv)
    lines = sys.stdin if args.log == "-" else Path(args.log).open(encoding="utf-8", errors="replace")
    events, malformed = parse_events(lines)
    report = analyze(events, malformed)
    if args.format == "json":
        print(json.dumps(report, indent=2, sort_keys=True))
    else:
        print(f"Events: {report['audit_events']} (malformed: {malformed})")
        print(
            f"Specs: {len(report['specifications'])}; pairs compared: {report['specification_pairs_compared']}; "
            f"overlapping spec lifetimes: {report['overlapping_specification_pairs']}"
        )
        print(f"Unattributed operations: {report['unattributed_operations']}")
        print(f"Shared-write candidates: {len(report['shared_write_candidates'])}")
        for finding in report["shared_write_candidates"][:40]:
            relation = finding["temporal_relation"]
            specs_text = " <> ".join(finding["specifications"])
            print(f"  {finding['system']} {finding['resource']} [{relation}] — {specs_text}")
        for note in LIMITATIONS:
            print(f"NOTE: {note}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
