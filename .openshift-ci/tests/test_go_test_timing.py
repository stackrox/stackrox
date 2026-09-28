import io
import json
import os
import unittest
from contextlib import redirect_stdout
from unittest.mock import patch

from go_test_timing import process_events


def go_event(action, package="example.test/pkg", test=None, timestamp=None, **extra):
    event = {"Action": action, "Package": package}
    if test is not None:
        event["Test"] = test
    if timestamp is not None:
        event["Time"] = timestamp
    event.update(extra)
    return json.dumps(event)


def timing_events(output):
    return [
        json.loads(line[len("e2e_timing "):])
        for line in output.splitlines()
        if line.startswith("e2e_timing ")
    ]


class TestGoTestTiming(unittest.TestCase):
    def setUp(self):
        self.env = patch.dict(
            os.environ,
            {
                "E2E_TIMING_ENABLED": "true",
                "E2E_TIMING_RUN_ID": "gha:123:1:go-tests",
                "E2E_TIMING_LANE_ID": "gha.e2e.nongroovy.gke",
            },
            clear=True,
        )
        self.env.start()

    def tearDown(self):
        self.env.stop()

    def test_emits_package_and_test_case_spans_and_preserves_verbose_output(self):
        fixture = [
            go_event("start", timestamp="2026-09-25T12:00:00Z"),
            go_event("run", test="TestAPI", timestamp="2026-09-25T12:00:01Z"),
            go_event(
                "output",
                test="TestAPI",
                Output="    api output\n",
            ),
            go_event(
                "pass",
                test="TestAPI",
                timestamp="2026-09-25T12:00:02Z",
                Elapsed=1.25,
            ),
            go_event(
                "pass",
                timestamp="2026-09-25T12:00:03Z",
                Elapsed=3.0,
            ),
        ]
        output = io.StringIO()
        with redirect_stdout(output):
            process_events(fixture)

        events = timing_events(output.getvalue())
        test_events = [event for event in events if event["phase"] == "test-case"]
        package_events = [event for event in events if event["phase"] == "test-suite"]
        self.assertEqual(["start", "end"], [event["event"] for event in test_events])
        self.assertEqual(test_events[0]["span_id"], test_events[1]["span_id"])
        self.assertEqual("go:example.test/pkg::TestAPI", test_events[0]["name"])
        self.assertEqual("gha:123:1:go-tests", test_events[0]["run_id"])
        self.assertEqual("gha.e2e.nongroovy.gke", test_events[0]["lane_id"])
        self.assertEqual("2026-09-25T12:00:01Z", test_events[0]["timestamp"])
        self.assertEqual("2026-09-25T12:00:02Z", test_events[1]["timestamp"])
        self.assertNotIn("duration_ms", test_events[1])
        self.assertEqual("success", test_events[1]["outcome"])
        self.assertEqual(["start", "end"], [event["event"] for event in package_events])
        self.assertIn("=== RUN   TestAPI", output.getvalue())
        self.assertIn("api output\n", output.getvalue())
        self.assertIn("--- PASS: TestAPI (1.25s)", output.getvalue())

    def test_nested_subtest_is_a_distinct_span_with_parent_metadata(self):
        fixture = [
            go_event("run", test="TestParent/child", timestamp="2026-09-25T12:00:01Z"),
            go_event(
                "pass",
                test="TestParent/child",
                timestamp="2026-09-25T12:00:02Z",
                Elapsed=0.5,
            ),
        ]
        output = io.StringIO()
        with redirect_stdout(output):
            process_events(fixture)

        event = timing_events(output.getvalue())[0]
        self.assertEqual("TestParent", event["attributes"]["parent_test"])
        self.assertEqual("2", event["attributes"]["test_depth"])

    def test_failure_closes_the_span_as_failure_and_keeps_failure_output(self):
        fixture = [
            go_event("run", test="TestBroken", timestamp="2026-09-25T12:00:01Z"),
            go_event(
                "output",
                test="TestBroken",
                Output="    assertion failed\n",
            ),
            go_event(
                "fail",
                test="TestBroken",
                timestamp="2026-09-25T12:00:02Z",
                Elapsed=1.0,
            ),
            go_event("fail", timestamp="2026-09-25T12:00:03Z", Elapsed=2.0),
        ]
        output = io.StringIO()
        with redirect_stdout(output):
            process_events(fixture)

        events = timing_events(output.getvalue())
        test_end = next(
            event for event in events
            if event["phase"] == "test-case" and event["event"] == "end"
        )
        self.assertEqual("failure", test_end["outcome"])
        self.assertIn("assertion failed", output.getvalue())
        self.assertIn("--- FAIL: TestBroken", output.getvalue())
        self.assertIn("FAIL\texample.test/pkg", output.getvalue())

    def test_activity_markers_become_shared_spans_and_are_removed_from_test_output(self):
        marker = {
            "event": "start",
            "span_id": "test-activity:123:1",
            "activity": "fixture_k8s_deployment_ready_wait",
            "helper": "waitForDeploymentReadyInK8s",
            "attributes": {
                "deployment_name": "fixture",
                "namespace": "qa",
                "test_name": "TestFixture",
            },
        }
        end_marker = {**marker, "event": "end"}
        fixture = [
            go_event("run", test="TestFixture", timestamp="2026-09-25T12:00:01Z"),
            go_event(
                "output",
                test="TestFixture",
                Output=(
                    "    common.go:1: e2e_timing_activity "
                    + json.dumps(marker)
                    + "\n"
                ),
                timestamp="2026-09-25T12:00:02Z",
            ),
            go_event(
                "output",
                test="TestFixture",
                Output=(
                    "    common.go:1: e2e_timing_activity "
                    + json.dumps(end_marker)
                    + "\n"
                ),
                timestamp="2026-09-25T12:00:04Z",
            ),
        ]
        output = io.StringIO()
        with redirect_stdout(output):
            process_events(fixture)

        events = [
            event
            for event in timing_events(output.getvalue())
            if event["phase"] == "test-activity"
        ]
        self.assertEqual(["start", "end"], [event["event"] for event in events])
        self.assertEqual(events[0]["span_id"], events[1]["span_id"])
        self.assertEqual(
            "activity:fixture_k8s_deployment_ready_wait:waitForDeploymentReadyInK8s",
            events[0]["name"],
        )
        self.assertEqual("TestFixture", events[0]["attributes"]["test_name"])
        self.assertEqual("2026-09-25T12:00:02Z", events[0]["timestamp"])
        self.assertEqual("2026-09-25T12:00:04Z", events[1]["timestamp"])
        self.assertNotIn("e2e_timing_activity", output.getvalue())

    def test_malformed_activity_attributes_are_preserved_as_regular_test_output(self):
        marker = {"event": "start", "span_id": "test-activity:123:1", "attributes": []}
        fixture = [
            go_event(
                "output",
                test="TestFixture",
                Output="    common.go:1: e2e_timing_activity " + json.dumps(marker) + "\n",
                timestamp="2026-09-25T12:00:02Z",
            ),
        ]
        output = io.StringIO()
        with redirect_stdout(output):
            process_events(fixture)

        self.assertIn("e2e_timing_activity", output.getvalue())
        self.assertFalse(timing_events(output.getvalue()))

    def test_non_event_json_line_is_preserved(self):
        line = '{"diagnostic":"not a go test event"}\n'
        output = io.StringIO()
        with redirect_stdout(output):
            process_events([line])

        self.assertEqual(line, output.getvalue())

    def test_cached_result_is_marked_without_an_execution_interval(self):
        output = io.StringIO()
        with redirect_stdout(output):
            process_events([go_event("pass", Elapsed=0.1)])

        event = timing_events(output.getvalue())[0]
        self.assertEqual("skipped", event["event"])
        self.assertEqual("go-test-result-without-run-event", event["reason"])
        self.assertNotIn("duration_ms", event)


if __name__ == "__main__":
    unittest.main()
