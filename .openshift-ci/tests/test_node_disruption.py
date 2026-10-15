"""Cluster-free tests for the shared node monitor and framework integration."""

import json
import os
from pathlib import Path
import queue
import signal
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import Mock, patch
import xml.etree.ElementTree as ET

import node_disruption as monitor
from ci_tests import NonGroovyE2e


def node(ready='True'):
    return {'metadata': {'name': 'worker-1'},
            'status': {'conditions': [{'type': 'Ready', 'status': ready}]}}


def snapshot(ready='True'):
    return {'kind': 'NodeList', 'metadata': {'resourceVersion': 'list-42'}, 'items': [node(ready)]}


class FakeWatch:
    def __init__(self, messages):
        self.messages = queue.Queue()
        for message in messages:
            self.messages.put(message)
        self.proxy = Mock()
        self.proxy.poll.return_value = None
        self.start = Mock()
        self.close = Mock()
        self.diagnostic = Mock(return_value='')


class BaselineAndEventTest(unittest.TestCase):
    def test_only_dedicated_true_enables_monitoring(self):
        for value in ('', 'false', 'true'):
            with self.subTest(value=value), patch.dict(os.environ, {
                    'GKE_SPOT': 'true', 'GKE_NODE_DISRUPTION_WATCH': value}):
                self.assertEqual(monitor.enabled(), value == 'true')

    def test_baseline_uses_collection_version(self):
        data = snapshot()
        data['items'][0]['metadata']['resourceVersion'] = 'node-7'
        self.assertEqual(monitor.baseline_resource_version(data), 'list-42')

    def test_invalid_baselines(self):
        cases = {
            'no nodes': {'items': []},
            'resourceVersion': {'metadata': {}, 'items': [node()]},
            'not Ready': snapshot('False'),
            'not an object': [],
        }
        for message, data in cases.items():
            with self.subTest(message=message), self.assertRaisesRegex(RuntimeError, message):
                monitor.baseline_resource_version(data)

    def test_disruptions_and_ignored_events(self):
        cases = {
            'deleted': ('DELETED', 'True', True),
            'false': ('MODIFIED', 'False', True),
            'unknown': ('MODIFIED', 'Unknown', True),
            'missing condition': ('MODIFIED', None, True),
            'healthy': ('MODIFIED', 'True', False),
            'added unhealthy': ('ADDED', 'False', False),
            'bookmark': ('BOOKMARK', 'True', False),
        }
        for name, (kind, ready, expected) in cases.items():
            with self.subTest(name=name):
                result = monitor.disruption({'type': kind, 'object': node(ready)})
                self.assertEqual(bool(result), expected)
        with self.assertRaisesRegex(RuntimeError, 'expired'):
            monitor.disruption({'type': 'ERROR', 'object': {'message': 'expired'}})
        with self.assertRaisesRegex(RuntimeError, 'name'):
            monitor.disruption({'type': 'DELETED', 'object': {}})


