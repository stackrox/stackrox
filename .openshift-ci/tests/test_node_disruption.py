"""Cluster-free tests for node-name polling and nongroovy integration."""

import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch
import xml.etree.ElementTree as ET

import node_disruption as monitor
from ci_tests import NonGroovyE2e


def snapshot(*names, ready="True"):
    return {
        "items": [
            {"metadata": {"name": name}, "status": {"conditions": [{"type": "Ready", "status": ready}]}}
            for name in names
        ]
    }


def reply(*names, ready="True", code=0, stderr=""):
    return subprocess.CompletedProcess("kubectl", code, json.dumps(snapshot(*names, ready=ready)), stderr)


def kubectl_responses(responses):
    """Emulate kubectl writing directly to the supplied artifact streams."""
    responses = iter(responses)

    def collect(*_args, **kwargs):
        result = next(responses)
        if isinstance(result, Exception):
            if isinstance(result, subprocess.TimeoutExpired):
                kwargs["stdout"].write((result.stdout or b"").decode())
                kwargs["stderr"].write((result.stderr or b"").decode())
            raise result
        kwargs["stdout"].write(result.stdout)
        kwargs["stderr"].write(result.stderr)
        return result

    return collect


class CollectionTest(unittest.TestCase):
    def test_report_is_discoverable_by_prow(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "node-names.txt").write_text("a\n", encoding="utf-8")
            monitor.write_report(root, 1, [])
            report = root / "junit-node-monitor.xml"
            self.assertTrue(report.is_file())
            self.assertRegex(report.name, r".*junit.*\.xml")
            self.assertEqual(ET.parse(report).getroot().get("failures"), "0")
            self.assertFalse((root / "junit-node-monitor.xml.tmp").exists())
            self.assertFalse((root / "node-monitor.xml").exists())

    def test_expected_count(self):
        cases = {
            "unset": ({}, None),
            "empty": ({"NUM_NODES": ""}, None),
            "one": ({"NUM_NODES": "1"}, 1),
            "many": ({"NUM_NODES": "3"}, 3),
        }
        for name, (env, expected) in cases.items():
            with self.subTest(name=name):
                self.assertEqual(monitor.expected_node_count(env), expected)
        invalid = {"zero": "0", "negative": "-1", "text": "three", "fraction": "1.5"}
        for name, value in invalid.items():
            with self.subTest(name=name), self.assertRaisesRegex(RuntimeError, "positive integer"):
                monitor.expected_node_count({"NUM_NODES": value})

    def test_junit_contains_only_outcome_and_failure_explanation(self):
        cases = {
            "passed": ([], False, None),
            "failed": (["kubectl failed"], False, "failure"),
            "cancelled": ([], True, "skipped"),
        }
        for name, (errors, cancelled, outcome) in cases.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory, patch("builtins.print") as log:
                directory = Path(directory)
                (directory / "node-names.txt").write_text("a\n")
                monitor.write_report(directory, 1, errors, cancelled)
                case = ET.parse(directory / "junit-node-monitor.xml").find(".//testcase")
                self.assertEqual([child.tag for child in case], [outcome] if outcome else [])
                log.assert_called_once()
                self.assertIn("Expected 1 nodes; observed 1 unique names: a", log.call_args.args[0])
                if outcome == "failure":
                    self.assertIn("kubectl failed", case.findtext("failure"))
                    self.assertIn("kubectl failed", log.call_args.args[0])
                if outcome == "skipped":
                    self.assertIsNone(case.find("skipped").text)

    def test_names_only_verdict(self):
        cases = {
            "stable duplicates": ([reply("a", "b"), reply("b", "a")], 2, False),
            "replacement": ([reply("a"), reply("b")], 1, True),
            "too few": ([reply("a")], 2, True),
            "too many": ([reply("a", "b")], 1, True),
            "not ready is ignored": ([reply("a", ready="False")], 1, False),
            "disappearance is ignored": ([reply("a", "b"), reply("a")], 2, False),
            "empty later sample is ignored": ([reply("a"), reply()], 1, False),
            "unobserved transient is ignored": ([reply("a"), reply("a")], 1, False),
        }
        for name, (replies, expected, mismatch) in cases.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory, patch.object(
                monitor.subprocess, "run", side_effect=kubectl_responses(replies)
            ) as run:
                directory = Path(directory)
                for _ in range(len(replies)):
                    monitor.sample_nodes(directory, {"KUBECONFIG": "test-config"})
                failed = monitor.write_report(directory, expected, [], False)
                self.assertEqual(failed, mismatch)
                report = ET.parse(directory / "junit-node-monitor.xml").getroot()
                self.assertEqual(report.get("failures"), str(int(mismatch)))
                self.assertEqual(len(list(directory.glob("node-snapshot-*.json"))), len(replies))
                self.assertEqual(run.call_args.kwargs["env"], {"KUBECONFIG": "test-config"})
                self.assertEqual(run.call_args.kwargs["timeout"], 10)
                self.assertEqual(
                    run.call_args.args[0], ["kubectl", "--request-timeout=10s", "get", "nodes", "-o", "json"]
                )


