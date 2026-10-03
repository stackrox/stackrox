"""Dispatcher behavior. These tests do not start a job."""

from __future__ import annotations

import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from decisionlib import Decision, parse_decision  # noqa: E402
from gha_dispatcher import gha_action  # noqa: E402
from prow_dispatcher import comment_requests, prow_action, requests_since  # noqa: E402

DEFAULTS = {
    "style-check": "skip",
    "go": "skip",
    "wait-for-images": "run",
}


def decision(*jobs: str) -> Decision:
    text = "".join(f"{job}\n" for job in jobs) or "# no job runs\n"
    return parse_decision(text)


class GhaTest(unittest.TestCase):
    def test_without_the_label_the_job_runs(self):
        self.assertEqual(gha_action("go", decision(), DEFAULTS, enforce=False), "run")

    def test_listed_job_runs_when_enforced(self):
        self.assertEqual(
            gha_action("style-check", decision("style-check"), DEFAULTS, enforce=True),
            "run",
        )

    def test_unlisted_enrolled_job_skips_when_enforced(self):
        self.assertEqual(gha_action("go", decision("style-check"), DEFAULTS, enforce=True), "skip")

    def test_a_job_with_no_default_stays_outside(self):
        self.assertEqual(
            gha_action("not-enrolled", decision(), DEFAULTS, enforce=True),
            "run",
        )

    def test_missing_decision_uses_the_default_line(self):
        self.assertEqual(gha_action("go", None, DEFAULTS, enforce=True), "skip")
        self.assertEqual(gha_action("wait-for-images", None, DEFAULTS, enforce=True), "run")


class ProwTest(unittest.TestCase):
    def test_a_listed_job_runs_when_enforced(self):
        self.assertEqual(
            prow_action("go", decision("go"), DEFAULTS, enforce=True, forced=False),
            "run",
        )

    def test_enforce_skips_a_job_left_off_the_list(self):
        self.assertEqual(
            prow_action("go", decision(), DEFAULTS, enforce=True, forced=False),
            "skip",
        )

    def test_test_comment_runs_a_job_the_decision_skipped(self):
        self.assertEqual(
            prow_action("go", decision(), DEFAULTS, enforce=True, forced=True),
            "run",
        )

    def test_without_the_label_the_job_runs(self):
        self.assertEqual(
            prow_action("go", decision(), DEFAULTS, enforce=False, forced=False),
            "run",
        )

    def test_a_job_with_no_default_stays_outside(self):
        self.assertEqual(
            prow_action("not-enrolled", decision(), DEFAULTS, enforce=True, forced=False),
            "run",
        )

    def test_comment_lines(self):
        self.assertTrue(comment_requests("gke-qa-e2e-tests", "/test gke-qa-e2e-tests\n"))
        self.assertTrue(comment_requests("gke-qa-e2e-tests", "/test all"))
        self.assertFalse(comment_requests("gke-qa-e2e-tests", "/test other-job"))
        self.assertFalse(comment_requests("gke-qa-e2e-tests", "please /test gke-qa-e2e-tests"))

    def test_only_a_comment_after_the_commit_counts(self):
        comments = [
            {"created_at": "2026-09-30T10:00:00Z", "body": "/test go"},
            {"created_at": "2026-09-30T12:00:00Z", "body": "/test go"},
        ]
        self.assertFalse(requests_since(comments[:1], "go", "2026-09-30T11:00:00Z"))
        self.assertTrue(requests_since(comments, "go", "2026-09-30T11:00:00Z"))


if __name__ == "__main__":
    unittest.main()
