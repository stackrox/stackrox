"""The GitHub Actions dispatcher turns resolver opinions into start, stop, or default."""

import json
from pathlib import Path

from dispatcher import (
    JobPlan,
    PullRequest,
    default_sentence,
    default_would_start,
    dispatch,
    format_plan,
    format_summary,
    github_commands,
    load_defaults,
    main,
)
from resolver import FileTrace, Selection, load_mapping, resolve

DEFAULTS = Path(__file__).with_name("gha_defaults.yaml")
MAPPING = Path(__file__).resolve().parents[1] / "test-domains.yaml"

READY = PullRequest(draft=False, fork=False, labels=frozenset(), files=())


def test_run_starts_whether_or_not_the_old_trigger_would():
    defaults = {
        "style-check": {"when": "always"},
        "e2e-qa-tests-gke": {"when": "label", "label": "e2e-qa-tests-gke"},
    }
    selection = resolve(
        ["sensor/common/foo.go"],
        load_mapping(MAPPING),
    )
    plan = dispatch(selection, defaults, READY)
    by_job = {item.job: item for item in plan}
    assert by_job["style-check"].action == "start"
    assert by_job["style-check"].would_run is True
    assert by_job["e2e-qa-tests-gke"].opinion == "unsure"


def test_run_starts_a_job_the_old_trigger_would_leave_off():
    defaults = {"e2e-qa-tests-gke": {"when": "label", "label": "e2e-qa-tests-gke"}}
    selection = Selection(
        run=frozenset({"e2e-qa-tests-gke"}),
        skip=frozenset(),
        unsure=frozenset(),
        execute=frozenset({"e2e-qa-tests-gke"}),
        reason="domains",
        matched_domains=frozenset(),
        unmatched_files=(),
        shadow=True,
    )
    plan = dispatch(selection, defaults, READY)
    job = plan[0]
    assert job.default_starts is False
    assert job.action == "start"
    assert job.would_run is True


def test_skip_stops_a_job_the_old_trigger_would_start():
    defaults = load_defaults(DEFAULTS)
    selection = resolve(["sensor/common/foo.go"], load_mapping(MAPPING))
    plan = dispatch(selection, defaults, READY)
    go_postgres = next(item for item in plan if item.job == "go-postgres")
    assert go_postgres.opinion == "skip"
    assert go_postgres.default_starts is True
    assert go_postgres.action == "stop"
    assert go_postgres.would_run is False


def test_unsure_does_not_run():
    defaults = load_defaults(DEFAULTS)
    selection = resolve(["sensor/common/foo.go"], load_mapping(MAPPING))
    draft = PullRequest(draft=True, fork=False, labels=frozenset(), files=("sensor/common/foo.go",))
    plan = dispatch(selection, defaults, draft)
    by_job = {item.job: item for item in plan}
    assert by_job["go"].action == "default"
    assert by_job["go"].default_starts is True
    assert by_job["go"].would_run is False
    assert by_job["e2e-byodb-tests"].action == "default"
    assert by_job["e2e-byodb-tests"].default_starts is False
    assert by_job["e2e-byodb-tests"].would_run is False
    assert by_job["e2e-qa-tests-gke"].default_starts is False


def test_label_is_the_old_trigger_for_optional_e2e():
    defaults = load_defaults(DEFAULTS)
    selection = resolve(["README.md"], load_mapping(MAPPING))
    labeled = PullRequest(
        draft=True,
        fork=False,
        labels=frozenset({"e2e-qa-tests-gke"}),
        files=("README.md",),
    )
    plan = dispatch(selection, defaults, labeled)
    job = next(item for item in plan if item.job == "e2e-qa-tests-gke")
    assert job.default_starts is True


