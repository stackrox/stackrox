#!/usr/bin/env python3
"""Classify CI jobs for one diff by walking a numbered rule list.

Rules are data in a TOML file. The list is walked from top to bottom.
A later run replaces an earlier skip. A later skip does not replace an
earlier run: that is a conflict, the plan warns, and the job runs.
A job no rule has mentioned is unsure until the last rule, which must
skip every remaining job. Shadow mode still executes every known job.
Pass shadow=False (CLI: --enforce) to execute only run.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
import tomllib
from dataclasses import dataclass, field
from pathlib import Path

_WHEN = frozenset({"all", "any", "always", "label", "no-files", "remaining"})


@dataclass(frozen=True)
class Rule:
    number: int
    name: str
    when: str
    paths: tuple[re.Pattern[str], ...]
    path_patterns: tuple[str, ...]
    label: str
    run: frozenset[str]
    skip: frozenset[str]
    run_all: bool
    skip_all: bool

    def condition(self) -> str:
        if self.when == "always":
            return "every pull request"
        if self.when == "no-files":
            return "the diff has no files"
        if self.when == "label":
            return f"label {self.label} is set"
        if self.when == "all":
            return "every changed file matches"
        if self.when == "any":
            return "any changed file matches"
        return "every job no earlier rule decided"


@dataclass(frozen=True)
class Conflict:
    job: str
    run_rules: tuple[int, ...]
    skip_rules: tuple[int, ...]


@dataclass(frozen=True)
class Decision:
    job: str
    opinion: str
    rules: tuple[int, ...]


@dataclass
class _Votes:
    run_rules: list[int] = field(default_factory=list)
    skip_rules: list[int] = field(default_factory=list)


@dataclass(frozen=True)
class Mapping:
    jobs: frozenset[str]
    rules: tuple[Rule, ...]
    by_number: dict[int, Rule]
    # Key requires every job in the tuple. A run of the key is a run of each.
    requires: dict[str, tuple[str, ...]]


@dataclass(frozen=True)
class FileTrace:
    """One changed path. The rule list, not the file, decides the jobs."""

    path: str
    kind: str = "file"
    domain: str = ""
    explicit_runs: tuple[str, ...] = ()
    explicit_skips: tuple[str, ...] = ()


@dataclass(frozen=True)
class Selection:
    run: frozenset[str]
    skip: frozenset[str]
    unsure: frozenset[str]
    execute: frozenset[str]
    reason: str
    shadow: bool
    files: tuple[FileTrace, ...] = ()
    required_runs: frozenset[str] = frozenset()
    matched_rules: tuple[int, ...] = ()
    decisions: tuple[Decision, ...] = ()
    conflicts: tuple[Conflict, ...] = ()

    def decision_for(self, job: str) -> Decision | None:
        for decision in self.decisions:
            if decision.job == job:
                return decision
        return None

    def conflict_for(self, job: str) -> Conflict | None:
        for conflict in self.conflicts:
            if conflict.job == job:
                return conflict
        return None

    def to_dict(self) -> dict[str, object]:
        return {
            "shadow": self.shadow,
            "reason": self.reason,
            "run": sorted(self.run),
            "skip": sorted(self.skip),
            "unsure": sorted(self.unsure),
            "execute": sorted(self.execute),
            "matched_rules": list(self.matched_rules),
            "required_runs": sorted(self.required_runs),
            "conflicts": [
                {
                    "job": conflict.job,
                    "run_rules": list(conflict.run_rules),
                    "skip_rules": list(conflict.skip_rules),
                }
                for conflict in self.conflicts
            ],
            "decisions": [
                {
                    "job": decision.job,
                    "opinion": decision.opinion,
                    "rules": list(decision.rules),
                }
                for decision in self.decisions
            ],
            "files": [trace.path for trace in self.files],
        }


def load_mapping(path: str | Path) -> Mapping:
    text = Path(path).read_text(encoding="utf-8")
    if not str(path).endswith(".toml"):
        raise ValueError("mapping must be a TOML file")
    data = tomllib.loads(text)
    if not isinstance(data, dict):
        raise ValueError("mapping must be a TOML table")
    return parse_mapping(data)


def parse_mapping(data: dict) -> Mapping:
    if data.get("version") != 1:
        raise ValueError(f"unsupported mapping version: {data.get('version')!r}")

    jobs = _string_tuple(data, "jobs")
    if len(jobs) != len(set(jobs)):
        raise ValueError("duplicate job name")
    job_set = frozenset(jobs)
    rules = _parse_rules(data.get("rules"), job_set)
    return Mapping(
        jobs=job_set,
        rules=rules,
        by_number={rule.number: rule for rule in rules},
        requires=_parse_requires(data, job_set),
    )


def resolve(
    changed_files: list[str] | None,
    mapping: Mapping,
    labels: list[str] | None = None,
    *,
    shadow: bool = True,
) -> Selection:
    """Classify mapping.jobs by walking mapping.rules from top to bottom."""
    files = _dedupe(changed_files) if changed_files else []
    label_set = set(labels or [])
    votes = {job: _Votes() for job in mapping.jobs}
    matched: list[int] = []

    for rule in mapping.rules:
        if rule.when == "remaining":
            pending = [
                job
                for job, vote in votes.items()
                if not vote.run_rules and not vote.skip_rules
            ]
            if not pending:
                continue
            matched.append(rule.number)
            _record(votes, pending, "skip", rule.number)
            continue
        if not _rule_matches(rule, files, label_set):
            continue
        matched.append(rule.number)
        if rule.run_all or rule.run:
            _record(votes, _targets(rule, "run", mapping.jobs), "run", rule.number)
        if rule.skip_all or rule.skip:
            _record(votes, _targets(rule, "skip", mapping.jobs), "skip", rule.number)

    run: set[str] = set()
    skip: set[str] = set()
    unsure: set[str] = set()
    decisions: list[Decision] = []
    conflicts: list[Conflict] = []
    for job in sorted(mapping.jobs):
        vote = votes[job]
        if vote.run_rules and vote.skip_rules:
            conflicts.append(
                Conflict(job, tuple(vote.run_rules), tuple(vote.skip_rules))
            )
            run.add(job)
            decisions.append(Decision(job, "run", tuple(vote.run_rules)))
        elif vote.run_rules:
            run.add(job)
            decisions.append(Decision(job, "run", tuple(vote.run_rules)))
        elif vote.skip_rules:
            skip.add(job)
            decisions.append(Decision(job, "skip", tuple(vote.skip_rules)))
        else:
            unsure.add(job)

    run_set, skip_set, unsure_set, required = _apply_requires(
        frozenset(run), frozenset(skip), frozenset(unsure), mapping.requires
    )
    execute = mapping.jobs if shadow else run_set
    return Selection(
        run=run_set,
        skip=skip_set,
        unsure=unsure_set,
        execute=execute,
        reason="rules",
        shadow=shadow,
        files=tuple(FileTrace(path) for path in files),
        required_runs=required,
        matched_rules=tuple(matched),
        decisions=tuple(decisions),
        conflicts=tuple(conflicts),
    )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Classify CI jobs for a diff.")
    parser.add_argument("--mapping", required=True, help="Path to the rule list TOML")
    parser.add_argument("--labels", default="", help="Comma-separated PR labels")
    parser.add_argument("--format", choices=("json", "human"), default="json")
    parser.add_argument(
        "--enforce",
        action="store_true",
        help="Execute run only, and drop skip. The default is shadow mode.",
    )
    parser.add_argument("files", nargs="*", help="Changed paths. '-' reads stdin.")
    args = parser.parse_args(argv)

    files = list(args.files)
    if files == ["-"]:
        files = [line.strip() for line in sys.stdin if line.strip()]
    labels = [label for label in args.labels.split(",") if label]
    mapping = load_mapping(args.mapping)
    selection = resolve(files, mapping, labels, shadow=not args.enforce)
    if args.format == "json":
        json.dump(selection.to_dict(), sys.stdout, indent=2)
        sys.stdout.write("\n")
    else:
        _print_human(selection, mapping)
    return 0


def _record(
    votes: dict[str, _Votes],
    jobs: frozenset[str] | list[str],
    opinion: str,
    number: int,
) -> None:
    for job in jobs:
        vote = votes[job]
        bucket = vote.run_rules if opinion == "run" else vote.skip_rules
        if number not in bucket:
            bucket.append(number)


def _targets(rule: Rule, opinion: str, jobs: frozenset[str]) -> frozenset[str]:
    if opinion == "run":
        return jobs if rule.run_all else rule.run
    return jobs if rule.skip_all else rule.skip


def _rule_matches(rule: Rule, files: list[str], labels: set[str]) -> bool:
    if rule.when == "always":
        return True
    if rule.when == "no-files":
        return not files
    if rule.when == "label":
        return rule.label in labels
    if rule.when == "any":
        return any(_matches(path, rule.paths) for path in files)
    if rule.when == "all":
        return bool(files) and all(_matches(path, rule.paths) for path in files)
    return False


def _matches(path: str, patterns: tuple[re.Pattern[str], ...]) -> bool:
    return any(pattern.search(path) for pattern in patterns)


def _apply_requires(
    run: frozenset[str],
    skip: frozenset[str],
    unsure: frozenset[str],
    requires: dict[str, tuple[str, ...]],
) -> tuple[frozenset[str], frozenset[str], frozenset[str], frozenset[str]]:
    """A running job pulls in the jobs it requires. A skip does not."""
    running = set(run)
    skipped = set(skip)
    unknown = set(unsure)
    pulled: set[str] = set()
    changed = True
    while changed:
        changed = False
        for job in list(running):
            for needed in requires.get(job, ()):
                if needed in running:
                    continue
                running.add(needed)
                skipped.discard(needed)
                unknown.discard(needed)
                pulled.add(needed)
                changed = True
    return frozenset(running), frozenset(skipped), frozenset(unknown), frozenset(pulled)


def _parse_rules(raw: object, jobs: frozenset[str]) -> tuple[Rule, ...]:
    if not isinstance(raw, list) or not raw:
        raise ValueError("rules must be a non-empty list")
    rules: list[Rule] = []
    names: set[str] = set()
    for index, body in enumerate(raw, start=1):
        if not isinstance(body, dict):
            raise ValueError(f"rule {index} must be a table")
        rule = _parse_rule(body, index, jobs)
        if rule.name in names:
            raise ValueError(f"duplicate rule name {rule.name}")
        names.add(rule.name)
        rules.append(rule)
    last = rules[-1]
    if last.when != "remaining" or not last.skip_all or last.run or last.run_all:
        raise ValueError("the last rule must skip every job no earlier rule decided")
    return tuple(rules)


def _parse_rule(body: dict, index: int, jobs: frozenset[str]) -> Rule:
    number = body.get("number")
    if number != index:
        raise ValueError(f"rule {index} must be numbered {index}")
    name = body.get("name")
    if not isinstance(name, str) or not name:
        raise ValueError(f"rule {index} needs a name")
    when = body.get("when")
    if when not in _WHEN:
        raise ValueError(f"rule {number} {name} has unknown when: {when!r}")
    if "unsure" in body:
        raise ValueError(f"rule {number} {name} cannot say unsure")
    patterns = _optional_strings(body, "paths", number, name)
    label = body.get("label", "")
    if not isinstance(label, str):
        raise ValueError(f"rule {number} {name} label must be a string")
    _check_when_fields(when, patterns, label, number, name)
    run, run_all = _job_list(body, "run", number, name, jobs)
    skip, skip_all = _job_list(body, "skip", number, name, jobs)
    if not run and not run_all and not skip and not skip_all:
        raise ValueError(f"rule {number} {name} must run or skip a job")
    overlap = run & skip
    if overlap or (run_all and skip_all):
        raise ValueError(f"rule {number} {name} both runs and skips a job")
    if run_all and skip or skip_all and run:
        raise ValueError(f"rule {number} {name} both runs and skips a job")
    return Rule(
        number=number,
        name=name,
        when=when,
        paths=tuple(_compile(pattern, f"rule {number} {name}") for pattern in patterns),
        path_patterns=patterns,
        label=label,
        run=run,
        skip=skip,
        run_all=run_all,
        skip_all=skip_all,
    )


def _check_when_fields(
    when: str,
    patterns: tuple[str, ...],
    label: str,
    number: int,
    name: str,
) -> None:
    where = f"rule {number} {name}"
    if when in {"any", "all"} and not patterns:
        raise ValueError(f"{where} needs paths")
    if when == "label" and not label:
        raise ValueError(f"{where} needs a label")
    if when not in {"any", "all"} and patterns:
        raise ValueError(f"{where} does not take paths")
    if when != "label" and label:
        raise ValueError(f"{where} does not take a label")


def _job_list(
    body: dict, key: str, number: int, name: str, jobs: frozenset[str]
) -> tuple[frozenset[str], bool]:
    raw = body.get(key, [])
    if raw is None:
        raw = []
    if not isinstance(raw, list) or not all(isinstance(item, str) for item in raw):
        raise ValueError(f"rule {number} {name} {key} must be a list of jobs")
    if not raw:
        return frozenset(), False
    if "*" in raw:
        if raw != ["*"]:
            raise ValueError(f"rule {number} {name} {key} mixes * with job names")
        return frozenset(), True
    chosen = frozenset(raw)
    unknown = chosen - jobs
    if unknown:
        listed = ", ".join(sorted(unknown))
        raise ValueError(f"unknown job {listed} in rule {number} {name}")
    if len(raw) != len(chosen):
        raise ValueError(f"rule {number} {name} repeats a job in {key}")
    return chosen, False


def _parse_requires(data: dict, jobs: frozenset[str]) -> dict[str, tuple[str, ...]]:
    raw = data.get("requires", {})
    if raw is None:
        raw = {}
    if not isinstance(raw, dict):
        raise ValueError("requires must be a table")
    requires: dict[str, tuple[str, ...]] = {}
    for job, needed in raw.items():
        if job not in jobs:
            raise ValueError(f"unknown job {job} in requires")
        if not isinstance(needed, list) or not needed:
            raise ValueError(f"{job} requires a non-empty list")
        names = tuple(str(item) for item in needed)
        if len(names) != len(set(names)):
            raise ValueError(f"duplicate requirement for {job}")
        for name in names:
            if name not in jobs:
                raise ValueError(f"unknown job {name} required by {job}")
            if name == job:
                raise ValueError(f"{job} requires itself")
        requires[str(job)] = names
    _detect_require_cycles(requires)
    return requires


def _detect_require_cycles(requires: dict[str, tuple[str, ...]]) -> None:
    visiting: set[str] = set()
    visited: set[str] = set()

    def walk(name: str) -> None:
        if name in visited:
            return
        if name in visiting:
            raise ValueError(f"requires cycle at {name}")
        visiting.add(name)
        for needed in requires.get(name, ()):
            walk(needed)
        visiting.remove(name)
        visited.add(name)

    for name in requires:
        walk(name)


def _optional_strings(body: dict, key: str, number: int, name: str) -> tuple[str, ...]:
    raw = body.get(key, [])
    if raw is None:
        raw = []
    if not isinstance(raw, list) or not all(isinstance(item, str) for item in raw):
        raise ValueError(f"rule {number} {name} {key} must be a list of strings")
    return tuple(raw)


def _string_tuple(data: dict, key: str) -> tuple[str, ...]:
    value = data.get(key)
    if not isinstance(value, list) or not all(isinstance(item, str) for item in value):
        raise ValueError(f"{key} must be a list of strings")
    return tuple(value)


def _compile(pattern: str, where: str) -> re.Pattern[str]:
    try:
        return re.compile(pattern)
    except re.error as exc:
        raise ValueError(f"invalid pattern {pattern!r} in {where}") from exc


def _dedupe(files: list[str]) -> list[str]:
    seen: set[str] = set()
    unique: list[str] = []
    for name in files:
        if name in seen:
            continue
        seen.add(name)
        unique.append(name)
    return unique


def _print_human(selection: Selection, mapping: Mapping) -> None:
    print("Rules:")
    if not selection.matched_rules:
        print("  none")
    for number in selection.matched_rules:
        rule = mapping.by_number[number]
        print(f"  {number} {rule.name}")
        print(f"    {rule.condition()}")
    print("Conflicts:")
    if not selection.conflicts:
        print("  none")
    for conflict in selection.conflicts:
        print(f"  {conflict.job}")
        print("    run wins")
    print(f"Run ({len(selection.run)}):")
    _print_jobs(selection.run, selection, mapping)
    print(f"Skip ({len(selection.skip)}):")
    _print_jobs(selection.skip, selection, mapping)


def _print_jobs(jobs: frozenset[str], selection: Selection, mapping: Mapping) -> None:
    if not jobs:
        print("  -")
        return
    for job in sorted(jobs):
        decision = selection.decision_for(job)
        if job in selection.required_runs:
            note = "required by a running job"
        elif decision is None:
            note = "no rule"
        else:
            note = ", ".join(
                f"{number} {mapping.by_number[number].name}" for number in decision.rules
            )
        print(f"  {job} ({note})")


if __name__ == "__main__":
    sys.exit(main())
