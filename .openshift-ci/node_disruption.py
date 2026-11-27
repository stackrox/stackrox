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
    suite = ET.Element("testsuite", name="ci.node", tests="1", failures="0", skipped=str(int(skipped)))
    case = ET.SubElement(suite, "testcase", classname="ci.node", name="NoNodeDisruption")
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


def compare_snapshot(initial, names, stage, errors):
    if initial is None:
        if not names:
            errors.append(f"{stage}: node snapshot is empty")
            return None
        return names
    if names != initial:
        change = f"Nodes added: {sorted(names - initial)}; removed: {sorted(initial - names)}"
        if change not in errors:
            errors.append(change)
    return initial


def start_polling(stopped, sample, errors):
    def poll():
        while not stopped.wait(POLL_INTERVAL):
            sample("periodic")

    try:
        worker = threading.Thread(target=poll, name="node-monitor", daemon=True)
        worker.start()
        return worker
    except RuntimeError as exc:
        errors.append(str(exc))
        return None


def artifact_directory(env, phase):
    return Path(env.get("ARTIFACT_DIR", "junit-reports")) / (phase or "")


@contextmanager
def monitor(env=None, *, phase=None):
    """Collect node diagnostics and yield the list of advisory errors for the final report."""
    env = dict(os.environ if env is None else env)
    errors = []
    directory = artifact_directory(env, phase)
    try:
        directory.mkdir(parents=True, exist_ok=True)
    except OSError as exc:
        print(f"WARNING: Could not create node monitor directory: {exc}", flush=True)
        yield errors
        return
    stopped = threading.Event()
    worker = None
    cancelled = False

    initial = None

    def sample(stage):
        nonlocal initial
        try:
            names = set(sample_nodes(directory, env))
            initial = compare_snapshot(initial, names, stage, errors)
        except (OSError, ValueError, KeyError, TypeError, RuntimeError, subprocess.SubprocessError) as exc:
            errors.append(f"{stage}: {exc}")

    ready = False
    try:
        (directory / NODE_NAMES).write_text("", encoding="utf-8")
        (directory / MONITOR_LOG).write_text("", encoding="utf-8")
        ready = True
    except (OSError, ValueError, RuntimeError) as exc:
        errors.append(f"setup: {exc}")

    if ready:
        sample("startup")
        worker = start_polling(stopped, sample, errors)
    try:
        yield errors
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
        expected = len(initial) if initial else None
        finish_report(directory, expected, errors, cancelled)


def start_actions_monitor(state):
    state.mkdir()
    with (state / "worker.log").open("w") as log:
        # The detached worker must outlive this start step.
        worker = subprocess.Popen(  # pylint: disable=consider-using-with
            [sys.executable, __file__, "worker", *sys.argv[2:]],
            stdin=subprocess.DEVNULL, stdout=log, stderr=log, start_new_session=True,
        )
    deadline = time.monotonic() + 45
    while not (state / "ready").exists():
        if worker.poll() is not None:
            error = f"Node monitor worker exited {worker.returncode} before completing its startup probe"
            break
        if time.monotonic() >= deadline:
            error = "Node monitor startup probe was not acknowledged within 45 seconds"
            break
        time.sleep(0.25)
    else:
        return
    (state / "startup-error").write_text(error)
    print(f"WARNING: {error}", flush=True)


def run_actions_worker(state, args):
    stop = state / "stop"
    try:
        with monitor(phase=args.phase) as errors:
            (state / "ready").touch()
            while not stop.exists():
                time.sleep(0.25)
            if (state / "startup-error").exists():
                errors.append((state / "startup-error").read_text())
            if stop.read_text() == "cancelled":
                raise KeyboardInterrupt
    except KeyboardInterrupt:
        pass
    finally:
        (state / "done").touch()


def finish_actions_monitor(state, directory, args):
    stop = state / "stop"
    done = state / "done"
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
        errors = ["Background node monitor did not publish a report"]
        if (state / "startup-error").exists():
            errors.append((state / "startup-error").read_text())
        finish_report(directory, None, errors)


def actions_monitor(args):
    state = Path(os.environ["RUNNER_TEMP"]) / (
        f"node-monitor-{os.environ['GITHUB_RUN_ID']}-{os.environ['GITHUB_RUN_ATTEMPT']}-"
        f"{os.environ['GITHUB_JOB']}-{args.phase or 'main'}"
    )
    if args.action == "start":
        start_actions_monitor(state)
    elif args.action == "worker":
        run_actions_worker(state, args)
    else:
        finish_actions_monitor(state, artifact_directory(os.environ, args.phase), args)


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("start", "worker", "finish"))
    parser.add_argument("--phase")
    parser.add_argument("--outcome", default="")
    try:
        actions_monitor(parser.parse_args())
    except (OSError, ValueError, RuntimeError, KeyError) as exc:
        print(f"WARNING: Node monitor: {exc}", flush=True)
