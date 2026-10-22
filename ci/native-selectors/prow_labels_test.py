"""Check the bounded label-to-command helper without posting comments."""

import json
from pathlib import Path
import unittest

from prow.label_requests import requested_commands


class ProwLabelsTest(unittest.TestCase):
    def test_label_requests(self):
        commands = json.loads(Path('prow/selectors.json').read_text())['commands']
        cases = {
            'no request': ([], {}, []),
            'UI label': (['area/ui'], {}, ['/test gke-ui-e2e-tests']),
            'run all preserves current automatic inventory': (
                ['ci-run-all-tests'], {}, ['/test gke-qa-e2e-tests', '/test gke-ui-e2e-tests']),
            'overlapping labels deduplicate': (
                ['ci-run-all-tests', 'area/ui'], {}, ['/test gke-qa-e2e-tests', '/test gke-ui-e2e-tests']),
            'pending UI stays running': (['area/ui'], {'ci/prow/gke-ui-e2e-tests': 'pending'}, []),
            'successful UI stays complete': (['area/ui'], {'ci/prow/gke-ui-e2e-tests': 'success'}, []),
            'failed UI may rerun': (['area/ui'], {'ci/prow/gke-ui-e2e-tests': 'failure'}, ['/test gke-ui-e2e-tests']),
            'QA test selection does not schedule jobs': (['ci-all-qa-tests'], {}, []),
        }
        for name, (labels, statuses, expected) in cases.items():
            with self.subTest(name=name):
                self.assertEqual(requested_commands(labels, commands, statuses), expected)
        self.assertEqual(set(requested_commands(['ci-run-all-tests'], commands, {}, True)), set(commands.values()))
