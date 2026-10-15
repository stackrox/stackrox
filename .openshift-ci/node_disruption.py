"""Opt-in node monitoring shared by nongroovy CI entrypoints.

Use the raw NodeList: kubectl's generic JSON list printer discards collection
metadata. Watches must start at the collection version, not a node's version.
"""

from dataclasses import dataclass
import http.client
import json
import os
from pathlib import Path
import queue
import signal
import socket
import subprocess
import tempfile
import threading
import time
import xml.etree.ElementTree as ET
from urllib.parse import urlencode

API_TIMEOUT = 10
TERM_GRACE = 5
WATCH_TIMEOUT = 300
WATCH_RENEW_DELAY = 0.1


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
    if kind in ('ADDED', 'BOOKMARK'):
        return None
    if kind not in ('DELETED', 'MODIFIED'):
        raise RuntimeError(f'Unexpected Kubernetes watch event type: {kind!r}')

    name = node.get('metadata', {}).get('name')
    if not name:
        raise RuntimeError(f'{kind} node event has no node name')
    ready = ready_status(node)
    if kind == 'DELETED' or ready != 'True':
        return {'type': kind, 'node': name, 'ready': ready}
    return None


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


class UnixConnection(http.client.HTTPConnection):
    """HTTP over a private kubectl proxy socket; kubectl owns authentication."""

    def __init__(self, path):
        super().__init__('localhost', timeout=API_TIMEOUT)
        self.path = str(path)
        self.transport_socket = None

    def connect(self):
        self.sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.sock.settimeout(self.timeout)
        self.transport_socket = self.sock
        self.sock.connect(self.path)


def baseline_resource_version(snapshot):
    if not isinstance(snapshot, dict):
        raise RuntimeError('node baseline is not an object')
    nodes = snapshot.get('items', [])
    if not nodes:
        raise RuntimeError('node baseline returned no nodes')
    version = snapshot.get('metadata', {}).get('resourceVersion')
    if not isinstance(version, str) or not version:
        raise RuntimeError('NodeList metadata.resourceVersion is missing')
    not_ready = [node.get('metadata', {}).get('name', '<unnamed>')
                 for node in nodes if ready_status(node) != 'True']
    if not_ready:
        raise RuntimeError('Nodes not Ready before E2E tests: ' + ', '.join(not_ready))
    return version


def response_or_error(connection, path):
    connection.request('GET', path)
    response = connection.getresponse()
    if response.status != 200:
        # Status/reason suffice; never dump authentication responses or headers.
        raise RuntimeError(f'Kubernetes API HTTP {response.status} {response.reason}')
    return response


class NodeWatch:
    """Own the proxy and blocking HTTP reader; deliver outcomes to the main thread."""

    def __init__(self):
        self.messages = queue.Queue(maxsize=256)
        self.stopped = threading.Event()
        self.connection = None
        self.proxy = None
        self.reader = None
        self.temporary = None
        self.stderr = None

    def start(self, cancellation):
        # Resources span startup, monitoring and close(), rather than one scope.
        # pylint: disable=consider-using-with
        self.temporary = tempfile.TemporaryDirectory(prefix='node-watch-')
        path = Path(self.temporary.name) / 'api.sock'
        self.stderr = tempfile.TemporaryFile()
        self.proxy = subprocess.Popen(
            ['kubectl', 'proxy', '--unix-socket', str(path),
             '--accept-paths=^/api/v1/nodes$',
             '--reject-methods=^(POST|PUT|PATCH|DELETE|CONNECT|OPTIONS|TRACE|HEAD)$'],
            stdout=subprocess.DEVNULL, stderr=self.stderr, start_new_session=True,
        )
        deadline = time.monotonic() + API_TIMEOUT
        while not path.exists():
            if cancellation.requested:
                return
            if self.proxy.poll() is not None:
                raise RuntimeError('kubectl proxy exited during startup')
            if time.monotonic() >= deadline:
                raise RuntimeError('kubectl proxy startup timed out')
            time.sleep(0.05)
        self.reader = threading.Thread(target=self.read, args=(path,), daemon=True)
        self.reader.start()

    def emit(self, kind, value=None):
        while not self.stopped.is_set():
            try:
                self.messages.put((kind, value), timeout=0.1)
                return
            except queue.Full:
                continue

    def read(self, path):
        phase = 'node baseline'
        try:
            self.connection = UnixConnection(path)
            response = response_or_error(self.connection, '/api/v1/nodes')
            version = baseline_resource_version(json.loads(response.read()))
            self.connection.close()
            phase = 'node watch'
            self.read_watches(path, version)
        except (OSError, ValueError, TypeError, AttributeError, RuntimeError,
                http.client.HTTPException) as exc:
            self.emit('error', f'{phase}: {exc}')

    def read_watches(self, path, version):
        """Resume cleanly expired watches without replacing the original baseline."""
        first = True
        while not self.stopped.is_set():
            self.connection = UnixConnection(path)
            query = urlencode({'watch': 'true', 'resourceVersion': version,
                               'allowWatchBookmarks': 'true', 'timeoutSeconds': WATCH_TIMEOUT})
            try:
                response = response_or_error(self.connection, '/api/v1/nodes?' + query)
                self.connection.transport_socket.settimeout(WATCH_TIMEOUT + API_TIMEOUT)
                if first:
                    self.emit('ready')
                    first = False
                version = self.read_events(response, version)
            finally:
                self.connection.close()
            if version is None:
                return
            # Clean HTTP EOF is normal watch expiration. A bounded pause avoids
            # spinning on servers that repeatedly return empty streams.
            self.stopped.wait(WATCH_RENEW_DELAY)

    def read_events(self, response, version):
        while not self.stopped.is_set():
            line = response.readline()
            if not line:
                return version
            event = json.loads(line)
            disrupted = disruption(event)
            if disrupted:
                self.emit('event', disrupted)
                return None
            current = event.get('object', {}).get('metadata', {}).get('resourceVersion')
            if not isinstance(current, str) or not current:
                raise RuntimeError('watch event metadata.resourceVersion is missing')
            # Resource versions are opaque tokens, including on ignored ADDED
            # events and BOOKMARKs. Never order them or relist after HTTP 410:
            # relisting would hide disruptions in a lost-history interval.
            version = current
        return None

    def diagnostic(self):
        if self.stderr is None:
            return ''
        self.stderr.seek(0, os.SEEK_END)
        size = self.stderr.tell()
        self.stderr.seek(max(0, size - 1000))
        return self.stderr.read().decode('utf-8', errors='replace').strip()

    def close(self):
        self.stopped.set()
        # Stop upstream first, then unblock a reader even if HTTP handed socket
        # ownership to a Connection: close response.
        terminate(self.proxy, grace=1)
        if self.connection is not None:
            sock = self.connection.transport_socket
            if sock is not None:
                try:
                    sock.shutdown(socket.SHUT_RDWR)
                except OSError:
                    pass
        if self.reader is not None:
            self.reader.join(timeout=API_TIMEOUT + 1)
        if self.connection is not None:
            self.connection.close()
        if self.stderr is not None:
            self.stderr.close()
        if self.temporary is not None:
            self.temporary.cleanup()


