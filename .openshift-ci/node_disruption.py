"""Node-name polling shared by nongroovy CI entrypoints."""

from contextlib import contextmanager
import json
import os
from pathlib import Path
import threading
import subprocess
import time
import xml.etree.ElementTree as ET

API_TIMEOUT = 10
POLL_INTERVAL = 5 * 60


def expected_node_count(env):
    value = env.get("NUM_NODES")
    if not value:
        return None
    try:
        count = int(value)
        if count > 0:
            return count
    except ValueError:
        pass
    raise RuntimeError("NUM_NODES must be a positive integer")


def sample_nodes(directory, env):
    """Keep raw kubectl output for diagnosis and append names for the final check."""
    snapshot = directory / f"node-snapshot-{time.time_ns()}.json"
    with snapshot.open("w", encoding="utf-8") as stdout, (directory / "node-monitor.log").open(
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
    with (directory / "node-names.txt").open("a", encoding="utf-8") as stream:
        stream.writelines(name + "\n" for name in names)
    return names


def write_report(directory, expected, errors, cancelled=False):
    names_file = directory / "node-names.txt"
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
    temporary = directory / "node-monitor.xml.tmp"
    temporary.write_text(ET.tostring(suite, encoding="unicode", xml_declaration=True), encoding="utf-8")
    temporary.replace(directory / "node-monitor.xml")
    return failed


def prepare_monitor(directory, env, sample, errors):
    try:
        expected = expected_node_count(env)
        if expected is not None:
            directory.mkdir(parents=True, exist_ok=True)
            (directory / "node-names.txt").write_text("", encoding="utf-8")
            (directory / "node-monitor.log").write_text("", encoding="utf-8")
            if sample() == []:
                errors.append("initial node sample returned no nodes")
        return expected
    except (OSError, ValueError, RuntimeError) as exc:
        errors.append(str(exc))
        return None


def finish_report(directory, env, expected, errors, cancelled=False):
    """Write the advisory report without letting artifact failures skip cleanup."""
    if not env.get("NUM_NODES"):
        return False
    try:
        directory.mkdir(parents=True, exist_ok=True)
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
            thread = threading.Thread(target=poll, name="node-monitor", daemon=True)
            thread.start()
            worker = thread
        except RuntimeError as exc:
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
