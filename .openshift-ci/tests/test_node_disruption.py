"""Cluster-free tests for node-name polling and nongroovy integration."""

import json
import os
from pathlib import Path
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


def snapshot(*names, ready='True'):
    return {'items': [{'metadata': {'name': name},
                       'status': {'conditions': [{'type': 'Ready', 'status': ready}]}}
                      for name in names]}


def reply(*names, ready='True', code=0, stderr=''):
    return subprocess.CompletedProcess('kubectl', code, json.dumps(snapshot(*names, ready=ready)), stderr)


class Clock:
    def __init__(self):
        self.now = 0

    def monotonic(self):
        return self.now

    def sleep(self, seconds):
        self.now += seconds


class CollectionTest(unittest.TestCase):
    def test_only_dedicated_true_enables_monitoring(self):
        cases = {'unset': '', 'disabled': 'false', 'enabled': 'true'}
        for name, value in cases.items():
            with self.subTest(name=name), patch.dict(os.environ, {
                    'GKE_SPOT': 'true', 'GKE_NODE_DISRUPTION_WATCH': value}):
                self.assertEqual(monitor.enabled(), value == 'true')

    def test_expected_count(self):
        cases = {'unset': ({}, None), 'one': ({'NUM_NODES': '1'}, 1),
                 'many': ({'NUM_NODES': '3'}, 3)}
        for name, (env, expected) in cases.items():
            with self.subTest(name=name):
                self.assertEqual(monitor.expected_node_count(env), expected)
        invalid = {'empty': '', 'zero': '0', 'negative': '-1', 'text': 'three', 'fraction': '1.5'}
        for name, value in invalid.items():
            with self.subTest(name=name), self.assertRaisesRegex(RuntimeError, 'positive integer'):
                monitor.expected_node_count({'NUM_NODES': value})

    def test_invalid_snapshots(self):
        cases = {'not an object': [], 'missing items': {}, 'invalid items': {'items': None},
                 'invalid node': {'items': [None]}, 'missing name': {'items': [{}]},
                 'invalid name': snapshot(''), 'multiline name': snapshot('node\nother')}
        for name, data in cases.items():
            with self.subTest(name=name), self.assertRaises(ValueError):
                monitor.node_names(data)

    def test_names_only_verdict(self):
        cases = {
            'stable duplicates': ([reply('a', 'b'), reply('b', 'a')], 2, False),
            'replacement': ([reply('a'), reply('b')], 1, True),
            'too few': ([reply('a')], 2, True),
            'too many': ([reply('a', 'b')], 1, True),
            'not ready is ignored': ([reply('a', ready='False')], 1, False),
            'disappearance is ignored': ([reply('a', 'b'), reply('a')], 2, False),
            'empty later sample is ignored': ([reply('a'), reply()], 1, False),
            'unobserved transient is ignored': ([reply('a'), reply('a')], 1, False),
        }
        for name, (replies, expected, mismatch) in cases.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory, \
                    patch.dict(os.environ, ARTIFACT_DIR=directory), \
                    patch.object(monitor.subprocess, 'run', side_effect=replies) as run:
                poller = monitor.NodePoll({'KUBECONFIG': 'test-config'})
                for _ in range(len(replies)):
                    poller.sample()
                verdict = poller.verdict(expected)
                self.assertEqual(verdict['mismatch'], mismatch)
                self.assertIn(f'Expected {expected}', verdict['summary'])
                self.assertEqual(poller.errors, [])
                records = [json.loads(line) for line in poller.snapshots.read_text().splitlines()]
                self.assertEqual(len(records), len(replies))
                self.assertIn('items', json.loads(records[0]['stdout']))
                self.assertIn('timestamp', records[0])
                self.assertEqual(run.call_args.kwargs['env'], {'KUBECONFIG': 'test-config'})
                self.assertEqual(run.call_args.kwargs['timeout'], 10)
                self.assertEqual(run.call_args.args[0],
                                 ['kubectl', '--request-timeout=10s', 'get', 'nodes', '-o', 'json'])

    def test_poll_errors_are_retained(self):
        cases = {
            'exit': (reply(code=1, stderr='forbidden'), 'exited 1'),
            'malformed': (subprocess.CompletedProcess('kubectl', 0, 'broken', ''), 'Expecting value'),
            'timeout': (subprocess.TimeoutExpired('kubectl', 10, b'partial', b'timeout'), 'timed out'),
            'missing executable': (FileNotFoundError('kubectl not found'), 'not found'),
        }
        for name, (response, error) in cases.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory, \
                    patch.dict(os.environ, ARTIFACT_DIR=directory), \
                    patch.object(monitor.subprocess, 'run', side_effect=[response]):
                poller = monitor.NodePoll({})
                self.assertIsNone(poller.sample())
                self.assertIn(error, poller.errors[0])
                record = json.loads(poller.snapshots.read_text())
                self.assertIn(error, record['error'])
                self.assertEqual(poller.names.read_text(), '')
                if name == 'timeout':
                    self.assertEqual(record['stdout'], 'partial')
                    self.assertEqual(record['stderr'], 'timeout')
                if name == 'exit':
                    self.assertEqual(record['stderr'], 'forbidden')

    def test_new_run_clears_old_observations(self):
        with tempfile.TemporaryDirectory() as directory, patch.dict(os.environ, ARTIFACT_DIR=directory):
            poller = monitor.NodePoll({})
            poller.names.write_text('old-node\n')
            poller.snapshots.write_text('old snapshot')
            fresh = monitor.NodePoll({})
            self.assertEqual(fresh.names.read_text(), '')
            self.assertEqual(fresh.snapshots.read_text(), '')