class NodeMonitorTest(unittest.TestCase):
    def run_fake(self, messages, code=0, cancel=False, timeout=None):
        watch = FakeWatch(messages)
        test = Mock(pid=123, returncode=code)
        test.poll.return_value = code
        with tempfile.TemporaryDirectory() as directory, \
                patch.dict(os.environ, ARTIFACT_DIR=directory), \
                patch.object(monitor, 'NodeWatch', return_value=watch), \
                patch.object(monitor.subprocess, 'Popen', return_value=test) as popen, \
                patch.object(monitor, 'terminate') as terminate:
            if cancel:
                watch.start.side_effect = lambda c: c.handle(signal.SIGTERM, None)
            result = monitor.run(['product'], timeout=timeout, run_url='https://ci.example/run')
            report = ET.parse(Path(directory) / 'node-monitor.xml').getroot()
        watch.close.assert_called_once()
        return result, report, popen, terminate

    def test_product_exit_codes_and_passing_junit(self):
        for code in (0, 42, -signal.SIGTERM):
            with self.subTest(code=code):
                result, report, popen, _ = self.run_fake([('ready', None)], code)
                self.assertEqual(result.exit_code, code if code >= 0 else 128 - code)
                self.assertFalse(result.monitor_failed)
                self.assertEqual(report.get('failures'), '0')
                self.assertIn('https://ci.example/run', report.findtext('.//system-out'))
                popen.assert_called_once_with(['product'], env=None, start_new_session=True)

    def test_startup_errors_do_not_launch_product(self):
        for error in ('missing metadata.resourceVersion', 'HTTP 403 Forbidden', 'watch timeout'):
            with self.subTest(error=error):
                result, report, popen, _ = self.run_fake([('error', error)])
                self.assertTrue(result.monitor_failed)
                self.assertIn(error, report.findtext('.//failure'))
                popen.assert_not_called()

    def test_buffered_disruption_prevents_product_start(self):
        event = {'type': 'DELETED', 'node': 'worker-1', 'ready': 'True'}
        result, report, popen, _ = self.run_fake([('ready', None), ('event', event)])
        self.assertTrue(result.monitor_failed)
        self.assertIn('worker-1', report.findtext('.//failure'))
        popen.assert_not_called()

    def test_cancellation_is_skipped(self):
        result, report, popen, _ = self.run_fake([], cancel=True)
        self.assertEqual(result.exit_code, 143)
        self.assertEqual(report.get('skipped'), '1')
        self.assertIsNotNone(report.find('.//skipped'))
        popen.assert_not_called()

    def test_errors_and_disruptions_stop_running_product(self):
        for kind, value in {'error': 'watch ended', 'event': {
                'type': 'MODIFIED', 'node': 'worker-1', 'ready': 'False'}}.items():
            with self.subTest(kind=kind):
                watch = FakeWatch([(kind, value)])
                result = {}
                test = Mock()
                monitor.monitor_test(test, watch, result, monitor.Cancellation(), None)
                self.assertEqual(result[kind], value)
                test.poll.assert_not_called()

    def test_timeout_and_running_cancellation(self):
        watch = FakeWatch([])
        test = Mock()
        test.poll.return_value = None
        result = {}
        monitor.monitor_test(test, watch, result, monitor.Cancellation(), 0)
        self.assertTrue(result['timed_out'])
        cancellation = monitor.Cancellation()
        cancellation.handle(signal.SIGINT, None)
        monitor.monitor_test(test, watch, {}, cancellation, None)

    def test_watch_establishment_deadline(self):
        watch = FakeWatch([])
        result = {}
        with patch.object(monitor.time, 'monotonic', side_effect=[0, 100]):
            self.assertFalse(monitor.await_ready(watch, result, monitor.Cancellation()))
        self.assertIn('timed out', result['error'])

    def test_http_failure_diagnostic(self):
        connection = Mock()
        connection.getresponse.return_value.status = 403
        connection.getresponse.return_value.reason = 'Forbidden'
        with self.assertRaisesRegex(RuntimeError, 'HTTP 403 Forbidden'):
            monitor.response_or_error(connection, '/api/v1/nodes')

    def test_raw_list_watch_reader(self):
        baseline = Mock(status=200)
        baseline.read.return_value = json.dumps(snapshot()).encode()
        response = Mock(status=200)
        response.readline.side_effect = [
            json.dumps({'type': 'ADDED', 'object': node('False')}).encode() + b'\n',
            json.dumps({'type': 'DELETED', 'object': node()}).encode() + b'\n',
        ]
        connections = [Mock(), Mock()]
        for connection, reply in zip(connections, (baseline, response)):
            connection.getresponse.return_value = reply
        watch = monitor.NodeWatch()
        with patch.object(monitor, 'UnixConnection', side_effect=connections):
            watch.read('/unused')
        self.assertEqual(watch.messages.get_nowait(), ('ready', None))
        self.assertEqual(watch.messages.get_nowait()[0], 'event')
        query = connections[1].request.call_args.args[1]
        self.assertIn('resourceVersion=list-42', query)
        self.assertIn('watch=true', query)

    def test_reader_eof_malformed_json_and_api_error(self):
        for line in (b'', b'broken\n', b'{"type":"ERROR","object":{"message":"expired"}}\n'):
            with self.subTest(line=line):
                baseline = Mock(status=200)
                baseline.read.return_value = json.dumps(snapshot()).encode()
                response = Mock(status=200)
                response.readline.return_value = line
                connection = Mock()
                connection.getresponse.side_effect = [baseline, response]
                watch = monitor.NodeWatch()
                with patch.object(monitor, 'UnixConnection', return_value=connection):
                    watch.read('/unused')
                self.assertEqual(watch.messages.get_nowait()[0], 'ready')
                self.assertEqual(watch.messages.get_nowait()[0], 'error')

    def test_running_monitor_failure_terminates_product_and_restores_handlers(self):
        watch = FakeWatch([('ready', None)])
        test = Mock(pid=123, returncode=-15)
        previous = signal.getsignal(signal.SIGTERM)

        def launch(*_args, **_kwargs):
            watch.messages.put(('error', 'stream failed after launch'))
            return test

        with tempfile.TemporaryDirectory() as directory, \
                patch.dict(os.environ, ARTIFACT_DIR=directory), \
                patch.object(monitor, 'NodeWatch', return_value=watch), \
                patch.object(monitor.subprocess, 'Popen', side_effect=launch), \
                patch.object(monitor, 'terminate') as terminate:
            result = monitor.run(['product'])
            self.assertTrue(result.monitor_failed)
            terminate.assert_called_once_with(test)
            self.assertEqual(signal.getsignal(signal.SIGTERM), previous)
            self.assertIn('after launch', ET.parse(Path(directory) / 'node-monitor.xml').findtext('.//failure'))

    def test_running_cancellation_terminates_product_and_skips_junit(self):
        watch = FakeWatch([('ready', None)])
        test = Mock(pid=123, returncode=-15)
        test.poll.return_value = None
        original_next = monitor.next_message

        def next_then_cancel(current):
            if current.messages.empty():
                signal.getsignal(signal.SIGINT)(signal.SIGINT, None)
                return None, None
            return original_next(current)

        with tempfile.TemporaryDirectory() as directory, \
                patch.dict(os.environ, ARTIFACT_DIR=directory), \
                patch.object(monitor, 'NodeWatch', return_value=watch), \
                patch.object(monitor, 'next_message', side_effect=next_then_cancel), \
                patch.object(monitor.subprocess, 'Popen', return_value=test), \
                patch.object(monitor, 'terminate') as terminate:
            result = monitor.run(['product'])
            self.assertEqual(result.exit_code, 130)
            self.assertFalse(result.monitor_failed)
            terminate.assert_called_once_with(test)
            report = ET.parse(Path(directory) / 'node-monitor.xml').getroot()
            self.assertEqual(report.get('skipped'), '1')
            self.assertEqual(report.get('failures'), '0')

    def test_cleanup_survives_report_write_failure(self):
        watch = FakeWatch([('ready', None)])
        test = Mock(pid=123, returncode=0)
        test.poll.return_value = 0
        previous = signal.getsignal(signal.SIGTERM)
        with patch.object(monitor, 'NodeWatch', return_value=watch), \
                patch.object(monitor.subprocess, 'Popen', return_value=test), \
                patch.object(monitor, 'write_report', side_effect=OSError('disk full')), \
                patch.object(monitor, 'terminate') as terminate, \
                self.assertRaisesRegex(OSError, 'disk full'):
            monitor.run(['product'])
        terminate.assert_called_once_with(test)
        watch.close.assert_called_once()
        self.assertEqual(signal.getsignal(signal.SIGTERM), previous)

    def test_reader_baseline_errors_are_phase_specific(self):
        for error in (TimeoutError('request timed out'), OSError('connection refused')):
            with self.subTest(error=error):
                watch = monitor.NodeWatch()
                with patch.object(monitor, 'UnixConnection', side_effect=error):
                    watch.read('/unused')
                kind, detail = watch.messages.get_nowait()
                self.assertEqual(kind, 'error')
                self.assertIn('node baseline:', detail)

    def test_termination_escalates_after_five_seconds(self):
        process = Mock(pid=123)
        process.wait.side_effect = [subprocess.TimeoutExpired('product', 5), -9]
        with patch.object(monitor.os, 'killpg') as killpg:
            monitor.terminate(process)
        self.assertEqual(killpg.call_args_list[0].args, (123, signal.SIGTERM))
        self.assertEqual(killpg.call_args_list[-1].args, (123, signal.SIGKILL))
        process.wait.assert_any_call(timeout=5)

    def test_real_process_group_with_resistant_descendant(self):
        # The leader exits on TERM, while its child ignores it. KILL must still
        # target the process group after the leader has been reaped.
        script = '''import os, signal, time
if os.fork() == 0:
    signal.signal(signal.SIGTERM, signal.SIG_IGN)
    print('ready', flush=True)
    time.sleep(60)
else:
    time.sleep(60)
'''
        # Explicit finally below must also kill descendants if an assertion fails.
        # pylint: disable=consider-using-with
        process = subprocess.Popen([sys.executable, '-c', script], start_new_session=True,
                                   stdout=subprocess.PIPE, text=True)
        try:
            self.assertEqual(process.stdout.readline().strip(), 'ready')
            start = time.monotonic()
            monitor.terminate(process, grace=0.2)
            self.assertIsNotNone(process.returncode)
            self.assertGreaterEqual(time.monotonic() - start, 0.19)
        finally:
            monitor.kill_group(process.pid, signal.SIGKILL)
            process.wait()
            process.stdout.close()


