#!/usr/bin/env python3
"""Decide which CI jobs a pull request runs.

Rules live in a TOML file and are walked from top to bottom. A target that
one rule runs and another skips takes its line from decision-defaults.
A target no rule has mentioned takes the last rule. When that rule says
default, the same file supplies the decision.
"""

from __future__ import annotations

import argparse
import re
import sys
import tomllib
from dataclasses import dataclass, field
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from decisionlib import (  # noqa: E402
    DecisionError,
    check_defaults,
    decision_from_defaults,
    format_decision,
    parse_defaults,
    read_list,
)

_WHEN = frozenset(
    {
        "all-files-changed",
        "any-file-changed",
        "always",
        "label-exists",
        "no-file-changed",
        "remaining",
    }
)
_PATH_WHEN = frozenset({"all-files-changed", "any-file-changed"})
_OPINIONS = ("run", "skip", "default")


@dataclass(frozen=True)
class Rule:
    number: int
    name: str
    when: str
    paths: tuple[re.Pattern[str], ...]
    label: str
    run: frozenset[str]
    skip: frozenset[str]
    default: frozenset[str]
    run_all: bool
    skip_all: bool
    default_all: bool


@dataclass(frozen=True)
class Mapping:
    jobs: tuple[str, ...]
    rules: tuple[Rule, ...]
    requires: dict[str, tuple[str, ...]]


@dataclass(frozen=True)
class Hit:
    rule: str
    reason: str


@dataclass
class _Vote:
    run: list[Hit] = field(default_factory=list)
    skip: list[Hit] = field(default_factory=list)
    default: list[Hit] = field(default_factory=list)

    def opinion(self, name: str) -> list[Hit]:
        return getattr(self, name)


@dataclass(frozen=True)
class Result:
    """Result is the decision text and the log that explains it."""

    decision_text: str
    log_text: str
    jobs: frozenset[str]


