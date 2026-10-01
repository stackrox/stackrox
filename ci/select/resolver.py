#!/usr/bin/env python3
"""Decide which CI jobs a pull request runs.

Rules live in a TOML file and are walked from top to bottom. A target that
one rule runs and another skips runs: a clash must not drop a job a rule
asked for. A target no rule has mentioned takes the last rule. When that
rule says default, decision-defaults supplies the decision.
"""

from __future__ import annotations

import argparse
import re
import sys
import tomllib
from dataclasses import dataclass, field
from datetime import datetime
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
        "every-file-matches",
        "any-file-matches",
        "always",
        "label-exists",
        "no-file-changed",
        "remaining",
    }
)
_PATH_WHEN = frozenset({"every-file-matches", "any-file-matches"})
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
        """opinion returns the run, skip, or default hits recorded for this target."""
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
            outcomes[job] = "run"
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
        actions = _rule_actions(rule, mapping.jobs, votes, clashes)
        # A clash line names the rules and leaves out the files and the label.
        matched = _match_event(rule, paths) if _rule_matched(rule, votes) else None
        if not actions and not matched:
            continue
        if matched:
            events.append(matched)
        events.extend(_rule_event(rule.name, verb, job) for verb, job in actions)

    for job in clashes:
        vote = votes[job]
        events.append(
            f"clash on target {_q(job)}; "
            f"{_says(_rule_names(vote.run), 'run')}, "
            f"{_says(_rule_names(vote.skip), 'skip')}; "
            "run wins"
        )

    for job in from_default:
        if job in running and job not in originally_run:
            continue
        rule_name = _rule_names(votes[job].default)[0]
        events.append(
            f"rule {_q(rule_name)} takes default {_q(outcomes[job])} for target {_q(job)}"
        )

    for needed, because in added:
        if needed in originally_run:
            continue
        events.append(
            f"added target {_q(needed)} because target {_q(because)} requires it"
        )

    chosen: list[str] = []
    for job in mapping.jobs:
        if job not in originally_run:
            continue
        chosen.append(job)
    seen_added: set[str] = set()
    for needed, _because in added:
        if needed in originally_run or needed in seen_added:
            continue
        seen_added.add(needed)
        chosen.append(needed)

    decision_text = format_decision(chosen)
    log_text = _format_log(repo, pr, commit, events)
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
    """_record_opinions stores this rule's run, skip, or default on each named target."""
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
    """_named_targets lists the jobs this rule mentions. A star means every job."""
    if rule.run_all or rule.skip_all or rule.default_all:
        return list(jobs)
    named = rule.run | rule.skip | rule.default
    return [job for job in jobs if job in named]


def _match_reasons(rule: Rule, files: list[str], labels: set[str]) -> list[str] | None:
    """_match_reasons returns a hit when the rule applies, or None when it does not.

    The log prints the files once, so this does not repeat them onto each target.
    """
    if rule.when == "always":
        return [""]
    if rule.when == "no-file-changed":
        if files:
            return None
        return [""]
    if rule.when == "label-exists":
        if rule.label not in labels:
            return None
        return [""]
    if rule.when == "any-file-matches":
        if any(_matches(path, rule.paths) for path in files):
            return [""]
        return None
    if rule.when == "every-file-matches":
        if files and all(_matches(path, rule.paths) for path in files):
            return [""]
        return None
    return None


def _matches(path: str, patterns: tuple[re.Pattern[str], ...]) -> bool:
    """_matches reports whether one changed path matches any of the rule's patterns."""
    return any(pattern.search(path) for pattern in patterns)


def _format_log(repo: str, pr: str, commit: str, events: list[str]) -> str:
    """_format_log keeps the commit and the run time in the header, and numbers events from 001."""
    when = datetime.now().astimezone().isoformat(timespec="seconds")
    width = max(3, len(str(len(events))))
    body = "".join(f"{index:0{width}d}. {event}\n" for index, event in enumerate(events, start=1))
    return f"# {repo} PR {pr} {commit[:7]}\n# {when}\n{body}"


def _rule_matched(rule: Rule, votes: dict[str, _Vote]) -> bool:
    """_rule_matched reports whether this rule recorded a vote on any target."""
    for vote in votes.values():
        for opinion in _OPINIONS:
            if any(hit.rule == rule.name for hit in vote.opinion(opinion)):
                return True
    return False


def _rule_actions(
    rule: Rule,
    jobs: tuple[str, ...],
    votes: dict[str, _Vote],
    clashes: list[str],
) -> list[tuple[str, str]]:
    """_rule_actions lists this rule's run and skip lines, leaving clashes to their own lines."""
    actions: list[tuple[str, str]] = []
    for job in jobs:
        if job in clashes:
            continue
        for opinion, verb in (("run", "runs"), ("skip", "skips")):
            if any(hit.rule == rule.name for hit in votes[job].opinion(opinion)):
                actions.append((verb, job))
    return actions


def _match_event(rule: Rule, files: list[str]) -> str | None:
    """_match_event states why the rule matched, once, with files grouped by pattern."""
    if rule.when == "label-exists":
        return f"rule {_q(rule.name)} matches because label {_q(rule.label)} is set"
    if rule.when == "no-file-changed":
        return f"rule {_q(rule.name)} matches because the diff has no files"
    if rule.when not in _PATH_WHEN:
        return None
    groups, matched = _file_groups(rule, files)
    if not groups:
        return None
    header = f"rule {_q(rule.name)} matches"
    if rule.when == "every-file-matches":
        header += " every changed file"
    elif matched > 1 and matched == len(files):
        header += f" every changed file ({matched})"
    return header + "\n" + "\n".join(f"     {group}" for group in groups)


