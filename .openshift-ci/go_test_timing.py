#!/usr/bin/env python3

"""Translate `go test -json` lifecycle events into CI timing spans."""

import json
import sys
import uuid
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Dict, Iterable, List, Tuple

from e2e_timing import emit_span_event, is_enabled, record_skipped


@dataclass
class ActiveSpan:
    span_id: str
    phase: str
    name: str
    attributes: Dict[str, str]


SpanKey = Tuple[str, str, str]
_ACTIVITY_MARKER = "e2e_timing_activity "


def _timestamp(event: dict) -> str:
    timestamp = event.get("Time")
    if isinstance(timestamp, str) and timestamp:
        return timestamp
    return datetime.now(timezone.utc).isoformat(timespec="milliseconds").replace(
        "+00:00", "Z"
    )


def _span_details(package: str, test: str) -> Tuple[str, str, Dict[str, str]]:
    if test:
        phase = "test-case"
        name = f"go:{package}::{test}"
        attributes = {
            "framework": "go",
            "package": package,
            "test_name": test,
            "test_depth": str(test.count("/") + 1),
        }
        if "/" in test:
            attributes["parent_test"] = test.rsplit("/", 1)[0]
        return phase, name, attributes

    return "test-suite", f"go:{package}", {
        "framework": "go",
        "package": package,
    }


def _elapsed(event: dict) -> float:
    try:
        return float(event.get("Elapsed") or 0.0)
    except (TypeError, ValueError):
        return 0.0


def _key(phase: str, package: str, test: str) -> SpanKey:
    return phase, package, test


def _start_span(event: dict, active: Dict[SpanKey, List[ActiveSpan]]) -> None:
    package = event.get("Package", "")
    test = event.get("Test", "")
    phase, name, attributes = _span_details(package, test)
    span = ActiveSpan(
        span_id=f"{phase}:{uuid.uuid4().hex}",
        phase=phase,
        name=name,
        attributes=attributes,
    )
    active.setdefault(_key(phase, package, test), []).append(span)
    emit_span_event(
        span.phase,
        span.name,
        span.span_id,
        "start",
        _timestamp(event),
        attributes=span.attributes,
    )


def _finish_span(
    event: dict,
    action: str,
    active: Dict[SpanKey, List[ActiveSpan]],
) -> None:
    package = event.get("Package", "")
    test = event.get("Test", "")
    phase, name, attributes = _span_details(package, test)
    spans = active.get(_key(phase, package, test), [])
    if not spans:
        # A cached Go test has a result but no `run` event. Mark it distinctly
        # rather than presenting it as an executed test interval.
        if action in {"pass", "skip"}:
            reason = (
                "go-test-result-without-run-event"
                if action == "pass"
                else "go-test-skipped-without-run-event"
            )
            record_skipped(phase, name, reason, attributes=attributes)
        return

    span = spans.pop()
    if not spans:
        active.pop(_key(phase, package, test), None)

    outcome = {"pass": "success", "fail": "failure", "skip": "skipped"}[action]
    emit_span_event(
        span.phase,
        span.name,
        span.span_id,
        "end",
        _timestamp(event),
        attributes=span.attributes,
        outcome=outcome,
    )


def _activity_span(event: dict, marker: dict, active: Dict[str, ActiveSpan]) -> bool:
    action = marker.get("event")
    span_id = marker.get("span_id")
    if action not in {"start", "end"} or not isinstance(span_id, str) or not span_id:
        return False

    category = marker.get("activity", "unknown")
    helper = marker.get("helper", "unknown")
    if not isinstance(category, str) or not isinstance(helper, str):
        return False
    details = marker.get("attributes")
    if details is None:
        details = {}
    if not isinstance(details, dict):
        return False
    attributes = {
        "framework": "go",
        "activity": category,
        "helper": helper,
    }
    attributes.update({str(key): str(value) for key, value in details.items()})
    test = event.get("Test") or attributes.get("test_name", "")
    if test:
        attributes.setdefault("test_name", test)
    span_name = f"activity:{category}:{helper}"

    if action == "start":
        span = ActiveSpan(
            span_id=span_id,
            phase="test-activity",
            name=span_name,
            attributes=attributes,
        )
        active[span_id] = span
        emit_span_event(
            span.phase,
            span.name,
            span.span_id,
            "start",
            _timestamp(event),
            attributes=span.attributes,
        )
    else:
        span = active.pop(span_id, None)
        if span is None:
            return False
        emit_span_event(
            span.phase,
            span.name,
            span.span_id,
            "end",
            _timestamp(event),
            attributes=span.attributes,
            outcome=marker.get("outcome"),
        )
    return True


