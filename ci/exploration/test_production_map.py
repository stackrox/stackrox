"""The three production rules in ci/test-domains.yaml."""

from pathlib import Path

from resolver import load_mapping, resolve

PRODUCTION = Path(__file__).resolve().parents[1] / "test-domains.yaml"

JOBS = frozenset(
    {
        "style-check",
        "sensor-integration-tests",
        "go-postgres",
    }
)


def decide(files, labels=None):
    return resolve(files, load_mapping(PRODUCTION), labels)


def test_docs_and_changelog_run_only_style():
    result = decide(["README.md", "CHANGELOG.md", "docs/guide.md"])
    assert result.reason == "docs-only"
    assert result.run == frozenset({"style-check"})
    assert result.skip == JOBS - frozenset({"style-check"})
    assert result.unsure == frozenset()


def test_sensor_change_runs_sensor_integration_and_skips_central_postgres():
    result = decide(["sensor/common/foo.go"])
    assert result.reason == "domains"
    assert result.matched_domains == frozenset({"sensor"})
    assert result.run == frozenset({"style-check", "sensor-integration-tests"})
    assert result.skip == frozenset({"go-postgres"})
    assert result.unsure == frozenset()


def test_central_policy_runs_postgres_tests_and_skips_sensor_integration():
    result = decide(["central/policy/service.go"])
    assert result.reason == "domains"
    assert result.matched_domains == frozenset({"central-policy"})
    assert result.run == frozenset({"style-check", "go-postgres"})
    assert result.skip == frozenset({"sensor-integration-tests"})
    assert result.unsure == frozenset()


def test_production_shadow_still_executes_every_job():
    result = decide(["sensor/common/foo.go"])
    assert result.shadow is True
    assert result.execute == JOBS
    assert result.skip
