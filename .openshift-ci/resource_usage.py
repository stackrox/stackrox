"""Temporary, opt-in resource measurements for the single-node Prow pilot."""

from datetime import datetime, timezone
import json
from pathlib import Path
import subprocess
import threading
import time


class ResourceSampler:
    """Record usage every 30 seconds and scheduling snapshots every 60 seconds."""

    COMMAND_TIMEOUT = 5
    INTERVAL = 30

    def __init__(self, output_dir):
        self.output_dir = Path(output_dir)
        self.stopped = threading.Event()
        self.thread = None

    def start(self):
        self.output_dir.mkdir(parents=True, exist_ok=True)
        self.thread = threading.Thread(target=self._run, daemon=True)
        self.thread.start()

    def stop(self):
        self.stopped.set()
        if self.thread is not None:
            self.thread.join(timeout=self.COMMAND_TIMEOUT + 1)
            if self.thread.is_alive():
                print("WARNING: Resource sampler did not stop within its command timeout")

    def _run(self):
        iteration = 0
        try:
            while not self.stopped.is_set():
                started = time.monotonic()
                self.sample(include_specs=iteration % 2 == 0)
                iteration += 1
                self.stopped.wait(max(0, self.INTERVAL - (time.monotonic() - started)))
        except Exception as err:  # Sampling must never replace the test's failure.
            print(f"WARNING: Resource sampling stopped: {err}")

    def sample(self, include_specs):
        commands = {
            "node-usage": ["top", "nodes", "--no-headers"],
            "pod-usage": ["top", "pods", "--all-namespaces", "--containers", "--no-headers"],
        }
        if include_specs:
            commands.update({
                "nodes": ["get", "nodes", "-o", "json"],
                "pods": ["get", "pods", "--all-namespaces", "-o", "json"],
                "events": ["get", "events", "--all-namespaces", "-o", "json"],
            })
        for kind, args in commands.items():
            if self.stopped.is_set():
                break
            record = {"timestamp": datetime.now(timezone.utc).isoformat(), "kind": kind}
            try:
                result = subprocess.run(
                    ["kubectl", "--request-timeout=4s", *args],
                    capture_output=True, text=True, check=False, timeout=self.COMMAND_TIMEOUT,
                )
                record["returncode"] = result.returncode
                if result.returncode:
                    record["error"] = result.stderr
                elif kind == "pods":
                    # Pod environments may contain credentials. Only publish scheduling resources and status.
                    record["output"] = pod_resources(json.loads(result.stdout))
                else:
                    record["output"] = result.stdout
            except (OSError, subprocess.TimeoutExpired, ValueError) as err:
                record["error"] = str(err)
            with (self.output_dir / "samples.jsonl").open("a", encoding="utf8") as output:
                output.write(json.dumps(record) + "\n")


def pod_resources(pods):
    """Keep request/placement information without exposing container environments."""
    result = []
    for pod in pods.get("items", []):
        metadata = pod.get("metadata", {})
        spec = pod.get("spec", {})
        filtered_spec = {key: spec[key] for key in (
            "nodeName", "nodeSelector", "affinity", "tolerations", "topologySpreadConstraints",
            "overhead", "resources", "priority", "schedulerName",
        ) if key in spec}
        for key in ("containers", "initContainers"):
            filtered_spec[key] = [{field: container[field] for field in ("name", "resources", "restartPolicy")
                                   if field in container} for container in spec.get(key, [])]
        result.append({
            "metadata": {key: metadata[key] for key in ("name", "namespace", "uid", "ownerReferences")
                         if key in metadata},
            "spec": filtered_spec,
            "status": pod.get("status", {}),
        })
    return result
