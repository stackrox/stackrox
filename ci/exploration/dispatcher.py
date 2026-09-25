"""Turn resolver opinions into GitHub Actions start, stop, or not start.

Only opinion run starts a job. Skip and unsure do not. The plan names the
changed files behind each opinion. A default line is the trigger recorded
for that job.
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
    plans = []
    for job in sorted(defaults):
        opinion = _opinion(selection, job)
        starts = default_would_start(defaults[job], pull_request)
        if opinion == "run":
            action = "start"
        elif opinion == "skip":
            action = "stop"
        else:
            action = "default"
        plans.append(
            JobPlan(job, opinion, starts, action)
        )
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


def github_commands(plans: tuple[JobPlan, ...]) -> tuple[str, ...]:
    """github_commands is one notice, plus a warning when the plan drops a default start."""
    start = sum(plan.would_run for plan in plans)
    skip = sum(plan.action == "stop" for plan in plans)
    unsure = sum(plan.action == "default" for plan in plans)
    dropped = sum(plan.default_starts and not plan.would_run for plan in plans)
    added = sum(plan.would_run and not plan.default_starts for plan in plans)
    notice = f"Would start {start}, skip {skip}, unsure {unsure}."
    if added:
        notice += f" This plan would start {_jobs(added)} the default would leave off."
    commands = [f"::notice title=Dispatcher plan::{notice}"]
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
    for command in github_commands(plans):
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
            default_sentence(defaults[plan.job], pull_request),
        )
        for plan in plans
    ]
    body = _render_jobs(entries) or ["  none"]
    return _Section(
        f"{title} ({len(entries)})",
        tuple(body),
        collapse=_collapses(title, len(entries)),
    )


def _unsure_section(plans, selection, defaults, pull_request):
    blocked = []
    plain = []
    plain_reason = _plain_unsure_reason(selection)
    for plan in plans:
        clause = default_sentence(defaults[plan.job], pull_request)
        reasons = _blocked_reasons(plan.job, selection)
        if reasons is None:
            plain.append((plan.job, plain_reason, clause))
        else:
            blocked.append((plan.job, reasons, clause))
    body = _render_jobs(blocked) + _render_jobs(plain)
    if not body:
        body = ["  none"]
    return _Section(
        f"Unsure ({len(plans)})",
        tuple(body),
        collapse=_collapses("Unsure", len(plans)),
    )


def _default_sections(plans: tuple[JobPlan, ...]) -> list[_Section]:
    dropped = [plan.job for plan in plans if plan.default_starts and not plan.would_run]
    added = [plan.job for plan in plans if plan.would_run and not plan.default_starts]
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
    return title.startswith(("Unsure", "Will skip", "Default would start", "This plan would start"))


def _render_jobs(entries: list[tuple[str, tuple[str, ...], str]]) -> list[str]:
    if not entries:
        return []
    grouped: dict[tuple[str, ...], list[tuple[str, str]]] = {}
    for job, reasons, clause in entries:
        grouped.setdefault(reasons, []).append((job, clause))
    lines: list[str] = []
    for reasons, members in grouped.items():
        if len(members) == 1:
            job, clause = members[0]
            lines.append(f"  {job}")
            lines.extend(f"    {reason}" for reason in reasons)
            lines.append(f"    default: {clause}")
            continue
        lines.extend(f"  {reason}" for reason in reasons)
        by_clause: dict[str, list[str]] = {}
        for job, clause in members:
            by_clause.setdefault(clause, []).append(job)
        for clause, jobs in by_clause.items():
            lines.append(f"    default: {clause}")
            lines.extend(f"      {job}" for job in jobs)
    return lines


def _run_reasons(job: str, selection: Selection, mapping: Mapping) -> tuple[str, ...]:
    if selection.reason == "label":
        return (f"label {mapping.run_all_label} is set, so this job runs",)
    if selection.reason == "no-diff":
        return ("the diff was missing, so this job runs",)
    if selection.reason == "always-run-all":
        paths = [trace.path for trace in selection.files if trace.kind == "always-run-all"]
        matched = _with_verb(paths, "matches", "match")
        return (f"{matched} a run-everything pattern, so this job runs",)
    if selection.reason == "docs-only":
        return ("every changed file is documentation, so this job runs",)
    lines = []
    run_paths = _voted(selection, job, "explicit_runs")
    skip_paths = _voted(selection, job, "explicit_skips")
    if run_paths:
        lines.append(_with_verb(run_paths, "says run", "say run"))
    elif job in mapping.code_always_run:
        lines.append("every code change runs this job")
    else:
        lines.append("a matched file says run")
    if skip_paths:
        lines.append(_with_verb(skip_paths, "says skip", "say skip"))
        lines.append("run wins")
    return tuple(lines)


def _skip_reasons(job: str, selection: Selection, _mapping: Mapping) -> tuple[str, ...]:
    if selection.reason == "docs-only":
        return ("every changed file is documentation",)
    skip_paths = _voted(selection, job, "explicit_skips")
    if skip_paths:
        return (_with_verb(skip_paths, "says skip", "say skip"),)
    return ("every matched file says skip",)


def _blocked_reasons(job: str, selection: Selection) -> tuple[str, ...] | None:
    skip_paths = _voted(selection, job, "explicit_skips")
    if not skip_paths:
        return None
    lines = [_with_verb(skip_paths, "says skip", "say skip")]
    silent = [
        trace.path
        for trace in selection.files
        if trace.kind == "domain"
        and job not in trace.explicit_runs
        and job not in trace.explicit_skips
    ]
    blockers = []
    if silent:
        blockers.append(_with_verb(silent, "does not say skip", "do not say skip"))
    if selection.unmatched_files:
        blockers.append(_unmatched_clause(selection.unmatched_files))
    if blockers:
        lines.append(f"{_join_clauses(blockers)}, so the skip does not stick")
    else:
        lines.append("the skip does not stick")
    return tuple(lines)


def _plain_unsure_reason(selection: Selection) -> tuple[str, ...]:
    if selection.reason == "unmatched":
        return ("No changed file matches a domain.",)
    return ("No matched file asks to run or skip these jobs.",)


def _changed_files(selection: Selection) -> tuple[str, ...]:
    lines: list[str] = []
    unmatched: list[str] = []
    for trace in selection.files:
        if trace.kind == "unmatched":
            unmatched.append(trace.path)
            continue
        lines.append(f"  {trace.path}")
        lines.append(f"    {_file_line(trace, selection)}")
    if unmatched:
        noun = "file matches" if len(unmatched) == 1 else "files match"
        lines.append(f"  {len(unmatched)} {noun} no domain")
        lines.extend(f"    {path}" for path in unmatched)
    if not lines:
        return ("  (no file list)",)
    return tuple(lines)


def _file_line(trace, selection: Selection) -> str:
    if trace.kind == "docs":
        if selection.reason == "docs-only":
            return "documentation or changelog"
        return "docs; ignored because other files change code"
    if trace.kind == "always-run-all":
        return "matches a run-everything pattern"
    if trace.kind != "domain":
        return trace.kind
    label = "domains" if "," in trace.domain else "domain"
    parts = [f"{label} {trace.domain.replace(',', ', ')}"]
    if trace.explicit_runs:
        parts.append("runs " + ", ".join(trace.explicit_runs))
    if trace.explicit_skips:
        parts.append("skips " + ", ".join(trace.explicit_skips))
    return ", ".join(parts)


def _voted(selection: Selection, job: str, field: str) -> list[str]:
    return [trace.path for trace in selection.files if job in getattr(trace, field)]


def _header(shadow: bool) -> tuple[str, ...]:
    if shadow:
        return (
            "Shadow. Other GitHub workflows still start themselves.",
            f"The switch is the pull request label {ENFORCE_LABEL}.",
            "Prow jobs keep their own config. This plan is GitHub Actions only.",
            "Only a job whose opinion is run is started. Unsure does not run.",
        )
    return (
        "Enforce. This dispatcher is the starter for these GitHub Actions jobs.",
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


def _unmatched_clause(paths: tuple[str, ...]) -> str:
    if len(paths) == 1:
        return f"{paths[0]} matches no domain"
    if len(paths) <= 3:
        return f"{_join_files(list(paths))} match no domain"
    return f"{len(paths)} files match no domain"


def _jobs(count: int) -> str:
    if count == 1:
        return "1 job"
    return f"{count} jobs"


if __name__ == "__main__":
    sys.exit(main())
