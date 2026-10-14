import ast
import os
from pathlib import Path
import unittest
from unittest.mock import Mock, patch

from clusters import GKECluster


class TestGKEPilot(unittest.TestCase):
    @patch("clusters.signal.signal")
    @patch("clusters.subprocess.run")
    @patch("clusters.subprocess.Popen")
    def test_provision_exports_job_overrides(self, popen, _run, _signal):
        popen.return_value.__enter__.return_value.wait.return_value = 0
        with patch.dict("os.environ", {"NUM_NODES": "3", "MACHINE_TYPE": "e2-standard-4"}):
            cluster = GKECluster("pilot", num_nodes=1, machine_type="e2-standard-8")
            cluster._provision()  # pylint: disable=protected-access
            self.assertEqual("1", os.environ["NUM_NODES"])
            self.assertEqual("e2-standard-8", os.environ["MACHINE_TYPE"])

    def test_selected_jobs_opt_in(self):
        root = Path(__file__).resolve().parents[2]
        for family in ("qa", "nongroovy", "ui"):
            with self.subTest(family=family):
                tree = ast.parse((root / "scripts/ci/jobs" / f"gke_{family}_e2e_tests.py").read_text())
                calls = [node for node in ast.walk(tree) if isinstance(node, ast.Call)
                         and isinstance(node.func, ast.Name) and node.func.id == "GKECluster"]
                self.assertEqual(1, len(calls))
                options = {keyword.arg: ast.literal_eval(keyword.value) for keyword in calls[0].keywords}
                self.assertEqual({"sample_resources": True}, options)

    @patch("clusters.set_ci_shared_export")
    @patch.object(GKECluster, "_provision")
    @patch("clusters.ResourceSampler")
    def test_sampler_starts_after_provision(self, sampler, provision, _export):
        order = Mock()
        order.attach_mock(provision, "provision")
        order.attach_mock(sampler.return_value.start, "start")
        with patch.dict("os.environ", {"ARTIFACT_DIR": "/tmp/artifacts"}):
            GKECluster("pilot", sample_resources=True).provision()
        self.assertEqual(["provision", "start"], [call[0] for call in order.mock_calls])
        sampler.assert_called_once_with(Path("/tmp/artifacts/resource-usage"))

    @patch("clusters.set_ci_shared_export")
    @patch.object(GKECluster, "_provision")
    @patch("clusters.ResourceSampler")
    def test_other_clusters_do_not_sample(self, sampler, _provision, _export):
        GKECluster("other").provision()
        sampler.assert_not_called()

    @patch("clusters.set_ci_shared_export")
    @patch.object(GKECluster, "_provision", side_effect=RuntimeError("provision failed"))
    @patch("clusters.ResourceSampler")
    def test_failed_provision_does_not_start_sampler(self, sampler, _provision, _export):
        with self.assertRaisesRegex(RuntimeError, "provision failed"):
            GKECluster("pilot", sample_resources=True).provision()
        sampler.assert_not_called()

    @patch("clusters.set_ci_shared_export")
    @patch.object(GKECluster, "_provision")
    @patch("clusters.ResourceSampler", side_effect=OSError("cannot write artifacts"))
    def test_sampler_failure_does_not_fail_provision(self, _sampler, _provision, _export):
        with patch.dict("os.environ", {"ARTIFACT_DIR": "/tmp/artifacts"}):
            GKECluster("pilot", sample_resources=True).provision()

    @patch("clusters.set_ci_shared_export")
    @patch.object(GKECluster, "_teardown", side_effect=RuntimeError("test teardown failure"))
    def test_cancel_stops_sampler_before_teardown(self, teardown, _export):
        cluster = GKECluster("pilot")
        cluster.resource_sampler = Mock()
        order = Mock()
        order.attach_mock(cluster.resource_sampler.stop, "stop")
        order.attach_mock(teardown, "teardown")
        with self.assertRaisesRegex(RuntimeError, "test teardown failure"):
            cluster.sigint_handler(None, None)
        self.assertEqual(["stop", "teardown"], [call[0] for call in order.mock_calls])
        teardown.assert_called_once_with(True)
