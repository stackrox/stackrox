"""Node-name polling shared by nongroovy CI entrypoints."""

from contextlib import contextmanager
import json
import os
from pathlib import Path
import threading
import subprocess
import time
import xml.etree.ElementTree as ET

API_TIMEOUT = 30
NODE_NAMES = "node-names.txt"
MONITOR_LOG = "node-monitor.log"
JUNIT_REPORT = "junit-node-monitor.xml"
POLL_INTERVAL = 5 * 60


def expected_node_count(env):
    value = env.get("NUM_NODES")
    if not value:
        return None
    try:
        count = int(value)
    except ValueError as exc:
        raise RuntimeError("NUM_NODES must be an integer") from exc
    if count > 0:
        return count
    raise RuntimeError("NUM_NODES must be a positive integer")


def sample_nodes(directory, env):
    """Keep raw kubectl output for diagnosis and append names for the final check."""
    snapshot = directory / f"node-snapshot-{time.time_ns()}.json"
    with snapshot.open("w", encoding="utf-8") as stdout, (directory / MONITOR_LOG).open(
        "a", encoding="utf-8"
    ) as stderr:
        stderr.write(f"\nCollecting {snapshot.name}\n")
        stderr.flush()
        completed = subprocess.run(
            ["kubectl", f"--request-timeout={API_TIMEOUT}s", "get", "nodes", "-o", "json"],
            env=env,
            stdout=stdout,
            stderr=stderr,
            timeout=API_TIMEOUT,
            check=False,
        )
    if completed.returncode:
        raise RuntimeError(f"kubectl get nodes exited {completed.returncode}")
    names = [node["metadata"]["name"] for node in json.loads(snapshot.read_text(encoding="utf-8"))["items"]]
    if any(not isinstance(name, str) or not name or any(char.isspace() for char in name) for name in names):
        raise ValueError("node snapshot contains an invalid name")
    with (directory / NODE_NAMES).open("a", encoding="utf-8") as stream:
        stream.writelines(name + "\n" for name in names)
    return names


def write_report(directory, expected, errors, cancelled=False):
    names_file = directory / NODE_NAMES
    names = sorted(set(names_file.read_text(encoding="utf-8").splitlines())) if names_file.exists() else []
    failed = bool(errors) or (not cancelled and len(names) != expected)
    skipped = cancelled and not failed
    detail = f"Expected {expected} nodes; observed {len(names)} unique names: " f"{', '.join(names)}.\n" + "\n".join(
        errors
    )
    print(("WARNING: " if failed else "") + detail.strip(), flush=True)
    suite = ET.Element("testsuite", name="ci.gke.node", tests="1", failures=str(int(failed)), skipped=str(int(skipped)))
    case = ET.SubElement(suite, "testcase", classname="ci.gke.node", name="NoNodeDisruption")
    if failed:
        ET.SubElement(case, "failure").text = detail
    elif skipped:
        ET.SubElement(case, "skipped")
    (directory / JUNIT_REPORT).write_text(
        ET.tostring(suite, encoding="unicode", xml_declaration=True), encoding="utf-8"
    )
    return failed


def prepare_monitor(directory, env, sample, errors):
    try:
        expected = expected_node_count(env)
        if expected is not None:
            (directory / NODE_NAMES).write_text("", encoding="utf-8")
            (directory / MONITOR_LOG).write_text("", encoding="utf-8")
            sample()
        return expected
    except (OSError, ValueError, RuntimeError) as exc:
        errors.append(str(exc))
        return None


def finish_report(directory, env, expected, errors, cancelled=False):
    """Write the advisory report without letting artifact failures skip cleanup."""
    if not env.get("NUM_NODES"):
        return False
    try:
        return write_report(directory, expected, errors, cancelled)
    except (OSError, ValueError, RuntimeError) as exc:
        print(f"WARNING: Could not write node monitor report: {exc}", flush=True)
        return True


@contextmanager
def monitor(env=None):
    """Collect advisory node diagnostics without owning the product process."""
    env = dict(os.environ if env is None else env)
    if not env.get("NUM_NODES"):
        yield
        return
    directory = Path(env.get("ARTIFACT_DIR", "junit-reports"))
    try:
        directory.mkdir(parents=True, exist_ok=True)
    except OSError as exc:
        print(f"WARNING: Could not create node monitor directory: {exc}", flush=True)
        yield
        return
    errors = []
    stopped = threading.Event()
    worker = None
    cancelled = False

    def sample():
        try:
            return sample_nodes(directory, env)
        except (OSError, ValueError, KeyError, TypeError, RuntimeError, subprocess.SubprocessError) as exc:
            errors.append(str(exc))
            return None

    def poll():
        while not stopped.wait(POLL_INTERVAL):
            sample()

    expected = prepare_monitor(directory, env, sample, errors)
    if expected is not None:
        try:
            worker = threading.Thread(target=poll, name="node-monitor", daemon=True)
            worker.start()
        except RuntimeError as exc:
            # Cleanup must not join a thread that failed to start.
            worker = None
            errors.append(str(exc))
    try:
        yield
    except (KeyboardInterrupt, SystemExit):
        cancelled = True
        raise
    finally:
        stopped.set()
        if worker is not None:
            worker.join()
        if expected is not None and not cancelled:
            sample()
        finish_report(directory, env, expected, errors, cancelled)
