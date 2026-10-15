#!/usr/bin/env python3
"""Run E2E tests while watching Kubernetes nodes for disruption."""

import argparse
import json
import os
from pathlib import Path
import select
import signal
import subprocess
import time
import xml.etree.ElementTree as ET

API_TIMEOUT = 10
TERM_GRACE = 5
BASELINE_ERRORS = (
    AttributeError,
    KeyError,
    TypeError,
    ValueError,
    OSError,
    RuntimeError,
    subprocess.SubprocessError,
)


class Cancellation:
    """Record termination signals so the wrapper can finish its JUnit report."""

    def __init__(self):
        self.signal = None

    @property
    def requested(self):
        return self.signal is not None

    def handle(self, signum, _frame):
        self.signal = signum


def ready_status(node):
    for condition in node.get('status', {}).get('conditions', []):
        if condition.get('type') == 'Ready':
            return condition.get('status')
    return 'Unknown'


def disruption(event):
    """Return concise detail for deletion or a MODIFIED node that is not Ready."""
    if not isinstance(event, dict):
        raise RuntimeError('Kubernetes watch event is not an object')

    kind = event.get('type')
    node = event.get('object', {})
    if not isinstance(node, dict):
        raise RuntimeError('Kubernetes watch event has no node object')
    if kind == 'ERROR':
        raise RuntimeError(node.get('message', 'Kubernetes watch returned an error'))
    if kind not in ('DELETED', 'MODIFIED'):
        return None

    name = node.get('metadata', {}).get('name')
    if not name:
        return None
    ready = ready_status(node)
    if kind == 'DELETED' or ready != 'True':
        return {'type': kind, 'node': name, 'ready': ready}
    return None


def terminate(process, grace=TERM_GRACE):
    """Stop a subprocess group with TERM, escalating to KILL after grace."""
    if process is None or process.poll() is not None:
        return

    deadline = time.monotonic() + grace
    try:
        os.killpg(process.pid, signal.SIGTERM)
    except ProcessLookupError:
        return

    try:
        process.wait(timeout=grace)
    except subprocess.TimeoutExpired:
        pass
    else:
        if not wait_for_group_exit(process.pid, deadline):
            kill_group(process.pid, signal.SIGKILL)
            if process.poll() is None:
                process.wait()
        return

    kill_group(process.pid, signal.SIGKILL)
    process.wait()


def wait_for_group_exit(process_group, deadline):
    """Wait out the grace period if descendants outlive the group leader."""
    try:
        os.killpg(process_group, 0)
    except ProcessLookupError:
        return True

    remaining = deadline - time.monotonic()
    if remaining > 0:
        time.sleep(remaining)
    return False


def kill_group(process_group, signum):
    try:
        os.killpg(process_group, signum)
    except ProcessLookupError:
        pass


def run_link():
    run_id = os.getenv('GITHUB_RUN_ID')
    if not run_id:
        return ''
    server = os.getenv('GITHUB_SERVER_URL', 'https://github.com').rstrip('/')
    repository = os.getenv('GITHUB_REPOSITORY', '')
    attempt = os.getenv('GITHUB_RUN_ATTEMPT', '1')
    return f'{server}/{repository}/actions/runs/{run_id}/attempts/{attempt}'


def report_detail(result):
    if result.get('event'):
        event = result['event']
        detail = (f"Node disruption during E2E tests: node={event['node']} "
                  f"event={event['type']} ready={event['ready']}")
        return 'failure', 'Node disruption during E2E tests', detail
    if result.get('error'):
        return 'failure', 'Node monitor error', f"Node monitor error: {result['error']}"
    if result.get('cancelled'):
        detail = f"Node monitoring cancelled (signal={result.get('signal', 'unknown')})"
        return 'skipped', 'Workflow cancelled', detail
    return 'passed', '', 'No node disruption detected during E2E tests.'


def write_report(result):
    """Write the single stable JUnit result for node monitoring."""
    status, message, detail = report_detail(result)
    duration = f"{result.get('duration_seconds', 0):.3f}"
    suite = ET.Element(
        'testsuite', name='ci.gke.node', tests='1',
        failures=str(int(status == 'failure')),
        skipped=str(int(status == 'skipped')), time=duration,
    )
    case = ET.SubElement(
        suite, 'testcase', classname='ci.gke.node',
        name='NoNodeDisruption', time=duration,
    )

    if status == 'failure':
        ET.SubElement(case, 'failure', message=message).text = detail
    elif status == 'skipped':
        ET.SubElement(case, 'skipped', message=message).text = detail

    link = run_link()
    if link:
        detail += f'\nRun: {link}'
    ET.SubElement(case, 'system-out').text = detail

    artifact_dir = Path(os.getenv('ARTIFACT_DIR', 'junit-reports'))
    artifact_dir.mkdir(parents=True, exist_ok=True)
    temporary = artifact_dir / 'node-monitor.xml.tmp'
    temporary.write_text(ET.tostring(suite, encoding='unicode', xml_declaration=True),
                         encoding='utf-8')
    temporary.replace(artifact_dir / 'node-monitor.xml')


def publish_output(failed):
    output = os.getenv('GITHUB_OUTPUT')
    if not output:
        return
    with open(output, 'a', encoding='utf-8') as stream:
        stream.write(f"failed={str(failed).lower()}\n")


