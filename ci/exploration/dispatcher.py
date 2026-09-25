"""Turn resolver opinions into GitHub Actions start, stop, or not start.

Only opinion run starts a job. Skip and unsure do not. Shadow mode only
prints the plan. The label ci-dispatcher-enforce applies it to the
example jobs in the shadow workflow.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from dataclasses import dataclass
from pathlib import Path

import yaml

from resolver import Selection, load_mapping, resolve

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
    """Whether today's workflow would start this job without the dispatcher."""
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


def format_plan(plans: tuple[JobPlan, ...], *, shadow: bool, title: str) -> str:
    lines = []
    if shadow:
        lines.append("Shadow. Other GitHub workflows still start themselves.")
        lines.append(f"The switch is the pull request label {ENFORCE_LABEL}.")
        lines.append("Prow jobs keep their own config. This plan is GitHub Actions only.")
        lines.append("An unsure job is not started.")
    else:
        lines.append("Enforce. This dispatcher is the starter for these GitHub Actions jobs.")
        lines.append("Only a job whose opinion is run is started. Unsure does not run.")
    lines.append("")
    lines.append(title)
    _bucket(lines, "dispatcher would start:", [item for item in plans if item.action == "start"])
    _bucket(lines, "dispatcher would stop:", [item for item in plans if item.action == "stop"])
    _bucket(lines, "dispatcher would not start:", [item for item in plans if item.action == "default"])
    return "\n".join(lines) + "\n"


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
    selection = resolve(
        files,
        load_mapping(args.mapping),
        list(labels),
        shadow=not args.enforce,
    )
    plans = dispatch(selection, load_defaults(args.defaults), pull_request)
    if args.format == "json":
        json.dump(_json(plans), sys.stdout, indent=2)
        sys.stdout.write("\n")
    else:
        sys.stdout.write(format_plan(plans, shadow=not args.enforce, title=args.title))
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


def _bucket(lines: list[str], label: str, items: list[JobPlan], indent: str = "  ") -> None:
    lines.append(f"{indent}{label}")
    if not items:
        lines.append(f"{indent}  -")
        return
    for item in items:
        old = "would start" if item.default_starts else "would stay off"
        lines.append(f"{indent}  {item.job}")
        lines.append(f"{indent}    opinion {item.opinion}, old trigger {old}")


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


if __name__ == "__main__":
    sys.exit(main())
