"""Read and write the decision file and the decision-defaults file.

The decision file lists jobs to run, one name on a line. A line that starts
with # is a comment and is not a job. An empty file is not a decision.

decision-defaults gives every target its own fallback. The last word on a
line is run or skip. The words before that are the target name.
"""

from __future__ import annotations

from dataclasses import dataclass


class DecisionError(Exception):
    """The text is not a usable decision or defaults file."""


@dataclass(frozen=True)
class Decision:
    """Decision is the set of jobs the dispatcher should run."""

    jobs: frozenset[str]


def read_list(text: str) -> list[str]:
    """read_list returns the non-blank lines, in order."""
    return [line.strip() for line in text.splitlines() if line.strip()]


def parse_decision(text: str) -> Decision:
    """parse_decision reads a decision file.

    Empty text is not a decision. A file that only has comments is a
    decision to run nothing.
    """
    if not text.strip():
        raise DecisionError("empty decision")
    jobs: list[str] = []
    for lineno, raw in enumerate(text.splitlines(), start=1):
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        if "#" in line:
            raise DecisionError(f"line {lineno} mixes a job name and a comment")
        jobs.append(line)
    return Decision(jobs=frozenset(jobs))


def format_decision(jobs: list[str]) -> str:
    """format_decision writes one job name per line.

    An empty list still writes one comment, so the file is a real decision
    to run nothing. A blank file is not a decision. A person may add other
    # comments; this writer does not.
    """
    if not jobs:
        return "# no job runs\n"
    lines: list[str] = []
    for job in jobs:
        if not job or "\n" in job or job.startswith("#"):
            raise DecisionError(f"bad job name {job!r}")
        lines.append(job)
    return "\n".join(lines) + "\n"


def parse_defaults(text: str) -> dict[str, str]:
    """parse_defaults reads one target per line.

    The line is the target name, then run or skip. The target may contain
    spaces, so the decision is the last word.
    """
    if not text.strip():
        raise DecisionError("decision-defaults is empty")
    defaults: dict[str, str] = {}
    for lineno, raw in enumerate(text.splitlines(), start=1):
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        if "#" in line:
            raise DecisionError(f"line {lineno} mixes a target and a comment")
        parts = line.rsplit(None, 1)
        if len(parts) != 2:
            raise DecisionError(f"line {lineno} needs a target and a decision")
        target, opinion = parts
        if opinion not in ("run", "skip"):
            raise DecisionError(f"line {lineno} decision must be run or skip")
        if target in defaults:
            raise DecisionError(f"duplicate default for {target}")
        defaults[target] = opinion
    if not defaults:
        raise DecisionError("decision-defaults lists no targets")
    return defaults


def decision_from_defaults(defaults: dict[str, str]) -> str:
    """decision_from_defaults is the decision file implied by the defaults.

    A target whose default is run is listed. A target whose default is skip
    is left off the list.
    """
    names = [name for name, opinion in defaults.items() if opinion == "run"]
    return format_decision(names)


def check_defaults(jobs: list[str], defaults: dict[str, str]) -> None:
    """check_defaults requires one line in decision-defaults for every target."""
    job_set = set(jobs)
    missing = [job for job in jobs if job not in defaults]
    extra = [name for name in defaults if name not in job_set]
    if missing:
        raise DecisionError("decision-defaults is missing " + ", ".join(missing))
    if extra:
        raise DecisionError("decision-defaults has unknown targets " + ", ".join(extra))
