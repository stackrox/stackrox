"""The production rule list in ci/test-domains.toml."""

from pathlib import Path

from resolver import load_mapping, main, resolve

PRODUCTION = Path(__file__).resolve().parents[1] / "test-domains.toml"

IMAGE_WAIT = frozenset({"wait-for-images", "should-dispatch"})


def decide(files, labels=None):
    mapping = load_mapping(PRODUCTION)
    return mapping, resolve(files, mapping, labels)


def test_docs_and_changelog_skip_go():
    mapping, result = decide(["README.md", "CHANGELOG.md", "docs/guide.md"])
    assert result.unsure == frozenset()
    assert "style-check" in result.run
    assert "go" in result.skip
    assert result.decision_for("go").rules == (4,)
    assert result.conflicts == ()
    assert IMAGE_WAIT <= result.run
    assert result.required_runs == frozenset({"should-dispatch"})
    assert result.skip == mapping.jobs - result.run


def test_go_source_runs_go_unit_tests():
    _mapping, result = decide(["sensor/common/foo.go"])
    assert "go" in result.run
    assert result.decision_for("go").rules == (5,)
    assert "sensor-integration-tests" in result.run
    assert "go-postgres" in result.skip
    assert result.conflict_for("go") is None


def test_sensor_and_central_conflict_and_run_wins():
    _mapping, result = decide(
        ["sensor/test-selection-shadow.txt", "central/policy/test-selection-shadow.txt"]
    )
    assert "go" in result.skip
    assert "sensor-integration-tests" in result.run
    assert "go-postgres" in result.run
    assert result.conflict_for("go-postgres") is not None
    assert result.conflict_for("sensor-integration-tests") is not None


def test_dispatch_change_runs_one_prow_suite_and_skips_the_others():
    _mapping, result = decide([".openshift-ci/dispatch.sh"])
    assert "gke-nongroovy-e2e-tests" in result.run
    assert result.decision_for("gke-nongroovy-e2e-tests").rules == (10,)
    assert "gke-ui-e2e-tests" in result.skip
    assert result.decision_for("gke-ui-e2e-tests").rules == (11,)
    assert "gke-qa-e2e-tests" in result.skip
    assert result.decision_for("gke-qa-e2e-tests").rules == (11,)


def test_job_flag_exits_for_the_prow_demonstration(capsys):
    skip = main(
        [
            "--mapping",
            str(PRODUCTION),
            "--job",
            "gke-ui-e2e-tests",
            "--enforce",
            ".openshift-ci/dispatch.sh",
        ]
    )
    assert skip == 10
    assert "skip" in capsys.readouterr().err
    run = main(
        [
            "--mapping",
            str(PRODUCTION),
            "--job",
            "gke-nongroovy-e2e-tests",
            "--enforce",
            ".openshift-ci/dispatch.sh",
        ]
    )
    assert run == 0
    assert "prow-demo-run" in capsys.readouterr().err
    outside = main(
        [
            "--mapping",
            str(PRODUCTION),
            "--job",
            "test-binary-build-commands",
            "--enforce",
            ".openshift-ci/dispatch.sh",
        ]
    )
    assert outside == 0


def test_ci_only_change_skips_go_and_runs_style_and_the_image_wait():
    mapping, result = decide(["ci/test-domains.toml", ".github/workflows/style.yaml"])
    assert "go" in result.skip
    assert result.decision_for("go").rules == (12,)
    assert "style-check" in result.run
    assert IMAGE_WAIT <= result.run
    assert result.unsure == frozenset()
    assert result.skip == mapping.jobs - result.run