def test_defaults_cover_the_github_actions_jobs():
    mapping = load_mapping(MAPPING)
    defaults = load_defaults(DEFAULTS)
    prow = {
        "aks-qa-e2e-tests",
        "aro-qa-e2e-tests",
        "eks-qa-e2e-tests",
        "gke-external-pg-17-qa-e2e-tests",
        "gke-nongroovy-compatibility-tests",
        "gke-nongroovy-e2e-tests",
        "gke-operator-e2e-tests",
        "gke-qa-e2e-tests",
        "gke-race-condition-qa-e2e-tests",
        "gke-scale-tests",
        "gke-scanner-v4-install-tests",
        "gke-ui-e2e-tests",
        "gke-version-compatibility-tests",
        "ibmcloudz-qa-e2e-tests",
        "ocp-compliance-e2e-tests",
        "ocp-nongroovy-e2e-tests",
        "ocp-operator-e2e-tests",
        "ocp-perf-scale-tests",
        "ocp-qa-e2e-tests",
        "ocp-scanner-v4-install-tests",
        "ocp-ui-e2e-tests",
        "ocp-vm-scanning-e2e-tests",
        "osd-aws-qa-e2e-tests",
        "osd-gcp-qa-e2e-tests",
        "powervs-qa-e2e-tests",
        "rosa-qa-e2e-tests",
        "ui-component-tests",
    }
    assert set(defaults) == mapping.jobs - prow


def test_not_ui_sentence_names_a_short_diff_and_not_a_long_one():
    one = PullRequest(False, False, frozenset(), ("sensor/common/foo.go",))
    assert default_sentence({"when": "not-ui"}, one) == (
        "starts because sensor/common/foo.go is outside ui/"
    )
    many = PullRequest(False, False, frozenset(), tuple(f"pkg/{i}.go" for i in range(6)))
    assert default_sentence({"when": "not-ui"}, many) == (
        "starts because a changed file is outside ui/"
    )


def test_default_sentence_agrees_with_whether_the_default_starts():
    defaults = load_defaults(DEFAULTS)
    pull_requests = [
        PullRequest(False, False, frozenset(), ("sensor/common/foo.go",)),
        PullRequest(True, False, frozenset(), ("sensor/common/foo.go",)),
        PullRequest(False, True, frozenset(), ("sensor/common/foo.go",)),
        PullRequest(False, False, frozenset(), ("ui/src/App.tsx",)),
        PullRequest(False, False, frozenset(), ()),
        PullRequest(False, False, frozenset({"e2e-qa-tests-gke"}), ("README.md",)),
        PullRequest(False, True, frozenset({"e2e-qa-tests-gke"}), ("README.md",)),
        PullRequest(False, False, frozenset(), (".github/workflows/style.yaml",)),
        PullRequest(
            False, False, frozenset({"e2e-qa-tests-gke-konflux"}), ("sensor/a.go",)
        ),
        PullRequest(
            False, True, frozenset({"e2e-qa-tests-gke-konflux"}), ("sensor/a.go",)
        ),
        PullRequest(False, False, frozenset({"konflux-build"}), ("sensor/a.go",)),
        PullRequest(True, False, frozenset({"konflux-build"}), ("sensor/a.go",)),
        PullRequest(False, False, frozenset({"konflux-build"}), ("ui/App.tsx",)),
    ]
    for pull_request in pull_requests:
        for rule in defaults.values():
            sentence = default_sentence(rule, pull_request)
            if default_would_start(rule, pull_request):
                assert sentence.startswith("starts"), sentence
                assert "stays off" not in sentence
            else:
                assert sentence.startswith("stays off"), sentence


def test_plan_text_for_a_sensor_change():
    text, summary = _report(("sensor/common/foo.go",))
    assert text.index("Changed files") < text.index("Will run (")
    assert text.index("Will run (") < text.index("Will skip (")
    assert text.index("Will skip (") < text.index("Unsure (")
    assert text.index("Unsure (") < text.index("Default would start these")
    assert "today" not in text.lower()
    assert "Shadow." in text
    assert "ci-dispatcher-enforce" in text
    assert "::group::Unsure (" in text
    assert "::group::Will run (" not in text

    changed = _between(text, "Changed files", "Will run (")
    assert "domain sensor, runs sensor-integration-tests, skips go-postgres" in changed

    run = _between(text, "Will run (", "Will skip (")
    assert run.index("sensor-integration-tests") < run.index("style-check")
    assert "sensor/common/foo.go says run" in run
    assert "every code change runs this job" in run
    assert (
        text.count("default: starts because sensor/common/foo.go is outside ui/") == 2
    )

    skip = _between(text, "Will skip (", "Unsure (")
    assert skip.index("go-postgres") < skip.index("sensor/common/foo.go says skip")
    assert "default: starts on every pull request" in skip

    unsure = _between(text, "Unsure (", "Default would start")
    assert unsure.count("default: starts on every pull request") == 1
    assert "\n      go\n" in unsure
    assert "No matched file asks to run or skip these jobs." in unsure

    dropped = _between(text, "Default would start these", "\n\n")
    assert "\n  go-postgres\n" in dropped
    assert "\n  go\n" in dropped

    assert "## Will run (" in summary
    assert "<summary>Unsure (" in summary
    assert "::group::" not in summary


