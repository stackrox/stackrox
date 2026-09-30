"""Resolver behavior for the checked-in rules and for small fixtures."""

from __future__ import annotations

import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from decisionlib import parse_decision, parse_defaults  # noqa: E402
from resolver import load_mapping, resolve  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]
RULES = ROOT / "test-domains.toml"
DEFAULTS = ROOT / "decision-defaults"
SELECT = Path(__file__).resolve().parent


def decide(files, labels=None, defaults=None, mapping=None):
    mapping = mapping or load_mapping(RULES)
    if defaults is None:
        defaults = parse_defaults(DEFAULTS.read_text(encoding="utf-8"))
    return resolve(
        files,
        labels or [],
        mapping,
        defaults,
        repo="stackrox/stackrox",
        pr="23035",
        commit="a1b2c3d",
    )


class ResolverTest(unittest.TestCase):
    def test_readme_matches_the_design_example(self):
        result = decide(["README.md"])
        self.assertEqual(
            result.decision_text,
            "\n".join(
                [
                    "# docs-only and style",
                    "style-check",
                    "# image-wait",
                    "wait-for-images",
                    "# prerequisite of wait-for-images",
                    "should-dispatch",
                    "",
                ]
            ),
        )
        log = result.log_text
        self.assertTrue(log.startswith("# stackrox/stackrox PR 23035\n"))
        self.assertIn(
            'a1b2c3d: rule "docs-only" runs target "style-check" because file "README.md" changed',
            log,
        )
        self.assertIn(
            'a1b2c3d: rule "docs-only" skips target "go" because file "README.md" changed',
            log,
        )
        self.assertIn(
            'a1b2c3d: rule "style" runs target "style-check" because file "README.md" changed',
            log,
        )
        self.assertIn(
            'a1b2c3d: rule "image-wait" runs target "wait-for-images"',
            log,
        )
        self.assertIn(
            'a1b2c3d: added target "should-dispatch" because target "wait-for-images" requires it',
            log,
        )
        self.assertIn(
            'a1b2c3d: rule "remaining" takes default "skip" for target "go-postgres"',
            log,
        )
        self.assertNotIn("go\n", result.decision_text)

    def test_clash_takes_the_default_skip(self):
        result = decide(["sensor/common/foo.go", "central/policy/service.go"])
        self.assertNotIn("go-postgres", result.jobs)
        self.assertNotIn("sensor-integration-tests", result.jobs)
        self.assertIn("go", result.jobs)
        self.assertIn(
            'a1b2c3d: clash on target "go-postgres"; '
            'rule "central-policy" says run, rule "sensor" says skip; default "skip"',
            result.log_text,
        )
        self.assertIn(
            'a1b2c3d: clash on target "sensor-integration-tests"; '
            'rule "sensor" says run, rule "central-policy" says skip; default "skip"',
            result.log_text,
        )

    def test_clash_takes_a_run_default_when_the_file_says_run(self):
        defaults = parse_defaults(DEFAULTS.read_text(encoding="utf-8"))
        defaults["go-postgres"] = "run"
        result = decide(
            ["sensor/common/foo.go", "central/policy/service.go"],
            defaults=defaults,
        )
        self.assertIn("go-postgres", result.jobs)
        self.assertIn('default "run"', result.log_text)

    def test_run_all_label_and_a_skip_take_the_default(self):
        # docs-only skips go, and the label runs go. That clash uses the default.
        result = decide(["README.md"], labels=["ci-run-all-tests"])
        self.assertNotIn("go", result.jobs)
        self.assertIn("style-check", result.jobs)
        self.assertIn("gke-qa-e2e-tests", result.jobs)
        self.assertIn('clash on target "go"', result.log_text)

    def test_go_mod_runs_every_target(self):
        result = decide(["go.mod"])
        self.assertEqual(result.jobs, frozenset(load_mapping(RULES).jobs))

    def test_unmatched_file_does_not_run_the_go_job(self):
        result = decide(["notes/todo.txt"])
        self.assertEqual(
            result.jobs,
            frozenset({"style-check", "wait-for-images", "should-dispatch"}),
        )

    def test_check_name_pulls_in_the_job_it_requires(self):
        mapping = load_mapping_text(
            """
version = 1
jobs = ["go", "go-check"]
[[rules]]
name = "matrix"
when = "always"
run = ["go-check"]
[[rules]]
name = "remaining"
when = "remaining"
default = ["*"]
[requires]
go-check = ["go"]
"""
        )
        defaults = {"go": "skip", "go-check": "skip"}
        result = decide([], defaults=defaults, mapping=mapping)
        self.assertEqual(result.jobs, frozenset({"go", "go-check"}))
        self.assertIn(
            'a1b2c3d: added target "go" because target "go-check" requires it',
            result.log_text,
        )

    def test_cli_writes_both_files(self):
        with tempfile.TemporaryDirectory() as tmp:
            folder = Path(tmp)
            labels = folder / "labels"
            files = folder / "files"
            decision = folder / "decision"
            log = folder / "log"
            labels.write_text("", encoding="utf-8")
            files.write_text("README.md\n", encoding="utf-8")
            completed = subprocess.run(
                [
                    sys.executable,
                    str(SELECT / "resolver.py"),
                    "--rules",
                    str(RULES),
                    "--defaults",
                    str(DEFAULTS),
                    "--labels-file",
                    str(labels),
                    "--files-file",
                    str(files),
                    "--repo",
                    "stackrox/stackrox",
                    "--pr",
                    "23035",
                    "--commit",
                    "a1b2c3d",
                    "--decision-out",
                    str(decision),
                    "--log-out",
                    str(log),
                ],
                check=False,
                text=True,
                capture_output=True,
            )
            self.assertEqual(completed.returncode, 0, completed.stderr)
            self.assertIn("style-check", parse_decision(decision.read_text(encoding="utf-8")).jobs)
            self.assertIn("PR 23035", log.read_text(encoding="utf-8"))


def load_mapping_text(text: str):
    import tomllib

    from resolver import parse_mapping

    return parse_mapping(tomllib.loads(text))


if __name__ == "__main__":
    unittest.main()
