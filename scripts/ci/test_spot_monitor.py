"""Tests for the Kubernetes node-disruption watcher; no cluster is required."""

import importlib.util
import json
import os
from pathlib import Path
import signal
import subprocess
import tempfile
import unittest
from unittest.mock import patch
import xml.etree.ElementTree as ET

spec = importlib.util.spec_from_file_location('spot_monitor', Path(__file__).with_name('spot_monitor.py'))
monitor = importlib.util.module_from_spec(spec)
spec.loader.exec_module(monitor)


def node(name='worker-1', ready='True'):
    return {'metadata': {'name': name}, 'status': {'conditions': [
        {'type': 'Ready', 'status': ready},
    ]}}


def event(kind, obj):
    return (json.dumps({'type': kind, 'object': obj}) + '\n').encode()


def fake_process(pid, stdout=None, returncode=None):
    process = unittest.mock.Mock(pid=pid, stdout=stdout, returncode=returncode)
    process.poll.side_effect = lambda: process.returncode

    def wait(timeout=None):
        if process.returncode is None:
            process.returncode = -signal.SIGTERM
        return process.returncode

    process.wait.side_effect = wait
    return process


def xml_report(result, temp, **env):
    with patch.dict(os.environ, {'ARTIFACT_DIR': temp, **env}):
        monitor.write_report(result)
    return ET.parse(Path(temp) / 'node-monitor.xml').getroot()


