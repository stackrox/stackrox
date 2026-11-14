"""Cluster-free tests for node-name polling and nongroovy integration."""

import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import Mock, patch
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


class Clock:
    def __init__(self):
        self.now = 0

    def monotonic(self):
        return self.now

    def sleep(self, seconds):
        self.now += seconds


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
                cancellation = monitor.Cancellation()
                if cancelled:
                    cancellation.handle(signal.SIGTERM, None)
                monitor.write_report(directory, 1, errors, cancellation)
                case = ET.parse(directory / "node-monitor.xml").find(".//testcase")
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
                failed = monitor.write_report(directory, expected, [], monitor.Cancellation())
                self.assertEqual(failed, mismatch)
                report = ET.parse(directory / "node-monitor.xml").getroot()
                self.assertEqual(report.get("failures"), str(int(mismatch)))
                self.assertEqual(len(list(directory.glob("node-snapshot-*.json"))), len(replies))
                self.assertEqual(run.call_args.kwargs["env"], {"KUBECONFIG": "test-config"})
                self.assertEqual(run.call_args.kwargs["timeout"], 10)
                self.assertEqual(
                    run.call_args.args[0], ["kubectl", "--request-timeout=10s", "get", "nodes", "-o", "json"]
                )


class NodeMonitorTest(unittest.TestCase):
    def run_fake(self, replies, **options):
        clock = Clock()
        test = Mock(pid=123, returncode=options.get("code", 0))

        def product_poll():
            cancel_at = options.get("cancel_at")
            if cancel_at is not None and clock.now >= cancel_at:
                signal.getsignal(signal.SIGTERM)(signal.SIGTERM, None)
            return options.get("code", 0) if clock.now >= options.get("duration", 0) else None

        test.poll.side_effect = product_poll
        with tempfile.TemporaryDirectory() as directory, patch.dict(os.environ, ARTIFACT_DIR=directory), patch.object(
            monitor.subprocess, "run", side_effect=kubectl_responses(replies)
        ) as collect, patch.object(monitor.subprocess, "Popen", return_value=test) as launch, patch.object(
            monitor, "terminate"
        ) as terminate, patch.object(
            monitor.time, "monotonic", side_effect=clock.monotonic
        ), patch.object(
            monitor.time, "sleep", side_effect=clock.sleep
        ):
            result = monitor.run(
                ["product"], timeout=options.get("timeout"), env=options.get("env", {"NUM_NODES": "1"})
            )
            report = ET.parse(Path(directory) / "node-monitor.xml").getroot()
        return result, report, collect, launch, terminate

    def test_five_minute_cadence_and_final_sample(self):
        self.assertEqual(monitor.POLL_INTERVAL, 300)
        result, _, collect, _, _ = self.run_fake([reply("a"), reply("a"), reply("b"), reply("b")], duration=601)
        self.assertEqual(collect.call_count, 4)  # Initial, 5m, 10m, final.
        self.assertTrue(result.monitor_failed)

    def test_startup_errors_are_advisory(self):
        cases = {
            "failed poll": ([reply(code=1), reply("a")], {"NUM_NODES": "1"}, "exited 1"),
            "empty baseline": ([reply(), reply("a")], {"NUM_NODES": "1"}, "no nodes"),
            "invalid configuration": ([], {"NUM_NODES": "0"}, "positive integer"),
        }
        for name, (replies, env, error) in cases.items():
            with self.subTest(name=name):
                result, report, _, launch, _ = self.run_fake(replies, env=env)
                self.assertTrue(result.monitor_failed)
                self.assertIn(error, report.findtext(".//failure"))
                self.assertEqual(result.exit_code, 0)
                launch.assert_called_once()

    def test_timeout_and_cancellation(self):
        cases = {
            "product signal": (None, None, 143, "0", 2),
            "timeout": (1, None, 124, "0", 2),
            "cancelled": (None, 1, 143, "1", 1),
        }
        for name, (timeout, cancel_at, code, skipped, polls) in cases.items():
            with self.subTest(name=name):
                previous = signal.getsignal(signal.SIGTERM)
                result, report, collect, _, terminate = self.run_fake(
                    [reply("a")] * polls,
                    duration=100,
                    timeout=timeout,
                    cancel_at=cancel_at,
                    code=-signal.SIGTERM if name == "product signal" else 0,
                )
                self.assertEqual(result.exit_code, code)
                self.assertEqual(result.timed_out, timeout is not None)
                self.assertEqual(report.get("skipped"), skipped)
                self.assertEqual(collect.call_count, polls)
                terminate.assert_called_once()
                self.assertEqual(signal.getsignal(signal.SIGTERM), previous)

    def test_missing_node_count_skips_all_monitoring(self):
        cases = {"unset success": ({}, 0), "unset failure": ({}, 42), "empty": ({"NUM_NODES": ""}, 0)}
        for name, (env, code) in cases.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory, patch.dict(
                os.environ, ARTIFACT_DIR=directory
            ), patch.object(monitor, "sample_nodes") as poller, patch.object(
                monitor, "write_report"
            ) as report, patch.object(
                monitor.subprocess, "Popen", return_value=Mock(returncode=code)
            ) as launch, patch.object(
                monitor, "terminate"
            ):
                result = monitor.run(["product"], env=env)
                self.assertEqual(result.exit_code, code)
                self.assertFalse(result.monitor_failed)
                poller.assert_not_called()
                report.assert_not_called()
                launch.assert_called_once_with(["product"], env=env, start_new_session=True)
                self.assertEqual(list(Path(directory).iterdir()), [])

    def test_launch_error_is_logged_without_monitoring(self):
        cases = {"unset": {}, "empty": {"NUM_NODES": ""}}
        for name, env in cases.items():
            with self.subTest(name=name), patch.object(
                monitor.subprocess, "Popen", side_effect=FileNotFoundError("missing command")
            ), patch.object(monitor, "write_report") as report, patch("builtins.print") as log:
                result = monitor.run(["missing-command"], env=env)
                self.assertEqual(result.exit_code, 1)
                report.assert_not_called()
                log.assert_called_once_with("ERROR: Could not run test command: missing command", flush=True)

    def test_cleanup_survives_report_write_failure(self):
        previous = signal.getsignal(signal.SIGTERM)
        with tempfile.TemporaryDirectory() as directory, patch.dict(os.environ, ARTIFACT_DIR=directory), patch.object(
            monitor.subprocess, "run", side_effect=kubectl_responses([reply("a"), reply("a")])
        ), patch.object(monitor.subprocess, "Popen", return_value=Mock(returncode=0)) as launch, patch.object(
            monitor, "write_report", side_effect=OSError("disk full")
        ), patch.object(
            monitor, "terminate"
        ) as terminate:
            result = monitor.run(["product"], env={"NUM_NODES": "1"})
            self.assertEqual(result.exit_code, 0)
            self.assertTrue(result.monitor_failed)
        terminate.assert_called_once_with(launch.return_value)
        self.assertEqual(signal.getsignal(signal.SIGTERM), previous)


