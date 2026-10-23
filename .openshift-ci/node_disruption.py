"""Opt-in node-name polling shared by nongroovy CI entrypoints."""

from dataclasses import dataclass
import json
import os
from pathlib import Path
import signal
import subprocess
import time
import xml.etree.ElementTree as ET

API_TIMEOUT = 10
TERM_GRACE = 5
POLL_INTERVAL = 5 * 60


def enabled():
    return os.getenv('GKE_NODE_DISRUPTION_WATCH') == 'true'


@dataclass
class RunResult:
    exit_code: int
    monitor_failed: bool
    timed_out: bool = False


class Cancellation:
    def __init__(self):
        self.signal = None

    @property
    def requested(self):
        return self.signal is not None

    def handle(self, signum, _frame):
        self.signal = signum


def terminate(process, grace=TERM_GRACE):
    """Stop a subprocess group with TERM, escalating to KILL after grace."""
    if process is None:
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


def report_detail(result):
    detail = result.get('summary', '')
    if result.get('error'):
        detail += f"\nNode monitor error: {result['error']}"
    if result.get('error') or result.get('mismatch'):
        return 'failure', 'Node name change or monitor error', detail.strip()
    if result.get('cancelled'):
        detail = f"Node monitoring cancelled (signal={result.get('signal', 'unknown')})"
        return 'skipped', 'Workflow cancelled', detail
    return 'passed', '', 'No node name change detected. ' + detail


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

    link = result.get('run_url', '')
    if link:
        detail += f'\nRun: {link}'
    ET.SubElement(case, 'system-out').text = detail

    artifact_dir = Path(os.getenv('ARTIFACT_DIR', 'junit-reports'))
    artifact_dir.mkdir(parents=True, exist_ok=True)
    temporary = artifact_dir / 'node-monitor.xml.tmp'
    temporary.write_text(ET.tostring(suite, encoding='unicode', xml_declaration=True),
                         encoding='utf-8')
    temporary.replace(artifact_dir / 'node-monitor.xml')


def expected_node_count(env):
    value = env.get('NUM_NODES')
    if value is None:
        return None
    try:
        count = int(value)
    except ValueError as exc:
        raise RuntimeError('NUM_NODES must be a positive integer') from exc
    if count <= 0:
        raise RuntimeError('NUM_NODES must be a positive integer')
    return count


def node_names(snapshot):
    """Only names affect the verdict; readiness and per-snapshot counts do not."""
    if not isinstance(snapshot, dict) or not isinstance(snapshot.get('items'), list):
        raise ValueError('node snapshot must contain an items list')
    names = []
    for node in snapshot['items']:
        name = node.get('metadata', {}).get('name') if isinstance(node, dict) else None
        if not isinstance(name, str) or not name or any(char.isspace() for char in name):
            raise ValueError('node snapshot contains an invalid node name')
        names.append(name)
    return names


def output_text(value):
    # TimeoutExpired may hold bytes even when subprocess.run uses text=True.
    return value.decode('utf-8', errors='replace') if isinstance(value, bytes) else value or ''