class NodeMonitorTest(unittest.TestCase):
    def test_lifecycle(self):
        cases = {
            "success": (None, False, False),
            "product failure": (RuntimeError("product"), False, False),
            "timeout": (subprocess.TimeoutExpired("product", 1), False, False),
            "interrupt": (KeyboardInterrupt(), False, True),
            "exit": (SystemExit(2), False, True),
            "report failure": (RuntimeError("product"), True, False),
        }
        for name, (error, report_error, skipped) in cases.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory, patch.object(
                monitor, "sample_nodes", return_value=["a"]
            ) as sample, patch.object(
                monitor, "write_report", side_effect=OSError("disk full") if report_error else None
            ) as report, patch.object(monitor.threading, "Thread") as thread:
                try:
                    with monitor.monitor({"NUM_NODES": "1", "ARTIFACT_DIR": directory}):
                        if error:
                            raise error
                except BaseException as caught:
                    self.assertIs(caught, error)
                else:
                    self.assertIsNone(error)
                self.assertEqual(sample.call_count, 1 if skipped else 2)
                thread.return_value.join.assert_called_once()
                self.assertEqual(report.call_args.args[-1], skipped)

    def test_disabled(self):
        for env in ({}, {"NUM_NODES": ""}):
            with self.subTest(env=env), patch.object(monitor, "sample_nodes") as sample:
                with monitor.monitor(env):
                    pass
                sample.assert_not_called()

    def test_errors_are_advisory(self):
        cases = {
            "invalid count": ("0", ["a"], None),
            "empty baseline": ("1", [], None),
            "poll failure": ("1", None, OSError("kubectl")),
            "thread failure": ("1", ["a"], None),
        }
        for name, (count, names, error) in cases.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory, patch.object(
                monitor, "sample_nodes", return_value=names, side_effect=error
            ), patch.object(monitor.threading, "Thread") as thread:
                if name == "thread failure":
                    thread.return_value.start.side_effect = RuntimeError("thread unavailable")
                with monitor.monitor({"NUM_NODES": count, "ARTIFACT_DIR": directory}):
                    pass
                report = ET.parse(Path(directory) / "junit-node-monitor.xml").getroot()
                self.assertEqual(report.get("failures"), "1")

    def test_periodic_polling_and_shutdown(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(
            monitor, "sample_nodes", return_value=["a"]
        ) as sample, patch.object(monitor.threading, "Event") as event, patch.object(
            monitor.threading, "Thread"
        ) as thread:
            event.return_value.wait.side_effect = [False, False, True]
            thread.side_effect = lambda **kwargs: unittest.mock.Mock(start=kwargs["target"])
            with monitor.monitor({"NUM_NODES": "1", "ARTIFACT_DIR": directory}):
                pass
            self.assertEqual(sample.call_count, 4)
            self.assertEqual(event.return_value.wait.call_count, 3)
            event.return_value.wait.assert_called_with(300)
            event.return_value.set.assert_called_once()


class NonGroovyIntegrationTest(unittest.TestCase):
    def test_original_runner_and_exception_preserved(self):
        for error in (None, RuntimeError("product"), subprocess.TimeoutExpired("product", 1)):
            with self.subTest(error=error), patch.object(monitor, "monitor") as context, patch.object(
                NonGroovyE2e, "run_with_graceful_kill", side_effect=error
            ) as run:
                context.return_value.__exit__.return_value = False
                test = NonGroovyE2e()
                if error:
                    with self.assertRaises(type(error)) as caught:
                        test.run()
                    self.assertIs(caught.exception, error)
                else:
                    test.run()
                run.assert_called_once_with(
                    ["tests/e2e/run.sh", test.TEST_OUTPUT_DIR],
                    test.TEST_TIMEOUT, output_dir=test.TEST_OUTPUT_DIR,
                )
                context.return_value.__exit__.assert_called_once()


if __name__ == "__main__":
    unittest.main()