class NodeMonitorTest(unittest.TestCase):
    def run_fake(self, replies, **options):
        clock = Clock()
        test = Mock(pid=123, returncode=options.get('code', 0))

        def product_poll():
            cancel_at = options.get('cancel_at')
            if cancel_at is not None and clock.now >= cancel_at:
                signal.getsignal(signal.SIGTERM)(signal.SIGTERM, None)
            return options.get('code', 0) if clock.now >= options.get('duration', 0) else None

        test.poll.side_effect = product_poll
        with tempfile.TemporaryDirectory() as directory, \
                patch.dict(os.environ, ARTIFACT_DIR=directory), \
                patch.object(monitor.subprocess, 'run', side_effect=replies) as collect, \
                patch.object(monitor.subprocess, 'Popen', return_value=test) as launch, \
                patch.object(monitor, 'terminate') as terminate, \
                patch.object(monitor.time, 'monotonic', side_effect=clock.monotonic), \
                patch.object(monitor.time, 'sleep', side_effect=clock.sleep):
            result = monitor.run(['product'], timeout=options.get('timeout'), env=options.get('env', {}),
                                 run_url='https://ci.example/run')
            report = ET.parse(Path(directory) / 'node-monitor.xml').getroot()
            records = [json.loads(line) for line in (Path(directory) / 'node-snapshots.jsonl').read_text().splitlines()]
            names = (Path(directory) / 'node-names.txt').read_text()
        return result, report, collect, launch, terminate, records, names

    def test_product_exit_codes_and_passing_junit(self):
        cases = {'success': 0, 'failure': 42, 'signal': -signal.SIGTERM}
        for name, code in cases.items():
            with self.subTest(name=name):
                result, report, collect, launch, _, records, names = self.run_fake(
                    [reply('a'), reply('a')], code=code)
                self.assertEqual(result.exit_code, code if code >= 0 else 128 - code)
                self.assertFalse(result.monitor_failed)
                self.assertEqual(report.get('failures'), '0')
                self.assertEqual(report.find('.//testcase').get('name'), 'NoNodeDisruption')
                self.assertIn('https://ci.example/run', report.findtext('.//system-out'))
                launch.assert_called_once_with(['product'], env={}, start_new_session=True)
                self.assertEqual(collect.call_count, 2)
                self.assertEqual(len(records), 2)
                self.assertEqual(names, 'a\na\n')

    def test_five_minute_cadence_and_final_sample(self):
        self.assertEqual(monitor.POLL_INTERVAL, 300)
        result, _, collect, _, _, _, _ = self.run_fake(
            [reply('a'), reply('a'), reply('b'), reply('b')], duration=601)
        self.assertEqual(collect.call_count, 4)  # Initial, 5m, 10m, final.
        self.assertTrue(result.monitor_failed)

    def test_short_test_still_has_final_sample(self):
        result, report, collect, _, _, _, _ = self.run_fake([reply('a'), reply('b')], duration=1)
        self.assertEqual(collect.call_count, 2)
        self.assertTrue(result.monitor_failed)
        self.assertIn('observed 2 unique names: a, b', report.findtext('.//failure'))

    def test_expected_size_configuration_and_fallback(self):
        cases = {'explicit match': ({'NUM_NODES': '2'}, ['a', 'b'], False),
                 'explicit mismatch': ({'NUM_NODES': '2'}, ['a'], True),
                 'fallback': ({}, ['a', 'b'], False)}
        for name, (env, nodes, failed) in cases.items():
            with self.subTest(name=name):
                result, _, _, launch, _, _, _ = self.run_fake([reply(*nodes), reply(*nodes)], env=env)
                self.assertEqual(result.monitor_failed, failed)
                launch.assert_called_once()  # Count mismatches are evaluated at the end.

    def test_startup_errors_do_not_launch_product(self):
        cases = {'failed poll': ([reply(code=1)], {}, 'exited 1'),
                 'empty baseline': ([reply()], {}, 'no nodes'),
                 'invalid configuration': ([], {'NUM_NODES': '0'}, 'positive integer')}
        for name, (replies, env, error) in cases.items():
            with self.subTest(name=name):
                result, report, _, launch, _, _, _ = self.run_fake(replies, env=env)
                self.assertTrue(result.monitor_failed)
                self.assertIn(error, report.findtext('.//failure'))
                launch.assert_not_called()

    def test_failed_polls_do_not_stop_product_and_never_pass(self):
        cases = {'intermediate': ([reply('a'), reply(code=1), reply('a')], 301),
                 'final': ([reply('a'), reply(code=1)], 0)}
        for name, (replies, duration) in cases.items():
            with self.subTest(name=name):
                result, report, collect, launch, _, _, _ = self.run_fake(replies, duration=duration)
                self.assertEqual(collect.call_count, len(replies))
                self.assertTrue(result.monitor_failed)
                self.assertIn('exited 1', report.findtext('.//failure'))
                launch.assert_called_once()

    def test_timeout_and_cancellation(self):
        cases = {'timeout': (1, None, 124, '0', 2),
                 'cancelled': (None, 1, 143, '1', 1)}
        for name, (timeout, cancel_at, code, skipped, polls) in cases.items():
            with self.subTest(name=name):
                previous = signal.getsignal(signal.SIGTERM)
                result, report, collect, _, terminate, _, _ = self.run_fake(
                    [reply('a')] * polls, duration=100, timeout=timeout, cancel_at=cancel_at)
                self.assertEqual(result.exit_code, code)
                self.assertEqual(result.timed_out, timeout is not None)
                self.assertEqual(report.get('skipped'), skipped)
                self.assertEqual(collect.call_count, polls)
                terminate.assert_called_once()
                self.assertEqual(signal.getsignal(signal.SIGTERM), previous)

    def test_cancellation_does_not_hide_recorded_poll_error(self):
        result, report, _, _, _, _, _ = self.run_fake(
            [reply('a'), reply(code=1)], duration=999, cancel_at=301)
        self.assertTrue(result.monitor_failed)
        self.assertEqual(report.get('failures'), '1')
        self.assertEqual(report.get('skipped'), '0')

    def test_cancellation_during_startup(self):
        def cancel(_poller):
            signal.getsignal(signal.SIGINT)(signal.SIGINT, None)
            return ['a']

        with patch.object(monitor.NodePoll, 'sample', autospec=True, side_effect=cancel):
            result, report, _, launch, _, _, _ = self.run_fake([])
        self.assertEqual(result.exit_code, 130)
        self.assertEqual(report.get('skipped'), '1')
        launch.assert_not_called()

    def test_cleanup_survives_report_write_failure(self):
        previous = signal.getsignal(signal.SIGTERM)
        with tempfile.TemporaryDirectory() as directory, \
                patch.dict(os.environ, ARTIFACT_DIR=directory), \
                patch.object(monitor.subprocess, 'run', return_value=reply('a')), \
                patch.object(monitor.subprocess, 'Popen', return_value=Mock(returncode=0)) as launch, \
                patch.object(monitor, 'write_report', side_effect=OSError('disk full')), \
                patch.object(monitor, 'terminate') as terminate, \
                self.assertRaisesRegex(OSError, 'disk full'):
            monitor.run(['product'], env={})
        terminate.assert_called_once_with(launch.return_value)
        self.assertEqual(signal.getsignal(signal.SIGTERM), previous)


class TerminationTest(unittest.TestCase):
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
