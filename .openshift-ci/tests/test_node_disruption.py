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


def kubectl_responses(responses):
    """Emulate kubectl writing directly to the supplied artifact streams."""
    responses = iter(responses)

    def collect(*_args, **kwargs):
        response = next(responses)
        if isinstance(response, Exception):
            if isinstance(response, subprocess.TimeoutExpired):
                kwargs['stdout'].write((response.stdout or b'').decode())
                kwargs['stderr'].write((response.stderr or b'').decode())
            raise response
        kwargs['stdout'].write(response.stdout)
        kwargs['stderr'].write(response.stderr)
        return response

    return collect


class CollectionTest(unittest.TestCase):
    def test_expected_count(self):
        cases = {'unset': ({}, None), 'empty': ({'NUM_NODES': ''}, None), 'one': ({'NUM_NODES': '1'}, 1),
                 'many': ({'NUM_NODES': '3'}, 3)}
        for name, (env, expected) in cases.items():
            with self.subTest(name=name):
                self.assertEqual(monitor.expected_node_count(env), expected)
        invalid = {'zero': '0', 'negative': '-1', 'text': 'three', 'fraction': '1.5'}
        for name, value in invalid.items():
            with self.subTest(name=name), self.assertRaisesRegex(RuntimeError, 'positive integer'):
                monitor.expected_node_count({'NUM_NODES': value})

    def test_junit_contains_only_outcome_and_failure_explanation(self):
        cases = {'passed': ([], False, None), 'failed': (['kubectl failed'], False, 'failure'),
                 'cancelled': ([], True, 'skipped')}
        for name, (errors, cancelled, outcome) in cases.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory, \
                    patch('builtins.print') as log:
                directory = Path(directory)
                (directory / 'node-names.txt').write_text('a\n')
                cancellation = monitor.Cancellation()
                if cancelled:
                    cancellation.handle(signal.SIGTERM, None)
                monitor.write_report(directory, 1, errors, cancellation)
                case = ET.parse(directory / 'node-monitor.xml').find('.//testcase')
                self.assertEqual([child.tag for child in case], [outcome] if outcome else [])
                log.assert_called_once()
                self.assertIn('Expected 1 nodes; observed 1 unique names: a', log.call_args.args[0])
                if outcome == 'failure':
                    self.assertIn('kubectl failed', case.findtext('failure'))
                    self.assertIn('kubectl failed', log.call_args.args[0])
                if outcome == 'skipped':
                    self.assertIsNone(case.find('skipped').text)

    def test_invalid_snapshots(self):
        cases = {'not an object': [], 'missing items': {}, 'invalid items': {'items': None},
                 'invalid node': {'items': [None]}, 'missing name': {'items': [{}]},
                 'invalid name': snapshot(''), 'multiline name': snapshot('node\nother')}
        for name, data in cases.items():
            completed = subprocess.CompletedProcess('kubectl', 0, json.dumps(data), '')
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory, \
                    patch.object(monitor.subprocess, 'run', side_effect=kubectl_responses([completed])), \
                    self.assertRaises((ValueError, KeyError, TypeError)):
                monitor.sample_nodes(Path(directory), {})

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
                    patch.object(monitor.subprocess, 'run', side_effect=kubectl_responses(replies)) as run:
                directory = Path(directory)
                for _ in range(len(replies)):
                    monitor.sample_nodes(directory, {'KUBECONFIG': 'test-config'})
                failed = monitor.write_report(directory, expected, [], monitor.Cancellation())
                self.assertEqual(failed, mismatch)
                report = ET.parse(directory / 'node-monitor.xml').getroot()
                self.assertEqual(report.get('failures'), str(int(mismatch)))
                self.assertEqual(len(list(directory.glob('node-snapshot-*.json'))), len(replies))
                self.assertEqual(run.call_args.kwargs['env'], {'KUBECONFIG': 'test-config'})
                self.assertEqual(run.call_args.kwargs['timeout'], 10)
                self.assertEqual(run.call_args.args[0],
                                 ['kubectl', '--request-timeout=10s', 'get', 'nodes', '-o', 'json'])

    def test_poll_diagnostics_are_retained(self):
        cases = {
            'exit': reply(code=1, stderr='forbidden'),
            'malformed': subprocess.CompletedProcess('kubectl', 0, 'broken', ''),
            'timeout': subprocess.TimeoutExpired('kubectl', 10, b'partial', b'timeout'),
            'missing executable': FileNotFoundError('kubectl not found'),
        }
        for name, response in cases.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory, \
                    patch.object(monitor.subprocess, 'run', side_effect=kubectl_responses([response])), \
                    self.assertRaises((OSError, RuntimeError, ValueError, subprocess.TimeoutExpired)):
                directory = Path(directory)
                try:
                    monitor.sample_nodes(directory, {})
                finally:
                    self.assertEqual(len(list(directory.glob('node-snapshot-*.json'))), 1)
                    if name == 'timeout':
                        self.assertIn('timeout', (directory / 'node-monitor.log').read_text())
                    if name == 'exit':
                        self.assertIn('forbidden', (directory / 'node-monitor.log').read_text())


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
                patch.object(monitor.subprocess, 'run', side_effect=kubectl_responses(replies)) as collect, \
                patch.object(monitor.subprocess, 'Popen', return_value=test) as launch, \
                patch.object(monitor, 'terminate') as terminate, \
                patch.object(monitor.time, 'monotonic', side_effect=clock.monotonic), \
                patch.object(monitor.time, 'sleep', side_effect=clock.sleep):
            result = monitor.run(
                ['product'], timeout=options.get('timeout'), env=options.get('env', {'NUM_NODES': '1'}))
            report = ET.parse(Path(directory) / 'node-monitor.xml').getroot()
            records = list(Path(directory).glob('node-snapshot-*.json'))
            names = (Path(directory) / 'node-names.txt').read_text() if records else ''
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
                self.assertEqual(list(report.find('.//testcase')), [])
                launch.assert_called_once_with(['product'], env={'NUM_NODES': '1'}, start_new_session=True)
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

    def test_expected_size_configuration(self):
        cases = {'explicit match': ({'NUM_NODES': '2'}, ['a', 'b'], False),
                 'explicit mismatch': ({'NUM_NODES': '2'}, ['a'], True)}
        for name, (env, nodes, failed) in cases.items():
            with self.subTest(name=name):
                result, _, _, launch, _, _, _ = self.run_fake([reply(*nodes), reply(*nodes)], env=env)
                self.assertEqual(result.monitor_failed, failed)
                launch.assert_called_once()  # Count mismatches are evaluated at the end.

    def test_startup_errors_do_not_launch_product(self):
        cases = {'failed poll': ([reply(code=1)], {'NUM_NODES': '1'}, 'exited 1'),
                 'empty baseline': ([reply()], {'NUM_NODES': '1'}, 'no nodes'),
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
        def cancel(_directory, _env):
            signal.getsignal(signal.SIGINT)(signal.SIGINT, None)
            return ['a']

        with patch.object(monitor, 'sample_nodes', autospec=True, side_effect=cancel):
            result, report, _, launch, _, _, _ = self.run_fake([])
        self.assertEqual(result.exit_code, 130)
        self.assertEqual(report.get('skipped'), '1')
        launch.assert_not_called()

    def test_missing_node_count_skips_all_monitoring(self):
        cases = {'unset success': ({}, 0), 'unset failure': ({}, 42),
                 'empty': ({'NUM_NODES': ''}, 0)}
        for name, (env, code) in cases.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory, \
                    patch.dict(os.environ, ARTIFACT_DIR=directory), \
                    patch.object(monitor, 'sample_nodes') as poller, \
                    patch.object(monitor, 'write_report') as report, \
                    patch.object(monitor.subprocess, 'Popen', return_value=Mock(returncode=code)) as launch, \
                    patch.object(monitor, 'terminate'):
                result = monitor.run(['product'], env=env)
                self.assertEqual(result.exit_code, code)
                self.assertFalse(result.monitor_failed)
                poller.assert_not_called()
                report.assert_not_called()
                launch.assert_called_once_with(['product'], env=env, start_new_session=True)
                self.assertEqual(list(Path(directory).iterdir()), [])

    def test_cleanup_survives_report_write_failure(self):
        previous = signal.getsignal(signal.SIGTERM)
        with tempfile.TemporaryDirectory() as directory, \
                patch.dict(os.environ, ARTIFACT_DIR=directory), \
                patch.object(monitor.subprocess, 'run', side_effect=kubectl_responses([reply('a'), reply('a')])), \
                patch.object(monitor.subprocess, 'Popen', return_value=Mock(returncode=0)) as launch, \
                patch.object(monitor, 'write_report', side_effect=OSError('disk full')), \
                patch.object(monitor, 'terminate') as terminate, \
                self.assertRaisesRegex(OSError, 'disk full'):
            monitor.run(['product'], env={'NUM_NODES': '1'})
        terminate.assert_called_once_with(launch.return_value)
        self.assertEqual(signal.getsignal(signal.SIGTERM), previous)


class TerminationTest(unittest.TestCase):
    def test_termination_escalates_after_five_seconds(self):
        process = Mock(pid=123)
        process.wait.side_effect = [subprocess.TimeoutExpired('product', 5), -9]
        with patch.object(monitor.os, 'killpg') as killpg, \
                patch.object(monitor.time, 'monotonic', side_effect=[0, 5]):
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
            try:
                os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            process.wait()
            process.stdout.close()


class NonGroovyIntegrationTest(unittest.TestCase):
    def test_shared_runner_is_used_independently_of_spot(self):
        cases = {
            'unset': {},
            'gke non-spot': {'KUBERNETES_PROVIDER': 'gke', 'GKE_SPOT': 'false'},
            'gke spot': {'KUBERNETES_PROVIDER': 'gke', 'GKE_SPOT': 'true'},
            'openshift': {'KUBERNETES_PROVIDER': 'ocp'},
            'legacy disabled flag': {'GKE_NODE_DISRUPTION_WATCH': 'false'},
        }
        for name, env in cases.items():
            with self.subTest(name=name), patch.dict(os.environ, env, clear=True), \
                    patch.object(monitor, 'run', return_value=monitor.RunResult(0, False)) as run:
                test = NonGroovyE2e()
                test.run()
                run.assert_called_once_with(
                    ['tests/e2e/run.sh', test.TEST_OUTPUT_DIR], timeout=test.TEST_TIMEOUT,
                    env=dict(env, ROX_CI_OUTPUT_DIR=test.TEST_OUTPUT_DIR))
                self.assertIn(test.TEST_OUTPUT_DIR, test.test_outputs)

    def test_monitoring_preserves_outputs_timeout_and_failure(self):
        cases = {'passed': monitor.RunResult(0, False), 'product failure': monitor.RunResult(42, False),
                 'monitor failure': monitor.RunResult(1, True), 'timeout': monitor.RunResult(124, False, True)}
        for name, result in cases.items():
            with self.subTest(name=name), \
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
