import io
import json
import os
import subprocess
import unittest
from contextlib import redirect_stdout
from unittest.mock import patch

from e2e_timing import is_enabled, record_skipped, timed_span
from post_tests import FinalPost, PostClusterTest, RunWithBestEffortMixin
from run_timed import main as run_timed_main


def events_from_output(output):
    return [
        json.loads(line[len("e2e_timing "):])
        for line in output.getvalue().splitlines()
        if line.startswith("e2e_timing ")
    ]


class TestTimedSpan(unittest.TestCase):
    def setUp(self):
        self.env = patch.dict(
            os.environ,
            {
                "E2E_TIMING_ENABLED": "true",
                "CI_JOB_NAME": "unit-test-lane",
                "GITHUB_ACTIONS": "true",
                "GITHUB_RUN_ID": "12345",
                "GITHUB_RUN_ATTEMPT": "2",
                "GITHUB_JOB": "unit-test-job",
            },
            clear=True,
        )
        self.env.start()

    def tearDown(self):
        self.env.stop()

    def test_emits_matching_start_and_end_records_to_log(self):
        output = io.StringIO()
        with redirect_stdout(output):
            with timed_span("post-test-collection", "fake-collector"):
                pass

        events = events_from_output(output)
        self.assertEqual(["start", "end"], [event["event"] for event in events])
        self.assertEqual(events[0]["span_id"], events[1]["span_id"])
        self.assertEqual("gha:12345:2:unit-test-lane", events[0]["run_id"])
        self.assertEqual("unit-test-lane", events[0]["lane_id"])
        self.assertGreaterEqual(events[1]["duration_ms"], 0)
        self.assertEqual("success", events[1]["outcome"])

    def test_exception_is_recorded_and_re_raised(self):
        output = io.StringIO()
        with redirect_stdout(output):
            with self.assertRaisesRegex(ValueError, "test failure"):
                with timed_span("post-test-collection", "fake-collector"):
                    raise ValueError("test failure")

        end_event = events_from_output(output)[-1]
        self.assertEqual("failure", end_event["outcome"])
        self.assertEqual("ValueError", end_event["reason"])

    def test_timeout_is_recorded_as_timeout(self):
        output = io.StringIO()
        with redirect_stdout(output):
            with self.assertRaises(subprocess.TimeoutExpired):
                with timed_span("post-test-collection", "fake-collector"):
                    raise subprocess.TimeoutExpired(["fake"], 1)

        self.assertEqual("timeout", events_from_output(output)[-1]["outcome"])

    def test_skipped_work_has_a_marker_instead_of_a_zero_duration_span(self):
        output = io.StringIO()
        with redirect_stdout(output):
            record_skipped("test-execution", "qa-part-1", "e2e-infra-only")

        event = events_from_output(output)[0]
        self.assertEqual("skipped", event["event"])
        self.assertEqual("e2e-infra-only", event["reason"])
        self.assertNotIn("duration_ms", event)

    def test_disabled_emitter_does_not_write(self):
        with patch.dict(os.environ, {"E2E_TIMING_ENABLED": "false"}):
            output = io.StringIO()
            with redirect_stdout(output):
                with timed_span("post-test-collection", "fake-collector"):
                    pass
                record_skipped("test-execution", "qa-part-1", "e2e-infra-only")

        self.assertEqual("", output.getvalue())

    def test_only_mvp_github_lanes_are_enabled_by_default(self):
        with patch.dict(
            os.environ,
            {"GITHUB_ACTIONS": "true", "CI_JOB_NAME": "gke-qa-e2e-tests"},
            clear=True,
        ):
            self.assertTrue(is_enabled())
            output = io.StringIO()
            with redirect_stdout(output):
                with timed_span("post-test-collection", "fake-collector"):
                    pass
        self.assertIn('"lane_id":"gha.qa.gke"', output.getvalue())

        with patch.dict(
            os.environ,
            {"GITHUB_ACTIONS": "true", "CI_JOB_NAME": "gke-byodb-e2e-tests"},
            clear=True,
        ):
            self.assertFalse(is_enabled())

        with patch.dict(
            os.environ,
            {"OPENSHIFT_CI": "true", "CI_JOB_NAME": "gke-qa-e2e-tests"},
            clear=True,
        ):
            self.assertFalse(is_enabled())


