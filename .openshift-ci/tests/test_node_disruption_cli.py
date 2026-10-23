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

    def test_monitor_findings_do_not_change_exit_code_or_publish_outputs(self):
        cases = {'passed with disruption': (True, 0), 'failed with disruption': (True, 42),
                 'product failure': (False, 42), 'cancelled': (False, 143)}
        for name, (failed, code) in cases.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory:
                output = Path(directory) / 'output'
                with patch.dict(os.environ, GITHUB_OUTPUT=str(output)), \
                        patch('sys.argv', ['node_disruption.py', 'run', '--', 'product']), \
                        patch.object(monitor, 'run', return_value=monitor.RunResult(code, failed)) as run:
                    self.assertEqual(monitor.main(), code)
                self.assertFalse(output.exists())
                run.assert_called_once_with(['product'])

    def test_missing_command_is_rejected(self):
        with patch('sys.argv', ['node_disruption.py', 'run', '--']), self.assertRaises(SystemExit):
            monitor.main()


if __name__ == '__main__':
    unittest.main()
