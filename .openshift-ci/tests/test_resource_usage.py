import json
from pathlib import Path
import subprocess
import tempfile
import threading
import unittest
from unittest.mock import Mock, patch

from resource_usage import ResourceSampler


class TestResourceSampler(unittest.TestCase):
    def setUp(self):
        # pylint: disable=consider-using-with
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.sampler = ResourceSampler(self.directory.name)

    def records(self):
        return [json.loads(line) for line in
                Path(self.directory.name, "samples.jsonl").read_text(encoding="utf8").splitlines()]

    @patch("resource_usage.subprocess.run")
    def test_usage_only(self, run):
        run.return_value = subprocess.CompletedProcess([], 0, "usage", "")
        self.sampler.sample(include_specs=False)
        records = self.records()
        self.assertEqual(2, len(records))
        self.assertEqual(["node-usage", "pod-usage"], [record["kind"] for record in records])
        for record in records:
            self.assertEqual("usage", record["output"])
            self.assertIn("timestamp", record)
        self.assertEqual(5, run.call_args.kwargs["timeout"])

    @patch("resource_usage.subprocess.run")
    def test_timeout_does_not_stop_sampling(self, run):
        run.side_effect = [subprocess.TimeoutExpired("kubectl", 5),
                           subprocess.CompletedProcess([], 1, "", "metrics unavailable")]
        self.sampler.sample(include_specs=False)
        records = self.records()
        self.assertIn("timed out", records[0]["error"])
        self.assertEqual(1, records[1]["returncode"])
        self.assertEqual("metrics unavailable", records[1]["error"])

    @patch("resource_usage.subprocess.run")
    def test_pod_snapshot_excludes_environment(self, run):
        pods = {"items": [{"metadata": {"name": "central", "namespace": "stackrox"},
                           "spec": {"nodeName": "node", "containers": [
                               {"name": "central", "env": [{"name": "PASSWORD", "value": "secret"}],
                                "resources": {"requests": {"cpu": "1"}}}],
                                    "initContainers": [{"name": "init", "resources": {"requests": {"cpu": "2"}}}]},
                           "status": {"phase": "Running"}}]}
        run.return_value = subprocess.CompletedProcess([], 0, json.dumps(pods), "")
        self.sampler.sample(include_specs=True)
        record = next(record for record in self.records() if record["kind"] == "pods")
        self.assertNotIn("secret", json.dumps(record))
        self.assertEqual({"requests": {"cpu": "1"}}, record["output"][0]["spec"]["containers"][0]["resources"])
        self.assertEqual("2", record["output"][0]["spec"]["initContainers"][0]["resources"]["requests"]["cpu"])

    def test_start_and_stop(self):
        sampled_event = threading.Event()
        sampled = Mock(side_effect=lambda **_kwargs: sampled_event.set())
        with patch.object(self.sampler, "sample", sampled):
            self.sampler.start()
            self.assertTrue(sampled_event.wait(timeout=1))
            self.sampler.stop()
        self.assertFalse(self.sampler.thread.is_alive())
        self.assertGreaterEqual(sampled.call_count, 1)

    def test_snapshots_every_other_sample(self):
        flags = []

        def sample(include_specs):
            flags.append(include_specs)
            if len(flags) == 4:
                self.sampler.stopped.set()

        with patch.object(ResourceSampler, "INTERVAL", 0), patch.object(self.sampler, "sample", side_effect=sample):
            self.sampler._run()  # pylint: disable=protected-access
        self.assertEqual([True, False, True, False], flags)

    @patch("resource_usage.subprocess.run")
    def test_stop_during_command_prevents_further_commands(self, run):
        def command(*_args, **_kwargs):
            self.sampler.stopped.set()
            return subprocess.CompletedProcess([], 0, "usage", "")

        run.side_effect = command
        self.sampler.sample(include_specs=True)
        self.assertEqual(1, run.call_count)

    @patch("resource_usage.subprocess.run")
    def test_stopped_sampler_runs_no_commands(self, run):
        self.sampler.stop()
        self.sampler.sample(include_specs=True)
        run.assert_not_called()
