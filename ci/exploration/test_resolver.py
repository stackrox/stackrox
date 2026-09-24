"""Behavior of the exploration test-selection resolver."""

import json
from pathlib import Path

import pytest

from resolver import load_mapping, main, parse_mapping, resolve

FIXTURE = Path(__file__).with_name("fixture.yaml")

JOBS = frozenset(
    {
        "style",
        "go-unit-tests",
        "build",
        "build-operator",
        "gke-qa-e2e-tests",
        "gke-nongroovy-e2e-tests",
        "gke-ui-e2e-tests",
        "gke-scale-tests",
        "ocp-4-12-qa-e2e-tests",
    }
)

SENSOR_RUN = frozenset(
    {"style", "go-unit-tests", "build", "gke-nongroovy-e2e-tests"}
)
SENSOR_SKIP = frozenset(
    {
        "build-operator",
        "gke-ui-e2e-tests",
        "gke-scale-tests",
        "ocp-4-12-qa-e2e-tests",
    }
)
SENSOR_UNSURE = frozenset({"gke-qa-e2e-tests"})


@pytest.fixture
def mapping():
    return load_mapping(FIXTURE)


def decide(mapping, files, labels=None, shadow=True):
    return resolve(files, mapping, labels, shadow=shadow)


def assert_split(result, run, skip, unsure):
    assert result.run == frozenset(run)
    assert result.skip == frozenset(skip)
    assert result.unsure == frozenset(unsure)
    assert result.run | result.skip | result.unsure == JOBS
    assert result.run.isdisjoint(result.skip)
    assert result.run.isdisjoint(result.unsure)
    assert result.skip.isdisjoint(result.unsure)


def assert_run_all(result):
    assert_split(result, JOBS, set(), set())
    assert result.matched_domains == frozenset()
    assert result.execute == JOBS


def test_fixture_uses_ten_path_patterns(mapping):
    patterns = list(mapping.always_run_all_patterns)
    patterns.extend(mapping.skip_all_patterns)
    for domain in mapping.domains.values():
        patterns.extend(domain.path_patterns)
    assert len(patterns) == 10


def test_label_runs_every_job_even_for_docs(mapping):
    result = decide(mapping, ["README.md"], labels=["ci-run-all-tests"])
    assert result.reason == "label"
    assert_run_all(result)


@pytest.mark.parametrize("files", [None, []])
def test_missing_diff_runs_every_job(mapping, files):
    result = decide(mapping, files)
    assert result.reason == "no-diff"
    assert_run_all(result)


@pytest.mark.parametrize("path", ["go.mod", "proto/storage/alert.proto"])
def test_always_run_all_patterns_run_every_job(mapping, path):
    result = decide(mapping, [path, "sensor/common/foo.go"])
    assert result.reason == "always-run-all"
    assert_run_all(result)


def test_go_mod_pattern_does_not_match_nested_files(mapping):
    result = decide(mapping, ["third_party/go.mod"])
    assert result.reason == "unmatched"
    assert result.unmatched_files == ("third_party/go.mod",)


def test_docs_only_skips_every_job_except_style(mapping):
    result = decide(mapping, ["README.md", "docs/guide.md"])
    assert result.reason == "docs-only"
    assert result.matched_domains == frozenset()
    assert_split(result, {"style"}, JOBS - {"style"}, set())


def test_docs_mixed_with_code_use_the_code_domain(mapping):
    result = decide(mapping, ["README.md", "sensor/common/foo.go"])
    assert result.reason == "domains"
    assert result.matched_domains == frozenset({"sensor"})
    assert_split(result, SENSOR_RUN, SENSOR_SKIP, SENSOR_UNSURE)


def test_unmatched_file_runs_every_job(mapping):
    result = decide(mapping, ["pkg/booleanpolicy/foo.go"])
    assert result.reason == "unmatched"
    assert result.unmatched_files == ("pkg/booleanpolicy/foo.go",)
    assert_run_all(result)


def test_one_unmatched_file_discards_a_domain_match(mapping):
    result = decide(
        mapping, ["sensor/common/foo.go", "pkg/booleanpolicy/foo.go"]
    )
    assert result.reason == "unmatched"
    assert result.unmatched_files == ("pkg/booleanpolicy/foo.go",)
    assert_run_all(result)


def test_longer_prefix_overrides_parent_instead_of_union(mapping):
    result = decide(
        mapping, ["sensor/kubernetes/complianceoperator/foo.go"]
    )
    assert result.reason == "domains"
    assert result.matched_domains == frozenset({"sensor-ocp"})
    assert_split(
        result,
        {
            "style",
            "go-unit-tests",
            "build",
            "gke-qa-e2e-tests",
            "ocp-4-12-qa-e2e-tests",
        },
        {
            "build-operator",
            "gke-nongroovy-e2e-tests",
            "gke-ui-e2e-tests",
            "gke-scale-tests",
        },
        set(),
    )


def test_shorter_prefix_keeps_the_parent_domain(mapping):
    result = decide(mapping, ["sensor/common/foo.go"])
    assert result.matched_domains == frozenset({"sensor"})
    assert_split(result, SENSOR_RUN, SENSOR_SKIP, SENSOR_UNSURE)