def load_mapping(path: str | Path) -> Mapping:
    """load_mapping reads the rule file."""
    data = tomllib.loads(Path(path).read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise DecisionError("rules must be a TOML table")
    return parse_mapping(data)


def parse_mapping(data: dict) -> Mapping:
    """parse_mapping checks the rule file and returns the rules in order."""
    if data.get("version") != 1:
        raise DecisionError(f"unsupported rules version: {data.get('version')!r}")
    jobs = _string_list(data, "jobs")
    if len(jobs) != len(set(jobs)):
        raise DecisionError("duplicate job name")
    if not jobs:
        raise DecisionError("jobs must list at least one target")
    job_set = set(jobs)
    rules = _parse_rules(data.get("rules"), job_set)
    return Mapping(jobs=tuple(jobs), rules=rules, requires=_parse_requires(data, job_set))


def resolve(
    files: list[str],
    labels: list[str],
    mapping: Mapping,
    defaults: dict[str, str],
    *,
    repo: str,
    pr: str,
    commit: str,
) -> Result:
    """resolve walks the rules and returns the jobs to run, plus the log."""
    check_defaults(list(mapping.jobs), defaults)
    paths = _dedupe(files)
    label_set = set(labels)
    short = commit[:7]
    votes = {job: _Vote() for job in mapping.jobs}

    for rule in mapping.rules:
        if rule.when == "remaining":
            pending = [
                job
                for job in mapping.jobs
                if not votes[job].run and not votes[job].skip and not votes[job].default
            ]
            _record_opinions(votes, rule, pending, [""])
            continue
        reasons = _match_reasons(rule, paths, label_set)
        if reasons is None:
            continue
        _record_opinions(votes, rule, _named_targets(rule, mapping.jobs), reasons)

    outcomes: dict[str, str] = {}
    clashes: list[str] = []
    from_default: list[str] = []
    for job in mapping.jobs:
        vote = votes[job]
        if vote.run and vote.skip:
            outcomes[job] = defaults[job]
            clashes.append(job)
        elif vote.run:
            outcomes[job] = "run"
        elif vote.skip:
            outcomes[job] = "skip"
        elif vote.default:
            outcomes[job] = defaults[job]
            from_default.append(job)
        else:
            raise DecisionError(f"{job} has no decision")

    originally_run = {job for job, opinion in outcomes.items() if opinion == "run"}
    running = set(originally_run)
    added: list[tuple[str, str]] = []
    added_by: dict[str, list[str]] = {}
    changed = True
    while changed:
        changed = False
        for job in mapping.jobs:
            if job not in running:
                continue
            for needed in mapping.requires.get(job, ()):
                if needed in running:
                    continue
                running.add(needed)
                outcomes[needed] = "run"
                added.append((needed, job))
                added_by.setdefault(needed, []).append(job)
                changed = True

    events: list[str] = []
    for rule in mapping.rules:
        if rule.when == "remaining":
            continue
        for job in mapping.jobs:
            if job in clashes:
                continue
            for opinion, verb in (("run", "runs"), ("skip", "skips")):
                for hit in votes[job].opinion(opinion):
                    if hit.rule != rule.name:
                        continue
                    events.append(_rule_event(short, rule.name, verb, job, hit.reason))

    for job in clashes:
        vote = votes[job]
        events.append(
            f"{short}: clash on target {_q(job)}; "
            f"{_says(_rule_names(vote.run), 'run')}, "
            f"{_says(_rule_names(vote.skip), 'skip')}; "
            f"default {_q(defaults[job])}"
        )

    for job in from_default:
        if job in running and job not in originally_run:
            continue
        rule_name = _rule_names(votes[job].default)[0]
        events.append(
            f"{short}: rule {_q(rule_name)} takes default {_q(outcomes[job])} for target {_q(job)}"
        )

    for needed, because in added:
        if needed in originally_run:
            continue
        events.append(
            f"{short}: added target {_q(needed)} because target {_q(because)} requires it"
        )

    items: list[tuple[str, str]] = []
    for job in mapping.jobs:
        if job not in originally_run:
            continue
        items.append((_run_comment(job, votes[job], clashes, from_default), job))
    seen_added: set[str] = set()
    for needed, _because in added:
        if needed in originally_run or needed in seen_added:
            continue
        seen_added.add(needed)
        sources = ", ".join(added_by[needed])
        items.append((f"prerequisite of {sources}", needed))

    decision_text = format_decision(items)
    log_text = f"# {repo} PR {pr}\n" + "".join(f"{event}\n" for event in events)
    return Result(decision_text=decision_text, log_text=log_text, jobs=frozenset(running))


def settle(
    files: list[str] | None,
    labels: list[str],
    mapping: Mapping,
    defaults: dict[str, str],
    *,
    repo: str,
    pr: str,
    commit: str,
) -> tuple[str, str]:
    """settle returns decision text and a log.

    files is None when the changed-file list could not be read. That is not
    a decision from the rules, so the defaults file is used. The same happens
    when a rule file is wrong or a target has no default.
    """
    try:
        if files is None:
            raise DecisionError("changed files are unknown")
        result = resolve(
            files, labels, mapping, defaults, repo=repo, pr=pr, commit=commit
        )
    except DecisionError as exc:
        return decision_from_defaults(defaults), f"resolver failed: {exc}\nusing decision-defaults\n"
    return result.decision_text, result.log_text


def open_decision(
    rules_path: str | Path,
    defaults_path: str | Path,
    files: list[str] | None,
    labels: list[str],
    *,
    repo: str,
    pr: str,
    commit: str,
) -> tuple[str, str, dict[str, str]]:
    """open_decision reads the rule files and returns decision text, log, and defaults.

    The defaults dict is returned even when the rules fail, so a dispatcher
    can still tell which targets this project owns.
    """
    defaults = parse_defaults(Path(defaults_path).read_text(encoding="utf-8"))
    try:
        mapping = load_mapping(rules_path)
    except (OSError, DecisionError, tomllib.TOMLDecodeError) as exc:
        text = decision_from_defaults(defaults)
        return text, f"resolver failed: {exc}\nusing decision-defaults\n", defaults
    text, log = settle(
        files, labels, mapping, defaults, repo=repo, pr=str(pr), commit=commit
    )
    return text, log, defaults


def main(argv: list[str] | None = None) -> int:
    """main writes the decision file and the log, or reports an error."""
    parser = argparse.ArgumentParser(description="Decide which CI jobs a pull request runs.")
    parser.add_argument("--rules", required=True)
    parser.add_argument("--defaults", required=True)
    parser.add_argument("--labels-file", required=True)
    parser.add_argument("--files-file", required=True)
    parser.add_argument("--repo", required=True)
    parser.add_argument("--pr", required=True)
    parser.add_argument("--commit", required=True)
    parser.add_argument("--decision-out", required=True)
    parser.add_argument("--log-out", required=True)
    args = parser.parse_args(argv)

    try:
        mapping = load_mapping(args.rules)
        defaults = parse_defaults(Path(args.defaults).read_text(encoding="utf-8"))
        labels = read_list(Path(args.labels_file).read_text(encoding="utf-8"))
        files = read_list(Path(args.files_file).read_text(encoding="utf-8"))
        result = resolve(
            files,
            labels,
            mapping,
            defaults,
            repo=args.repo,
            pr=str(args.pr),
            commit=args.commit,
        )
    except (OSError, DecisionError, tomllib.TOMLDecodeError) as exc:
        print(f"resolver failed: {exc}", file=sys.stderr)
        return 1

    Path(args.decision_out).write_text(result.decision_text, encoding="utf-8")
    Path(args.log_out).write_text(result.log_text, encoding="utf-8")
    return 0


def _record_opinions(
    votes: dict[str, _Vote],
    rule: Rule,
    jobs: list[str],
    reasons: list[str],
) -> None:
    for opinion in _OPINIONS:
        if opinion == "run":
            chosen = jobs if rule.run_all else [job for job in jobs if job in rule.run]
        elif opinion == "skip":
            chosen = jobs if rule.skip_all else [job for job in jobs if job in rule.skip]
        else:
            chosen = jobs if rule.default_all else [job for job in jobs if job in rule.default]
        for job in chosen:
            bucket = votes[job].opinion(opinion)
            for reason in reasons:
                bucket.append(Hit(rule.name, reason))


def _named_targets(rule: Rule, jobs: tuple[str, ...]) -> list[str]:
    if rule.run_all or rule.skip_all or rule.default_all:
        return list(jobs)
    named = rule.run | rule.skip | rule.default
    return [job for job in jobs if job in named]


def _match_reasons(rule: Rule, files: list[str], labels: set[str]) -> list[str] | None:
    if rule.when == "always":
        return [""]
    if rule.when == "no-file-changed":
        if files:
            return None
        return ["because the diff has no files"]
    if rule.when == "label-exists":
        if rule.label not in labels:
            return None
        return [f"because label {_q(rule.label)} is set"]
    if rule.when == "any-file-changed":
        matched = [path for path in files if _matches(path, rule.paths)]
        if not matched:
            return None
        return [f"because file {_q(path)} changed" for path in matched]
    if rule.when == "all-files-changed":
        if not files or not all(_matches(path, rule.paths) for path in files):
            return None
        return [f"because file {_q(path)} changed" for path in files]
    return None


def _matches(path: str, patterns: tuple[re.Pattern[str], ...]) -> bool:
    return any(pattern.search(path) for pattern in patterns)


def _rule_event(commit: str, rule: str, verb: str, job: str, reason: str) -> str:
    line = f"{commit}: rule {_q(rule)} {verb} target {_q(job)}"
    if reason:
        return f"{line} {reason}"
    return line


def _rule_names(hits: list[Hit]) -> list[str]:
    names: list[str] = []
    for hit in hits:
        if hit.rule not in names:
            names.append(hit.rule)
    return names


def _says(names: list[str], opinion: str) -> str:
    quoted = " and ".join(f"rule {_q(name)}" for name in names)
    verb = "says" if len(names) == 1 else "say"
    return f"{quoted} {verb} {opinion}"


def _run_comment(job: str, vote: _Vote, clashes: list[str], from_default: list[str]) -> str:
    if job in clashes or job in from_default:
        return "default decision"
    return " and ".join(_rule_names(vote.run))


def _q(value: str) -> str:
    escaped = value.replace("\\", "\\\\").replace('"', '\\"')
    return f'"{escaped}"'


def _parse_rules(raw: object, jobs: set[str]) -> tuple[Rule, ...]:
    if not isinstance(raw, list) or not raw:
        raise DecisionError("rules must be a non-empty list")
    rules: list[Rule] = []
    names: set[str] = set()
    for index, body in enumerate(raw, start=1):
        if not isinstance(body, dict):
            raise DecisionError(f"rule {index} must be a table")
        rule = _parse_rule(body, index, jobs)
        if rule.name in names:
            raise DecisionError(f"duplicate rule name {rule.name}")
        names.add(rule.name)
        rules.append(rule)
    last = rules[-1]
    if last.when != "remaining":
        raise DecisionError("the last rule must decide every target no earlier rule decided")
    covers = sum((last.run_all, last.skip_all, last.default_all))
    if covers != 1:
        raise DecisionError("the last rule must run, skip, or take the default for every remaining target")
    return tuple(rules)


def _parse_rule(body: dict, index: int, jobs: set[str]) -> Rule:
    name = body.get("name")
    if not isinstance(name, str) or not name:
        raise DecisionError(f"rule {index} needs a name")
    when = body.get("when")
    if when not in _WHEN:
        raise DecisionError(f"rule {index} {name} has unknown when: {when!r}")
    patterns = _optional_strings(body, "paths", index, name)
    label = body.get("label", "")
    if not isinstance(label, str):
        raise DecisionError(f"rule {index} {name} label must be a string")
    _check_when_fields(when, patterns, label, index, name)
    run, run_all = _job_list(body, "run", index, name, jobs)
    skip, skip_all = _job_list(body, "skip", index, name, jobs)
    default, default_all = _job_list(body, "default", index, name, jobs)
    if not any((run, run_all, skip, skip_all, default, default_all)):
        raise DecisionError(f"rule {index} {name} must run, skip, or take the default")
    _reject_overlap(run, skip, default, run_all, skip_all, default_all, index, name)
    return Rule(
        number=index,
        name=name,
        when=when,
        paths=tuple(_compile(pattern, f"rule {index} {name}") for pattern in patterns),
        label=label,
        run=run,
        skip=skip,
        default=default,
        run_all=run_all,
        skip_all=skip_all,
        default_all=default_all,
    )


def _reject_overlap(
    run: frozenset[str],
    skip: frozenset[str],
    default: frozenset[str],
    run_all: bool,
    skip_all: bool,
    default_all: bool,
    index: int,
    name: str,
) -> None:
    starred = sum((run_all, skip_all, default_all))
    if starred > 1:
        raise DecisionError(f"rule {index} {name} uses * more than once")
    if starred and (run or skip or default):
        raise DecisionError(f"rule {index} {name} mixes * with job names")
    if run & skip or run & default or skip & default:
        raise DecisionError(f"rule {index} {name} gives one job two opinions")


def _check_when_fields(
    when: str,
    patterns: tuple[str, ...],
    label: str,
    index: int,
    name: str,
) -> None:
    where = f"rule {index} {name}"
    if when in _PATH_WHEN and not patterns:
        raise DecisionError(f"{where} needs paths")
    if when == "label-exists" and not label:
        raise DecisionError(f"{where} needs a label")
    if when not in _PATH_WHEN and patterns:
        raise DecisionError(f"{where} does not take paths")
    if when != "label-exists" and label:
        raise DecisionError(f"{where} does not take a label")


def _job_list(
    body: dict, key: str, index: int, name: str, jobs: set[str]
) -> tuple[frozenset[str], bool]:
    raw = body.get(key, [])
    if raw is None:
        raw = []
    if not isinstance(raw, list) or not all(isinstance(item, str) for item in raw):
        raise DecisionError(f"rule {index} {name} {key} must be a list of jobs")
    if not raw:
        return frozenset(), False
    if "*" in raw:
        if raw != ["*"]:
            raise DecisionError(f"rule {index} {name} {key} mixes * with job names")
        return frozenset(), True
    chosen = frozenset(raw)
    unknown = chosen - jobs
    if unknown:
        raise DecisionError(
            f"unknown job {', '.join(sorted(unknown))} in rule {index} {name}"
        )
    if len(raw) != len(chosen):
        raise DecisionError(f"rule {index} {name} repeats a job in {key}")
    return chosen, False


def _parse_requires(data: dict, jobs: set[str]) -> dict[str, tuple[str, ...]]:
    raw = data.get("requires", {})
    if raw is None:
        raw = {}
    if not isinstance(raw, dict):
        raise DecisionError("requires must be a table")
    requires: dict[str, tuple[str, ...]] = {}
    for job, needed in raw.items():
        if job not in jobs:
            raise DecisionError(f"unknown job {job} in requires")
        if not isinstance(needed, list) or not needed:
            raise DecisionError(f"{job} requires a non-empty list")
        names = tuple(str(item) for item in needed)
        if len(names) != len(set(names)):
            raise DecisionError(f"duplicate requirement for {job}")
        for name in names:
            if name not in jobs:
                raise DecisionError(f"unknown job {name} required by {job}")
            if name == job:
                raise DecisionError(f"{job} requires itself")
        requires[str(job)] = names
    _detect_cycles(requires)
    return requires


def _detect_cycles(requires: dict[str, tuple[str, ...]]) -> None:
    visiting: set[str] = set()
    visited: set[str] = set()

    def walk(name: str) -> None:
        if name in visited:
            return
        if name in visiting:
            raise DecisionError(f"requires cycle at {name}")
        visiting.add(name)
        for needed in requires.get(name, ()):
            walk(needed)
        visiting.remove(name)
        visited.add(name)

    for name in requires:
        walk(name)


def _optional_strings(body: dict, key: str, index: int, name: str) -> tuple[str, ...]:
    raw = body.get(key, [])
    if raw is None:
        raw = []
    if not isinstance(raw, list) or not all(isinstance(item, str) for item in raw):
        raise DecisionError(f"rule {index} {name} {key} must be a list of strings")
    return tuple(raw)


def _string_list(data: dict, key: str) -> list[str]:
    value = data.get(key)
    if not isinstance(value, list) or not all(isinstance(item, str) for item in value):
        raise DecisionError(f"{key} must be a list of strings")
    return list(value)


def _compile(pattern: str, where: str) -> re.Pattern[str]:
    try:
        return re.compile(pattern)
    except re.error as exc:
        raise DecisionError(f"invalid pattern {pattern!r} in {where}") from exc


def _dedupe(files: list[str]) -> list[str]:
    seen: set[str] = set()
    unique: list[str] = []
    for name in files:
        if name in seen:
            continue
        seen.add(name)
        unique.append(name)
    return unique


if __name__ == "__main__":
    sys.exit(main())
