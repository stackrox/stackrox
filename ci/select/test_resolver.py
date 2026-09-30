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
                    "style-check",
                    "wait-for-images",
                    "should-dispatch",
                    "",
                ]
            ),
        )
        log = result.log_text
        self.assertRegex(
            log,
            r"^# stackrox/stackrox PR 23035 a1b2c3d\n"
            r"# \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}\n",
        )
        self.assertIn(
            '001. rule "docs-only" matches every changed file\n     README.md\n',
            log,
        )
        self.assertIn('002. rule "docs-only" skips target "go"', log)
        self.assertIn('009. rule "docs-only" skips target "e2e-nongroovy-tests"', log)
        self.assertNotIn("because file", log)
        self.assertIn(
            '017. clash on target "style-check"; '
            'rule "style" says run, rule "docs-only" says skip; default "run"',
            log,
        )
        self.assertIn(
            '018. clash on target "wait-for-images"; '
            'rule "image-wait" says run, rule "docs-only" says skip; default "run"',
            log,
        )
        self.assertIn(
            '019. added target "should-dispatch" because target "wait-for-images" requires it',
            log,
        )
        self.assertNotIn("e2e-nongroovy-tests\n", result.decision_text)
        self.assertNotIn("go\n", result.decision_text)

    def test_clash_takes_the_default_run(self):
        result = decide(["sensor/common/foo.go", "central/policy/service.go"])
        self.assertIn("go-postgres", result.jobs)
        self.assertIn("sensor-integration-tests", result.jobs)
        self.assertIn("go", result.jobs)
        self.assertIn(
            '006. clash on target "go-postgres"; '
            'rule "central-policy" says run, rule "sensor" says skip; default "run"',
            result.log_text,
        )
        self.assertIn(
            '007. clash on target "sensor-integration-tests"; '
            'rule "sensor" says run, rule "central-policy" says skip; default "run"',
            result.log_text,
        )

    def test_clash_takes_a_skip_default_when_the_file_says_skip(self):
        defaults = parse_defaults(DEFAULTS.read_text(encoding="utf-8"))
        defaults["go-postgres"] = "skip"
        result = decide(
            ["sensor/common/foo.go", "central/policy/service.go"],
            defaults=defaults,
        )
        self.assertNotIn("go-postgres", result.jobs)
        self.assertIn('default "skip"', result.log_text)

    def test_run_all_label_and_a_skip_take_the_default(self):
        # docs-only skips every target, and the label runs every target.
        # Each clash uses that target's default.
        result = decide(["README.md"], labels=["ci-run-all-tests"])
        self.assertIn("go", result.jobs)
        self.assertIn("style-check", result.jobs)
        self.assertNotIn("gke-qa-e2e-tests", result.jobs)
        self.assertIn(
            'clash on target "go"; rule "run-all-label" says run, rule "docs-only" says skip; default "run"',
            result.log_text,
        )
        self.assertIn(
            'clash on target "gke-qa-e2e-tests"; '
            'rule "run-all-label" says run, rule "docs-only" says skip; default "skip"',
            result.log_text,
        )

    def test_ci_tooling_skips_end_to_end_jobs(self):
        files = [
            ".github/workflows/ci-decision.yaml",
            ".github/workflows/e2e-dispatch.yaml",
            ".openshift-ci/dispatch.sh",
            "ci/decision-defaults",
            "ci/select/resolver.py",
            "ci/test-domains.toml",
        ]
        result = decide(files)
        for job in (
            "e2e-byodb-tests",
            "e2e-nongroovy-tests",
            "gke-nongroovy-e2e-tests",
            "gke-ui-e2e-tests",
            "ocp-vm-scanning-e2e-tests",
        ):
            self.assertNotIn(job, result.jobs)
        self.assertIn("style-check", result.jobs)
        self.assertIn("go", result.jobs)
        self.assertIn(
            "001. rule \"ci-tooling\" matches every changed file\n"
            "     .github/workflows/ (2)\n"
            "     .openshift-ci/ (1)\n"
            "     ci/select/ (1)\n"
            "     ci/decision-defaults\n"
            "     ci/test-domains.toml\n",
            result.log_text,
        )
        self.assertIn('002. rule "ci-tooling" skips target "e2e-byodb-tests"', result.log_text)
        self.assertNotIn('skips target "e2e-byodb-tests" because file', result.log_text)

    def test_one_product_file_keeps_end_to_end_jobs(self):
        result = decide([".github/workflows/style.yaml", "sensor/common/foo.go"])
        self.assertIn("e2e-byodb-tests", result.jobs)
        self.assertIn(
            'rule "sensor" matches\n     sensor/common/foo.go\n',
            result.log_text,
        )
        self.assertIn(
            'rule "style" matches every changed file (2)\n'
            "     .github/workflows/style.yaml\n"
            "     sensor/common/foo.go\n",
            result.log_text,
        )

    def test_go_mod_runs_every_target(self):
        result = decide(["go.mod"])
        self.assertEqual(result.jobs, frozenset(load_mapping(RULES).jobs))

    def test_unmatched_file_runs_the_ready_pull_request_defaults(self):
        result = decide(["notes/todo.txt"])
        defaults = parse_defaults(DEFAULTS.read_text(encoding="utf-8"))
        expected = {name for name, opinion in defaults.items() if opinion == "run"}
        self.assertEqual(result.jobs, frozenset(expected))

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
            '002. added target "go" because target "go-check" requires it',
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
