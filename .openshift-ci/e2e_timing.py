#!/usr/bin/env python3

"""Small, best-effort e2e timing event emitter shared by CI helper scripts."""

import contextlib
import json
import os
import subprocess
import time
import uuid
from datetime import datetime, timezone
from typing import Dict, Iterator, Optional, Tuple


_TRUE_VALUES = {"1", "true", "yes", "on"}
_MVP_GHA_LANES = {
    "gke-qa-e2e-tests": "gha.qa.gke",
    "gke-nongroovy-e2e-tests": "gha.e2e.nongroovy.gke",
}


def is_enabled() -> bool:
    """Enable timing on the two MVP GHA lanes, with explicit overrides."""
    configured = os.getenv("E2E_TIMING_ENABLED")
    if configured is not None:
        return configured.lower() in _TRUE_VALUES
    is_github_actions = os.getenv("GITHUB_ACTIONS", "").lower() in _TRUE_VALUES
    return is_github_actions and os.getenv("CI_JOB_NAME") in _MVP_GHA_LANES


def _provider_and_run_id() -> Tuple[str, str]:
    explicit_run_id = os.getenv("E2E_TIMING_RUN_ID")
    if os.getenv("GITHUB_ACTIONS", "").lower() in _TRUE_VALUES:
        run_id = explicit_run_id or ":".join(
            (
                "gha",
                os.getenv("GITHUB_RUN_ID", "unknown-run"),
                os.getenv("GITHUB_RUN_ATTEMPT", "1"),
                os.getenv("CI_JOB_NAME") or os.getenv("GITHUB_JOB", "unknown-job"),
            )
        )
        return "github-actions", run_id

    if os.getenv("OPENSHIFT_CI", "").lower() in _TRUE_VALUES:
        run_id = explicit_run_id or ":".join(
            (
                "prow",
                os.getenv("BUILD_ID", "unknown-build"),
                os.getenv("JOB_NAME", "unknown-job"),
            )
        )
        return "openshift-ci", run_id

    return "local", explicit_run_id or f"local:{os.getpid()}"


def _lane_id() -> str:
    configured_lane = os.getenv("E2E_TIMING_LANE_ID")
    if configured_lane:
        return configured_lane
    ci_job_name = os.getenv("CI_JOB_NAME")
    if ci_job_name in _MVP_GHA_LANES:
        return _MVP_GHA_LANES[ci_job_name]
    return (
        ci_job_name
        or os.getenv("JOB_NAME")
        or os.getenv("GITHUB_JOB")
        or "unknown-lane"
    )


def _timestamp() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="milliseconds").replace(
        "+00:00", "Z"
    )


def _base_event(phase: str, name: str, attributes: Optional[Dict[str, str]]):
    provider, run_id = _provider_and_run_id()
    event: Dict[str, object] = {
        "schema_version": 1,
        "run_id": run_id,
        "lane_id": _lane_id(),
        "provider": provider,
        "span_id": f"{phase}:{uuid.uuid4().hex}",
        "phase": phase,
        "name": name,
    }
    if attributes:
        event["attributes"] = attributes
    return event


def _write_event(event: Dict[str, object]) -> None:
    line = "e2e_timing " + json.dumps(event, separators=(",", ":"), sort_keys=True)
    try:
        print(line, flush=True)
    except OSError:
        # Timing telemetry must not affect test or cleanup behavior.
        pass


def child_timing_environment() -> Optional[Dict[str, str]]:
    """Propagate the resolved timing identity to instrumented child processes."""
    if not is_enabled():
        return None

    provider, run_id = _provider_and_run_id()
    environment = os.environ.copy()
    environment["E2E_TIMING_ENABLED"] = "true"
    environment.setdefault("E2E_TIMING_PROVIDER", provider)
    environment.setdefault("E2E_TIMING_RUN_ID", run_id)
    environment.setdefault("E2E_TIMING_LANE_ID", _lane_id())
    return environment


def emit_span_event(
    phase: str,
    name: str,
    span_id: str,
    event_name: str,
    timestamp: str,
    attributes: Optional[Dict[str, str]] = None,
    duration_ms: Optional[int] = None,
    outcome: Optional[str] = None,
) -> None:
    """Emit one half of a span using a timestamp supplied by a test framework."""
    if not is_enabled() or event_name not in {"start", "end"}:
        return

    event = _base_event(phase, name, attributes)
    event.update(
        {
            "span_id": span_id,
            "event": event_name,
            "timestamp": timestamp,
        }
    )
    if event_name == "end":
        if duration_ms is not None:
            event["duration_ms"] = max(0, duration_ms)
        if outcome:
            event["outcome"] = outcome
    _write_event(event)


def record_skipped(
    phase: str,
    name: str,
    reason: str,
    attributes: Optional[Dict[str, str]] = None,
) -> None:
    """Emit an explicit skip marker; skipped work is not a zero-duration span."""
    if not is_enabled():
        return
    event = _base_event(phase, name, attributes)
    event.update({"event": "skipped", "timestamp": _timestamp(), "reason": reason})
    _write_event(event)


@contextlib.contextmanager
def timed_span(
    phase: str,
    name: str,
    attributes: Optional[Dict[str, str]] = None,
) -> Iterator[None]:
    """Emit start/end JSONL records around work without changing its outcome."""
    if not is_enabled():
        yield
        return

    base_event = _base_event(phase, name, attributes)
    start_time = time.perf_counter_ns()
    _write_event({**base_event, "event": "start", "timestamp": _timestamp()})
    outcome = "success"
    reason = None
    try:
        yield
    except BaseException as err:
        if isinstance(err, (TimeoutError, subprocess.TimeoutExpired)):
            outcome = "timeout"
        elif isinstance(err, (KeyboardInterrupt, SystemExit)):
            outcome = "interrupted"
        else:
            outcome = "failure"
        reason = type(err).__name__
        raise
    finally:
        end_time = time.perf_counter_ns()
        end_event: Dict[str, object] = {
            **base_event,
            "event": "end",
            "timestamp": _timestamp(),
            "duration_ms": max(0, (end_time - start_time) // 1_000_000),
            "outcome": outcome,
        }
        if reason:
            end_event["reason"] = reason
        _write_event(end_event)