def test_plan_text_shows_a_run_winning_over_a_skip():
    text, _summary = _report(
        (
            "sensor/test-selection-shadow.txt",
            "central/policy/test-selection-shadow.txt",
            ".github/workflows/style.yaml",
        )
    )
    changed = _between(text, "Changed files", "Will run (")
    assert "domain sensor, runs sensor-integration-tests, skips go-postgres" in changed
    assert (
        "domain central-policy, runs go-postgres, skips sensor-integration-tests"
        in changed
    )
    assert "1 file matches no domain" in changed
    assert ".github/workflows/style.yaml" in changed

    run = _between(text, "Will run (", "Will skip (")
    assert "matches no domain" not in run
    postgres = run.split("sensor-integration-tests", 1)[0]
    assert "central/policy/test-selection-shadow.txt says run" in postgres
    assert "sensor/test-selection-shadow.txt says skip" in postgres
    assert "run wins" in postgres
    sensor = run.split("sensor-integration-tests", 1)[1].split("style-check", 1)[0]
    assert "sensor/test-selection-shadow.txt says run" in sensor
    assert "central/policy/test-selection-shadow.txt says skip" in sensor
    assert "run wins" in sensor

    skip = _between(text, "Will skip (", "Unsure (")
    assert "go-postgres" not in skip
    assert "none" in skip


def test_plan_text_says_when_an_unmatched_file_blocks_a_skip():
    text, _summary = _report(
        ("sensor/common/foo.go", ".github/workflows/style.yaml")
    )
    skip = _between(text, "Will skip (", "Unsure (")
    assert "none" in skip
    assert "go-postgres" not in skip
    unsure = _between(text, "Unsure (", "Default would start")
    blocked = unsure.split("No matched file asks", 1)[0]
    assert blocked.index("go-postgres") < blocked.index("says skip")
    assert (
        ".github/workflows/style.yaml matches no domain, so the skip does not stick"
        in blocked
    )


def test_plan_text_for_docs_only():
    text, summary = _report(("README.md",))
    assert "ignored because other files" not in text
    assert "documentation or changelog" in text
    run = _between(text, "Will run (", "Will skip (")
    assert "style-check" in run
    assert "every changed file is documentation, so this job runs" in run
    skip = _between(text, "Will skip (", "Unsure (")
    assert skip.count("every changed file is documentation") == 1
    assert "go-postgres" in skip
    assert "::group::Will skip (" in text
    assert "Unsure (0)" in text
    assert "::group::Unsure" not in text
    assert "<summary>Will skip (" in summary


def test_plan_text_for_files_that_match_nothing():
    text, _summary = _report((".github/workflows/style.yaml",))
    assert "Will run (0)" in text
    assert "Will skip (0)" in text
    unsure = _between(text, "Unsure (", "Default would start")
    assert "No changed file matches a domain." in unsure
    assert text.count(".github/workflows/style.yaml matches the path rule") == 1
    assert "github-actions-lint" in unsure
    assert "github-actions-shellcheck" in unsure


