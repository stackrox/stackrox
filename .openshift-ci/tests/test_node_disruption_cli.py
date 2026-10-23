"""CLI and GitHub Actions reporting tests for the shared node monitor."""

import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import node_disruption as monitor


class CLITest(unittest.TestCase):
    def test_cli_delegates_independently_of_spot(self):
        cases = {'unset': {}, 'non-spot': {'GKE_SPOT': 'false'}, 'spot': {'GKE_SPOT': 'true'},
                 'legacy disabled flag': {'GKE_NODE_DISRUPTION_WATCH': 'false'}}
        for name, env in cases.items():
            with self.subTest(name=name), patch.dict(os.environ, env, clear=True), \
                    patch('sys.argv', ['node_disruption.py', 'run', '--', 'product']), \
                    patch.object(monitor, 'run',
                                 return_value=monitor.RunResult(42, False)) as run:
                self.assertEqual(monitor.main(), 42)
                run.assert_called_once_with(['product'])

    def test_output_distinguishes_product_failure_from_monitor_failure(self):
        cases = {'product failure': (False, 42), 'monitor failure': (True, 1), 'cancelled': (False, 143)}
        for name, (failed, code) in cases.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory:
                output = Path(directory) / 'output'
                with patch.dict(os.environ, {
                        'GKE_SPOT': 'false',
                        'GITHUB_OUTPUT': str(output)}), \
                        patch('sys.argv', ['node_disruption.py', 'run', '--', 'product']), \
                        patch.object(monitor, 'run',
                                     return_value=monitor.RunResult(code, failed)) as run:
                    self.assertEqual(monitor.main(), code)
                self.assertEqual(output.read_text(), f'failed={str(failed).lower()}\n')
                run.assert_called_once_with(['product'])

    def test_reporting_failure_still_sets_recovery_skip_output(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / 'output'
            with patch.dict(os.environ, GITHUB_OUTPUT=str(output)), \
                    patch('sys.argv', ['node_disruption.py', 'run', '--', 'product']), \
                    patch.object(monitor, 'run', side_effect=OSError('disk full')):
                self.assertEqual(monitor.main(), 1)
            self.assertEqual(output.read_text(), 'failed=true\n')

    def test_missing_command_is_rejected(self):
        with patch('sys.argv', ['node_disruption.py', 'run', '--']), self.assertRaises(SystemExit):
            monitor.main()


if __name__ == '__main__':
    unittest.main()