def _file_groups(rule: Rule, files: list[str]) -> tuple[list[str], int]:
    """_file_groups buckets matched paths under the first pattern each one hits.

    A lone file is its path. A directory prefix with other files beside it is
    the prefix and a count, including a count of one.
    """
    assigned: list[list[str]] = [[] for _ in rule.paths]
    for path in files:
        for index, pattern in enumerate(rule.paths):
            if pattern.search(path):
                assigned[index].append(path)
                break
    matched = sum(len(paths) for paths in assigned)
    lines: list[str] = []
    for pattern, paths in zip(rule.paths, assigned, strict=True):
        if not paths:
            continue
        kind, label = _pattern_kind(pattern.pattern)
        if kind == "exact":
            lines.append(label)
        elif kind == "prefix" and matched != 1:
            lines.append(f"{label} ({len(paths)})")
        elif len(paths) == 1:
            lines.append(paths[0])
        else:
            lines.extend(paths)
    return lines, matched


def _pattern_kind(source: str) -> tuple[str, str]:
    """_pattern_kind classifies a path pattern as prefix, exact, or other."""
    anchored_end = source.endswith("$")
    body = source[1:] if source.startswith("^") else source
    if anchored_end:
        body = body[:-1]
    literal = _regex_literal(body)
    if literal is None:
        return "other", source
    if source.startswith("^") and anchored_end:
        return "exact", literal
    if source.startswith("^"):
        return "prefix", literal
    return "other", literal


def _regex_literal(body: str) -> str | None:
    """_regex_literal returns the path text when the pattern has no wildcards."""
    out: list[str] = []
    index = 0
    while index < len(body):
        char = body[index]
        if char == "\\":
            if index + 1 >= len(body):
                return None
            out.append(body[index + 1])
            index += 2
            continue
        if char in ".+*?[](){}|":
            return None
        out.append(char)
        index += 1
    return "".join(out)


def _rule_event(rule: str, verb: str, job: str) -> str:
    """_rule_event formats one log line for a rule that runs or skips a target."""
    return f"rule {_q(rule)} {verb} target {_q(job)}"


def _rule_names(hits: list[Hit]) -> list[str]:
    """_rule_names lists the rule names behind these hits, once each, in order."""
    names: list[str] = []
    for hit in hits:
        if hit.rule not in names:
            names.append(hit.rule)
    return names


def _says(names: list[str], opinion: str) -> str:
    """_says joins rule names into a clash phrase, such as rule "docs-only" says skip."""
    quoted = " and ".join(f"rule {_q(name)}" for name in names)
    verb = "says" if len(names) == 1 else "say"
    return f"{quoted} {verb} {opinion}"


def _q(value: str) -> str:
    """_q wraps a value in quotes for the log."""
    escaped = value.replace("\\", "\\\\").replace('"', '\\"')
    return f'"{escaped}"'


def _parse_rules(raw: object, jobs: set[str]) -> tuple[Rule, ...]:
    """_parse_rules reads the rules in order and allows remaining only as the last rule.

    An earlier one votes on targets still open, and a later rule can clash those votes back to run.
    """
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
    for rule in rules[:-1]:
        if rule.when == "remaining":
            raise DecisionError(
                f"rule {rule.number} {rule.name} uses remaining, which only the last rule may use"
            )
    covers = sum((last.run_all, last.skip_all, last.default_all))
    if covers != 1:
        raise DecisionError("the last rule must run, skip, or take the default for every remaining target")
    return tuple(rules)


def _parse_rule(body: dict, index: int, jobs: set[str]) -> Rule:
    """_parse_rule turns one TOML rule table into a Rule."""
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
    """_reject_overlap rejects a rule that gives one job two opinions, or mixes * with names."""
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
    """_check_when_fields requires paths or a label only for the when values that use them."""
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
    """_job_list reads a run, skip, or default list. The second value is true when the list is *."""
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
    """_parse_requires reads which running job pulls other jobs onto the list."""
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
    """_detect_cycles rejects a requires graph that loops."""
    visiting: set[str] = set()
    visited: set[str] = set()

    def walk(name: str) -> None:
        """walk rejects a requires path that returns to a job already on the path."""
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
    """_optional_strings reads a string list that may be absent, such as paths."""
    raw = body.get(key, [])
    if raw is None:
        raw = []
    if not isinstance(raw, list) or not all(isinstance(item, str) for item in raw):
        raise DecisionError(f"rule {index} {name} {key} must be a list of strings")
    return tuple(raw)


def _string_list(data: dict, key: str) -> list[str]:
    """_string_list reads a required list of strings, such as jobs."""
    value = data.get(key)
    if not isinstance(value, list) or not all(isinstance(item, str) for item in value):
        raise DecisionError(f"{key} must be a list of strings")
    return list(value)


def _compile(pattern: str, where: str) -> re.Pattern[str]:
    """_compile turns one path pattern into a regular expression."""
    try:
        return re.compile(pattern)
    except re.error as exc:
        raise DecisionError(f"invalid pattern {pattern!r} in {where}") from exc


def _dedupe(files: list[str]) -> list[str]:
    """_dedupe drops repeated paths and keeps the first occurrence."""
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
