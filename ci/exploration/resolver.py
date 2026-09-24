#!/usr/bin/env python3
"""Classify CI jobs for one diff into run, skip, and unsure.

Shadow mode is the default: the three sets are reported, and execute
still lists every known job. Pass shadow=False (CLI: --enforce) to
execute only run and unsure.

The mapping is data. Adding a domain or a job does not require a new
branch in this module. Requires PyYAML.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import dataclass
from pathlib import Path

import yaml

_OPINIONS = frozenset({"run", "skip", "unsure"})


@dataclass(frozen=True)
class Domain:
    name: str
    path_patterns: tuple[str, ...]
    paths: tuple[re.Pattern[str], ...]
    jobs: dict[str, str]
    extends: str | None


@dataclass(frozen=True)
class Mapping:
    jobs: frozenset[str]
    code_always_run: frozenset[str]
    docs_run: frozenset[str]
    run_all_label: str
    always_run_all_patterns: tuple[str, ...]
    always_run_all: tuple[re.Pattern[str], ...]
    skip_all_patterns: tuple[str, ...]
    skip_all: tuple[re.Pattern[str], ...]
    domains: dict[str, Domain]


@dataclass(frozen=True)
class FileTrace:
    """One changed file and the jobs its own rule explicitly runs."""

    path: str
    kind: str
    domain: str
    explicit_runs: tuple[str, ...]


@dataclass(frozen=True)
class Selection:
    run: frozenset[str]
    skip: frozenset[str]
    unsure: frozenset[str]
    execute: frozenset[str]
    reason: str
    matched_domains: frozenset[str]
    unmatched_files: tuple[str, ...]
    shadow: bool
    files: tuple[FileTrace, ...] = ()

    def to_dict(self) -> dict[str, object]:
        return {
            "shadow": self.shadow,
            "reason": self.reason,
            "run": sorted(self.run),
            "skip": sorted(self.skip),
            "unsure": sorted(self.unsure),
            "execute": sorted(self.execute),
            "matched_domains": sorted(self.matched_domains),
            "unmatched_files": list(self.unmatched_files),
            "files": [
                {
                    "path": trace.path,
                    "kind": trace.kind,
                    "domain": trace.domain,
                    "explicit_runs": list(trace.explicit_runs),
                }
                for trace in self.files
            ],
        }


def load_mapping(path: str | Path) -> Mapping:
    data = yaml.safe_load(Path(path).read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise ValueError("mapping must be a YAML mapping")
    return parse_mapping(data)


def parse_mapping(data: dict) -> Mapping:
    if data.get("version") != 1:
        raise ValueError(f"unsupported mapping version: {data.get('version')!r}")

    jobs = _string_tuple(data, "jobs")
    if len(jobs) != len(set(jobs)):
        raise ValueError("duplicate job name")
    job_set = frozenset(jobs)

    domains = _parse_domains(data.get("domains"), job_set)
    _detect_cycles(domains)

    return Mapping(
        jobs=job_set,
        code_always_run=_subset(data, "code_always_run", job_set),
        docs_run=_subset(data, "docs_run", job_set),
        run_all_label=_required_str(data, "run_all_label"),
        always_run_all_patterns=_string_tuple(data, "always_run_all"),
        always_run_all=_compile_all(data, "always_run_all"),
        skip_all_patterns=_string_tuple(data, "skip_all"),
        skip_all=_compile_all(data, "skip_all"),
        domains=domains,
    )


def resolve(
    changed_files: list[str] | None,
    mapping: Mapping,
    labels: list[str] | None = None,
    *,
    shadow: bool = True,
) -> Selection:
    """Classify mapping.jobs for this diff.

    Precedence: run-all label, missing diff, always_run_all, docs-only,
    an unmatched file, then per-domain opinions. The longest matching
    path wins for a file; a shorter parent does not also vote.
    An unmatched file does not name a test, so every known job is unsure
    and still executed.
    """
    files = _dedupe(changed_files) if changed_files else []
    traces = _traces(files, mapping)

    if mapping.run_all_label in (labels or []):
        return _everything(mapping, reason="label", shadow=shadow, files=traces)

    if not files:
        return _everything(mapping, reason="no-diff", shadow=shadow)

    if any(_matches(path, mapping.always_run_all) for path in files):
        return _everything(mapping, reason="always-run-all", shadow=shadow, files=traces)

    significant = [path for path in files if not _matches(path, mapping.skip_all)]
    if not significant:
        return _docs_only(mapping, shadow=shadow, files=traces)

    per_file: list[dict[str, str]] = []
    matched_domains: list[str] = []
    unmatched: list[str] = []
    for path in significant:
        winners = _winning_domains(path, mapping)
        if not winners:
            unmatched.append(path)
            continue
        matched_domains.extend(domain.name for domain in winners)
        per_file.append(_merge_domain_votes(winners, mapping))

    if unmatched:
        return _selection(
            mapping,
            run=frozenset(),
            skip=frozenset(),
            unsure=mapping.jobs,
            reason="unmatched",
            shadow=shadow,
            unmatched_files=tuple(unmatched),
            files=traces,
        )

    opinions = _merge_vote_dicts(per_file, mapping.jobs)
    for job in mapping.code_always_run:
        opinions[job] = "run"

    run: set[str] = set()
    skip: set[str] = set()
    unsure: set[str] = set()
    for job, opinion in opinions.items():
        if opinion == "run":
            run.add(job)
        elif opinion == "skip":
            skip.add(job)
        else:
            unsure.add(job)

    return _selection(
        mapping,
        run=frozenset(run),
        skip=frozenset(skip),
        unsure=frozenset(unsure),
        reason="domains",
        shadow=shadow,
        matched_domains=frozenset(matched_domains),
        files=traces,
    )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Classify CI jobs for a diff.")
    parser.add_argument("--mapping", required=True, help="Path to the domain mapping YAML")
    parser.add_argument("--labels", default="", help="Comma-separated PR labels")
    parser.add_argument("--format", choices=("json", "human"), default="json")
    parser.add_argument(
        "--enforce",
        action="store_true",
        help="Execute run and unsure, and drop skip. The default is shadow mode.",
    )
    parser.add_argument("files", nargs="*", help="Changed paths. '-' reads stdin.")
    args = parser.parse_args(argv)

    files = list(args.files)
    if files == ["-"]:
        files = [line.strip() for line in sys.stdin if line.strip()]
    labels = [label for label in args.labels.split(",") if label]
    selection = resolve(
        files,
        load_mapping(args.mapping),
        labels,
        shadow=not args.enforce,
    )
    if args.format == "json":
        json.dump(selection.to_dict(), sys.stdout, indent=2)
        sys.stdout.write("\n")
    else:
        _print_human(selection)
    return 0


def _everything(
    mapping: Mapping,
    *,
    reason: str,
    shadow: bool,
    files: tuple[FileTrace, ...] = (),
    unmatched: tuple[str, ...] = (),
) -> Selection:
    return _selection(
        mapping,
        run=mapping.jobs,
        skip=frozenset(),
        unsure=frozenset(),
        reason=reason,
        shadow=shadow,
        unmatched_files=unmatched,
        files=files,
    )


def _docs_only(
    mapping: Mapping,
    *,
    shadow: bool,
    files: tuple[FileTrace, ...],
) -> Selection:
    return _selection(
        mapping,
        run=mapping.docs_run,
        skip=mapping.jobs - mapping.docs_run,
        unsure=frozenset(),
        reason="docs-only",
        shadow=shadow,
        files=files,
    )


def _selection(
    mapping: Mapping,
    *,
    run: frozenset[str],
    skip: frozenset[str],
    unsure: frozenset[str],
    reason: str,
    shadow: bool,
    matched_domains: frozenset[str] = frozenset(),
    unmatched_files: tuple[str, ...] = (),
    files: tuple[FileTrace, ...] = (),
) -> Selection:
    # Shadow mode records the split and still runs every known job.
    execute = mapping.jobs if shadow else run | unsure
    return Selection(
        run=run,
        skip=skip,
        unsure=unsure,
        execute=execute,
        reason=reason,
        matched_domains=matched_domains,
        unmatched_files=unmatched_files,
        shadow=shadow,
        files=files,
    )


def _traces(files: list[str], mapping: Mapping) -> tuple[FileTrace, ...]:
    return tuple(_trace(path, mapping) for path in files)


def _trace(path: str, mapping: Mapping) -> FileTrace:
    if _matches(path, mapping.always_run_all):
        return FileTrace(path, "always-run-all", "", tuple(sorted(mapping.jobs)))
    if _matches(path, mapping.skip_all):
        return FileTrace(path, "docs", "", ())
    winners = _winning_domains(path, mapping)
    if not winners:
        return FileTrace(path, "unmatched", "", ())
    votes = _merge_domain_votes(winners, mapping)
    explicit = tuple(sorted(job for job, opinion in votes.items() if opinion == "run"))
    domain = ",".join(sorted(winner.name for winner in winners))
    return FileTrace(path, "domain", domain, explicit)


def _winning_domains(path: str, mapping: Mapping) -> list[Domain]:
    """Domains with the longest match. A shorter parent does not also vote."""
    scored: list[tuple[int, Domain]] = []
    for domain in mapping.domains.values():
        best = -1
        for pattern in domain.paths:
            match = pattern.search(path)
            if match is not None:
                best = max(best, match.end() - match.start())
        if best >= 0:
            scored.append((best, domain))
    if not scored:
        return []
    top = max(score for score, _ in scored)
    return [domain for score, domain in scored if score == top]


def _merge_domain_votes(domains: list[Domain], mapping: Mapping) -> dict[str, str]:
    votes = [_opinions_for(domain, mapping) for domain in domains]
    return _merge_vote_dicts(votes, mapping.jobs)


def _opinions_for(domain: Domain, mapping: Mapping) -> dict[str, str]:
    """Parent opinions, then this domain. Child keys replace parent keys."""
    opinions: dict[str, str] = {}
    if domain.extends is not None:
        opinions.update(_opinions_for(mapping.domains[domain.extends], mapping))
    opinions.update(domain.jobs)
    return opinions


def _merge_vote_dicts(votes: list[dict[str, str]], jobs: frozenset[str]) -> dict[str, str]:
    merged: dict[str, str] = {}
    for job in jobs:
        merged[job] = _merge_votes([vote.get(job) for vote in votes])
    return merged


def _merge_votes(votes: list[str | None]) -> str:
    """Run wins. Skip counts only when every vote says skip.

    A missing vote is unsure: a domain that never mentions a job has
    not agreed to drop it.
    """
    if any(vote == "run" for vote in votes):
        return "run"
    if any(vote is None or vote == "unsure" for vote in votes):
        return "unsure"
    return "skip"


def _matches(path: str, patterns: tuple[re.Pattern[str], ...]) -> bool:
    return any(pattern.search(path) for pattern in patterns)


def _dedupe(files: list[str]) -> list[str]:
    seen: set[str] = set()
    unique: list[str] = []
    for name in files:
        if name in seen:
            continue
        seen.add(name)
        unique.append(name)
    return unique


def _parse_domains(raw: object, jobs: frozenset[str]) -> dict[str, Domain]:
    if not isinstance(raw, dict):
        raise ValueError("domains must be a mapping")
    domains: dict[str, Domain] = {}
    for name, body in raw.items():
        if not isinstance(name, str) or not isinstance(body, dict):
            raise ValueError(f"domain {name!r} must be a mapping")
        patterns = _string_tuple(body, "paths")
        if not patterns:
            raise ValueError(f"domain {name} has no paths")
        extends = body.get("extends")
        if extends is not None and not isinstance(extends, str):
            raise ValueError(f"domain {name} extends must be a string")
        domains[name] = Domain(
            name=name,
            path_patterns=patterns,
            paths=tuple(_compile(pattern, f"domain {name}") for pattern in patterns),
            jobs=_parse_opinions(body.get("jobs", {}), name, jobs),
            extends=extends,
        )
    return domains


def _parse_opinions(raw: object, domain: str, jobs: frozenset[str]) -> dict[str, str]:
    if not isinstance(raw, dict):
        raise ValueError(f"domain {domain} jobs must be a mapping")
    opinions: dict[str, str] = {}
    for job, opinion in raw.items():
        if job not in jobs:
            raise ValueError(f"unknown job {job} in domain {domain}")
        if opinion not in _OPINIONS:
            raise ValueError(f"unknown opinion {opinion} for job {job}")
        opinions[str(job)] = str(opinion)
    return opinions


def _detect_cycles(domains: dict[str, Domain]) -> None:
    visiting: set[str] = set()
    visited: set[str] = set()

    def walk(name: str) -> None:
        if name in visited:
            return
        if name in visiting:
            raise ValueError(f"domain extends cycle at {name}")
        visiting.add(name)
        parent = domains[name].extends
        if parent is not None:
            if parent not in domains:
                raise ValueError(f"domain {name} extends unknown domain {parent}")
            walk(parent)
        visiting.remove(name)
        visited.add(name)

    for name in domains:
        walk(name)


def _subset(data: dict, key: str, jobs: frozenset[str]) -> frozenset[str]:
    chosen = frozenset(_string_tuple(data, key))
    unknown = chosen - jobs
    if unknown:
        listed = ", ".join(sorted(unknown))
        raise ValueError(f"unknown job {listed} in {key}")
    return chosen


def _string_tuple(data: dict, key: str) -> tuple[str, ...]:
    value = data.get(key)
    if not isinstance(value, list) or not all(isinstance(item, str) for item in value):
        raise ValueError(f"{key} must be a list of strings")
    return tuple(value)


def _required_str(data: dict, key: str) -> str:
    value = data.get(key)
    if not isinstance(value, str) or not value:
        raise ValueError(f"{key} must be a non-empty string")
    return value


def _compile_all(data: dict, key: str) -> tuple[re.Pattern[str], ...]:
    return tuple(_compile(pattern, key) for pattern in _string_tuple(data, key))


def _compile(pattern: str, where: str) -> re.Pattern[str]:
    try:
        return re.compile(pattern)
    except re.error as err:
        raise ValueError(f"invalid pattern {pattern!r} in {where}: {err}") from err


def _print_human(selection: Selection) -> None:
    if selection.shadow:
        print("Shadow mode. This report does not skip any other CI job.")
        print("Unsure jobs still run. Skip is only a prediction.")
    else:
        print("Enforce mode. Execute is run plus unsure. Skip is dropped.")
    print()
    print("Summary")
    print(f"  reason: {selection.reason}")
    print(f"  {_reason_sentence(selection)}")
    explicit = {job for trace in selection.files for job in trace.explicit_runs}
    noted = sorted(selection.run - explicit) if selection.reason == "domains" else []
    _print_bucket("run", selection.run, note_jobs=noted)
    _print_bucket("skip", selection.skip)
    _print_bucket("unsure", selection.unsure)
    if selection.shadow:
        print("  execute: every job above (shadow mode runs run, skip, and unsure)")
    else:
        _print_bucket("execute", selection.execute)
    print()
    print("Changed files")
    if not selection.files:
        print("  (no file list)")
        return
    for trace in selection.files:
        print(f"  {trace.path}")
        print(f"    {_trace_sentence(trace)}")
        _print_bucket("explicit runs", trace.explicit_runs, indent="    ")


def _reason_sentence(selection: Selection) -> str:
    sentences = {
        "label": "The ci-run-all-tests label is set, so every known job runs.",
        "no-diff": "The diff was missing, so every known job runs.",
        "always-run-all": "A changed file matches a run-everything pattern, so every known job runs.",
        "docs-only": "Every changed file is documentation, so only the docs jobs run.",
        "unmatched": "A changed file matches no rule. It names no test, so every known job is unsure and all of them run.",
        "domains": "Each file voted. One run is enough. Skip sticks only when every matched file says skip.",
    }
    return sentences.get(selection.reason, selection.reason)


def _trace_sentence(trace: FileTrace) -> str:
    if trace.kind == "docs":
        return "documentation or changelog; ignored when other files change code"
    if trace.kind == "unmatched":
        return "no rule matches this path"
    if trace.kind == "always-run-all":
        return "matches a run-everything pattern"
    if trace.domain:
        return f"domain {trace.domain}"
    return trace.kind


def _print_bucket(
    label: str,
    jobs: frozenset[str] | tuple[str, ...],
    *,
    note_jobs: list[str] | None = None,
    indent: str = "  ",
) -> None:
    names = list(jobs) if isinstance(jobs, tuple) else sorted(jobs)
    notes = set(note_jobs or [])
    print(f"{indent}{label}:")
    if not names:
        print(f"{indent}  -")
        return
    for name in names:
        suffix = " (every code change)" if name in notes else ""
        print(f"{indent}  {name}{suffix}")


if __name__ == "__main__":
    sys.exit(main())
