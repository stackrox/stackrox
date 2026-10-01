"""Resolver behavior for the checked-in rules and for small fixtures."""

from __future__ import annotations

import contextlib
import io
import subprocess
import textwrap
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from decisionlib import DecisionError, parse_decision, parse_defaults  # noqa: E402
from resolver import load_mapping, main, open_decision, resolve, settle  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]
RULES = ROOT / "test-domains.toml"
DEFAULTS = ROOT / "decision-defaults"
SELECT = Path(__file__).resolve().parent
REPO = Path(__file__).resolve().parents[2]


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
            'rule "style" says run, rule "docs-only" says skip; run wins',
            log,
        )
        self.assertIn(
            '018. clash on target "wait-for-images"; '
            'rule "image-wait" says run, rule "docs-only" says skip; run wins',
            log,
        )
        self.assertIn(
            '019. added target "should-dispatch" because target "wait-for-images" requires it',
            log,
        )
        self.assertNotIn("e2e-nongroovy-tests\n", result.decision_text)
        self.assertNotIn("go\n", result.decision_text)

    def test_product_rules_that_disagree_both_run(self):
        result = decide(["sensor/common/foo.go", "central/policy/service.go"])
        self.assertIn("go-postgres", result.jobs)
        self.assertIn("sensor-integration-tests", result.jobs)
        self.assertIn("go", result.jobs)
        self.assertIn(
            '004. rule "sensor" matches\n     sensor/common/foo.go\n',
            result.log_text,
        )
        self.assertIn(
            '005. rule "central-policy" matches\n     central/policy/service.go\n',
            result.log_text,
        )
        self.assertIn(
            '007. clash on target "go-postgres"; '
            'rule "central-policy" says run, rule "sensor" says skip; run wins',
            result.log_text,
        )
        self.assertIn(
            '008. clash on target "sensor-integration-tests"; '
            'rule "sensor" says run, rule "central-policy" says skip; run wins',
            result.log_text,
        )

    def test_clash_runs_when_the_ready_pull_request_line_is_skip(self):
        # decision-defaults describes a boring pull request. A rule that
        # named this job as run is evidence the suite is relevant.
        defaults = parse_defaults(DEFAULTS.read_text(encoding="utf-8"))
        defaults["go-postgres"] = "skip"
        result = decide(
            ["sensor/common/foo.go", "central/policy/service.go"],
            defaults=defaults,
        )
        self.assertIn("go-postgres", result.jobs)
        self.assertIn(
            'clash on target "go-postgres"; '
            'rule "central-policy" says run, rule "sensor" says skip; run wins',
            result.log_text,
        )

    def test_run_all_label_wins_a_clash_with_a_skip(self):
        # The label votes run for every target. A docs-only skip clashes,
        # and the clash runs the job, including targets a ready pull request skips.
        result = decide(["README.md"], labels=["ci-run-all-tests"])
        self.assertIn("go", result.jobs)
        self.assertIn("style-check", result.jobs)
        self.assertIn("gke-qa-e2e-tests", result.jobs)
        self.assertIn(
            'clash on target "go"; rule "run-all-label" says run, rule "docs-only" says skip; run wins',
            result.log_text,
        )
        self.assertIn(
            'clash on target "gke-qa-e2e-tests"; '
            'rule "run-all-label" says run, rule "docs-only" says skip; run wins',
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
        self.assertIn('rule "style" runs target "style-check"', result.log_text)

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

    def test_edge_cases(self):
        """test_edge_cases checks each row's decision and log against that row's files and rules."""
        for case in _EDGE_CASES:
            with self.subTest(case["name"]):
                self.assertIn(" should ", case["name"])
                given = case["input"]
                if "error" in case["expect"]:
                    with self.assertRaises(DecisionError) as caught:
                        _load_rules(given["rules"])
                    self.assertEqual(str(caught.exception), case["expect"]["error"])
                    continue
                decision, log = _run_edge(given)
                self.assertEqual(decision, textwrap.dedent(case["expect"]["decision"]).lstrip("\n"))
                self.assertEqual(_stable_log(log), textwrap.dedent(case["expect"]["log"]).lstrip("\n"))

    def test_rejected_rules(self):
        """test_rejected_rules checks that a bad rule file is refused before a decision exists."""
        for case in _REJECTED_RULES:
            with self.subTest(case["name"]):
                self.assertIn(" should ", case["name"])
                with self.assertRaises(DecisionError) as caught:
                    _load_rules(case["input"]["rules"])
                self.assertEqual(str(caught.exception), case["expect"]["error"])

    def test_open_decision_uses_the_rules_or_the_defaults(self):
        """test_open_decision_uses_the_rules_or_the_defaults covers a readable diff and a rules file that cannot be loaded."""
        text, log, defaults = open_decision(
            REPO / "ci/select/testdata/corners.toml",
            REPO / "ci/select/testdata/corners-defaults",
            [],
            [],
            repo="stackrox/stackrox",
            pr="23035",
            commit="a1b2c3d",
        )
        self.assertEqual(text, "unit\n")
        self.assertIn('rule "nothing-changed" matches because the diff has no files', log)
        self.assertIn("unit", defaults)

        text, log, owned = open_decision(
            REPO / "ci/select/testdata/invalid/bad-version.toml",
            REPO / "ci/select/testdata/corners-defaults",
            ["README.md"],
            [],
            repo="stackrox/stackrox",
            pr="23035",
            commit="a1b2c3d",
        )
        self.assertEqual(text, "# no job runs\n")
        self.assertIn("resolver failed: unsupported rules version: 2", log)
        self.assertEqual(owned["unit"], "skip")

        text, log, _owned = open_decision(
            REPO / "ci/select/testdata/missing.toml",
            REPO / "ci/select/testdata/corners-defaults",
            [],
            [],
            repo="stackrox/stackrox",
            pr="23035",
            commit="a1b2c3d",
        )
        self.assertEqual(text, "# no job runs\n")
        self.assertIn("resolver failed:", log)

        text, log, _owned = open_decision(
            REPO / "ci/select/testdata/invalid/broken.toml",
            REPO / "ci/select/testdata/corners-defaults",
            [],
            [],
            repo="stackrox/stackrox",
            pr="23035",
            commit="a1b2c3d",
        )
        self.assertEqual(text, "# no job runs\n")
        self.assertIn("resolver failed:", log)

    def test_main_writes_the_decision_and_reports_a_bad_rules_file(self):
        """test_main_writes_the_decision_and_reports_a_bad_rules_file runs the resolver entry point in this process."""
        with tempfile.TemporaryDirectory() as tmp:
            folder = Path(tmp)
            labels = folder / "labels"
            files = folder / "files"
            decision = folder / "decision"
            log = folder / "log"
            labels.write_text("", encoding="utf-8")
            files.write_text("", encoding="utf-8")
            code = main(
                [
                    "--rules",
                    str(REPO / "ci/select/testdata/corners.toml"),
                    "--defaults",
                    str(REPO / "ci/select/testdata/corners-defaults"),
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
                ]
            )
            self.assertEqual(code, 0)
            self.assertEqual(decision.read_text(encoding="utf-8"), "unit\n")
            self.assertIn("nothing-changed", log.read_text(encoding="utf-8"))
            with contextlib.redirect_stderr(io.StringIO()):
                code = main(
                    [
                        "--rules",
                        str(REPO / "ci/select/testdata/invalid/bad-version.toml"),
                    "--defaults",
                    str(REPO / "ci/select/testdata/corners-defaults"),
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
                ]
            )
            self.assertEqual(code, 1)


# Each row is one edge case. input is what the resolver reads. expect is what it writes.
# input.files is the changed-path list. None means that list could not be read.
# input.labels is the label list.
# input.rules and input.defaults are the config: a fixture path under ci/select/testdata, or the text itself.
# expect.decision and expect.log are that output. The log clock line is written as # <when>.
# expect.error is set when the rules are rejected before a decision exists.
_EDGE_CASES = (
        {
            "name": 'A sensor file and a Central policy file should run go-postgres when its ready line says skip',
            "input": {
                "files": ['sensor/common/foo.go', 'central/policy/service.go'],
                "labels": [],
                "rules": 'ci/select/testdata/rules.toml',
                "defaults": 'ci/select/testdata/defaults-go-postgres-skip',
            },
            "expect": {
                "decision": """
            style-check
            go
            go-postgres
            sensor-integration-tests
            should-dispatch
            wait-for-images
            e2e-byodb-tests
            e2e-db-backup-restore-tests
            e2e-gke-upgrade-tests
            e2e-nongroovy-tests
            gke-nongroovy-e2e-tests
            gke-ui-e2e-tests
        """,
                "log": """
            # stackrox/stackrox PR 23035 a1b2c3d
            # <when>
            001. rule "go-sources" matches all 2 changed files
                 sensor/common/foo.go
                 central/policy/service.go
            002. rule "go-sources" runs target "go"
            003. rule "style" runs target "style-check"
            004. rule "sensor" matches
                 sensor/common/foo.go
            005. rule "central-policy" matches
                 central/policy/service.go
            006. rule "image-wait" runs target "wait-for-images"
            007. clash on target "go-postgres"; rule "central-policy" says run, rule "sensor" says skip; run wins
            008. clash on target "sensor-integration-tests"; rule "sensor" says run, rule "central-policy" says skip; run wins
            009. rule "remaining" takes default "run" for target "should-dispatch"
            010. rule "remaining" takes default "run" for target "e2e-byodb-tests"
            011. rule "remaining" takes default "run" for target "e2e-db-backup-restore-tests"
            012. rule "remaining" takes default "run" for target "e2e-gke-upgrade-tests"
            013. rule "remaining" takes default "run" for target "e2e-nongroovy-tests"
            014. rule "remaining" takes default "skip" for target "e2e-qa-tests-gke"
            015. rule "remaining" takes default "skip" for target "e2e-qa-tests-gke-konflux"
            016. rule "remaining" takes default "skip" for target "e2e-qa-tests-ocp-4-22"
            017. rule "remaining" takes default "run" for target "gke-nongroovy-e2e-tests"
            018. rule "remaining" takes default "skip" for target "gke-qa-e2e-tests"
            019. rule "remaining" takes default "run" for target "gke-ui-e2e-tests"
            020. rule "remaining" takes default "skip" for target "ocp-vm-scanning-e2e-tests"
        """,
            },
        },
        {
            "name": 'A sensor file and a Central policy file should run both suites when both ready lines say run',
            "input": {
                "files": ['sensor/common/foo.go', 'central/policy/service.go'],
                "labels": [],
                "rules": 'ci/select/testdata/rules.toml',
                "defaults": 'ci/select/testdata/defaults',
            },
            "expect": {
                "decision": """
            style-check
            go
            go-postgres
            sensor-integration-tests
            should-dispatch
            wait-for-images
            e2e-byodb-tests
            e2e-db-backup-restore-tests
            e2e-gke-upgrade-tests
            e2e-nongroovy-tests
            gke-nongroovy-e2e-tests
            gke-ui-e2e-tests
        """,
                "log": """
            # stackrox/stackrox PR 23035 a1b2c3d
            # <when>
            001. rule "go-sources" matches all 2 changed files
                 sensor/common/foo.go
                 central/policy/service.go
            002. rule "go-sources" runs target "go"
            003. rule "style" runs target "style-check"
            004. rule "sensor" matches
                 sensor/common/foo.go
            005. rule "central-policy" matches
                 central/policy/service.go
            006. rule "image-wait" runs target "wait-for-images"
            007. clash on target "go-postgres"; rule "central-policy" says run, rule "sensor" says skip; run wins
            008. clash on target "sensor-integration-tests"; rule "sensor" says run, rule "central-policy" says skip; run wins
            009. rule "remaining" takes default "run" for target "should-dispatch"
            010. rule "remaining" takes default "run" for target "e2e-byodb-tests"
            011. rule "remaining" takes default "run" for target "e2e-db-backup-restore-tests"
            012. rule "remaining" takes default "run" for target "e2e-gke-upgrade-tests"
            013. rule "remaining" takes default "run" for target "e2e-nongroovy-tests"
            014. rule "remaining" takes default "skip" for target "e2e-qa-tests-gke"
            015. rule "remaining" takes default "skip" for target "e2e-qa-tests-gke-konflux"
            016. rule "remaining" takes default "skip" for target "e2e-qa-tests-ocp-4-22"
            017. rule "remaining" takes default "run" for target "gke-nongroovy-e2e-tests"
            018. rule "remaining" takes default "skip" for target "gke-qa-e2e-tests"
            019. rule "remaining" takes default "run" for target "gke-ui-e2e-tests"
            020. rule "remaining" takes default "skip" for target "ocp-vm-scanning-e2e-tests"
        """,
            },
        },
        {
            "name": 'README.md alone should skip go and run style-check',
            "input": {
                "files": ['README.md'],
                "labels": [],
                "rules": 'ci/select/testdata/rules.toml',
                "defaults": 'ci/select/testdata/defaults',
            },
            "expect": {
                "decision": """
            style-check
            wait-for-images
            should-dispatch
        """,
                "log": """
            # stackrox/stackrox PR 23035 a1b2c3d
            # <when>
            001. rule "docs-only" matches every changed file
                 README.md
            002. rule "docs-only" skips target "go"
            003. rule "docs-only" skips target "go-postgres"
            004. rule "docs-only" skips target "sensor-integration-tests"
            005. rule "docs-only" skips target "should-dispatch"
            006. rule "docs-only" skips target "e2e-byodb-tests"
            007. rule "docs-only" skips target "e2e-db-backup-restore-tests"
            008. rule "docs-only" skips target "e2e-gke-upgrade-tests"
            009. rule "docs-only" skips target "e2e-nongroovy-tests"
            010. rule "docs-only" skips target "e2e-qa-tests-gke"
            011. rule "docs-only" skips target "e2e-qa-tests-gke-konflux"
            012. rule "docs-only" skips target "e2e-qa-tests-ocp-4-22"
            013. rule "docs-only" skips target "gke-nongroovy-e2e-tests"
            014. rule "docs-only" skips target "gke-qa-e2e-tests"
            015. rule "docs-only" skips target "gke-ui-e2e-tests"
            016. rule "docs-only" skips target "ocp-vm-scanning-e2e-tests"
            017. clash on target "style-check"; rule "style" says run, rule "docs-only" says skip; run wins
            018. clash on target "wait-for-images"; rule "image-wait" says run, rule "docs-only" says skip; run wins
            019. added target "should-dispatch" because target "wait-for-images" requires it
        """,
            },
        },
        {
            "name": 'README.md listed twice should skip go and run style-check the same way',
            "input": {
                "files": ['README.md', 'README.md'],
                "labels": [],
                "rules": 'ci/select/testdata/rules.toml',
                "defaults": 'ci/select/testdata/defaults',
            },
            "expect": {
                "decision": """
            style-check
            wait-for-images
            should-dispatch
        """,
                "log": """
            # stackrox/stackrox PR 23035 a1b2c3d
            # <when>
            001. rule "docs-only" matches every changed file
                 README.md
            002. rule "docs-only" skips target "go"
            003. rule "docs-only" skips target "go-postgres"
            004. rule "docs-only" skips target "sensor-integration-tests"
            005. rule "docs-only" skips target "should-dispatch"
            006. rule "docs-only" skips target "e2e-byodb-tests"
            007. rule "docs-only" skips target "e2e-db-backup-restore-tests"
            008. rule "docs-only" skips target "e2e-gke-upgrade-tests"
            009. rule "docs-only" skips target "e2e-nongroovy-tests"
            010. rule "docs-only" skips target "e2e-qa-tests-gke"
            011. rule "docs-only" skips target "e2e-qa-tests-gke-konflux"
            012. rule "docs-only" skips target "e2e-qa-tests-ocp-4-22"
            013. rule "docs-only" skips target "gke-nongroovy-e2e-tests"
            014. rule "docs-only" skips target "gke-qa-e2e-tests"
            015. rule "docs-only" skips target "gke-ui-e2e-tests"
            016. rule "docs-only" skips target "ocp-vm-scanning-e2e-tests"
            017. clash on target "style-check"; rule "style" says run, rule "docs-only" says skip; run wins
            018. clash on target "wait-for-images"; rule "image-wait" says run, rule "docs-only" says skip; run wins
            019. added target "should-dispatch" because target "wait-for-images" requires it
        """,
            },
        },
        {
            "name": 'A file no rule names should run the ready pull request and leave the skip lines off',
            "input": {
                "files": ['notes/todo.txt'],
                "labels": [],
                "rules": 'ci/select/testdata/rules.toml',
                "defaults": 'ci/select/testdata/defaults',
            },
            "expect": {
                "decision": """
            style-check
            go
            go-postgres
            sensor-integration-tests
            should-dispatch
            wait-for-images
            e2e-byodb-tests
            e2e-db-backup-restore-tests
            e2e-gke-upgrade-tests
            e2e-nongroovy-tests
            gke-nongroovy-e2e-tests
            gke-ui-e2e-tests
        """,
                "log": """
            # stackrox/stackrox PR 23035 a1b2c3d
            # <when>
            001. rule "style" runs target "style-check"
            002. rule "image-wait" runs target "wait-for-images"
            003. rule "remaining" takes default "run" for target "go"
            004. rule "remaining" takes default "run" for target "go-postgres"
            005. rule "remaining" takes default "run" for target "sensor-integration-tests"
            006. rule "remaining" takes default "run" for target "should-dispatch"
            007. rule "remaining" takes default "run" for target "e2e-byodb-tests"
            008. rule "remaining" takes default "run" for target "e2e-db-backup-restore-tests"
            009. rule "remaining" takes default "run" for target "e2e-gke-upgrade-tests"
            010. rule "remaining" takes default "run" for target "e2e-nongroovy-tests"
            011. rule "remaining" takes default "skip" for target "e2e-qa-tests-gke"
            012. rule "remaining" takes default "skip" for target "e2e-qa-tests-gke-konflux"
            013. rule "remaining" takes default "skip" for target "e2e-qa-tests-ocp-4-22"
            014. rule "remaining" takes default "run" for target "gke-nongroovy-e2e-tests"
            015. rule "remaining" takes default "skip" for target "gke-qa-e2e-tests"
            016. rule "remaining" takes default "run" for target "gke-ui-e2e-tests"
            017. rule "remaining" takes default "skip" for target "ocp-vm-scanning-e2e-tests"
        """,
            },
        },
        {
            "name": 'ci-run-all-tests on a file no rule names should run every target',
            "input": {
                "files": ['notes/todo.txt'],
                "labels": ['ci-run-all-tests'],
                "rules": 'ci/select/testdata/rules.toml',
                "defaults": 'ci/select/testdata/defaults',
            },
            "expect": {
                "decision": """
            style-check
            go
            go-postgres
            sensor-integration-tests
            should-dispatch
            wait-for-images
            e2e-byodb-tests
            e2e-db-backup-restore-tests
            e2e-gke-upgrade-tests
            e2e-nongroovy-tests
            e2e-qa-tests-gke
            e2e-qa-tests-gke-konflux
            e2e-qa-tests-ocp-4-22
            gke-nongroovy-e2e-tests
            gke-qa-e2e-tests
            gke-ui-e2e-tests
            ocp-vm-scanning-e2e-tests
        """,
                "log": """
            # stackrox/stackrox PR 23035 a1b2c3d
            # <when>
            001. rule "run-all-label" matches because label "ci-run-all-tests" is set
            002. rule "run-all-label" runs target "style-check"
            003. rule "run-all-label" runs target "go"
            004. rule "run-all-label" runs target "go-postgres"
            005. rule "run-all-label" runs target "sensor-integration-tests"
            006. rule "run-all-label" runs target "should-dispatch"
            007. rule "run-all-label" runs target "wait-for-images"
            008. rule "run-all-label" runs target "e2e-byodb-tests"
            009. rule "run-all-label" runs target "e2e-db-backup-restore-tests"
            010. rule "run-all-label" runs target "e2e-gke-upgrade-tests"
            011. rule "run-all-label" runs target "e2e-nongroovy-tests"
            012. rule "run-all-label" runs target "e2e-qa-tests-gke"
            013. rule "run-all-label" runs target "e2e-qa-tests-gke-konflux"
            014. rule "run-all-label" runs target "e2e-qa-tests-ocp-4-22"
            015. rule "run-all-label" runs target "gke-nongroovy-e2e-tests"
            016. rule "run-all-label" runs target "gke-qa-e2e-tests"
            017. rule "run-all-label" runs target "gke-ui-e2e-tests"
            018. rule "run-all-label" runs target "ocp-vm-scanning-e2e-tests"
            019. rule "style" runs target "style-check"
            020. rule "image-wait" runs target "wait-for-images"
        """,
            },
        },
        {
            "name": 'ci-run-all-tests on README.md should run every target',
            "input": {
                "files": ['README.md'],
                "labels": ['ci-run-all-tests'],
                "rules": 'ci/select/testdata/rules.toml',
                "defaults": 'ci/select/testdata/defaults',
            },
            "expect": {
                "decision": """
            style-check
            go
            go-postgres
            sensor-integration-tests
            should-dispatch
            wait-for-images
            e2e-byodb-tests
            e2e-db-backup-restore-tests
            e2e-gke-upgrade-tests
            e2e-nongroovy-tests
            e2e-qa-tests-gke
            e2e-qa-tests-gke-konflux
            e2e-qa-tests-ocp-4-22
            gke-nongroovy-e2e-tests
            gke-qa-e2e-tests
            gke-ui-e2e-tests
            ocp-vm-scanning-e2e-tests
        """,
                "log": """
            # stackrox/stackrox PR 23035 a1b2c3d
            # <when>
            001. rule "run-all-label" matches because label "ci-run-all-tests" is set
            002. rule "docs-only" matches every changed file
                 README.md
            003. clash on target "style-check"; rule "run-all-label" and rule "style" say run, rule "docs-only" says skip; run wins
            004. clash on target "go"; rule "run-all-label" says run, rule "docs-only" says skip; run wins
            005. clash on target "go-postgres"; rule "run-all-label" says run, rule "docs-only" says skip; run wins
            006. clash on target "sensor-integration-tests"; rule "run-all-label" says run, rule "docs-only" says skip; run wins
            007. clash on target "should-dispatch"; rule "run-all-label" says run, rule "docs-only" says skip; run wins
            008. clash on target "wait-for-images"; rule "run-all-label" and rule "image-wait" say run, rule "docs-only" says skip; run wins
            009. clash on target "e2e-byodb-tests"; rule "run-all-label" says run, rule "docs-only" says skip; run wins
            010. clash on target "e2e-db-backup-restore-tests"; rule "run-all-label" says run, rule "docs-only" says skip; run wins
            011. clash on target "e2e-gke-upgrade-tests"; rule "run-all-label" says run, rule "docs-only" says skip; run wins
            012. clash on target "e2e-nongroovy-tests"; rule "run-all-label" says run, rule "docs-only" says skip; run wins
            013. clash on target "e2e-qa-tests-gke"; rule "run-all-label" says run, rule "docs-only" says skip; run wins
            014. clash on target "e2e-qa-tests-gke-konflux"; rule "run-all-label" says run, rule "docs-only" says skip; run wins
            015. clash on target "e2e-qa-tests-ocp-4-22"; rule "run-all-label" says run, rule "docs-only" says skip; run wins
            016. clash on target "gke-nongroovy-e2e-tests"; rule "run-all-label" says run, rule "docs-only" says skip; run wins
            017. clash on target "gke-qa-e2e-tests"; rule "run-all-label" says run, rule "docs-only" says skip; run wins
            018. clash on target "gke-ui-e2e-tests"; rule "run-all-label" says run, rule "docs-only" says skip; run wins
            019. clash on target "ocp-vm-scanning-e2e-tests"; rule "run-all-label" says run, rule "docs-only" says skip; run wins
        """,
            },
        },
        {
            "name": 'ci-run-all-tests on CI tooling files should run the end-to-end jobs',
            "input": {
                "files": ['.github/workflows/ci-decision.yaml', '.openshift-ci/dispatch.sh', 'ci/select/resolver.py'],
                "labels": ['ci-run-all-tests'],
                "rules": 'ci/select/testdata/rules.toml',
                "defaults": 'ci/select/testdata/defaults',
            },
            "expect": {
                "decision": """
            style-check
            go
            go-postgres
            sensor-integration-tests
            should-dispatch
            wait-for-images
            e2e-byodb-tests
            e2e-db-backup-restore-tests
            e2e-gke-upgrade-tests
            e2e-nongroovy-tests
            e2e-qa-tests-gke
            e2e-qa-tests-gke-konflux
            e2e-qa-tests-ocp-4-22
            gke-nongroovy-e2e-tests
            gke-qa-e2e-tests
            gke-ui-e2e-tests
            ocp-vm-scanning-e2e-tests
        """,
                "log": """
            # stackrox/stackrox PR 23035 a1b2c3d
            # <when>
            001. rule "run-all-label" matches because label "ci-run-all-tests" is set
            002. rule "run-all-label" runs target "style-check"
            003. rule "run-all-label" runs target "go"
            004. rule "run-all-label" runs target "go-postgres"
            005. rule "run-all-label" runs target "sensor-integration-tests"
            006. rule "run-all-label" runs target "should-dispatch"
            007. rule "run-all-label" runs target "wait-for-images"
            008. rule "ci-tooling" matches every changed file
                 .github/workflows/ (1)
                 .openshift-ci/ (1)
                 ci/select/ (1)
            009. rule "style" runs target "style-check"
            010. rule "image-wait" runs target "wait-for-images"
            011. clash on target "e2e-byodb-tests"; rule "run-all-label" says run, rule "ci-tooling" says skip; run wins
            012. clash on target "e2e-db-backup-restore-tests"; rule "run-all-label" says run, rule "ci-tooling" says skip; run wins
            013. clash on target "e2e-gke-upgrade-tests"; rule "run-all-label" says run, rule "ci-tooling" says skip; run wins
            014. clash on target "e2e-nongroovy-tests"; rule "run-all-label" says run, rule "ci-tooling" says skip; run wins
            015. clash on target "e2e-qa-tests-gke"; rule "run-all-label" says run, rule "ci-tooling" says skip; run wins
            016. clash on target "e2e-qa-tests-gke-konflux"; rule "run-all-label" says run, rule "ci-tooling" says skip; run wins
            017. clash on target "e2e-qa-tests-ocp-4-22"; rule "run-all-label" says run, rule "ci-tooling" says skip; run wins
            018. clash on target "gke-nongroovy-e2e-tests"; rule "run-all-label" says run, rule "ci-tooling" says skip; run wins
            019. clash on target "gke-qa-e2e-tests"; rule "run-all-label" says run, rule "ci-tooling" says skip; run wins
            020. clash on target "gke-ui-e2e-tests"; rule "run-all-label" says run, rule "ci-tooling" says skip; run wins
            021. clash on target "ocp-vm-scanning-e2e-tests"; rule "run-all-label" says run, rule "ci-tooling" says skip; run wins
        """,
            },
        },
        {
            "name": 'An empty file list should run the ready pull request and leave the skip lines off',
            "input": {
                "files": [],
                "labels": [],
                "rules": 'ci/select/testdata/rules.toml',
                "defaults": 'ci/select/testdata/defaults',
            },
            "expect": {
                "decision": """
            style-check
            go
            go-postgres
            sensor-integration-tests
            should-dispatch
            wait-for-images
            e2e-byodb-tests
            e2e-db-backup-restore-tests
            e2e-gke-upgrade-tests
            e2e-nongroovy-tests
            gke-nongroovy-e2e-tests
            gke-ui-e2e-tests
        """,
                "log": """
            # stackrox/stackrox PR 23035 a1b2c3d
            # <when>
            001. rule "style" runs target "style-check"
            002. rule "image-wait" runs target "wait-for-images"
            003. rule "remaining" takes default "run" for target "go"
            004. rule "remaining" takes default "run" for target "go-postgres"
            005. rule "remaining" takes default "run" for target "sensor-integration-tests"
            006. rule "remaining" takes default "run" for target "should-dispatch"
            007. rule "remaining" takes default "run" for target "e2e-byodb-tests"
            008. rule "remaining" takes default "run" for target "e2e-db-backup-restore-tests"
            009. rule "remaining" takes default "run" for target "e2e-gke-upgrade-tests"
            010. rule "remaining" takes default "run" for target "e2e-nongroovy-tests"
            011. rule "remaining" takes default "skip" for target "e2e-qa-tests-gke"
            012. rule "remaining" takes default "skip" for target "e2e-qa-tests-gke-konflux"
            013. rule "remaining" takes default "skip" for target "e2e-qa-tests-ocp-4-22"
            014. rule "remaining" takes default "run" for target "gke-nongroovy-e2e-tests"
            015. rule "remaining" takes default "skip" for target "gke-qa-e2e-tests"
            016. rule "remaining" takes default "run" for target "gke-ui-e2e-tests"
            017. rule "remaining" takes default "skip" for target "ocp-vm-scanning-e2e-tests"
        """,
            },
        },
        {
            "name": 'An unreadable file list should run the ready pull request without walking the rules',
            "input": {
                "files": None,
                "labels": [],
                "rules": 'ci/select/testdata/rules.toml',
                "defaults": 'ci/select/testdata/defaults',
            },
            "expect": {
                "decision": """
            style-check
            go
            go-postgres
            sensor-integration-tests
            should-dispatch
            wait-for-images
            e2e-byodb-tests
            e2e-db-backup-restore-tests
            e2e-gke-upgrade-tests
            e2e-nongroovy-tests
            gke-nongroovy-e2e-tests
            gke-ui-e2e-tests
        """,
                "log": """
            resolver failed: changed files are unknown
            using decision-defaults
        """,
            },
        },
        {
            "name": 'A suite that requires images should run images after a rule skipped them',
            "input": {
                "files": [],
                "labels": [],
                "rules": """
            version = 1
            jobs = ["suite", "images"]
            [[rules]]
            name = "suite"
            when = "always"
            run = ["suite"]
            [[rules]]
            name = "hold-images"
            when = "always"
            skip = ["images"]
            [[rules]]
            name = "remaining"
            when = "remaining"
            default = ["*"]
            [requires]
            suite = ["images"]
        """,
                "defaults": """
            suite skip
            images skip
        """,
            },
            "expect": {
                "decision": """
            suite
            images
        """,
                "log": """
            # stackrox/stackrox PR 23035 a1b2c3d
            # <when>
            001. rule "suite" runs target "suite"
            002. rule "hold-images" skips target "images"
            003. added target "images" because target "suite" requires it
        """,
            },
        },
        {
            "name": 'Rules where left and right require each other should be rejected',
            "input": {
                "rules": """
            version = 1
            jobs = ["left", "right"]
            [[rules]]
            name = "run-left"
            when = "always"
            run = ["left"]
            [[rules]]
            name = "remaining"
            when = "remaining"
            default = ["*"]
            [requires]
            left = ["right"]
            right = ["left"]
        """,
            },
            "expect": {
                "error": 'requires cycle at left',
            },
        },
        {
            "name": 'Sensor listed before central-policy should run both jobs',
            "input": {
                "files": ['sensor/common/foo.go', 'central/policy/service.go'],
                "labels": [],
                "rules": """
            version = 1
            jobs = ["go-postgres", "sensor-integration-tests"]
            [[rules]]
            name = "sensor"
            when = "any-file-matches"
            paths = ['^sensor/']
            run = ["sensor-integration-tests"]
            skip = ["go-postgres"]
            [[rules]]
            name = "central-policy"
            when = "any-file-matches"
            paths = ['^central/policy/']
            run = ["go-postgres"]
            skip = ["sensor-integration-tests"]
            [[rules]]
            name = "remaining"
            when = "remaining"
            default = ["*"]
        """,
                "defaults": """
            go-postgres run
            sensor-integration-tests run
        """,
            },
            "expect": {
                "decision": """
            go-postgres
            sensor-integration-tests
        """,
                "log": """
            # stackrox/stackrox PR 23035 a1b2c3d
            # <when>
            001. rule "sensor" matches
                 sensor/common/foo.go
            002. rule "central-policy" matches
                 central/policy/service.go
            003. clash on target "go-postgres"; rule "central-policy" says run, rule "sensor" says skip; run wins
            004. clash on target "sensor-integration-tests"; rule "sensor" says run, rule "central-policy" says skip; run wins
        """,
            },
        },
        {
            "name": 'Central-policy listed before sensor should run both jobs',
            "input": {
                "files": ['sensor/common/foo.go', 'central/policy/service.go'],
                "labels": [],
                "rules": """
            version = 1
            jobs = ["go-postgres", "sensor-integration-tests"]
            [[rules]]
            name = "central-policy"
            when = "any-file-matches"
            paths = ['^central/policy/']
            run = ["go-postgres"]
            skip = ["sensor-integration-tests"]
            [[rules]]
            name = "sensor"
            when = "any-file-matches"
            paths = ['^sensor/']
            run = ["sensor-integration-tests"]
            skip = ["go-postgres"]
            [[rules]]
            name = "remaining"
            when = "remaining"
            default = ["*"]
        """,
                "defaults": """
            go-postgres run
            sensor-integration-tests run
        """,
            },
            "expect": {
                "decision": """
            go-postgres
            sensor-integration-tests
        """,
                "log": """
            # stackrox/stackrox PR 23035 a1b2c3d
            # <when>
            001. rule "central-policy" matches
                 central/policy/service.go
            002. rule "sensor" matches
                 sensor/common/foo.go
            003. clash on target "go-postgres"; rule "central-policy" says run, rule "sensor" says skip; run wins
            004. clash on target "sensor-integration-tests"; rule "sensor" says run, rule "central-policy" says skip; run wins
        """,
            },
        },
        {
            "name": 'An empty file list should run unit because no file changed',
            "input": {
                "files": [],
                "labels": [],
                "rules": 'ci/select/testdata/corners.toml',
                "defaults": 'ci/select/testdata/corners-defaults',
            },
            "expect": {
                "decision": """
            unit
        """,
                "log": """
            # stackrox/stackrox PR 23035 a1b2c3d
            # <when>
            001. rule "nothing-changed" matches because the diff has no files
            002. rule "nothing-changed" runs target "unit"
        """,
            },
        },
        {
            "name": 'cmd/main.go should run unit because the wildcard matches that path',
            "input": {
                "files": ['cmd/main.go'],
                "labels": [],
                "rules": 'ci/select/testdata/corners.toml',
                "defaults": 'ci/select/testdata/corners-defaults',
            },
            "expect": {
                "decision": """
            unit
        """,
                "log": """
            # stackrox/stackrox PR 23035 a1b2c3d
            # <when>
            001. rule "go-wildcard" matches
                 cmd/main.go
            002. rule "go-wildcard" runs target "unit"
        """,
            },
        },
        {
            "name": 'An exact path, a directory prefix, and two notes files should run lint',
            "input": {
                "files": ['keep/exact.go', 'pkg/a.go', 'pkg/b.go', 'notes/a', 'notes/b'],
                "labels": [],
                "rules": 'ci/select/testdata/corners.toml',
                "defaults": 'ci/select/testdata/corners-defaults',
            },
            "expect": {
                "decision": """
            unit
            lint
        """,
                "log": """
            # stackrox/stackrox PR 23035 a1b2c3d
            # <when>
            001. rule "go-wildcard" matches
                 keep/exact.go
                 pkg/a.go
                 pkg/b.go
            002. rule "go-wildcard" runs target "unit"
            003. rule "shapes" matches all 5 changed files
                 keep/exact.go
                 pkg/ (2)
                 notes/a
                 notes/b
            004. rule "shapes" runs target "lint"
        """,
            },
        },
        {
            "name": 'The ci-suite label should run suite and the jobs that chain requires',
            "input": {
                "files": ['README.md'],
                "labels": ['ci-suite'],
                "rules": 'ci/select/testdata/corners.toml',
                "defaults": 'ci/select/testdata/corners-defaults',
            },
            "expect": {
                "decision": """
            suite
            images
            extra
        """,
                "log": """
            # stackrox/stackrox PR 23035 a1b2c3d
            # <when>
            001. rule "run-suite" matches because label "ci-suite" is set
            002. rule "run-suite" runs target "suite"
            003. added target "images" because target "suite" requires it
            004. added target "extra" because target "images" requires it
        """,
            },
        },

)




# input.rules is a fixture that the parser must refuse. expect.error is the message.
_REJECTED_RULES = (
        {"name": 'A rules file whose version is not 1 should be rejected', "input": {"rules": 'ci/select/testdata/invalid/bad-version.toml'}, "expect": {"error": 'unsupported rules version: 2'}},
        {"name": 'A rules file that names a job twice should be rejected', "input": {"rules": 'ci/select/testdata/invalid/duplicate-job.toml'}, "expect": {"error": 'duplicate job name'}},
        {"name": 'A rules file with no jobs should be rejected', "input": {"rules": 'ci/select/testdata/invalid/no-jobs.toml'}, "expect": {"error": 'jobs must list at least one target'}},
        {"name": 'A rules file whose jobs value is not a list should be rejected', "input": {"rules": 'ci/select/testdata/invalid/jobs-not-a-list.toml'}, "expect": {"error": 'jobs must be a list of strings'}},
        {"name": 'A rules file with no rules should be rejected', "input": {"rules": 'ci/select/testdata/invalid/no-rules.toml'}, "expect": {"error": 'rules must be a non-empty list'}},
        {"name": 'A rule that is not a table should be rejected', "input": {"rules": 'ci/select/testdata/invalid/rule-not-a-table.toml'}, "expect": {"error": 'rule 1 must be a table'}},
        {"name": 'Two rules with the same name should be rejected', "input": {"rules": 'ci/select/testdata/invalid/duplicate-rule-name.toml'}, "expect": {"error": 'duplicate rule name same'}},
        {"name": 'A rules file whose last rule is not remaining should be rejected', "input": {"rules": 'ci/select/testdata/invalid/last-rule-not-remaining.toml'}, "expect": {"error": 'the last rule must decide every target no earlier rule decided'}},
        {"name": 'A remaining rule that is not last should be rejected', "input": {"rules": 'ci/select/testdata/invalid/remaining-not-last.toml'}, "expect": {"error": 'rule 1 early uses remaining, which only the last rule may use'}},
        {"name": 'A remaining rule that does not use a star should be rejected', "input": {"rules": 'ci/select/testdata/invalid/last-rule-not-star.toml'}, "expect": {"error": 'the last rule must run, skip, or take the default for every remaining target'}},
        {"name": 'A rule with an empty name should be rejected', "input": {"rules": 'ci/select/testdata/invalid/missing-name.toml'}, "expect": {"error": 'rule 1 needs a name'}},
        {"name": 'A rule with an unknown when should be rejected', "input": {"rules": 'ci/select/testdata/invalid/unknown-when.toml'}, "expect": {"error": "rule 1 odd has unknown when: 'sometimes'"}},
        {"name": 'A label that is not text should be rejected', "input": {"rules": 'ci/select/testdata/invalid/label-not-a-string.toml'}, "expect": {"error": 'rule 1 odd label must be a string'}},
        {"name": 'A rule that does not run, skip, or take the default should be rejected', "input": {"rules": 'ci/select/testdata/invalid/no-opinion.toml'}, "expect": {"error": 'rule 1 odd must run, skip, or take the default'}},
        {"name": 'A rule that uses a star twice should be rejected', "input": {"rules": 'ci/select/testdata/invalid/star-twice.toml'}, "expect": {"error": 'rule 1 odd uses * more than once'}},
        {"name": 'A rule that mixes a star with job names should be rejected', "input": {"rules": 'ci/select/testdata/invalid/star-and-names.toml'}, "expect": {"error": 'rule 1 odd mixes * with job names'}},
        {"name": 'A rule that runs and skips the same job should be rejected', "input": {"rules": 'ci/select/testdata/invalid/two-opinions.toml'}, "expect": {"error": 'rule 1 odd gives one job two opinions'}},
        {"name": 'A path rule without paths should be rejected', "input": {"rules": 'ci/select/testdata/invalid/paths-required.toml'}, "expect": {"error": 'rule 1 odd needs paths'}},
        {"name": 'A label rule without a label should be rejected', "input": {"rules": 'ci/select/testdata/invalid/label-required.toml'}, "expect": {"error": 'rule 1 odd needs a label'}},
        {"name": 'An always rule that lists paths should be rejected', "input": {"rules": 'ci/select/testdata/invalid/paths-not-allowed.toml'}, "expect": {"error": 'rule 1 odd does not take paths'}},
        {"name": 'An always rule that lists a label should be rejected', "input": {"rules": 'ci/select/testdata/invalid/label-not-allowed.toml'}, "expect": {"error": 'rule 1 odd does not take a label'}},
        {"name": 'A run list that is not job names should be rejected', "input": {"rules": 'ci/select/testdata/invalid/run-not-a-list.toml'}, "expect": {"error": 'rule 1 odd run must be a list of jobs'}},
        {"name": 'A run list that mixes a star with a job name should be rejected', "input": {"rules": 'ci/select/testdata/invalid/star-mixed-in-list.toml'}, "expect": {"error": 'rule 1 odd run mixes * with job names'}},
        {"name": 'A rule that names an unknown job should be rejected', "input": {"rules": 'ci/select/testdata/invalid/unknown-job.toml'}, "expect": {"error": 'unknown job missing in rule 1 odd'}},
        {"name": 'A rule that repeats a job should be rejected', "input": {"rules": 'ci/select/testdata/invalid/repeated-job.toml'}, "expect": {"error": 'rule 1 odd repeats a job in run'}},
        {"name": 'A requires value that is not a table should be rejected', "input": {"rules": 'ci/select/testdata/invalid/requires-not-a-table.toml'}, "expect": {"error": 'requires must be a table'}},
        {"name": 'A requires entry for an unknown job should be rejected', "input": {"rules": 'ci/select/testdata/invalid/requires-unknown-job.toml'}, "expect": {"error": 'unknown job missing in requires'}},
        {"name": 'A requires list that is empty should be rejected', "input": {"rules": 'ci/select/testdata/invalid/requires-empty.toml'}, "expect": {"error": 'unit requires a non-empty list'}},
        {"name": 'A requires list that repeats a job should be rejected', "input": {"rules": 'ci/select/testdata/invalid/requires-duplicate.toml'}, "expect": {"error": 'duplicate requirement for unit'}},
        {"name": 'A requires list that names an unknown job should be rejected', "input": {"rules": 'ci/select/testdata/invalid/requires-missing-target.toml'}, "expect": {"error": 'unknown job missing required by unit'}},
        {"name": 'A job that requires itself should be rejected', "input": {"rules": 'ci/select/testdata/invalid/requires-self.toml'}, "expect": {"error": 'unit requires itself'}},
        {"name": 'A paths value that is not a list of text should be rejected', "input": {"rules": 'ci/select/testdata/invalid/paths-not-strings.toml'}, "expect": {"error": 'rule 1 odd paths must be a list of strings'}},
        {"name": 'A path pattern that is not a regular expression should be rejected', "input": {"rules": 'ci/select/testdata/invalid/bad-pattern.toml'}, "expect": {"error": "invalid pattern '[' in rule 1 odd"}},
)


def _config_text(value: str) -> str:
    """_config_text reads a repository path, or returns the config text when value is the text."""
    if "\n" not in value:
        path = REPO / value
        if path.is_file():
            return path.read_text(encoding="utf-8")
    return value


def _load_rules(value: str):
    """_load_rules parses rules from a repository path or from config text."""
    text = _config_text(value)
    if text == value and "\n" in value:
        return load_mapping_text(text)
    path = REPO / value
    if "\n" not in value and path.is_file():
        return load_mapping(path)
    return load_mapping_text(text)


def _run_edge(given):
    """_run_edge resolves one row. files is None when the changed-file list could not be read."""
    rules = _load_rules(given["rules"])
    defaults = parse_defaults(_config_text(given["defaults"]))
    return settle(
        given["files"],
        given.get("labels") or [],
        rules,
        defaults,
        repo="stackrox/stackrox",
        pr="23035",
        commit="a1b2c3d",
    )


def _stable_log(log: str) -> str:
    """_stable_log replaces the clock line so the rest of the log can be compared exactly."""
    lines = log.splitlines(keepends=True)
    if len(lines) >= 2 and lines[0].startswith("# ") and lines[1].startswith("# "):
        lines[1] = "# <when>\n"
    return "".join(lines)



def load_mapping_text(text: str):
    import tomllib

    from resolver import parse_mapping

    return parse_mapping(tomllib.loads(text))


if __name__ == "__main__":
    unittest.main()