class NonGroovyIntegrationTest(unittest.TestCase):
    def test_prow_run_links(self):
        cases = {
            'explicit': ({'JOB_URL': 'https://ci.example/job'}, 'https://ci.example/job'),
            'periodic': ({'JOB_NAME': 'gke', 'BUILD_ID': '123'},
                         'https://prow.ci.openshift.org/view/gs/origin-ci-test/logs/gke/123'),
            'pull': ({'JOB_NAME': 'gke', 'BUILD_ID': '123', 'PULL_NUMBER': '42'},
                     'https://prow.ci.openshift.org/view/gs/origin-ci-test/pr-logs/pull/stackrox_stackrox/42/gke/123'),
            'missing': ({}, ''),
        }
        for name, (env, expected) in cases.items():
            with self.subTest(name=name), patch.dict(os.environ, env, clear=True):
                self.assertEqual(NonGroovyE2e().run_url(), expected)

    def test_disabled_path_is_unchanged(self):
        test = NonGroovyE2e()
        with patch.dict(os.environ, GKE_NODE_DISRUPTION_WATCH='false'), \
                patch.object(test, 'run_with_graceful_kill') as run:
            test.run()
        run.assert_called_once_with(['tests/e2e/run.sh', test.TEST_OUTPUT_DIR],
                                    test.TEST_TIMEOUT, output_dir=test.TEST_OUTPUT_DIR)

    def test_enabled_path_preserves_outputs_timeout_and_failure(self):
        for result in (monitor.RunResult(0, False), monitor.RunResult(42, False),
                       monitor.RunResult(1, True), monitor.RunResult(124, False, True)):
            with self.subTest(result=result), \
                    patch.dict(os.environ, GKE_NODE_DISRUPTION_WATCH='true'), \
                    patch.object(monitor, 'run', return_value=result) as run:
                test = NonGroovyE2e()
                if result.timed_out:
                    with self.assertRaises(subprocess.TimeoutExpired):
                        test.run()
                elif result.exit_code:
                    with self.assertRaisesRegex(RuntimeError, str(result.exit_code)):
                        test.run()
                else:
                    test.run()
                self.assertIn(test.TEST_OUTPUT_DIR, test.test_outputs)
                self.assertEqual(run.call_args.kwargs['timeout'], test.TEST_TIMEOUT)
                self.assertEqual(run.call_args.kwargs['env']['ROX_CI_OUTPUT_DIR'], test.TEST_OUTPUT_DIR)
