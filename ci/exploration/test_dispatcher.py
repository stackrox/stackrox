"""The GitHub Actions dispatcher turns resolver opinions into start, stop, or default."""

from pathlib import Path

from dispatcher import (
    PullRequest,
    dispatch,
    format_plan,
    load_defaults,
)
from resolver import Selection, load_mapping, resolve

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


def test_unsure_leaves_the_old_trigger_alone():
    defaults = load_defaults(DEFAULTS)
    selection = resolve(["sensor/common/foo.go"], load_mapping(MAPPING))
    draft = PullRequest(draft=True, fork=False, labels=frozenset(), files=("sensor/common/foo.go",))
    plan = dispatch(selection, defaults, draft)
    by_job = {item.job: item for item in plan}
    assert by_job["go"].action == "default"
    assert by_job["go"].default_starts is True
    assert by_job["go"].would_run is True
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


def test_plan_text_names_start_stop_and_default():
    defaults = load_defaults(DEFAULTS)
    selection = resolve(["sensor/common/foo.go"], load_mapping(MAPPING))
    text = format_plan(
        dispatch(selection, defaults, READY),
        shadow=True,
        title="Example",
    )
    assert "dispatcher would start:" in text
    assert "sensor-integration-tests" in text
    assert "dispatcher would stop:" in text
    stop_at = text.index("dispatcher would stop:")
    leave_at = text.index("leave to the old trigger:")
    assert "go-postgres" in text[stop_at:leave_at]
    assert "Shadow" in text
    assert "ci-dispatcher-enforce" in text