def next_message(watch):
    try:
        return watch.messages.get(timeout=0.1)
    except queue.Empty:
        if watch.proxy.poll() is not None:
            return 'error', 'kubectl proxy exited unexpectedly'
        return None, None


def await_ready(watch, result, cancellation):
    # Both baseline and watch-header requests have their own ten-second timeout.
    deadline = time.monotonic() + 2 * API_TIMEOUT + 1
    while not cancellation.requested:
        kind, value = next_message(watch)
        if kind == 'ready':
            # A buffered disruption/error must prevent launching product tests.
            if not watch.messages.empty():
                kind, value = watch.messages.get_nowait()
                result[kind] = value
                return False
            return True
        if kind == 'error' or time.monotonic() >= deadline:
            result['error'] = value or 'node monitoring establishment timed out'
            return False
    return False


def monitor_test(test, watch, result, cancellation, timeout):
    deadline = None if timeout is None else time.monotonic() + timeout
    while not cancellation.requested:
        kind, value = next_message(watch)
        if kind in ('error', 'event'):
            result[kind] = value
            return
        if test.poll() is not None and watch.messages.empty():
            return
        if deadline is not None and time.monotonic() >= deadline:
            result['timed_out'] = True
            return


def cleanup(test, watch, previous):
    try:
        terminate(test)
    finally:
        try:
            watch.close()
        finally:
            for sig, handler in previous.items():
                signal.signal(sig, handler)


def finish(result, test, cancellation):
    failed = bool(result.get('error') or result.get('event'))
    print(report_detail(result)[2], flush=True)
    if failed:
        return RunResult(1, True)
    if result.get('cancelled'):
        return RunResult(128 + cancellation.signal, False)
    if result.get('timed_out'):
        return RunResult(124, False, timed_out=True)
    code = test.returncode
    return RunResult(code if code >= 0 else 128 - code, False)


def run(command, timeout=None, env=None, run_url=''):
    """Run an already opted-in command; monitor failures are distinct from its exit."""
    if not command:
        raise ValueError('node monitoring requires a product command')
    result = {'started_at': time.monotonic(), 'run_url': run_url}
    cancellation = Cancellation()
    previous = {sig: signal.signal(sig, cancellation.handle)
                for sig in (signal.SIGTERM, signal.SIGINT)}
    watch = NodeWatch()
    test = None
    try:
        watch.start(cancellation)
        if await_ready(watch, result, cancellation) and not cancellation.requested:
            # cleanup() owns this group, including descendants after leader exit.
            test = subprocess.Popen(  # pylint: disable=consider-using-with
                command, env=env, start_new_session=True)
            monitor_test(test, watch, result, cancellation, timeout)
    except (OSError, RuntimeError, subprocess.SubprocessError) as exc:
        if not cancellation.requested:
            result['error'] = f'node monitor execution: {exc}'
    finally:
        if result.get('error'):
            diagnostic = watch.diagnostic()
            if diagnostic:
                result['error'] += '\n' + diagnostic
        if cancellation.requested and not (result.get('error') or result.get('event')):
            result.update(cancelled=True, signal=cancellation.signal)
        result['duration_seconds'] = time.monotonic() - result['started_at']
        try:
            # Publish evidence before potentially spending five seconds terminating tests.
            write_report(result)
        finally:
            cleanup(test, watch, previous)
    return finish(result, test, cancellation)
