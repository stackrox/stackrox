#!/usr/bin/env python3
"""Tests for the nightly ClairCore updater and its workflow contract."""

from contextlib import redirect_stdout
from io import StringIO
from pathlib import Path
import subprocess
import unittest
from unittest.mock import patch

from update_claircore import MODULE, main, update_claircore


class QueueRunner:
    def __init__(self, results):
        self.results = list(results)
        self.commands = []

    def __call__(self, command, **kwargs):
        self.commands.append(command)
        result = self.results.pop(0)
        if kwargs.get("check") and result.returncode:
            raise subprocess.CalledProcessError(result.returncode, command)
        return result


def completed(returncode=0, stdout=""):
    return subprocess.CompletedProcess([], returncode, stdout=stdout)


class ClairCoreUpdaterTest(unittest.TestCase):
    def test_unchanged_main_skips_go_get_and_git_diff(self):
        version = "v1.6.2"
        runner = QueueRunner([completed(stdout=version), completed(stdout=version)])

        self.assertFalse(update_claircore(runner))
        self.assertEqual(
            runner.commands,
            [
                ["go", "list", "-m", "-f={{.Version}}", f"{MODULE}@main"],
                ["go", "list", "-m", "-f={{.Version}}", MODULE],
            ],
        )

    def test_unreleased_main_revision_updates_go_mod_and_sum(self):
        runner = QueueRunner(
            [
                completed(stdout="v1.6.2-0.20261103120000-abcdef123456"),
                completed(stdout="v1.6.2"),
                completed(),
                completed(returncode=1),
            ]
        )

        self.assertTrue(update_claircore(runner))
        self.assertEqual(
            runner.commands[2],
            ["go", "get", f"{MODULE}@main"],
        )
        self.assertEqual(
            runner.commands[3],
            ["git", "diff", "--quiet", "--", "go.mod", "go.sum"],
        )

    def test_go_get_without_mod_file_changes_returns_unchanged(self):
        runner = QueueRunner(
            [
                completed(stdout="v1.6.2-0.20261103120000-abcdef123456"),
                completed(stdout="v1.6.2"),
                completed(),
                completed(),
            ]
        )

        self.assertFalse(update_claircore(runner))

    def test_version_lookup_failure_stops_before_go_get(self):
        runner = QueueRunner([completed(returncode=1)])

        with self.assertRaises(subprocess.CalledProcessError):
            update_claircore(runner)
        self.assertEqual(len(runner.commands), 1)

    def test_go_get_failure_stops_before_git_diff(self):
        runner = QueueRunner(
            [
                completed(stdout="v1.6.2-0.20261103120000-abcdef123456"),
                completed(stdout="v1.6.2"),
                completed(returncode=1),
            ]
        )

        with self.assertRaises(subprocess.CalledProcessError):
            update_claircore(runner)
        self.assertEqual(len(runner.commands), 3)

    def test_git_diff_error_fails_the_update(self):
        runner = QueueRunner(
            [
                completed(stdout="v1.6.2-0.20261103120000-abcdef123456"),
                completed(stdout="v1.6.2"),
                completed(),
                completed(returncode=2),
            ]
        )

        with self.assertRaises(subprocess.CalledProcessError):
            update_claircore(runner)

    def test_cli_reports_when_the_dependency_changed(self):
        output = StringIO()
        with patch("update_claircore.update_claircore", return_value=True):
            with redirect_stdout(output):
                main()
        self.assertEqual(output.getvalue().strip(), "true")

    def test_cli_reports_when_the_dependency_is_unchanged(self):
        output = StringIO()
        with patch("update_claircore.update_claircore", return_value=False):
            with redirect_stdout(output):
                main()
        self.assertEqual(output.getvalue().strip(), "false")


class WorkflowContractTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        repo_root = Path(__file__).resolve().parents[3]
        cls.e2e_workflow = (
            repo_root / ".github/workflows/scanner-e2e-test.yaml"
        ).read_text()
        cls.update_workflow = (
            repo_root / ".github/workflows/update_claircore_periodic.yaml"
        ).read_text()

    def test_scanner_e2e_runs_when_the_functional_test_label_is_added(self):
        self.assertRegex(self.e2e_workflow, r"(?m)^\s*- labeled\s*$")
        self.assertIn("github.event.action == 'labeled'", self.e2e_workflow)
        self.assertIn("github.event.action != 'labeled'", self.e2e_workflow)
        self.assertIn(
            "contains(github.event.pull_request.labels.*.name, 'scanner-functional-tests')",
            self.e2e_workflow,
        )
        self.assertIn(
            "github.event.label.name == 'scanner-functional-tests'",
            self.e2e_workflow,
        )

    def test_updater_uses_bot_identity_and_keeps_human_review(self):
        self.assertIn("RHACS_BOT_GITHUB_TOKEN", self.update_workflow)
        self.assertIn("persist-credentials: false", self.update_workflow)
        self.assertIn("group: update-claircore-main", self.update_workflow)
        self.assertIn("branch: update_claircore_main", self.update_workflow)
        self.assertIn("scanner-functional-tests", self.update_workflow)
        self.assertIn("team-reviewers: |\n            scanner", self.update_workflow)
        self.assertIn(
            'changed="$(python3 .github/workflows/scripts/update_claircore.py)"',
            self.update_workflow,
        )
        self.assertIn("if: steps.update.outputs.changed == 'true'", self.update_workflow)
        self.assertNotIn("enable-pull-request-automerge", self.update_workflow)
        self.assertNotRegex(self.update_workflow, r"(?m)^\s+auto-merge\s*$")


if __name__ == "__main__":
    unittest.main()
