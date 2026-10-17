"""GitHub-specific adapter tests; core monitoring lives in the Python framework."""

import importlib.util
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('spot_monitor', Path(__file__).with_name('spot_monitor.py'))
adapter = importlib.util.module_from_spec(spec)
spec.loader.exec_module(adapter)


class AdapterTest(unittest.TestCase):
    def test_disabled_mode_bypasses_monitor_and_report(self):
        with patch.dict(os.environ, GKE_NODE_DISRUPTION_WATCH='false', GKE_SPOT='true'), \
                patch('sys.argv', ['spot_monitor.py', 'run', '--', 'product']), \
                patch.object(adapter.subprocess, 'call', return_value=42) as call, \
                patch.object(adapter.node_disruption, 'run') as run:
            self.assertEqual(adapter.main(), 42)
        call.assert_called_once_with(['product'])
        run.assert_not_called()

    def test_output_distinguishes_product_failure_from_monitor_failure(self):
        for failed, code in ((False, 42), (True, 1), (False, 143)):
            with self.subTest(failed=failed, code=code), tempfile.TemporaryDirectory() as directory:
                output = Path(directory) / 'output'
                with patch.dict(os.environ, {
                        'GKE_NODE_DISRUPTION_WATCH': 'true', 'GKE_SPOT': 'false',
                        'GITHUB_OUTPUT': str(output), 'GITHUB_RUN_ID': '123',
                        'GITHUB_RUN_ATTEMPT': '2', 'GITHUB_REPOSITORY': 'stackrox/stackrox',
                        'GITHUB_SERVER_URL': 'https://github.com'}), \
                        patch('sys.argv', ['spot_monitor.py', 'run', '--', 'product']), \
                        patch.object(adapter.node_disruption, 'run',
                                     return_value=adapter.node_disruption.RunResult(code, failed)) as run:
                    self.assertEqual(adapter.main(), code)
                self.assertEqual(output.read_text(), f'failed={str(failed).lower()}\n')
                run.assert_called_once_with(
                    ['product'], run_url='https://github.com/stackrox/stackrox/actions/runs/123/attempts/2')

    def test_reporting_failure_still_sets_recovery_skip_output(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / 'output'
            with patch.dict(os.environ, GKE_NODE_DISRUPTION_WATCH='true', GITHUB_OUTPUT=str(output)), \
                    patch('sys.argv', ['spot_monitor.py', 'run', '--', 'product']), \
                    patch.object(adapter.node_disruption, 'run', side_effect=OSError('disk full')):
                self.assertEqual(adapter.main(), 1)
            self.assertEqual(output.read_text(), 'failed=true\n')

    def test_missing_command_is_rejected(self):
        with patch('sys.argv', ['spot_monitor.py', 'run', '--']), self.assertRaises(SystemExit):
            adapter.main()


if __name__ == '__main__':
    unittest.main()
