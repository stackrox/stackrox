"""Verify the nongroovy job provisions one node."""

import os
from pathlib import Path
import runpy
import tempfile
import unittest
from unittest.mock import Mock, patch

from clusters import GKECluster


class GKENonGroovyEntrypointTest(unittest.TestCase):
    def test_effective_node_count(self):
        cases = {
            "unset": ({}, "1"),
            "empty": ({"NUM_NODES": ""}, "1"),
            "override": ({"NUM_NODES": "5"}, "1"),
        }
        entrypoint = Path(__file__).resolve().parents[2] / "scripts/ci/jobs/gke_nongroovy_e2e_tests.py"
        for name, (env, expected) in cases.items():
            with self.subTest(name=name), patch.dict(os.environ, env, clear=True), patch(
                "runners.ClusterTestRunner"
            ) as runner:
                runpy.run_path(str(entrypoint))
                cluster = runner.call_args.kwargs["cluster"]
                self.assertIsInstance(cluster, GKECluster)
                self.assertEqual(cluster.num_nodes, int(expected))
                self.assertEqual(cluster.machine_type, "e2-standard-8")
                runner.return_value.run.assert_called_once()

                process = Mock()
                process.__enter__ = Mock(return_value=process)
                process.__exit__ = Mock(return_value=False)
                process.wait.return_value = 0

                def launch(*_args, expected=expected, process=process, **_kwargs):
                    self.assertEqual(os.environ["NUM_NODES"], expected)
                    return process

                with tempfile.TemporaryDirectory() as directory, patch.dict(
                    os.environ, SHARED_DIR=directory
                ), patch("clusters.subprocess.Popen", side_effect=launch), patch(
                    "clusters.subprocess.run"
                ), patch("clusters.signal.signal"):
                    cluster.provision()


if __name__ == "__main__":
    unittest.main()