def test_plan_text_names_a_job_the_plan_starts_and_the_default_leaves_off():
    defaults = {
        "e2e-qa-tests-gke": {"when": "label", "label": "e2e-qa-tests-gke"}
    }
    selection = Selection(
        run=frozenset({"e2e-qa-tests-gke"}),
        skip=frozenset(),
        unsure=frozenset(),
        execute=frozenset({"e2e-qa-tests-gke"}),
        reason="domains",
        matched_domains=frozenset({"sensor"}),
        unmatched_files=(),
        shadow=True,
        files=(
            FileTrace(
                "sensor/x.go",
                "domain",
                "sensor",
                ("e2e-qa-tests-gke",),
            ),
        ),
    )
    text, _summary = _report(("sensor/x.go",), defaults=defaults, selection=selection)
    assert "sensor/x.go says run" in text
    assert "default: stays off because label e2e-qa-tests-gke is absent" in text
    assert "This plan would start these, and the default would not" in text
    assert "Default would start these" not in text
    assert "Default matches this plan." not in text


def test_plan_text_says_when_the_plan_and_the_default_agree():
    defaults = {"style-check": {"when": "always"}}
    text, _summary = _report(("central/policy/service.go",), defaults=defaults)
    assert "every code change runs this job" in text
    assert "Default matches this plan." in text
    assert "Default would start these" not in text


def test_enforce_header_replaces_the_shadow_header():
    text, _summary = _report(("sensor/common/foo.go",), shadow=False)
    assert text.startswith("Enforce.")
    assert "Shadow." not in text


def test_github_commands_warn_when_the_default_would_start_a_stopped_job():
    plans = (
        JobPlan("style-check", "run", True, "start"),
        JobPlan("go-postgres", "skip", True, "stop"),
        JobPlan("e2e-qa-tests-gke", "unsure", False, "default"),
    )
    commands = github_commands(plans)
    assert len(commands) == 2
    assert commands[0].startswith("::notice title=Dispatcher plan::")
    assert "Would start 1, skip 1, unsure 1." in commands[0]
    assert commands[1].startswith("::warning title=Dispatcher plan::")
    assert "1 job" in commands[1]


def test_github_commands_skip_the_warning_when_nothing_is_dropped():
    plans = (JobPlan("e2e-qa-tests-gke", "run", False, "start"),)
    commands = github_commands(plans)
    assert len(commands) == 1
    assert commands[0].startswith("::notice title=Dispatcher plan::")
    assert "1 job the default would leave off" in commands[0]
    assert not any(command.startswith("::warning") for command in commands)


def test_cli_prints_the_plan_and_writes_the_summary(capsys, tmp_path, monkeypatch):
    summary = tmp_path / "summary.md"
    monkeypatch.setenv("GITHUB_STEP_SUMMARY", str(summary))
    exit_code = main(
        [
            "--mapping",
            str(MAPPING),
            "--defaults",
            str(DEFAULTS),
            "--title",
            "Example",
            "sensor/common/foo.go",
        ]
    )
    assert exit_code == 0
    out = capsys.readouterr().out
    assert "Will run (" in out
    assert "::notice title=Dispatcher plan::" in out
    written = summary.read_text()
    assert "<summary>Unsure (" in written
    assert "::group::" not in written


def test_cli_json_stays_json(capsys):
    exit_code = main(
        [
            "--mapping",
            str(MAPPING),
            "--defaults",
            str(DEFAULTS),
            "--format",
            "json",
            "sensor/common/foo.go",
        ]
    )
    assert exit_code == 0
    payload = json.loads(capsys.readouterr().out)
    assert "jobs" in payload


def _report(
    files,
    *,
    draft=False,
    fork=False,
    labels=(),
    shadow=True,
    title="Example",
    defaults=None,
    selection=None,
):
    pull_request = PullRequest(draft, fork, frozenset(labels), tuple(files))
    mapping = load_mapping(MAPPING)
    if defaults is None:
        defaults = load_defaults(DEFAULTS)
    if selection is None:
        selection = resolve(list(files), mapping, list(pull_request.labels))
    plans = dispatch(selection, defaults, pull_request)
    kwargs = {
        "shadow": shadow,
        "title": title,
        "selection": selection,
        "defaults": defaults,
        "pull_request": pull_request,
        "mapping": mapping,
    }
    return format_plan(plans, **kwargs), format_summary(plans, **kwargs)


def _between(text, start, end):
    begin = text.index(start)
    if end and end in text[begin + len(start) :]:
        return text[begin : text.index(end, begin + len(start))]
    return text[begin:]
