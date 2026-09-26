"""The three production rules in ci/test-domains.yaml."""

from pathlib import Path

from resolver import load_mapping, resolve

PRODUCTION = Path(__file__).resolve().parents[1] / "test-domains.yaml"

CI_FILES = [
    ".github/workflows/test-selection-shadow.yaml",
    ".gitignore",
    "ci/exploration/fixture.yaml",
    "ci/exploration/resolver.py",
    "ci/exploration/test_production_map.py",
    "ci/exploration/test_resolver.py",
    "ci/test-domains.yaml",
]


def decide(files, labels=None):
    mapping = load_mapping(PRODUCTION)
    return mapping, resolve(files, mapping, labels)


def test_docs_and_changelog_run_only_style():
    mapping, result = decide(["README.md", "CHANGELOG.md", "docs/guide.md"])
    assert result.reason == "docs-only"
    assert result.run == frozenset({"style-check"})
    assert result.skip == mapping.jobs - result.run
    assert result.unsure == frozenset()


def test_sensor_change_runs_sensor_integration_and_skips_central_postgres():
    mapping, result = decide(["sensor/common/foo.go"])
    assert result.reason == "domains"
    assert result.matched_domains == frozenset({"sensor"})
    assert result.run == frozenset({"style-check", "sensor-integration-tests"})
    assert result.skip == frozenset({"go-postgres"})
    assert result.unsure == mapping.jobs - result.run - result.skip
    assert result.files[0].explicit_runs == ("sensor-integration-tests",)
    assert result.files[0].explicit_skips == ("go-postgres",)


def test_central_policy_runs_postgres_tests_and_skips_sensor_integration():
    mapping, result = decide(["central/policy/service.go"])
    assert result.reason == "domains"
    assert result.matched_domains == frozenset({"central-policy"})
    assert result.run == frozenset({"style-check", "go-postgres"})
    assert result.skip == frozenset({"sensor-integration-tests"})
    assert result.unsure == mapping.jobs - result.run - result.skip
    assert result.files[0].explicit_runs == ("go-postgres",)
    assert result.files[0].explicit_skips == ("sensor-integration-tests",)


def test_matched_runs_stay_when_other_files_match_nothing():
    mapping, result = decide(
        CI_FILES
        + [
            "ci/exploration/samples/docs-note.md",
            "sensor/test-selection-shadow.txt",
            "central/policy/test-selection-shadow.txt",
        ]
    )
    assert result.reason == "domains"
    assert result.matched_domains == frozenset({"sensor", "central-policy"})
    assert result.run == frozenset(
        {"style-check", "sensor-integration-tests", "go-postgres"}
    )
    assert result.skip == frozenset()
    assert result.unsure == mapping.jobs - result.run
    assert result.execute == mapping.jobs


def test_ci_only_change_leaves_every_job_unsure_and_runs_them():
    mapping, result = decide(CI_FILES)
    assert result.reason == "unmatched"
    assert result.run == frozenset()
    assert result.skip == frozenset()
    assert result.unsure == mapping.jobs
    assert result.execute == mapping.jobs
    assert [trace.path for trace in result.files] == CI_FILES
    assert all(trace.explicit_runs == () for trace in result.files)


def test_production_shadow_still_executes_every_job():
    mapping, result = decide(["sensor/common/foo.go"])
    assert result.shadow is True
    assert result.execute == mapping.jobs
    assert result.skip