def baseline_resource_version():
    baseline = subprocess.run(
        ['kubectl', 'get', 'nodes', '-o', 'json'],
        capture_output=True, text=True, check=True, timeout=API_TIMEOUT,
    )
    snapshot = json.loads(baseline.stdout)
    nodes = snapshot.get('items', [])
    resource_version = snapshot.get('metadata', {}).get('resourceVersion')
    if not nodes:
        raise RuntimeError('kubectl returned no nodes')
    if not resource_version:
        raise RuntimeError('kubectl node snapshot did not include resourceVersion')

    not_ready = [node.get('metadata', {}).get('name', '<unnamed>')
                 for node in nodes if ready_status(node) != 'True']
    if not_ready:
        raise RuntimeError('Nodes not Ready before E2E tests: ' + ', '.join(not_ready))
    return resource_version


def install_signal_handlers(cancellation):
    return {
        signum: signal.signal(signum, cancellation.handle)
        for signum in (signal.SIGTERM, signal.SIGINT)
    }


def restore_signal_handlers(previous):
    for signum, handler in previous.items():
        signal.signal(signum, handler)


def start_test(command, result, cancellation):
    try:
        resource_version = baseline_resource_version()
    except BASELINE_ERRORS as exc:
        if not cancellation.requested:
            result['error'] = f'Could not establish node baseline: {exc}'
        return None, None

    if cancellation.requested:
        return None, None

    watch = subprocess.Popen(
        ['kubectl', 'get', 'nodes', '--watch-only', '--output-watch-events',
         '--resource-version', resource_version, '-o', 'json'],
        stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
        bufsize=0, start_new_session=True,
    )
    if watch.poll() is not None:
        result['error'] = 'kubectl node watch exited before E2E tests started'
        return watch, None
    if cancellation.requested:
        return watch, None

    result['started_at'] = time.time()
    try:
        test = subprocess.Popen(command, start_new_session=True)
    except OSError as exc:
        result['error'] = f'Could not start E2E command: {exc}'
        return watch, None
    return watch, test


def read_watch_event(watch, pending):
    chunk = os.read(watch.stdout.fileno(), 4096)
    if not chunk:
        return pending, None, 'kubectl node watch ended unexpectedly'

    pending += chunk
    while b'\n' in pending:
        line, pending = pending.split(b'\n', 1)
        try:
            node_event = disruption(json.loads(line))
        except (AttributeError, TypeError, json.JSONDecodeError, RuntimeError) as exc:
            return pending, None, f'Invalid or failed node watch event: {exc}'
        if node_event:
            return pending, node_event, None
    return pending, None, None


def report_monitor_error(result, error):
    result['error'] = error
    print(f'::error::{error}', flush=True)


def report_disruption(result, event):
    result['event'] = event
    print('::error::Kubernetes node disruption during E2E tests: '
          f"node={event['node']} type={event['type']} ready={event['ready']}",
          flush=True)


def monitor_test(test, watch, result, cancellation):
    pending = b''
    while True:
        if cancellation.requested:
            terminate(test)
            return

        test_finished = test.poll() is not None
        timeout = 0 if test_finished else 0.1
        readable, _, _ = select.select([watch.stdout], [], [], timeout)
        if readable:
            pending, event, error = read_watch_event(watch, pending)
            if error:
                report_monitor_error(result, error)
                terminate(test)
                return
            if event:
                report_disruption(result, event)
                terminate(test)
                return

        if watch.poll() is not None:
            report_monitor_error(result, 'kubectl node watch exited unexpectedly')
            terminate(test)
            return
        if test_finished and not readable:
            return


def cleanup(test, watch, previous_handlers):
    terminate(test)
    terminate(watch, grace=1)
    if watch is not None and watch.stdout:
        watch.stdout.close()
    restore_signal_handlers(previous_handlers)


def finish(result, test, cancellation):
    result['ended_at'] = time.time()
    result['duration_seconds'] = max(0, result['ended_at'] - result['started_at'])
    if cancellation.requested:
        result['cancelled'] = True
        result['signal'] = cancellation.signal
        result.pop('error', None)
        result.pop('event', None)

    write_report(result)
    publish_output(bool(result.get('event') or result.get('error')))
    if result.get('event') or result.get('error'):
        return 1
    if cancellation.requested:
        return 128 + cancellation.signal
    if test is None:
        return 1
    return test.returncode if test.returncode >= 0 else 128 - test.returncode


def run(command):
    if not command:
        raise ValueError('run requires a command after --')

    result = {'started_at': time.time()}
    cancellation = Cancellation()
    previous_handlers = install_signal_handlers(cancellation)
    watch = test = None
    try:
        watch, test = start_test(command, result, cancellation)
        if test is not None:
            monitor_test(test, watch, result, cancellation)
    except OSError as exc:
        if not cancellation.requested:
            report_monitor_error(result, f'Node monitor failed: {exc}')
        terminate(test)
    finally:
        cleanup(test, watch, previous_handlers)

    return finish(result, test, cancellation)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['run'])
    parser.add_argument('command', nargs=argparse.REMAINDER)
    args = parser.parse_args()
    command = args.command[1:] if args.command[:1] == ['--'] else args.command
    if os.getenv('GKE_SPOT') != 'true':
        return subprocess.call(command)
    return run(command)


if __name__ == '__main__':
    raise SystemExit(main())
