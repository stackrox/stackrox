"""Node-name polling shared by cluster-backed CI jobs."""

from contextlib import contextmanager
import json
import os
from pathlib import Path
import threading
import subprocess
import sys
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
    try:
        names = sorted(set(names_file.read_text(encoding="utf-8").splitlines())) if names_file.exists() else []
    except (OSError, ValueError) as exc:
        errors = [*errors, f"Could not read sampled names: {exc}"]
        names = []
    failed = bool(errors) or (not cancelled and len(names) != expected)
    skipped = cancelled or failed
    detail = f"Expected {expected} nodes; observed {len(names)} unique names: " f"{', '.join(names)}.\n" + "\n".join(
        errors
    )
    print(("WARNING: " if failed else "") + detail.strip(), flush=True)
    suite = ET.Element("testsuite", name="ci.gke.node", tests="1", failures="0", skipped=str(int(skipped)))
    case = ET.SubElement(suite, "testcase", classname="ci.gke.node", name="NoNodeDisruption")
    if failed:
        ET.SubElement(case, "skipped", message="Advisory node monitor finding").text = detail
    elif skipped:
        ET.SubElement(case, "skipped")
    temporary = directory / f".{JUNIT_REPORT}.{os.getpid()}.tmp"
    temporary.write_text(
        ET.tostring(suite, encoding="unicode", xml_declaration=True), encoding="utf-8"
    )
    temporary.replace(directory / JUNIT_REPORT)
    return failed


def finish_report(directory, expected, errors, cancelled=False):
    """Write the advisory report without letting artifact failures skip cleanup."""
    try:
        return write_report(directory, expected, errors, cancelled)
    except (OSError, ValueError, RuntimeError) as exc:
        print(f"WARNING: Could not write node monitor report: {exc}", flush=True)
        return True


@contextmanager
def monitor(env=None, *, baseline=False, phase=None):
    """Collect advisory node diagnostics without owning the product process."""
    env = dict(os.environ if env is None else env)
    if not baseline and not env.get("NUM_NODES"):
        yield
        return
    directory = Path(env.get("ARTIFACT_DIR", "junit-reports"))
    if phase:
        directory /= phase
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

    initial = None

    def sample(stage):
        nonlocal initial
        try:
            names = set(sample_nodes(directory, env))
            if initial is None:
                if not names:
                    errors.append(f"{stage}: node snapshot is empty")
                    return
                initial = names
                if expected is not None and len(names) != expected:
                    errors.append(f"{stage}: baseline contains {len(names)} nodes")
            elif names != initial:
                change = f"Nodes added: {sorted(names - initial)}; removed: {sorted(initial - names)}"
                if change not in errors:
                    errors.append(change)
            return names
        except (OSError, ValueError, KeyError, TypeError, RuntimeError, subprocess.SubprocessError) as exc:
            errors.append(f"{stage}: {exc}")

    def poll():
        while not stopped.wait(POLL_INTERVAL):
            sample("periodic")

    expected = None
    ready = False
    try:
        expected = None if baseline else expected_node_count(env)
        (directory / NODE_NAMES).write_text("", encoding="utf-8")
        (directory / MONITOR_LOG).write_text("", encoding="utf-8")
        ready = True
    except (OSError, ValueError, RuntimeError) as exc:
        errors.append(f"setup: {exc}")

    if ready:
        sample("startup")
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
        if ready and not cancelled:
            sample("final")
            if initial is None:
                errors.append("Could not establish a node baseline")
        if baseline and initial is not None:
            expected = len(initial)
        finish_report(directory, expected, errors, cancelled)


def actions_monitor(args):
    if not args.baseline and not os.environ.get("NUM_NODES"):
        return
    directory = Path(os.environ.get("ARTIFACT_DIR", "junit-reports"))
    if args.phase:
        directory /= args.phase
    state = Path(os.environ["RUNNER_TEMP"]) / (
        f"node-monitor-{os.environ['GITHUB_RUN_ID']}-{os.environ['GITHUB_RUN_ATTEMPT']}-"
        f"{os.environ['GITHUB_JOB']}-{args.phase or 'main'}"
    )
    stop = state / "stop"
    done = state / "done"
    if args.action == "start":
        state.mkdir()
        with (state / "worker.log").open("w") as log:
            subprocess.Popen(
                [sys.executable, __file__, "worker", *sys.argv[2:]],
                stdin=subprocess.DEVNULL, stdout=log, stderr=log, start_new_session=True,
            )
    elif args.action == "worker":
        try:
            with monitor(baseline=args.baseline, phase=args.phase):
                while not stop.exists():
                    time.sleep(0.25)
                if stop.read_text() == "cancelled":
                    raise KeyboardInterrupt
        except KeyboardInterrupt:
            pass
        finally:
            done.touch()
    else:
        if args.outcome == "skipped" and not state.exists():
            return
        if state.exists():
            request = state / "request"
            request.write_text(args.outcome)
            request.replace(stop)
            deadline = time.monotonic() + 90
            while not done.exists() and time.monotonic() < deadline:
                time.sleep(0.25)
        worker_log = state / "worker.log"
        if worker_log.is_file():
            print(worker_log.read_text(encoding="utf-8"), end="", flush=True)
        if not done.exists() or not (directory / JUNIT_REPORT).is_file():
            directory.mkdir(parents=True, exist_ok=True)
            finish_report(directory, None, ["Background node monitor did not publish a report"])


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("start", "worker", "finish"))
    parser.add_argument("--baseline", action="store_true")
    parser.add_argument("--phase")
    parser.add_argument("--outcome", default="")
    try:
        actions_monitor(parser.parse_args())
    except (OSError, ValueError, RuntimeError, KeyError) as exc:
        print(f"WARNING: Node monitor: {exc}", flush=True)
