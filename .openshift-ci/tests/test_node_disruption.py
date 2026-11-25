"""Cluster-free tests for node-name polling and nongroovy integration."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch
import xml.etree.ElementTree as ET

import node_disruption as monitor
from ci_tests import NonGroovyE2e
from clusters import GKECluster
from runners import ClusterTestRunner


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
            self.assertFalse((root / "node-monitor.xml").exists())

    def test_junit_contains_only_outcome_and_failure_explanation(self):
        cases = {
            "passed": ([], False, None),
            "failed": (["kubectl failed"], False, "skipped"),
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
                if errors:
                    self.assertIn("kubectl failed", case.findtext("skipped"))
                    self.assertIn("kubectl failed", log.call_args.args[0])
                if cancelled:
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
                self.assertEqual(report.get("skipped"), str(int(mismatch)))
                self.assertEqual(len(list(directory.glob("node-snapshot-*.json"))), len(replies))
                self.assertEqual(run.call_args.kwargs["env"], {"KUBECONFIG": "test-config"})
                self.assertEqual(run.call_args.kwargs["timeout"], 30)
                self.assertEqual(
                    run.call_args.args[0], ["kubectl", "--request-timeout=30s", "get", "nodes", "-o", "json"]
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

    def test_empty_baseline_is_advisory_failure(self):
        cases = {"recovered": ([reply(), reply("a")], "1"), "empty": ([reply(), reply()], "1")}
        for name, (responses, failures) in cases.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory, patch.object(
                monitor.subprocess, "run", side_effect=kubectl_responses(responses)
            ), patch.object(monitor.threading, "Thread"):
                with monitor.monitor({"NUM_NODES": "1", "ARTIFACT_DIR": directory}):
                    pass
                report = ET.parse(Path(directory) / monitor.JUNIT_REPORT).getroot()
                self.assertEqual(report.get("skipped"), failures)

    def test_observed_baseline(self):
        cases = {
            "stable": (("a", "b"), "0"),
            "stable with conflicting NUM_NODES": (("a", "b"), "0"),
            "addition": (("a", "b", "c"), "1"),
            "removal": (("a",), "1"),
            "replacement": (("a", "c"), "1"),
        }
        for name, (final, failures) in cases.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory, patch.object(
                monitor.subprocess, "run", side_effect=kubectl_responses([reply("a", "b"), reply(*final)])
            ), patch.object(monitor.threading, "Thread"):
                env = {"ARTIFACT_DIR": directory}
                if name == "stable with conflicting NUM_NODES":
                    env["NUM_NODES"] = "5"
                with monitor.monitor(env, phase="part-1"):
                    pass
                report = ET.parse(Path(directory) / "part-1" / monitor.JUNIT_REPORT)
                self.assertEqual(report.getroot().get("skipped"), failures)

    def test_directory_failure_is_advisory(self):
        with patch.object(Path, "mkdir", side_effect=OSError("disk full")), patch.object(
            monitor, "sample_nodes"
        ) as sample, patch("builtins.print") as log:
            with monitor.monitor({"NUM_NODES": "1"}):
                pass
            sample.assert_not_called()
            self.assertIn("disk full", log.call_args.args[0])

    def test_errors_are_advisory(self):
        cases = {
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
                self.assertEqual(report.get("skipped"), "1")
                if name == "thread failure":
                    thread.return_value.join.assert_not_called()
                if name == "poll failure":
                    self.assertIn("Could not establish a node baseline", report.findtext(".//skipped"))

    def test_polling_recovers_from_startup_probe(self):
        for startup in (subprocess.TimeoutExpired("kubectl", 30), reply()):
            with self.subTest(startup=startup), tempfile.TemporaryDirectory() as directory:
                with patch.object(
                    monitor.subprocess, "run", side_effect=kubectl_responses([startup, reply("a"), reply("b")])
                ) as run, patch.object(monitor.threading, "Event") as event, patch.object(
                    monitor.threading, "Thread"
                ) as thread:
                    event.return_value.wait.side_effect = [False, True]
                    thread.side_effect = lambda **kwargs: unittest.mock.Mock(start=kwargs["target"])
                    with monitor.monitor({"NUM_NODES": "1", "ARTIFACT_DIR": directory}):
                        pass
                self.assertEqual(run.call_count, 3)
                report = ET.parse(Path(directory) / monitor.JUNIT_REPORT)
                self.assertEqual(report.getroot().get("failures"), "0")
                reason = report.findtext(".//skipped")
                self.assertIn("startup:", reason)
                self.assertIn("Nodes added: ['b']; removed: ['a']", reason)

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


class ActionsMonitorTest(unittest.TestCase):
    def test_start_waits_for_first_probe(self):
        for startup, reason in [(snapshot("a"), "Nodes added: ['b']; removed: ['a']"), (None, "startup:")]:
            with self.subTest(startup=startup), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                kubectl = root / "kubectl"
                kubectl.write_text('#!/bin/sh\nsleep 0.2\ncat "$RUNNER_TEMP/nodes.json"\n')
                kubectl.chmod(0o755)
                nodes = root / "nodes.json"
                if startup:
                    nodes.write_text(json.dumps(startup))
                env = dict(os.environ, PATH=directory + os.pathsep + os.defpath, RUNNER_TEMP=directory,
                           ARTIFACT_DIR=directory, NUM_NODES="1", GITHUB_RUN_ID="1", GITHUB_RUN_ATTEMPT="1",
                           GITHUB_JOB="qa")
                command = [sys.executable, monitor.__file__]
                subprocess.run([*command, "start", "--phase", "part-1"], env=env, timeout=10, check=True)
                try:
                    self.assertTrue((root / "node-monitor-1-1-qa-part-1/ready").is_file())
                    nodes.write_text(json.dumps(snapshot("b")))
                finally:
                    subprocess.run([*command, "finish", "--phase", "part-1"], env=env, timeout=10, check=True)
                report = ET.parse(root / "part-1" / monitor.JUNIT_REPORT)
                self.assertIn(reason, report.findtext(".//skipped"))

    def test_missing_startup_acknowledgement(self):
        for exit_code, message in [(1, "worker exited 1"), (None, "within 45 seconds")]:
            with self.subTest(exit_code=exit_code), tempfile.TemporaryDirectory() as directory, patch.object(
                monitor.subprocess, "Popen"
            ) as spawn, patch.object(monitor.time, "monotonic", side_effect=[0, 46]):
                state = Path(directory) / "state"
                spawn.return_value.poll.return_value = exit_code
                spawn.return_value.returncode = exit_code
                monitor.start_actions_monitor(state)
                args = SimpleNamespace(phase=None, outcome="success")
                if exit_code is None:
                    # A late first probe must still produce an advisory finding.
                    (state / "stop").write_text(args.outcome)
                    with patch.dict(os.environ, NUM_NODES="1", ARTIFACT_DIR=directory), patch.object(
                        monitor.subprocess, "run", side_effect=kubectl_responses([reply("a"), reply("a")])
                    ):
                        monitor.run_actions_worker(state, args)
                with patch.object(monitor.time, "monotonic", side_effect=[0, 91]):
                    monitor.finish_actions_monitor(state, Path(directory), args)
                report = ET.parse(Path(directory) / monitor.JUNIT_REPORT)
                self.assertIn(message, report.findtext(".//skipped"))


class NonGroovyIntegrationTest(unittest.TestCase):
    def test_original_runner_and_exception_preserved(self):
        cases = {
            "success": None,
            "product failure": RuntimeError("product"),
            "timeout": subprocess.TimeoutExpired("product", 1),
        }
        for name, error in cases.items():
            with self.subTest(name=name), patch.object(
                NonGroovyE2e, "run_with_graceful_kill", side_effect=error
            ) as run:
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

    def test_product_outcome_and_report(self):
        def sample(directory, _env):
            (directory / monitor.NODE_NAMES).write_text("a\n", encoding="utf-8")
            return ["a"]

        cases = {
            "success": (None, False),
            "product failure": (RuntimeError("product"), False),
            "timeout": (subprocess.TimeoutExpired("product", 1), False),
            "monitor failure": (None, True),
            "both fail": (RuntimeError("product"), True),
        }
        for name, (error, sampling_error) in cases.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory, patch.dict(
                monitor.os.environ, NUM_NODES="1", ARTIFACT_DIR=directory
            ), patch.object(monitor.threading, "Thread"), patch.object(
                monitor, "sample_nodes", side_effect=OSError("kubectl") if sampling_error else sample
            ), patch.object(NonGroovyE2e, "run_with_graceful_kill", side_effect=error) as product:
                cluster = GKECluster("test")
                cluster.provision = lambda: None
                cluster.teardown = unittest.mock.create_autospec(cluster.teardown)
                test = ClusterTestRunner(cluster=cluster, test=NonGroovyE2e())
                if error:
                    with self.assertRaises(type(error)) as caught:
                        test.run()
                    self.assertIs(caught.exception, error)
                else:
                    test.run()
                product.assert_called_once_with(
                    ["tests/e2e/run.sh", "/tmp/e2e-test-logs"],
                    180 * 60,
                    output_dir="/tmp/e2e-test-logs",
                )
                report = ET.parse(Path(directory) / monitor.JUNIT_REPORT)
                self.assertEqual(report.find(".//testcase").get("name"), "NoNodeDisruption")
                self.assertEqual(report.getroot().get("skipped"), str(int(sampling_error)))


if __name__ == "__main__":
    unittest.main()