def test_ui_change_runs_build_and_skips_build_operator(mapping):
    result = decide(mapping, ["ui/src/App.tsx"])
    assert "build" in result.run
    assert "build-operator" in result.skip
    assert "build-operator" not in result.run
    assert_split(
        result,
        {"style", "go-unit-tests", "build", "gke-ui-e2e-tests"},
        {
            "build-operator",
            "gke-qa-e2e-tests",
            "gke-nongroovy-e2e-tests",
            "gke-scale-tests",
            "ocp-4-12-qa-e2e-tests",
        },
        set(),
    )


def test_scale_script_leaves_the_image_build_unsure(mapping):
    result = decide(mapping, ["tests/e2e/run-scale.sh"])
    assert result.matched_domains == frozenset({"scale"})
    assert_split(
        result,
        {"style", "go-unit-tests", "gke-scale-tests"},
        {
            "build-operator",
            "gke-qa-e2e-tests",
            "gke-nongroovy-e2e-tests",
            "gke-ui-e2e-tests",
            "ocp-4-12-qa-e2e-tests",
        },
        {"build"},
    )


def test_policy_path_is_its_own_domain(mapping):
    result = decide(mapping, ["central/policy/service.go"])
    assert result.matched_domains == frozenset({"central-policy"})
    assert "gke-qa-e2e-tests" in result.unsure
    assert "gke-nongroovy-e2e-tests" in result.run
    assert "gke-ui-e2e-tests" in result.skip


def test_mixed_domains_run_wins_and_silence_blocks_skip(mapping):
    result = decide(mapping, ["sensor/common/foo.go", "scanner/matcher/bar.go"])
    assert result.matched_domains == frozenset({"sensor", "scanner"})
    assert "gke-nongroovy-e2e-tests" in result.run
    assert "gke-ui-e2e-tests" in result.skip
    assert "gke-scale-tests" in result.unsure
    assert "gke-qa-e2e-tests" in result.unsure


def test_shadow_mode_executes_every_job_while_reporting_skips(mapping):
    result = decide(mapping, ["sensor/common/foo.go"], shadow=True)
    assert result.shadow is True
    assert result.skip
    assert result.execute == JOBS


def test_enforce_mode_executes_run_and_unsure_only(mapping):
    result = decide(mapping, ["sensor/common/foo.go"], shadow=False)
    assert result.shadow is False
    assert result.execute == SENSOR_RUN | SENSOR_UNSURE
    assert result.skip == SENSOR_SKIP
    assert result.execute.isdisjoint(result.skip)


def test_shadow_is_the_default(mapping):
    result = resolve(["sensor/common/foo.go"], mapping)
    assert result.shadow is True
    assert result.execute == JOBS


def test_code_always_run_overrides_a_domain_skip():
    mapping = parse_mapping(mapping_dict())
    result = resolve(["ui/a.tsx"], mapping, shadow=False)
    assert "style" in result.run


def test_rejects_unknown_mapping_version():
    data = mapping_dict()
    data["version"] = 2
    with pytest.raises(ValueError, match="version"):
        parse_mapping(data)


def test_rejects_opinion_for_an_unknown_job():
    data = mapping_dict()
    data["domains"] = {
        "ui": {
            "paths": ["^ui/"],
            "jobs": {"build-operator": "run"},
        }
    }
    with pytest.raises(ValueError, match="build-operator"):
        parse_mapping(data)


def test_rejects_an_unknown_opinion():
    data = mapping_dict()
    data["domains"]["ui"]["jobs"] = {"build": "maybe"}
    with pytest.raises(ValueError, match="maybe"):
        parse_mapping(data)


def test_rejects_an_unknown_parent_domain():
    data = mapping_dict()
    data["domains"]["child"] = {
        "extends": "missing",
        "paths": ["^child/"],
        "jobs": {},
    }
    with pytest.raises(ValueError, match="missing"):
        parse_mapping(data)


def test_cli_shadow_json_lists_three_buckets(capsys):
    exit_code = main(
        [
            "--mapping",
            str(FIXTURE),
            "--format",
            "json",
            "sensor/common/foo.go",
        ]
    )
    assert exit_code == 0
    payload = json.loads(capsys.readouterr().out)
    assert payload["shadow"] is True
    assert payload["reason"] == "domains"
    assert set(payload["execute"]) == set(payload["run"]) | set(
        payload["skip"]
    ) | set(payload["unsure"])
    assert "gke-nongroovy-e2e-tests" in payload["run"]
    assert "gke-ui-e2e-tests" in payload["skip"]
    assert "gke-qa-e2e-tests" in payload["unsure"]


def mapping_dict():
    return {
        "version": 1,
        "jobs": ["style", "build"],
        "code_always_run": ["style"],
        "docs_run": ["style"],
        "run_all_label": "ci-run-all-tests",
        "always_run_all": ["^go\\.mod$"],
        "skip_all": ["\\.md$"],
        "domains": {
            "ui": {
                "paths": ["^ui/"],
                "jobs": {"build": "run", "style": "skip"},
            }
        },
    }