class TestTimedCommand(unittest.TestCase):
    def test_records_command_duration_and_returns_command_status(self):
        output = io.StringIO()
        with patch.dict(
            os.environ,
            {"E2E_TIMING_ENABLED": "true", "E2E_INFRA_ONLY": "false"},
            clear=True,
        ), patch(
            "run_timed.subprocess.run",
            side_effect=subprocess.CalledProcessError(7, ["true"]),
        ) as run_command, redirect_stdout(output):
            status = run_timed_main(
                ["--phase", "test-lane-command", "--name", "qa-part-1", "--", "true"]
            )

        self.assertEqual(7, status)
        run_command.assert_called_once()
        self.assertEqual(["true"], run_command.call_args.args[0])
        self.assertTrue(run_command.call_args.kwargs["check"])
        self.assertEqual(
            "local",
            run_command.call_args.kwargs["env"]["E2E_TIMING_PROVIDER"],
        )
        self.assertEqual("true", run_command.call_args.kwargs["env"]["E2E_TIMING_ENABLED"])
        self.assertEqual("unknown-lane", run_command.call_args.kwargs["env"]["E2E_TIMING_LANE_ID"])
        events = events_from_output(output)
        self.assertEqual(["start", "end"], [e["event"] for e in events])
        self.assertEqual("failure", events[-1]["outcome"])

    def test_signal_exit_uses_shell_status_convention(self):
        with patch.dict(os.environ, {"E2E_TIMING_ENABLED": "false"}, clear=True), patch(
            "run_timed.subprocess.run",
            side_effect=subprocess.CalledProcessError(-9, ["true"]),
        ):
            status = run_timed_main(
                ["--phase", "test-lane-command", "--name", "qa-part-1", "--", "true"]
            )

        self.assertEqual(137, status)

    def test_infra_only_command_records_that_test_body_was_skipped(self):
        output = io.StringIO()
        with patch.dict(
            os.environ,
            {"E2E_TIMING_ENABLED": "true", "E2E_INFRA_ONLY": "true"},
            clear=True,
        ), patch(
            "run_timed.subprocess.run",
            return_value=subprocess.CompletedProcess(["true"], 0),
        ) as run_command, redirect_stdout(output):
            status = run_timed_main(
                ["--phase", "test-lane-command", "--name", "qa-part-1", "--", "true"]
            )

        self.assertEqual(0, status)
        run_command.assert_called_once()
        self.assertEqual("true", run_command.call_args.kwargs["env"]["E2E_TIMING_ENABLED"])
        events = events_from_output(output)
        self.assertEqual("skipped", events[0]["event"])
        self.assertEqual("test-execution", events[0]["phase"])
        self.assertEqual("post-test-collection", events[1]["phase"])
        self.assertEqual(["skipped", "skipped", "start", "end"], [e["event"] for e in events])
        self.assertTrue(all(event["attributes"]["infra_only"] == "true" for event in events))

    def test_infra_only_build_span_does_not_claim_test_phases_were_skipped(self):
        output = io.StringIO()
        with patch.dict(
            os.environ,
            {"E2E_TIMING_ENABLED": "true", "E2E_INFRA_ONLY": "true"},
            clear=True,
        ), patch(
            "run_timed.subprocess.run",
            return_value=subprocess.CompletedProcess(["true"], 0),
        ), redirect_stdout(output):
            status = run_timed_main(
                ["--phase", "test-build", "--name", "qa-backend-compile", "--", "true"]
            )

        self.assertEqual(0, status)
        events = events_from_output(output)
        self.assertEqual(["start", "end"], [event["event"] for event in events])
        self.assertTrue(all(event["phase"] == "test-build" for event in events))


class TestPostTestTiming(unittest.TestCase):
    def test_post_cluster_collection_emits_aggregate_stage_span(self):
        output = io.StringIO()
        post = PostClusterTest(
            collect_collector_metrics=False,
            collect_central_artifacts=False,
            collect_service_logs=False,
        )
        with patch.dict(
            os.environ,
            {"E2E_TIMING_ENABLED": "true", "GITHUB_ACTIONS": "true"},
            clear=True,
        ), redirect_stdout(output):
            post.run()

        events = events_from_output(output)
        stage_events = [event for event in events if event["phase"] == "post-test-stage"]
        self.assertEqual(["start", "end"], [event["event"] for event in stage_events])
        self.assertEqual("post-cluster-test", stage_events[0]["name"])
        self.assertGreaterEqual(stage_events[-1]["duration_ms"], 0)

    def test_final_post_emits_aggregate_stage_span(self):
        output = io.StringIO()
        post = FinalPost(handle_e2e_progress_failures=False)
        with patch.dict(
            os.environ,
            {"E2E_TIMING_ENABLED": "true", "GITHUB_ACTIONS": "true"},
            clear=True,
        ), patch.object(post, "run_with_best_effort", return_value=True), redirect_stdout(output):
            post.run()

        events = events_from_output(output)
        stage_events = [event for event in events if event["phase"] == "post-test-stage"]
        self.assertEqual(["start", "end"], [event["event"] for event in stage_events])
        self.assertEqual("final-post", stage_events[0]["name"])
        self.assertGreaterEqual(stage_events[-1]["duration_ms"], 0)

    def test_best_effort_command_emits_safe_operation_name(self):
        output = io.StringIO()
        test_runner = RunWithBestEffortMixin()
        with patch.dict(
            os.environ,
            {
                "E2E_TIMING_ENABLED": "true",
                "GITHUB_ACTIONS": "true",
                "GITHUB_RUN_ID": "12345",
                "CI_JOB_NAME": "unit-test-lane",
            },
            clear=True,
        ), patch("post_tests.subprocess.run") as run_command, redirect_stdout(output):
            self.assertTrue(
                test_runner.run_with_best_effort(
                    [
                        "scripts/ci/collect-service-logs.sh",
                        "stackrox",
                        "/tmp/private-path",
                    ],
                    timeout=10,
                )
            )

        self.assertTrue(run_command.called)
        events = events_from_output(output)
        self.assertEqual("collect-service-logs.sh", events[0]["name"])
        self.assertEqual({"namespace": "stackrox"}, events[0]["attributes"])
        self.assertNotIn("/tmp/private-path", json.dumps(events))
        self.assertEqual("success", events[-1]["outcome"])

    def test_best_effort_timeout_keeps_existing_failure_behavior(self):
        output = io.StringIO()
        test_runner = RunWithBestEffortMixin()
        with patch.dict(
            os.environ,
            {"E2E_TIMING_ENABLED": "true", "GITHUB_ACTIONS": "true"},
            clear=True,
        ), patch(
            "post_tests.subprocess.run",
            side_effect=subprocess.TimeoutExpired(["fake"], 1),
        ), redirect_stdout(output):
            self.assertFalse(
                test_runner.run_with_best_effort(
                    ["scripts/ci/collect-service-logs.sh", "stackrox"],
                    timeout=1,
                )
            )

        self.assertEqual(1, test_runner.exitstatus)
        events = events_from_output(output)
        self.assertEqual("timeout", events[-1]["outcome"])


if __name__ == "__main__":
    unittest.main()