class NodePoll:
    def __init__(self, env):
        self.env = env
        artifact_dir = Path(os.getenv('ARTIFACT_DIR', 'junit-reports'))
        artifact_dir.mkdir(parents=True, exist_ok=True)
        self.names = artifact_dir / 'node-names.txt'
        self.snapshots = artifact_dir / 'node-snapshots.jsonl'
        self.errors = []
        # Never mix observations from separate runs using the same artifact directory.
        self.names.write_text('', encoding='utf-8')
        self.snapshots.write_text('', encoding='utf-8')

    def sample(self):
        record = {'timestamp': time.time()}
        names = None
        try:
            completed = subprocess.run(
                ['kubectl', f'--request-timeout={API_TIMEOUT}s', 'get', 'nodes', '-o', 'json'],
                env=self.env, capture_output=True, text=True, errors='replace',
                timeout=API_TIMEOUT, check=False,
            )
            record.update(stdout=completed.stdout, stderr=completed.stderr,
                          returncode=completed.returncode)
            if completed.returncode != 0:
                raise RuntimeError(f'kubectl get nodes exited {completed.returncode}')
            names = node_names(json.loads(completed.stdout))
        except subprocess.TimeoutExpired as exc:
            record.update(error=f'kubectl get nodes timed out after {API_TIMEOUT}s',
                          stdout=output_text(exc.stdout), stderr=output_text(exc.stderr))
        except (OSError, ValueError, AttributeError, RuntimeError) as exc:
            record['error'] = str(exc)

        with self.snapshots.open('a', encoding='utf-8') as stream:
            stream.write(json.dumps(record) + '\n')
        if 'error' in record:
            self.errors.append(record['error'])
            return None
        with self.names.open('a', encoding='utf-8') as stream:
            stream.writelines(name + '\n' for name in names)
        return names

    def verdict(self, expected):
        names = sorted(set(self.names.read_text(encoding='utf-8').splitlines()))
        summary = (f'Expected {expected} nodes; observed {len(names)} unique names: '
                   f"{', '.join(names)}. Evidence: {self.names}, {self.snapshots}")
        return {'mismatch': len(names) != expected, 'summary': summary}


def monitor_test(test, poller, result, cancellation, timeout):
    deadline = None if timeout is None else time.monotonic() + timeout
    next_poll = time.monotonic() + POLL_INTERVAL
    while not cancellation.requested:
        if test.poll() is not None:
            return
        now = time.monotonic()
        if deadline is not None and now >= deadline:
            result['timed_out'] = True
            return
        if now >= next_poll:
            poller.sample()
            next_poll = time.monotonic() + POLL_INTERVAL
            continue
        # Check cancellation promptly without adding threads or signal-driven I/O.
        delay = min(0.1, next_poll - now)
        if deadline is not None:
            delay = min(delay, deadline - now)
        time.sleep(delay)


def cleanup(test, previous):
    try:
        terminate(test)
    finally:
        for sig, handler in previous.items():
            signal.signal(sig, handler)


def finish(result, test, cancellation):
    failed = bool(result.get('error') or result.get('mismatch'))
    status, _, detail = report_detail(result)
    print(('ERROR: ' if status == 'failure' else '') + detail, flush=True)
    if failed:
        return RunResult(1, True, timed_out=result.get('timed_out', False))
    if result.get('cancelled'):
        return RunResult(128 + cancellation.signal, False)
    if result.get('timed_out'):
        return RunResult(124, False, timed_out=True)
    code = test.returncode
    return RunResult(code if code >= 0 else 128 - code, False)


def run(command, timeout=None, env=None, run_url=''):
    """Sample names around a product command, deciding the verdict only at the end."""
    if not command:
        raise ValueError('node monitoring requires a product command')
    result = {'started_at': time.monotonic(), 'run_url': run_url}
    cancellation = Cancellation()
    previous = {sig: signal.signal(sig, cancellation.handle)
                for sig in (signal.SIGTERM, signal.SIGINT)}
    test, poller, expected = None, None, None
    try:
        environment = os.environ if env is None else env
        poller = NodePoll(environment)
        expected = expected_node_count(environment)
        initial = poller.sample()
        if initial == []:
            raise RuntimeError('initial node sample returned no nodes')
        if initial is not None and not cancellation.requested:
            expected = expected or len(set(initial))
            # cleanup() owns this group, including descendants after leader exit.
            test = subprocess.Popen(  # pylint: disable=consider-using-with
                command, env=env, start_new_session=True)
            monitor_test(test, poller, result, cancellation, timeout)
            if not cancellation.requested:
                poller.sample()
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError) as exc:
        if not cancellation.requested:
            result['error'] = f'node monitor execution: {exc}'
    finally:
        try:
            if poller is not None:
                if expected is not None and not cancellation.requested:
                    result.update(poller.verdict(expected))
                if poller.errors:
                    result['error'] = '\n'.join(filter(None, [result.get('error'), *poller.errors]))
            if cancellation.requested:
                result.update(cancelled=True, signal=cancellation.signal)
            result['duration_seconds'] = time.monotonic() - result['started_at']
            write_report(result)
        finally:
            cleanup(test, previous)
    return finish(result, test, cancellation)