class TerminationTest(unittest.TestCase):

    def test_real_process_group_with_resistant_descendant(self):
        # The leader exits on TERM, while its child ignores it. KILL must still
        # target the process group after the leader has been reaped.
        script = """import os, signal, time
if os.fork() == 0:
    signal.signal(signal.SIGTERM, signal.SIG_IGN)
    print('ready', flush=True)
    time.sleep(60)
else:
    time.sleep(60)
"""
        # Explicit finally below must also kill descendants if an assertion fails.
        # pylint: disable=consider-using-with
        process = subprocess.Popen(
            [sys.executable, "-c", script], start_new_session=True, stdout=subprocess.PIPE, text=True
        )
        try:
            self.assertEqual(process.stdout.readline().strip(), "ready")
            start = time.monotonic()
            monitor.terminate(process, grace=0.2)
            self.assertIsNotNone(process.returncode)
            self.assertGreaterEqual(time.monotonic() - start, 0.19)
        finally:
            try:
                os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            process.wait()
            process.stdout.close()


class NonGroovyIntegrationTest(unittest.TestCase):

    def test_monitoring_preserves_outputs_timeout_and_failure(self):
        cases = {
            "passed": monitor.RunResult(0, False),
            "product failure": monitor.RunResult(42, False),
            "monitor failure": monitor.RunResult(0, True),
            "timeout": monitor.RunResult(124, False, True),
        }
        for name, result in cases.items():
            with self.subTest(name=name), patch.dict(os.environ, {"NUM_NODES": "1"}, clear=True), patch.object(
                monitor, "run", return_value=result
            ) as run:
                test = NonGroovyE2e()
                if result.timed_out:
                    with self.assertRaises(subprocess.TimeoutExpired):
                        test.run()
                elif result.exit_code:
                    with self.assertRaisesRegex(RuntimeError, str(result.exit_code)):
                        test.run()
                else:
                    test.run()
                self.assertIn(test.TEST_OUTPUT_DIR, test.test_outputs)
                run.assert_called_once_with(
                    ["tests/e2e/run.sh", test.TEST_OUTPUT_DIR],
                    timeout=test.TEST_TIMEOUT,
                    env={"NUM_NODES": "1", "ROX_CI_OUTPUT_DIR": test.TEST_OUTPUT_DIR},
                )


