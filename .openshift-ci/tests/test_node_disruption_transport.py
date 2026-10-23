"""Exercise real subprocess polling with a fake kubectl, never a real cluster."""

import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch
import xml.etree.ElementTree as ET

import node_disruption as monitor


KUBECTL = '''#!{python}
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
'''


def response(*names):
    return {'stdout': json.dumps({'items': [{'metadata': {'name': name}} for name in names]})}


class PollingTransportTest(unittest.TestCase):
    def setUp(self):
        # This resource spans setUp and each test; addCleanup owns its lifetime.
        self.temporary = tempfile.TemporaryDirectory()  # pylint: disable=consider-using-with
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        kubectl = self.root / 'kubectl'
        kubectl.write_text(KUBECTL.format(python=sys.executable))
        kubectl.chmod(0o755)
        self.env = dict(os.environ, ARTIFACT_DIR=str(self.root / 'artifacts'),
                        FAKE_KUBECTL_ROOT=str(self.root),
                        PATH=str(self.root) + os.pathsep + os.environ.get('PATH', ''))
        self.env.pop('NUM_NODES', None)

    def run_monitor(self, responses, script='pass', timeout=None):
        (self.root / 'responses.json').write_text(json.dumps(responses))
        with patch.dict(os.environ, self.env, clear=True):
            result = monitor.run([sys.executable, '-c', script], env=self.env, timeout=timeout)
        report = ET.parse(self.root / 'artifacts/node-monitor.xml').getroot()
        records = [json.loads(line) for line in
                   (self.root / 'artifacts/node-snapshots.jsonl').read_text().splitlines()]
        return result, report, records

    def test_initial_and_final_snapshots_and_product_failure(self):
        result, report, records = self.run_monitor([response('a'), response('a')], 'raise SystemExit(42)')
        self.assertEqual(result.exit_code, 42)
        self.assertFalse(result.monitor_failed)
        self.assertEqual(report.get('failures'), '0')
        self.assertEqual(len(records), 2)
        self.assertEqual((self.root / 'artifacts/node-names.txt').read_text(), 'a\na\n')
        calls = [json.loads(line) for line in (self.root / 'requests.jsonl').read_text().splitlines()]
        self.assertEqual(calls, [['--request-timeout=10s', 'get', 'nodes', '-o', 'json']] * 2)

    def test_replacement_fails_after_product_finishes(self):
        marker = self.root / 'finished'
        script = f'from pathlib import Path; Path({str(marker)!r}).touch()'
        result, report, _ = self.run_monitor([response('old'), response('new')], script)
        self.assertTrue(marker.exists())
        self.assertTrue(result.monitor_failed)
        self.assertIn('new, old', report.findtext('.//failure'))

    def test_intermediate_failure_does_not_stop_product(self):
        marker = self.root / 'finished'
        script = f'import time; from pathlib import Path; time.sleep(0.5); Path({str(marker)!r}).touch()'
        replies = [response('a'), {'returncode': 1, 'stderr': 'forbidden'}, response('a')]
        with patch.object(monitor, 'POLL_INTERVAL', 0.15):
            result, report, records = self.run_monitor(replies, script)
        self.assertTrue(marker.exists())
        self.assertTrue(result.monitor_failed)
        self.assertGreaterEqual(len(records), 3)
        self.assertIn('exited 1', report.findtext('.//failure'))
        self.assertTrue(any('forbidden' in record['stderr'] for record in records))

    def test_initial_errors_prevent_product_launch(self):
        marker = self.root / 'started'
        script = f'from pathlib import Path; Path({str(marker)!r}).touch()'
        result, report, records = self.run_monitor([{'stdout': 'broken JSON'}], script)
        self.assertTrue(result.monitor_failed)
        self.assertFalse(marker.exists())
        self.assertEqual(report.get('failures'), '1')
        self.assertEqual(len(records), 1)
        self.assertEqual(records[0]['stdout'].strip(), 'broken JSON')

    def test_poll_subprocess_timeout_is_bounded(self):
        with patch.object(monitor, 'API_TIMEOUT', 0.5):
            result, report, records = self.run_monitor([{'delay': 60}])
        self.assertTrue(result.monitor_failed)
        self.assertIn('timed out', report.findtext('.//failure'))
        self.assertEqual(len(records), 1)
        self.assertIn('error', records[0])

    def test_product_timeout_keeps_final_sample(self):
        result, _, records = self.run_monitor([response('a')], 'import time; time.sleep(60)', timeout=0.1)
        self.assertTrue(result.timed_out)
        self.assertFalse(result.monitor_failed)
        self.assertEqual(result.exit_code, 124)
        self.assertEqual(len(records), 2)


if __name__ == '__main__':
    unittest.main()
