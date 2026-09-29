"""Turn resolver opinions into a run, skip, or unsure plan.

Only opinion run starts a job. The plan includes Prow jobs. A default line
is the GitHub Actions trigger recorded for that job.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
from dataclasses import dataclass
from pathlib import Path

import yaml

from resolver import Mapping, Selection, load_mapping, resolve

ENFORCE_LABEL = "ci-dispatcher-enforce"


@dataclass(frozen=True)
class PullRequest:
    draft: bool
    fork: bool
    labels: frozenset[str]
    files: tuple[str, ...]


@dataclass(frozen=True)
class JobPlan:
    job: str
    opinion: str
    default_starts: bool
    action: str
    # False for a Prow job: gha_defaults.yaml has no trigger to compare.
    has_default: bool = True

    @property
    def would_run(self) -> bool:
        # Unsure does not run. Only an explicit run starts the job.
        return self.action == "start"


def load_defaults(path: str | Path) -> dict[str, dict[str, str]]:
    data = yaml.safe_load(Path(path).read_text())
    jobs = data.get("jobs")
    if not isinstance(jobs, dict) or not jobs:
        raise ValueError("jobs must be a non-empty mapping")
    for name, rule in jobs.items():
        if not isinstance(rule, dict) or "when" not in rule:
            raise ValueError(f"{name} needs a when field")
    return jobs


def dispatch(
    selection: Selection,
    defaults: dict[str, dict[str, str]],
    pull_request: PullRequest,
) -> tuple[JobPlan, ...]:
    # Defaults cover GitHub Actions. The resolver also decides Prow jobs.
    names = set(defaults) | selection.run | selection.skip | selection.unsure
    plans = []
    for job in sorted(names):
        opinion = _opinion(selection, job)
        if opinion == "run":
            action = "start"
        elif opinion == "skip":
            action = "stop"
        else:
            action = "default"
        known = job in defaults
        starts = default_would_start(defaults[job], pull_request) if known else False
        plans.append(JobPlan(job, opinion, starts, action, has_default=known))
    return tuple(plans)


def default_would_start(rule: dict[str, str], pull_request: PullRequest) -> bool:
    """default_would_start reports whether the recorded trigger would start this job."""
    if pull_request.fork and rule.get("skip_fork"):
        return False
    when = rule["when"]
    if when == "always":
        return True
    if when == "not-ui":
        return not _ui_only(pull_request.files)
    if when == "ready":
        return _ready(pull_request)
    if when == "label":
        return rule["label"] in pull_request.labels and not pull_request.fork
    if when == "path":
        return _path_matches(rule["path"], pull_request.files)
    if when == "konflux":
        if "e2e-qa-tests-gke-konflux" in pull_request.labels and not pull_request.fork:
            return True
        return "konflux-build" in pull_request.labels and _ready(pull_request)
    raise ValueError(f"unknown when: {when}")


def default_sentence(rule: dict[str, str], pull_request: PullRequest) -> str:
    """default_sentence states why default_would_start returned its answer."""
    if pull_request.fork and rule.get("skip_fork"):
        return "stays off because this pull request is from a fork"
    when = rule["when"]
    if when == "always":
        return "starts on every pull request"
    if when == "not-ui":
        return _not_ui_sentence(pull_request)
    if when == "ready":
        return _ready_sentence(pull_request)
    if when == "label":
        return _label_sentence(rule["label"], pull_request)
    if when == "path":
        return _path_sentence(rule["path"], pull_request)
    if when == "konflux":
        return _konflux_sentence(pull_request)
    raise ValueError(f"unknown when: {when}")


@dataclass(frozen=True)
class _Section:
    title: str
    body: tuple[str, ...]
    collapse: bool = False


def format_plan(
    plans: tuple[JobPlan, ...],
    *,
    shadow: bool,
    title: str,
    selection: Selection,
    defaults: dict[str, dict[str, str]],
    pull_request: PullRequest,
    mapping: Mapping,
) -> str:
    """format_plan is the log text. Long sections fold with GitHub log groups."""
    sections = _sections(plans, selection, defaults, pull_request, mapping)
    return _render_text(shadow, title, sections)


def format_summary(
    plans: tuple[JobPlan, ...],
    *,
    shadow: bool,
    title: str,
    selection: Selection,
    defaults: dict[str, dict[str, str]],
    pull_request: PullRequest,
    mapping: Mapping,
) -> str:
    """format_summary is the same plan as Markdown for the job summary."""
    sections = _sections(plans, selection, defaults, pull_request, mapping)
    return _render_markdown(shadow, title, sections)


def github_commands(
    plans: tuple[JobPlan, ...],
    conflicts: tuple = (),
    mapping: Mapping | None = None,
) -> tuple[str, ...]:
    """github_commands is one notice, plus a warning per rule conflict and per dropped default."""
    start = sum(plan.would_run for plan in plans)
    skip = sum(plan.action == "stop" for plan in plans)
    unsure = sum(plan.action == "default" for plan in plans)
    dropped = sum(_default_dropped(plan) for plan in plans)
    added = sum(_default_added(plan) for plan in plans)
    notice = f"Would start {start}, skip {skip}, unsure {unsure}."
    if added:
        notice += f" This plan would start {_jobs(added)} the default would leave off."
    commands = [f"::notice title=Dispatcher plan::{notice}"]
    if mapping is not None:
        for conflict in conflicts:
            commands.append(
                "::warning title=Dispatcher plan::" + _conflict_text(conflict, mapping)
            )
    if dropped:
        commands.append(
            "::warning title=Dispatcher plan::"
            f"This plan would not start {_jobs(dropped)} the default would start."
        )
    return tuple(commands)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Plan GitHub Actions jobs for a pull request.")
    parser.add_argument("--mapping", required=True)
    parser.add_argument("--defaults", required=True)
    parser.add_argument("--labels", default="")
    parser.add_argument("--draft", action="store_true")
    parser.add_argument("--fork", action="store_true")
    parser.add_argument("--enforce", action="store_true")
    parser.add_argument("--title", default="This pull request")
    parser.add_argument("--format", choices=("human", "json"), default="human")
    parser.add_argument("files", nargs="*")
    args = parser.parse_args(argv)

    files = list(args.files)
    if files == ["-"]:
        files = [line.strip() for line in sys.stdin if line.strip()]
    labels = frozenset(label for label in args.labels.split(",") if label)
    pull_request = PullRequest(args.draft, args.fork, labels, tuple(files))
    mapping = load_mapping(args.mapping)
    selection = resolve(files, mapping, list(labels), shadow=not args.enforce)
    defaults = load_defaults(args.defaults)
    plans = dispatch(selection, defaults, pull_request)
    if args.format == "json":
        json.dump(_json(plans), sys.stdout, indent=2)
        sys.stdout.write("\n")
        return 0
    report = {
        "shadow": not args.enforce,
        "title": args.title,
        "selection": selection,
        "defaults": defaults,
        "pull_request": pull_request,
        "mapping": mapping,
    }
    sys.stdout.write(format_plan(plans, **report))
    for command in github_commands(plans, selection.conflicts, mapping):
        print(command)
    summary = os.environ.get("GITHUB_STEP_SUMMARY")
    if summary:
        with open(summary, "a", encoding="utf-8") as handle:
            handle.write(format_summary(plans, **report))
    return 0


def _json(plans: tuple[JobPlan, ...]) -> dict[str, object]:
    return {
        "jobs": [
            {
                "job": item.job,
                "opinion": item.opinion,
                "default_starts": item.default_starts,
                "has_default": item.has_default,
                "action": item.action,
                "would_run": item.would_run,
            }
            for item in plans
        ]
    }


def _opinion(selection: Selection, job: str) -> str:
    if job in selection.run:
        return "run"
    if job in selection.skip:
        return "skip"
    return "unsure"


def _ready(pull_request: PullRequest) -> bool:
    return not pull_request.draft and not pull_request.fork and not _ui_only(pull_request.files)


def _ui_only(files: tuple[str, ...]) -> bool:
    return bool(files) and all(path.startswith("ui/") for path in files)


def _path_matches(pattern: str, files: tuple[str, ...]) -> bool:
    if not files:
        return True
    compiled = re.compile(pattern)
    return any(compiled.search(path) for path in files)


def _sections(
    plans: tuple[JobPlan, ...],
    selection: Selection,
    defaults: dict[str, dict[str, str]],
    pull_request: PullRequest,
    mapping: Mapping,
) -> tuple[_Section, ...]:
    by_action = {"start": [], "stop": [], "default": []}
    for plan in plans:
        by_action[plan.action].append(plan)
    sections = [
        _Section("Changed files", _changed_files(selection)),
        _rules_section(selection, mapping),
        _conflicts_section(selection, mapping),
        _job_section(
            "Will run",
            by_action["start"],
            _run_reasons,
            selection,
            defaults,
            pull_request,
            mapping,
        ),
        _job_section(
            "Will skip",
            by_action["stop"],
            _skip_reasons,
            selection,
            defaults,
            pull_request,
            mapping,
        ),
        _unsure_section(by_action["default"], selection, defaults, pull_request),
    ]
    sections.extend(_default_sections(plans))
    return tuple(sections)


def _job_section(title, plans, reasons, selection, defaults, pull_request, mapping):
    entries = [
        (
            plan.job,
            reasons(plan.job, selection, mapping),
            _default_clause(plan, defaults, pull_request),
        )
        for plan in plans
    ]
    body = _render_jobs(entries) or ["  none"]
    return _Section(
        f"{title} ({len(entries)})",
        tuple(body),
        collapse=_collapses(title, len(entries)),
    )


def _unsure_section(plans, _selection, defaults, pull_request):
    reason = ("no rule decided these jobs",)
    entries = [
        (plan.job, reason, _default_clause(plan, defaults, pull_request))
        for plan in plans
    ]
    body = _render_jobs(entries) or ["  none"]
    return _Section(
        f"Unsure ({len(plans)})",
        tuple(body),
        collapse=_collapses("Unsure", len(plans)),
    )


def _default_clause(plan: JobPlan, defaults, pull_request: PullRequest) -> str:
    if not plan.has_default:
        return ""
    return default_sentence(defaults[plan.job], pull_request)


def _default_dropped(plan: JobPlan) -> bool:
    return plan.has_default and plan.default_starts and not plan.would_run


def _default_added(plan: JobPlan) -> bool:
    return plan.has_default and plan.would_run and not plan.default_starts


def _default_sections(plans: tuple[JobPlan, ...]) -> list[_Section]:
    dropped = [plan.job for plan in plans if _default_dropped(plan)]
    added = [plan.job for plan in plans if _default_added(plan)]
    sections = []
    if dropped:
        sections.append(
            _Section(
                f"Default would start these, and this plan would not ({len(dropped)})",
                tuple(f"  {job}" for job in dropped),
                collapse=_collapses("Default would start", len(dropped)),
            )
        )
    if added:
        sections.append(
            _Section(
                f"This plan would start these, and the default would not ({len(added)})",
                tuple(f"  {job}" for job in added),
                collapse=_collapses("This plan would start", len(added)),
            )
        )
    if not sections:
        sections.append(_Section("Default matches this plan.", ()))
    return sections


def _collapses(title: str, count: int) -> bool:
    if count <= 3:
        return False
    return title.startswith(
        ("Unsure", "Will skip", "Conflicts", "Default would start", "This plan would start")
    )


def _render_jobs(entries: list[tuple[str, tuple[str, ...], str]]) -> list[str]:
    if not entries:
        return []
    grouped: dict[tuple[str, ...], list[tuple[str, str]]] = {}
    for job, reasons, clause in entries:
        grouped.setdefault(reasons, []).append((job, clause))
    lines: list[str] = []
    for reasons, members in grouped.items():
        recorded = [(job, clause) for job, clause in members if clause]
        without_default = [job for job, clause in members if not clause]
        if len(members) == 1 and recorded:
            job, clause = recorded[0]
            lines.append(f"  {job}")
            lines.extend(f"    {reason}" for reason in reasons)
            lines.append(f"    default: {clause}")
            continue
        if len(members) == 1:
            lines.append(f"  {without_default[0]}")
            lines.extend(f"    {reason}" for reason in reasons)
            continue
        lines.extend(f"  {reason}" for reason in reasons)
        by_clause: dict[str, list[str]] = {}
        for job, clause in recorded:
            by_clause.setdefault(clause, []).append(job)
        for clause, jobs in by_clause.items():
            lines.append(f"    default: {clause}")
            lines.extend(f"      {job}" for job in jobs)
        if without_default:
            lines.append("    no GitHub Actions default")
            lines.extend(f"      {job}" for job in without_default)
    return lines


def _rules_section(selection: Selection, mapping: Mapping) -> _Section:
    lines: list[str] = []
    for number in selection.matched_rules:
        rule = mapping.by_number[number]
        lines.append(f"  {number} {rule.name}")
        lines.append(f"    {rule.condition()}")
    if not lines:
        lines.append("  none")
    return _Section(f"Rules ({len(selection.matched_rules)})", tuple(lines))


def _conflicts_section(selection: Selection, mapping: Mapping) -> _Section:
    if not selection.conflicts:
        return _Section("Conflicts (0)", ("  none",))
    lines: list[str] = []
    for conflict in selection.conflicts:
        lines.append(f"  {conflict.job}")
        lines.append(f"    {_conflict_text(conflict, mapping)}")
    return _Section(
        f"Conflicts ({len(selection.conflicts)})",
        tuple(lines),
        collapse=_collapses("Conflicts", len(selection.conflicts)),
    )


def _conflict_text(conflict, mapping: Mapping) -> str:
    ran = _rule_phrase(conflict.run_rules, mapping)
    skipped = _rule_phrase(conflict.skip_rules, mapping)
    return f"{ran} runs {conflict.job} and {skipped} skips it. Run wins."


def _rule_phrase(numbers: tuple[int, ...], mapping: Mapping) -> str:
    labels = [f"rule {number} {mapping.by_number[number].name}" for number in numbers]
    return _join_clauses(labels)


def _run_reasons(job: str, selection: Selection, mapping: Mapping) -> tuple[str, ...]:
    requirers = _requirers(job, selection, mapping)
    if job in selection.required_runs and requirers:
        return (f"{_with_verb(requirers, 'requires', 'require')} this job",)
    decision = selection.decision_for(job)
    if decision is None:
        return ("a rule says run",)
    lines = [_rule_phrase(decision.rules, mapping)]
    conflict = selection.conflict_for(job)
    if conflict is not None:
        lines.append(f"{_rule_phrase(conflict.skip_rules, mapping)} skips it")
        lines.append("run wins")
    return tuple(lines)


def _requirers(job: str, selection: Selection, mapping: Mapping) -> list[str]:
    return sorted(
        other
        for other, needs in mapping.requires.items()
        if job in needs and other in selection.run
    )


def _skip_reasons(job: str, selection: Selection, mapping: Mapping) -> tuple[str, ...]:
    decision = selection.decision_for(job)
    if decision is None:
        return ("no rule decided this job",)
    return (_rule_phrase(decision.rules, mapping),)


def _changed_files(selection: Selection) -> tuple[str, ...]:
    if not selection.files:
        return ("  (no file list)",)
    return tuple(f"  {trace.path}" for trace in selection.files)


def _header(shadow: bool) -> tuple[str, ...]:
    if shadow:
        return (
            "Shadow. Other GitHub workflows still start themselves.",
            f"The switch is the pull request label {ENFORCE_LABEL}.",
            *_plan_scope(),
        )
    return (
        "Enforce. This dispatcher is the starter for these GitHub Actions jobs.",
        *_plan_scope(),
    )


def _plan_scope() -> tuple[str, ...]:
    return (
        "Run, skip, and unsure include Prow. Default lines are GitHub Actions triggers.",
        "dispatch.sh applies this plan to Prow before the cluster is created.",
        "Only a job whose opinion is run is started. Unsure does not run.",
    )


def _render_text(shadow: bool, title: str, sections: tuple[_Section, ...]) -> str:
    chunks = [*_header(shadow), "", title, ""]
    for section in sections:
        chunks.extend(_text_section(section))
        chunks.append("")
    return "\n".join(chunks).rstrip() + "\n"


def _text_section(section: _Section) -> list[str]:
    if section.collapse:
        return [f"::group::{section.title}", *section.body, "::endgroup::"]
    return [section.title, *section.body]


def _render_markdown(shadow: bool, title: str, sections: tuple[_Section, ...]) -> str:
    parts = ["\n".join(_header(shadow)), "", f"## {title}", ""]
    for section in sections:
        parts.extend(_markdown_section(section))
    return "\n".join(parts).rstrip() + "\n"


def _markdown_section(section: _Section) -> list[str]:
    if not section.body:
        return [f"## {section.title}", ""]
    block = _fence(section.body)
    if section.collapse:
        return ["<details>", f"<summary>{section.title}</summary>", "", *block, "</details>", ""]
    return [f"## {section.title}", "", *block, ""]


def _fence(body: tuple[str, ...]) -> list[str]:
    return ["```", *body, "```"]


def _not_ui_sentence(pull_request: PullRequest) -> str:
    if _ui_only(pull_request.files):
        return "stays off because every changed file is under ui/"
    outside = [path for path in pull_request.files if not path.startswith("ui/")]
    if not outside:
        return "starts because the pull request is not ui-only"
    # A long diff does not explain this rule. One to three paths do.
    if len(outside) > 3:
        return "starts because a changed file is outside ui/"
    return f"starts because {_with_verb(outside, 'is outside ui/', 'are outside ui/')}"


def _ready_sentence(pull_request: PullRequest) -> str:
    if pull_request.draft:
        return "stays off because this pull request is a draft"
    if pull_request.fork:
        return "stays off because this pull request is from a fork"
    if _ui_only(pull_request.files):
        return "stays off because every changed file is under ui/"
    return "starts because this pull request is ready"


def _label_sentence(label: str, pull_request: PullRequest) -> str:
    if pull_request.fork:
        return "stays off because this pull request is from a fork"
    if label in pull_request.labels:
        return f"starts because this pull request has label {label}"
    return f"stays off because label {label} is absent"


def _path_sentence(pattern: str, pull_request: PullRequest) -> str:
    if not pull_request.files:
        return "starts because there is no file list, so the path rule matches"
    matched = [path for path in pull_request.files if re.search(pattern, path)]
    if matched:
        # The pattern stays out of this sentence. A long one hides the file that matched.
        return f"starts because {_with_verb(matched, 'matches', 'match')} the path rule"
    return f"stays off because no changed file matches {pattern}"


def _konflux_sentence(pull_request: PullRequest) -> str:
    if "e2e-qa-tests-gke-konflux" in pull_request.labels and not pull_request.fork:
        return "starts because this pull request has label e2e-qa-tests-gke-konflux"
    if pull_request.fork:
        return "stays off because this pull request is from a fork"
    if "konflux-build" not in pull_request.labels:
        return (
            "stays off because neither label e2e-qa-tests-gke-konflux nor konflux-build is set"
        )
    if pull_request.draft:
        return "stays off because this pull request is a draft"
    if _ui_only(pull_request.files):
        return "stays off because every changed file is under ui/"
    return "starts because this pull request has label konflux-build and is ready"


def _with_verb(paths: list[str], singular: str, plural: str) -> str:
    verb = singular if len(paths) == 1 else plural
    return f"{_join_files(paths)} {verb}"


def _join_files(paths: list[str]) -> str:
    if len(paths) == 1:
        return paths[0]
    if len(paths) == 2:
        return f"{paths[0]} and {paths[1]}"
    if len(paths) == 3:
        return f"{paths[0]}, {paths[1]}, and {paths[2]}"
    head = ", ".join(paths[:3])
    return f"{head}, and {len(paths) - 3} more"


def _join_clauses(parts: list[str]) -> str:
    if len(parts) == 1:
        return parts[0]
    if len(parts) == 2:
        return f"{parts[0]} and {parts[1]}"
    return f"{', '.join(parts[:-1])}, and {parts[-1]}"


def _jobs(count: int) -> str:
    if count == 1:
        return "1 job"
    return f"{count} jobs"


if __name__ == "__main__":
    sys.exit(main())