def _render_output(event: dict, active_activities: Dict[str, ActiveSpan]) -> None:
    output = event.get("Output", "")
    for line in output.splitlines(keepends=True):
        if _ACTIVITY_MARKER not in line:
            sys.stdout.write(line)
            continue
        marker_text = line.split(_ACTIVITY_MARKER, 1)[1].strip()
        try:
            marker = json.loads(marker_text)
        except (json.JSONDecodeError, TypeError):
            sys.stdout.write(line)
            continue
        if not isinstance(marker, dict) or not _activity_span(event, marker, active_activities):
            sys.stdout.write(line)


def _render_test_event(event: dict, action: str) -> None:
    test = event.get("Test", "")
    package = event.get("Package", "")
    elapsed = _elapsed(event)

    if action == "run" and test:
        sys.stdout.write(f"=== RUN   {test}\n")
    elif action == "pause" and test:
        sys.stdout.write(f"=== PAUSE {test}\n")
    elif action == "cont" and test:
        sys.stdout.write(f"=== CONT  {test}\n")
    elif action in {"pass", "fail", "skip"} and test:
        label = {"pass": "PASS", "fail": "FAIL", "skip": "SKIP"}[action]
        sys.stdout.write(f"--- {label}: {test} ({elapsed:.2f}s)\n")
    elif action == "pass" and package:
        sys.stdout.write("PASS\n")
        sys.stdout.write(f"ok\t{package}\t{elapsed:.2f}s\n")
    elif action == "fail" and package:
        sys.stdout.write("FAIL\n")
        sys.stdout.write(f"FAIL\t{package}\n")
    elif action == "skip" and package:
        sys.stdout.write(f"?\t{package}\t[no test files]\n")


def _handle_event(
    event: dict,
    active: Dict[SpanKey, List[ActiveSpan]],
    active_activities: Dict[str, ActiveSpan],
    timing_enabled: bool,
) -> None:
    action = event.get("Action", "")
    if action in {"output", "build-output", "build-fail"}:
        if action == "output" and timing_enabled:
            _render_output(event, active_activities)
        else:
            sys.stdout.write(event.get("Output", ""))
    elif action == "start" and not event.get("Test"):
        if timing_enabled:
            _start_span(event, active)
    elif action == "run" and event.get("Test"):
        if timing_enabled:
            _start_span(event, active)
        _render_test_event(event, action)
    elif action in {"pause", "cont"}:
        _render_test_event(event, action)
    elif action in {"pass", "fail", "skip"}:
        if timing_enabled:
            _finish_span(event, action, active)
        _render_test_event(event, action)
    sys.stdout.flush()


def process_events(lines: Iterable[str]) -> None:
    """Stream readable Go test output and timing sentinels without buffering."""
    active: Dict[SpanKey, List[ActiveSpan]] = {}
    active_activities: Dict[str, ActiveSpan] = {}
    timing_enabled = is_enabled()

    for line in lines:
        try:
            event = json.loads(line)
        except (json.JSONDecodeError, TypeError):
            sys.stdout.write(line)
            sys.stdout.flush()
            continue

        if not isinstance(event, dict) or not isinstance(event.get("Action"), str):
            sys.stdout.write(line)
            sys.stdout.flush()
            continue

        _handle_event(event, active, active_activities, timing_enabled)

    # If the test command exits or is terminated with unfinished tests, close
    # those intervals as interrupted rather than leaving dangling start events.
    for spans in active.values():
        for span in spans:
            emit_span_event(
                span.phase,
                span.name,
                span.span_id,
                "end",
                datetime.now(timezone.utc)
                .isoformat(timespec="milliseconds")
                .replace("+00:00", "Z"),
                attributes=span.attributes,
                outcome="interrupted",
            )
    for span in active_activities.values():
        emit_span_event(
            span.phase,
            span.name,
            span.span_id,
            "end",
            datetime.now(timezone.utc)
            .isoformat(timespec="milliseconds")
            .replace("+00:00", "Z"),
            attributes=span.attributes,
            outcome="interrupted",
        )


def main() -> int:
    process_events(sys.stdin)
    return 0


if __name__ == "__main__":
    sys.exit(main())
