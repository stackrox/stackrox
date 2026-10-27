import ast
import os
from pathlib import Path
import unittest
from unittest.mock import Mock, patch

from clusters import AutomationFlavorsCluster


class TestJobSampling(unittest.TestCase):
    def options(self, path, constructor):
        tree = ast.parse(path.read_text(encoding="utf8"))
        calls = [node for node in ast.walk(tree) if isinstance(node, ast.Call)
                 and isinstance(node.func, ast.Name) and node.func.id == constructor]
        return [{keyword.arg: ast.literal_eval(keyword.value) for keyword in call.keywords} for call in calls]

    def test_gke_sampling_preserves_sizing(self):
        root = Path(__file__).resolve().parents[2]
        cases = {
            "qa": {}, "nongroovy": {}, "ui": {}, "race_condition_qa": {},
            "external_pg_17_qa": {"machine_type": "e2-standard-8"},
            "scanner_v4_install": {"machine_type": "e2-standard-8"},
            "scale": {"machine_type": "e2-standard-8"},
            "operator": {"num_nodes": 4},
        }
        for family, sizing in cases.items():
            with self.subTest(family=family):
                suffix = "tests.py" if family in ("scale", "scanner_v4_install") else "e2e_tests.py"
                self.assertEqual([{**sizing, "sample_resources": True}], self.options(
                    root / "scripts/ci/jobs" / f"gke_{family}_{suffix}", "GKECluster"))
        self.assertEqual([{"machine_type": "e2-standard-8", "num_nodes": 2, "sample_resources": True}],
                         self.options(root / ".openshift-ci/compatibility_test.py", "GKECluster"))

    def test_only_ocp_automation_jobs_opt_in(self):
        jobs = Path(__file__).resolve().parents[2] / "scripts/ci/jobs"
        for path in jobs.glob("*_tests.py"):
            for options in self.options(path, "AutomationFlavorsCluster"):
                with self.subTest(job=path.name):
                    self.assertEqual({"sample_resources": True} if path.name.startswith("ocp_") else {}, options)


class TestOCPResourceSampling(unittest.TestCase):
    @patch("clusters.signal.signal")
    @patch("clusters.signal.getsignal")
    @patch("clusters.subprocess.run")
    @patch("clusters.ResourceSampler")
    def test_start_stop_and_restore_handler(self, sampler, run, getsignal, setsignal):
        previous_handler = Mock()
        getsignal.return_value = previous_handler
        order = Mock()
        order.attach_mock(run, "access")
        order.attach_mock(sampler.return_value.start, "start")
        with patch.dict(os.environ, {"KUBECONFIG": "/tmp/kubeconfig", "ARTIFACT_DIR": "/tmp/artifacts"}):
            cluster = AutomationFlavorsCluster(sample_resources=True).provision()
        self.assertEqual(["access", "start"], [call[0] for call in order.mock_calls])
        cluster.teardown()
        sampler.return_value.stop.assert_called_once()
        self.assertIs(previous_handler, setsignal.call_args.args[1])
        run.assert_called_once()  # Actual destruction remains owned by automation-flavors.

    @patch("clusters.signal.signal")
    @patch("clusters.subprocess.run")
    @patch("clusters.ResourceSampler")
    def test_other_flavours_do_not_sample(self, sampler, _run, setsignal):
        with patch.dict(os.environ, {"KUBECONFIG": "/tmp/kubeconfig"}):
            AutomationFlavorsCluster().provision().teardown()
        sampler.assert_not_called()
        setsignal.assert_not_called()

    @patch("clusters.signal.signal")
    @patch("clusters.signal.getsignal")
    @patch("clusters.subprocess.run")
    @patch("clusters.ResourceSampler")
    def test_cancel_stops_sampler_before_original_handler(self, sampler, _run, getsignal, _setsignal):
        previous_handler = Mock()
        getsignal.return_value = previous_handler
        order = Mock()
        order.attach_mock(sampler.return_value.stop, "stop")
        order.attach_mock(previous_handler, "previous_handler")
        with patch.dict(os.environ, {"KUBECONFIG": "/tmp/kubeconfig", "ARTIFACT_DIR": "/tmp/artifacts"}):
            cluster = AutomationFlavorsCluster(sample_resources=True).provision()
        cluster.sigint_handler(2, None)
        self.assertEqual(["stop", "previous_handler"], [call[0] for call in order.mock_calls])

    @patch("clusters.subprocess.run", side_effect=OSError("cannot access cluster"))
    @patch("clusters.ResourceSampler")
    def test_access_failure_does_not_start_sampler(self, sampler, _run):
        with patch.dict(os.environ, {"KUBECONFIG": "/tmp/kubeconfig"}):
            with self.assertRaisesRegex(OSError, "cannot access cluster"):
                AutomationFlavorsCluster(sample_resources=True).provision()
        sampler.assert_not_called()

    @patch("clusters.signal.signal")
    @patch("clusters.subprocess.run")
    @patch("clusters.ResourceSampler", side_effect=OSError("cannot write artifacts"))
    def test_sampling_failure_does_not_fail_provision(self, _sampler, _run, setsignal):
        with patch.dict(os.environ, {"KUBECONFIG": "/tmp/kubeconfig", "ARTIFACT_DIR": "/tmp/artifacts"}):
            AutomationFlavorsCluster(sample_resources=True).provision().teardown()
        setsignal.assert_not_called()