KUBECTL = """#!{python}
import json
import os
from pathlib import Path
import sys
import time

root = Path(os.environ['FAKE_KUBECTL_ROOT'])
with (root / 'requests.jsonl').open('a') as stream:
    stream.write(json.dumps(sys.argv[1:]) + '\\n')
counter = root / 'counter'
index = int(counter.read_text()) if counter.exists() else 0
counter.write_text(str(index + 1))
responses = json.loads((root / 'responses.json').read_text())
response = responses[min(index, len(responses) - 1)]
time.sleep(response.get('delay', 0))
print(response.get('stdout', ''), flush=True)
print(response.get('stderr', ''), file=sys.stderr, flush=True)
sys.exit(response.get('returncode', 0))
"""


def response(*names):
    return {"stdout": json.dumps({"items": [{"metadata": {"name": name}} for name in names]})}


class PollingTransportTest(unittest.TestCase):
    def setUp(self):
        # This resource spans setUp and each test; addCleanup owns its lifetime.
        self.temporary = tempfile.TemporaryDirectory()  # pylint: disable=consider-using-with
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        kubectl = self.root / "kubectl"
        kubectl.write_text(KUBECTL.format(python=sys.executable))
        kubectl.chmod(0o755)
        self.env = dict(
            os.environ,
            ARTIFACT_DIR=str(self.root / "artifacts"),
            NUM_NODES="1",
            FAKE_KUBECTL_ROOT=str(self.root),
            PATH=str(self.root) + os.pathsep + os.environ.get("PATH", ""),
        )

    def run_monitor(self, responses, script="pass", timeout=None):
        (self.root / "responses.json").write_text(json.dumps(responses))
        with patch.dict(os.environ, self.env, clear=True):
            result = monitor.run([sys.executable, "-c", script], env=self.env, timeout=timeout)
        report = ET.parse(self.root / "artifacts/node-monitor.xml").getroot()
        records = [path.read_text() for path in sorted((self.root / "artifacts").glob("node-snapshot-*.json"))]
        return result, report, records

    def test_poll_subprocess_timeout_is_bounded(self):
        with patch.object(monitor, "API_TIMEOUT", 0.5):
            result, report, records = self.run_monitor([{"delay": 60}])
        self.assertTrue(result.monitor_failed)
        self.assertIn("timed out", report.findtext(".//failure"))
        self.assertEqual(result.exit_code, 0)
        self.assertEqual(len(records), 2)
        self.assertTrue((self.root / "artifacts/node-monitor.log").exists())

    def test_cli_entrypoint_preserves_exit_code_without_outputs(self):
        cases = {
            "product success": ([response("a"), response("a")], 0, False),
            "malformed snapshot": ([{"stdout": "broken JSON"}], 0, True),
            "product failure": ([response("a"), response("a")], 42, False),
            "passed with replacement": ([response("old"), response("new")], 0, True),
            "initial poll error": ([{"returncode": 1, "stderr": "forbidden"}, response("a")], 0, True),
            "failed with replacement": ([response("old"), response("new")], 42, True),
        }
        for name, (replies, code, failed) in cases.items():
            with self.subTest(name=name):
                (self.root / "responses.json").write_text(json.dumps(replies))
                (self.root / "counter").write_text("0")
                output = self.root / "github-output"
                output.write_text("")
                env = dict(self.env, GITHUB_OUTPUT=str(output))
                completed = subprocess.run(
                    [
                        sys.executable,
                        str(Path(monitor.__file__).resolve()),
                        "run",
                        "--",
                        sys.executable,
                        "-c",
                        f"raise SystemExit({code})",
                    ],
                    env=env,
                    capture_output=True,
                    text=True,
                    timeout=10,
                    check=False,
                )
                self.assertEqual(completed.returncode, code, completed.stderr)
                self.assertEqual(output.read_text(), "")
                report = ET.parse(self.root / "artifacts/node-monitor.xml").getroot()
                self.assertEqual(report.get("failures"), str(int(failed)))
                self.assertIsNone(report.find(".//system-out"))
                self.assertNotIn("Run:", completed.stdout)
                self.assertNotIn("::error::", completed.stdout)
                self.assertNotIn("::warning::", completed.stdout)

    def test_cli_skips_monitoring_without_num_nodes(self):
        self.env.pop("NUM_NODES")
        output = self.root / "github-output"
        self.env["GITHUB_OUTPUT"] = str(output)
        completed = subprocess.run(
            [
                sys.executable,
                str(Path(monitor.__file__).resolve()),
                "run",
                "--",
                sys.executable,
                "-c",
                "raise SystemExit(42)",
            ],
            env=self.env,
            capture_output=True,
            text=True,
            timeout=10,
            check=False,
        )
        self.assertEqual(completed.returncode, 42, completed.stderr)
        self.assertFalse(output.exists())
        self.assertFalse((self.root / "requests.jsonl").exists())
        self.assertFalse((self.root / "artifacts").exists())

    def test_product_timeout_with_and_without_monitoring(self):
        cases = {"disabled": False, "enabled": True}
        for name, enabled in cases.items():
            with self.subTest(name=name):
                env = dict(self.env)
                if not enabled:
                    env.pop("NUM_NODES")
                (self.root / "responses.json").write_text(json.dumps([response("a")]))
                with patch.dict(os.environ, env, clear=True):
                    result = monitor.run([sys.executable, "-c", "import time; time.sleep(60)"], env=env, timeout=0.1)
                self.assertEqual(result.exit_code, 124)
                self.assertTrue(result.timed_out)
                self.assertFalse(result.monitor_failed)
                if enabled:
                    self.assertEqual(len(list((self.root / "artifacts").glob("node-snapshot-*.json"))), 2)
                else:
                    self.assertFalse((self.root / "requests.jsonl").exists())
                    self.assertFalse((self.root / "artifacts").exists())

    def test_missing_command_is_rejected(self):
        with patch("sys.argv", ["node_disruption.py", "run", "--"]), self.assertRaises(SystemExit):
            monitor.main()


if __name__ == "__main__":
    unittest.main()