class WatchTest(unittest.TestCase):
    def test_only_deletion_and_modified_not_ready_are_disruptions(self):
        self.assertIsNone(monitor.disruption({'type': 'ADDED', 'object': node()}))
        self.assertIsNone(monitor.disruption({'type': 'ADDED', 'object': node('new-worker', 'False')}))
        self.assertIsNone(monitor.disruption({'type': 'MODIFIED', 'object': node()}))
        self.assertEqual(monitor.disruption({
            'type': 'MODIFIED', 'object': node(ready='False'),
        }), {'type': 'MODIFIED', 'node': 'worker-1', 'ready': 'False'})
        self.assertEqual(monitor.disruption({
            'type': 'DELETED', 'object': node(),
        })['type'], 'DELETED')

    def test_watch_api_error_is_monitor_failure(self):
        with self.assertRaisesRegex(RuntimeError, 'watch failed'):
            monitor.disruption({'type': 'ERROR', 'object': {'message': 'watch failed'}})

    def test_baseline_requires_healthy_nodes_and_resource_version(self):
        for response, expected in (
                (subprocess.CompletedProcess([], 0, '{}', ''), 'no nodes'),
                (subprocess.CompletedProcess([], 0, json.dumps({
                    'metadata': {}, 'items': [node()],
                }), ''), 'resourceVersion'),
                (subprocess.CompletedProcess([], 0, json.dumps({
                    'metadata': {'resourceVersion': '42'}, 'items': [node(ready='False')],
                }), ''), 'not Ready')):
            with self.subTest(expected=expected), \
                    patch.object(monitor.subprocess, 'run', return_value=response):
                with self.assertRaisesRegex(RuntimeError, expected):
                    monitor.baseline_resource_version()

    def test_report_pass_failure_error_and_cancellation_xml(self):
        with tempfile.TemporaryDirectory() as temp:
            passed = xml_report({'duration_seconds': 2}, temp,
                                GITHUB_RUN_ID='123', GITHUB_RUN_ATTEMPT='2',
                                GITHUB_REPOSITORY='stackrox/stackrox')
            case = passed.find('testcase')
            self.assertEqual((passed.get('tests'), passed.get('failures'), passed.get('skipped')),
                             ('1', '0', '0'))
            self.assertEqual((case.get('classname'), case.get('name')),
                             ('ci.gke.node', 'NoNodeDisruption'))
            self.assertIn('https://github.com/stackrox/stackrox/actions/runs/123/attempts/2',
                          case.findtext('system-out'))
            self.assertIsNone(case.find('failure'))

            failed = xml_report({'event': {
                'node': 'worker-1', 'type': 'DELETED', 'ready': 'True',
            }}, temp)
            self.assertEqual((failed.get('tests'), failed.get('failures'), failed.get('skipped')),
                             ('1', '1', '0'))
            self.assertIn('node=worker-1', failed.findtext('.//failure'))

            monitor_error = xml_report({'error': 'watch failed'}, temp)
            self.assertEqual(len(monitor_error.findall('testcase')), 1)
            self.assertIn('watch failed', monitor_error.findtext('.//failure'))

            cancelled = xml_report({'cancelled': True, 'signal': signal.SIGTERM}, temp)
            self.assertEqual((cancelled.get('tests'), cancelled.get('failures'), cancelled.get('skipped')),
                             ('1', '0', '1'))
            self.assertIsNotNone(cancelled.find('.//skipped'))
            self.assertFalse((Path(temp) / 'node-monitor.json').exists())

    def test_baseline_error_fails_without_starting_test(self):
        with tempfile.TemporaryDirectory() as temp, patch.dict(os.environ, {
                'GKE_SPOT': 'true', 'ARTIFACT_DIR': temp,
                'GITHUB_OUTPUT': str(Path(temp) / 'output')}), \
                patch.object(monitor, 'baseline_resource_version',
                             side_effect=RuntimeError('baseline unavailable')), \
                patch.object(monitor.subprocess, 'Popen') as popen:
            self.assertEqual(monitor.run(['fake-e2e']), 1)
            popen.assert_not_called()
            root = ET.parse(Path(temp) / 'node-monitor.xml').getroot()
            self.assertIn('baseline unavailable', root.findtext('.//failure'))
            self.assertEqual((Path(temp) / 'output').read_text(), 'failed=true\n')

    def test_watch_error_fails_closed_and_stops_product_test(self):
        with tempfile.TemporaryDirectory() as temp:
            read_fd, write_fd = os.pipe()
            os.write(write_fd, event('ERROR', {'message': 'watch failed'}))
            os.close(write_fd)
            watch = fake_process(123, os.fdopen(read_fd, 'rb', buffering=0))
            test = fake_process(456)
            terminated = []
            with patch.dict(os.environ, {'GKE_SPOT': 'true', 'ARTIFACT_DIR': temp}), \
                    patch.object(monitor, 'baseline_resource_version', return_value='42'), \
                    patch.object(monitor.subprocess, 'Popen', side_effect=[watch, test]), \
                    patch.object(monitor, 'terminate',
                                 side_effect=lambda process, grace=5: (
                                     terminated.append(process), setattr(process, 'returncode', -15))), \
                    patch('builtins.print'):
                self.assertEqual(monitor.run(['fake-e2e']), 1)
            self.assertIn(test, terminated)
            root = ET.parse(Path(temp) / 'node-monitor.xml').getroot()
            self.assertEqual(len(root.findall('testcase')), 1)
            self.assertIn('watch failed', root.findtext('.//failure'))

    def test_deletion_stops_test_and_ignored_addition_does_not(self):
        with tempfile.TemporaryDirectory() as temp:
            read_fd, write_fd = os.pipe()
            os.write(write_fd, event('ADDED', node('new-worker', 'False')) +
                     event('DELETED', node()))
            os.close(write_fd)
            watch = fake_process(123, os.fdopen(read_fd, 'rb', buffering=0))
            test = fake_process(456)
            terminated = []
            with patch.dict(os.environ, {'GKE_SPOT': 'true', 'ARTIFACT_DIR': temp}), \
                    patch.object(monitor, 'baseline_resource_version', return_value='42'), \
                    patch.object(monitor.subprocess, 'Popen', side_effect=[watch, test]), \
                    patch.object(monitor, 'terminate',
                                 side_effect=lambda process, grace=5: (
                                     terminated.append(process), setattr(process, 'returncode', -15))), \
                    patch('builtins.print'):
                self.assertEqual(monitor.run(['fake-e2e']), 1)
            self.assertIn(test, terminated)
            root = ET.parse(Path(temp) / 'node-monitor.xml').getroot()
            self.assertIn('node=worker-1', root.findtext('.//failure'))

    def test_product_exit_code_is_preserved_when_monitoring_is_healthy(self):
        with tempfile.TemporaryDirectory() as temp, patch.dict(os.environ, {
                'GKE_SPOT': 'true', 'ARTIFACT_DIR': temp,
                'GITHUB_OUTPUT': str(Path(temp) / 'output')}), \
                patch.object(monitor, 'baseline_resource_version', return_value='42'), \
                patch.object(monitor.subprocess, 'Popen', side_effect=[
                    fake_process(123, stdout=unittest.mock.Mock()),
                    fake_process(456, returncode=42),
                ]), patch.object(monitor.select, 'select', return_value=([], [], [])), \
                patch.object(monitor, 'terminate'):
            self.assertEqual(monitor.run(['fake-e2e']), 42)
            root = ET.parse(Path(temp) / 'node-monitor.xml').getroot()
            self.assertEqual(root.get('failures'), '0')
            self.assertEqual((Path(temp) / 'output').read_text(), 'failed=false\n')

    def test_cancellation_terminates_test_and_writes_skipped_case(self):
        with tempfile.TemporaryDirectory() as temp:
            watch = fake_process(123, stdout=unittest.mock.Mock())
            test = fake_process(456)
            terminated = []

            def select_then_cancel(*_args):
                signal.getsignal(signal.SIGTERM)(signal.SIGTERM, None)
                return [], [], []

            with patch.dict(os.environ, {'GKE_SPOT': 'true', 'ARTIFACT_DIR': temp}), \
                    patch.object(monitor, 'baseline_resource_version', return_value='42'), \
                    patch.object(monitor.subprocess, 'Popen', side_effect=[watch, test]), \
                    patch.object(monitor.select, 'select', side_effect=select_then_cancel), \
                    patch.object(monitor, 'terminate',
                                 side_effect=lambda process, grace=5: (
                                     terminated.append(process), setattr(process, 'returncode', -15))):
                self.assertEqual(monitor.run(['fake-e2e']), 128 + signal.SIGTERM)
            self.assertIn(test, terminated)
            root = ET.parse(Path(temp) / 'node-monitor.xml').getroot()
            self.assertEqual((root.get('failures'), root.get('skipped')), ('0', '1'))
            self.assertIsNotNone(root.find('.//skipped'))

    def test_terminate_uses_process_group_then_kills_after_timeout(self):
        process = unittest.mock.Mock(pid=123, poll=unittest.mock.Mock(return_value=None),
                                     wait=unittest.mock.Mock(side_effect=[
                                         subprocess.TimeoutExpired('test', 5), -signal.SIGKILL,
                                     ]))
        with patch.object(monitor.os, 'killpg') as killpg:
            monitor.terminate(process)
        self.assertEqual(killpg.call_args_list, [
            unittest.mock.call(123, signal.SIGTERM),
            unittest.mock.call(123, signal.SIGKILL),
        ])
        process.wait.assert_any_call(timeout=monitor.TERM_GRACE)

    def test_terminate_waits_for_remaining_group_after_leader_exits(self):
        process = unittest.mock.Mock(pid=123,
                                     poll=unittest.mock.Mock(side_effect=[None, 0]),
                                     wait=unittest.mock.Mock(return_value=0))
        with patch.object(monitor.os, 'killpg') as killpg, \
                patch.object(monitor.time, 'monotonic', side_effect=[0, 1]), \
                patch.object(monitor.time, 'sleep') as sleep:
            monitor.terminate(process, grace=5)
        self.assertEqual(killpg.call_args_list, [
            unittest.mock.call(123, signal.SIGTERM),
            unittest.mock.call(123, 0),
            unittest.mock.call(123, signal.SIGKILL),
        ])
        sleep.assert_called_once_with(4)

    def test_disabled_mode_bypasses_watcher(self):
        with patch.dict(os.environ, GKE_SPOT='false'), \
                patch('sys.argv', ['spot_monitor.py', 'run', '--', 'test-command']), \
                patch.object(monitor.subprocess, 'call', return_value=42) as call, \
                patch.object(monitor, 'write_report') as write_report:
            self.assertEqual(monitor.main(), 42)
        call.assert_called_once_with(['test-command'])
        write_report.assert_not_called()


if __name__ == '__main__':
    unittest.main()
