"""Behavior of the ordered rule list."""

import json
from pathlib import Path

import pytest

from resolver import load_mapping, main, parse_mapping, resolve

FIXTURE = Path(__file__).with_name("fixture.toml")

JOBS = frozenset(
    {
        "style",
        "go",
        "sensor-integration-tests",
        "go-postgres",
        "wait-for-images",
        "should-dispatch",
        "other",
    }
)


@pytest.fixture
def mapping():
    return load_mapping(FIXTURE)


def decide(mapping, files, labels=None, shadow=True):
    return resolve(files, mapping, labels, shadow=shadow)


def test_docs_only_skips_go_and_runs_style(mapping):
    result = decide(mapping, ["README.md", "docs/guide.md"])
    assert result.reason == "rules"
    assert result.unsure == frozenset()
    assert "style" in result.run
    assert "go" in result.skip
    assert result.decision_for("go").rules == (3,)
    assert result.conflicts == ()
    assert "wait-for-images" in result.run
    assert "should-dispatch" in result.required_runs


def test_go_file_runs_go_even_when_a_directory_rule_also_matches(mapping):
    result = decide(mapping, ["sensor/common/foo.go"])
    assert "go" in result.run
    assert result.decision_for("go").rules == (4,)
    assert "sensor-integration-tests" in result.run
    assert "go-postgres" in result.skip
    assert result.decision_for("go-postgres").rules == (5,)
    assert result.conflicts == ()


def test_opposite_votes_warn_and_run_wins(mapping):
    result = decide(mapping, ["sensor/common/foo.go", "central/policy/service.go"])
    assert "go-postgres" in result.run
    assert "sensor-integration-tests" in result.run
    by_job = {conflict.job: conflict for conflict in result.conflicts}
    assert by_job["go-postgres"].run_rules == (6,)
    assert by_job["go-postgres"].skip_rules == (5,)
    assert by_job["sensor-integration-tests"].run_rules == (5,)
    assert by_job["sensor-integration-tests"].skip_rules == (6,)


def test_run_all_label_beats_a_later_skip(mapping):
    result = decide(mapping, ["README.md"], labels=["ci-run-all-tests"])
    assert result.run == JOBS
    assert result.skip == frozenset()
    go = result.conflict_for("go")
    assert go is not None
    assert go.run_rules == (1,)
    assert go.skip_rules == (3,)


def test_missing_diff_runs_every_job_and_docs_do_not_match(mapping):
    result = decide(mapping, [])
    assert result.run == JOBS
    assert result.conflicts == ()
    assert 3 not in result.matched_rules


def test_remaining_does_not_conflict_with_an_earlier_run(mapping):
    result = decide(mapping, ["README.md"])
    assert result.conflict_for("style") is None
    assert result.conflict_for("wait-for-images") is None
    assert result.decision_for("other").rules == (8,)


def test_enforce_executes_only_run(mapping):
    result = decide(mapping, ["sensor/common/foo.go"], shadow=False)
    assert result.execute == result.run
    assert result.execute.isdisjoint(result.skip)


def test_shadow_executes_every_job(mapping):
    result = decide(mapping, ["sensor/common/foo.go"])
    assert result.shadow is True
    assert result.execute == JOBS


def test_last_rule_must_skip_what_remains():
    data = _rules()
    data["rules"][-1]["skip"] = ["other"]
    with pytest.raises(ValueError, match="last rule"):
        parse_mapping(data)


def test_numbers_must_follow_the_list():
    data = _rules()
    data["rules"][1]["number"] = 9
    with pytest.raises(ValueError, match="numbered"):
        parse_mapping(data)


def test_a_rule_cannot_say_unsure():
    data = _rules()
    data["rules"][0]["unsure"] = ["other"]
    with pytest.raises(ValueError, match="unsure"):
        parse_mapping(data)


def test_human_report_lists_rules_then_conflicts(capsys):
    exit_code = main(
        [
            "--mapping",
            str(FIXTURE),
            "--format",
            "human",
            "sensor/common/foo.go",
            "central/policy/service.go",
        ]
    )
    assert exit_code == 0
    report = capsys.readouterr().out
    assert report.index("Rules:") < report.index("Conflicts:")
    assert report.index("Conflicts:") < report.index("Run (")
    assert "go-postgres" in report
    assert "run wins" in report


def test_cli_json_lists_conflicts(capsys):
    exit_code = main(
        ["--mapping", str(FIXTURE), "--format", "json", "sensor/a.go", "central/policy/b.go"]
    )
    assert exit_code == 0
    payload = json.loads(capsys.readouterr().out)
    assert payload["reason"] == "rules"
    assert payload["unsure"] == []
    jobs = {item["job"] for item in payload["conflicts"]}
    assert jobs == {"go-postgres", "sensor-integration-tests"}


def _rules():
    return {
        "version": 1,
        "jobs": ["style", "other"],
        "rules": [
            {"number": 1, "name": "style", "when": "always", "run": ["style"]},
            {"number": 2, "name": "remaining", "when": "remaining", "skip": ["*"]},
        ],
    }
