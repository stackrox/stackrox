#!/usr/bin/env python3
"""Run N3/N4 with supplied real artifacts and fresh, unexposed containers."""

import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import uuid

import lib


CASES = ("control", "manual-control", "N3a", "N3b", "N3c", "N3d-control", "N3d", "N4-missing", "N4-epoch", "N4-go-zero")
IMAGE = "quay.io/stackrox-io/apollo-ci:scanner-test-0.5.11@sha256:cea78f2b3a5a38c9ccda678712daaed90b7c372d6b5df963feb758b32dccebe1"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, required=True)
    parser.add_argument("--stream", required=True)
    parser.add_argument("--updater", type=Path, required=True)
    parser.add_argument("--updater-manifest", type=Path, required=True)
    parser.add_argument("--bundle", type=Path, required=True)
    parser.add_argument("--healthy-artifact", type=Path, required=True, help="The matching healthy stream artifact directory from assess-run.py")
    parser.add_argument("--evidence", type=Path, required=True)
    parser.add_argument("--runtime", choices=("docker", "podman"), default="docker")
    parser.add_argument("--case", choices=CASES, action="append")
    parser.add_argument("--timeout", type=int, default=3600)
    args = parser.parse_args()
    repo, updater, bundle = args.repo.resolve(), args.updater.resolve(), args.bundle.resolve()
    evidence = args.evidence.resolve()
    evidence.mkdir(parents=True, exist_ok=False)
    if not shutil.which(args.runtime):
        raise RuntimeError(f"Missing container runtime: {args.runtime}")
    provenance = json.loads(args.updater_manifest.read_text())
    if provenance["stream"] != args.stream or provenance["updater_sha256"] != lib.digest(updater):
        raise RuntimeError("Updater provenance does not match the supplied stream/binary")
    artifact = args.healthy_artifact.resolve()
    healthy = json.loads((artifact / "run.json").read_text())
    checked = json.loads((artifact / "object-check.json").read_text())
    built = json.loads((artifact / "updater-build.json").read_text())
    if healthy["implementation_commit"] != lib.BASE or healthy["case"] != "healthy" or healthy["stream"] != args.stream:
        raise RuntimeError("Healthy artifact does not belong to the requested implementation/stream")
    if checked["status"] != "Passed" or checked["http_status"] != 200 or checked["stream"] != args.stream or checked["readback_sha256"] != lib.digest(bundle):
        raise RuntimeError("Bundle does not match a verified healthy generation-matched read-back")
    if built["stream"] != args.stream or any(built[key] != provenance[key] for key in ("resolved_commit", "go_mod_sha256", "claircore")):
        raise RuntimeError("Local updater build does not reproduce the healthy stream's exact revision/dependency pin")
    lib.write_json(evidence / "inputs.json", {
        "implementation_commit": lib.BASE, "updater": provenance,
        "bundle_origin": checked["uri"], "bundle_generation": checked["metadata"]["generation"],
        "bundle_sha256": lib.digest(bundle), "healthy_run": healthy,
        "ci_image": IMAGE,
    })
    selected = args.case or list(CASES)
    cases = ["control"]
    for case in selected:
        prerequisites = {"N3c": "manual-control", "N3d": "N3d-control"}
        for candidate in (prerequisites.get(case), case):
            if candidate and candidate not in cases:
                cases.append(candidate)
    results = []
    passed = set()
    for case in cases:
        prerequisite = {"N3c": "manual-control", "N3d": "N3d-control"}.get(case, "control")
        if case != "control" and prerequisite not in passed:
            results.append({"case": case, "status": "Pending", "reason": f"Prerequisite {prerequisite} failed"})
            continue
        cell = evidence / case
        cell.mkdir()
        suffix = uuid.uuid4().hex[:12]
        network, database, client = (f"t1-{kind}-{suffix}" for kind in ("net", "pg", "client"))
        created_network = False
        try:
            lib.command([args.runtime, "network", "create", network])
            created_network = True
            lib.command([
                args.runtime, "run", "-d", "--name", database,
                "--network", network, "--network-alias", "postgres",
                "--label", "rox.t1.disposable=true",
                "-e", "POSTGRES_USER=test", "-e", "POSTGRES_PASSWORD=test",
                "-e", "POSTGRES_DB=matcher", "docker.io/library/postgres:15",
            ])
            command = [
                args.runtime, "run", "--name", client, "--network", network,
                "--user", f"{os.getuid()}:{os.getgid()}",
                "--label", "rox.t1.disposable=true",
                "-v", f"{repo}:/repo:ro", "-v", f"{cell}:/work",
                "-v", f"{updater}:/tmp/updater:ro",
                "-v", f"{bundle}:/inputs/vulnerabilities.zip:ro",
                "-e", "T1_ISOLATED_DATABASE=true", "-e", "PGHOST=postgres",
                "--workdir", "/work", IMAGE, "python3",
                "/repo/.github/workflows/scripts/t1-tests/integration-case.py",
                "--repo", "/repo", "--case", case, "--stream", args.stream,
                "--timeout", str(args.timeout),
            ]
            completed = subprocess.run(command, capture_output=True, text=True, timeout=args.timeout + 600)
            (cell / "container.log").write_text(completed.stdout + completed.stderr)
            report = cell / "evidence/result.json"
            if not report.exists():
                raise RuntimeError("Container did not produce a case result; this is a setup failure")
            result = json.loads(report.read_text())
            if completed.returncode != 0 or result["status"] != "Passed":
                result["status"] = "Failed"
            else:
                passed.add(case)
            results.append(result)
        except Exception as error:
            results.append({"case": case, "status": "Failed", "error": str(error)})
        finally:
            if created_network:
                logs = subprocess.run([args.runtime, "logs", database], capture_output=True)
                (cell / "postgres.log").write_bytes(logs.stdout + logs.stderr)
                cleanup = []
                for command in ([args.runtime, "rm", "-f", client], [args.runtime, "rm", "-f", database], [args.runtime, "network", "rm", network]):
                    outcome = subprocess.run(command, capture_output=True, text=True)
                    cleanup.append({"command": command, "exit": outcome.returncode, "output": outcome.stdout + outcome.stderr})
                lib.write_json(cell / "cleanup.json", cleanup)
            lib.write_json(evidence / "results.json", results)
    print(json.dumps(results, indent=2))
    return 0 if results and all(r["status"] == "Passed" for r in results) else 1


if __name__ == "__main__":
    sys.exit(main())
