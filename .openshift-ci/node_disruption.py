"""Node-name polling shared by nongroovy CI entrypoints."""

import argparse
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
    """Stop the whole product process group, including surviving descendants."""
    if process is None:
        return
    deadline = time.monotonic() + grace
    try:
        os.killpg(process.pid, signal.SIGTERM)
        try:
            process.wait(timeout=grace)
        except subprocess.TimeoutExpired:
            pass
        # A shell may exit before its children. Give the group the remaining grace.
        os.killpg(process.pid, 0)
        time.sleep(max(0, deadline - time.monotonic()))
        os.killpg(process.pid, signal.SIGKILL)
    except ProcessLookupError:
        pass
    process.wait()


def expected_node_count(env):
    value = env.get('NUM_NODES')
    if not value:
        return None
    try:
        count = int(value)
        if count > 0:
            return count
    except ValueError:
        pass
    raise RuntimeError('NUM_NODES must be a positive integer')


def sample_nodes(directory, env):
    """Keep raw kubectl output for diagnosis and append names for the final check."""
    snapshot = directory / f'node-snapshot-{time.time_ns()}.json'
    with snapshot.open('w', encoding='utf-8') as stdout, \
            (directory / 'node-monitor.log').open('a', encoding='utf-8') as stderr:
        stderr.write(f'\nCollecting {snapshot.name}\n')
        stderr.flush()
        completed = subprocess.run(
            ['kubectl', f'--request-timeout={API_TIMEOUT}s', 'get', 'nodes', '-o', 'json'],
            env=env, stdout=stdout, stderr=stderr, timeout=API_TIMEOUT, check=False)
    if completed.returncode:
        raise RuntimeError(f'kubectl get nodes exited {completed.returncode}')
    names = [node['metadata']['name'] for node in json.loads(snapshot.read_text(encoding='utf-8'))['items']]
    if any(not isinstance(name, str) or not name or any(char.isspace() for char in name) for name in names):
        raise ValueError('node snapshot contains an invalid name')
    with (directory / 'node-names.txt').open('a', encoding='utf-8') as stream:
        stream.writelines(name + '\n' for name in names)
    return names


def write_report(directory, expected, errors, cancellation):
    names_file = directory / 'node-names.txt'
    names = sorted(set(names_file.read_text(encoding='utf-8').splitlines())) if names_file.exists() else []
    failed = bool(errors) or (not cancellation.requested and len(names) != expected)
    skipped = cancellation.requested and not failed
    detail = (f'Expected {expected} nodes; observed {len(names)} unique names: '
              f"{', '.join(names)}.\n" + '\n'.join(errors))
    print(('ERROR: ' if failed else '') + detail.strip(), flush=True)
    suite = ET.Element('testsuite', name='ci.gke.node', tests='1',
                       failures=str(int(failed)), skipped=str(int(skipped)))
    case = ET.SubElement(suite, 'testcase', classname='ci.gke.node', name='NoNodeDisruption')
    if failed:
        ET.SubElement(case, 'failure').text = detail
    elif skipped:
        ET.SubElement(case, 'skipped')
    temporary = directory / 'node-monitor.xml.tmp'
    temporary.write_text(ET.tostring(suite, encoding='unicode', xml_declaration=True), encoding='utf-8')
    temporary.replace(directory / 'node-monitor.xml')
    return failed


def wait_for_test(test, sample, cancellation, timeout):
    deadline = float('inf') if timeout is None else time.monotonic() + timeout
    next_poll = time.monotonic() + POLL_INTERVAL
    while not cancellation.requested and test.poll() is None:
        now = time.monotonic()
        if now >= deadline:
            return True
        if sample and now >= next_poll:
            sample()
            next_poll = time.monotonic() + POLL_INTERVAL
        time.sleep(min(0.1, max(0, deadline - time.monotonic())))
    return False


def cleanup(test, previous):
    try:
        terminate(test)
    finally:
        for sig, handler in previous.items():
            signal.signal(sig, handler)


def finish(test, cancellation, timed_out, failed):
    if failed:
        code = 1
    elif cancellation.requested:
        code = 128 + cancellation.signal
    elif timed_out:
        code = 124
    else:
        code = test.returncode if test else 1
    return RunResult(code if code >= 0 else 128 - code, failed, timed_out)


def run(command, timeout=None, env=None):
    """Run tests; poll node names only when NUM_NODES is supplied."""
    if not command:
        raise ValueError('a product command is required')
    env = os.environ if env is None else env
    directory = Path(os.getenv('ARTIFACT_DIR', 'junit-reports'))
    cancellation = Cancellation()
    errors = []
    previous = {sig: signal.signal(sig, cancellation.handle) for sig in (signal.SIGTERM, signal.SIGINT)}
    test, expected, timed_out, failed = None, None, False, False

    def sample():
        try:
            return sample_nodes(directory, env)
        except (OSError, ValueError, KeyError, TypeError, RuntimeError, subprocess.SubprocessError) as exc:
            errors.append(str(exc))
            return None

    try:
        expected = expected_node_count(env)
        if expected is not None:
            directory.mkdir(parents=True, exist_ok=True)
            (directory / 'node-names.txt').write_text('', encoding='utf-8')
            (directory / 'node-monitor.log').write_text('', encoding='utf-8')
            if not sample():
                raise RuntimeError('initial node sample failed or returned no nodes')
        if not cancellation.requested:
            test = subprocess.Popen(  # pylint: disable=consider-using-with
                command, env=env, start_new_session=True)
            timed_out = wait_for_test(test, sample if expected else None, cancellation, timeout)
            if expected and not cancellation.requested:
                sample()
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError) as exc:
        if not cancellation.requested:
            errors.append(str(exc))
    finally:
        try:
            if env.get('NUM_NODES'):
                directory.mkdir(parents=True, exist_ok=True)
                failed = write_report(directory, expected, errors, cancellation)
        finally:
            cleanup(test, previous)
    return finish(test, cancellation, timed_out, failed)


def publish_output(failed):
    output = os.getenv('GITHUB_OUTPUT')
    if not output:
        return
    with open(output, 'a', encoding='utf-8') as stream:
        stream.write(f"failed={str(failed).lower()}\n")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['run'])
    parser.add_argument('command', nargs=argparse.REMAINDER)
    args = parser.parse_args()
    command = args.command[1:] if args.command[:1] == ['--'] else args.command
    if not command:
        parser.error('a command is required after --')
    try:
        result = run(command)
    except (OSError, RuntimeError) as exc:
        publish_output(True)
        print(f'::error::Node monitor could not finish: {exc}', flush=True)
        return 1
    publish_output(result.monitor_failed)
    if result.monitor_failed:
        print('::error::Node disruption or monitor error; see node-monitor.xml', flush=True)
    return result.exit_code


if __name__ == '__main__':
    raise SystemExit(main())
