#!/usr/bin/env python3
"""Build one real stream updater in its own clone, recording artifact provenance."""

import argparse
import json
import os
from pathlib import Path
import re
import shutil
import subprocess

import lib


IMAGE = "quay.io/stackrox-io/apollo-ci:scanner-test-0.5.11@sha256:cea78f2b3a5a38c9ccda678712daaed90b7c372d6b5df963feb758b32dccebe1"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source-repo", type=Path, required=True)
    parser.add_argument("--stream", required=True)
    parser.add_argument("--revision", required=True, help="Exact resolved stream-build SHA from the healthy workflow artifact")
    parser.add_argument("--directory", type=Path, required=True)
    parser.add_argument("--runtime", choices=("docker", "podman"), default="docker")
    args = parser.parse_args()
    directory = args.directory.resolve()
    directory.mkdir(parents=True, exist_ok=False)
    mapping = lib.original(args.source_repo.resolve(), "scanner/updater/version/VULNERABILITY_BUNDLE_VERSION").decode()
    refs = {fields[0]: fields[1] for line in mapping.splitlines() if len(fields := line.split()) >= 2 and not fields[0].startswith("#")}
    if args.stream not in refs or not re.fullmatch(r"[a-zA-Z0-9_-]+", args.stream):
        raise RuntimeError("Unknown/unsafe stream")
    if not re.fullmatch(r"[0-9a-f]{40}", args.revision):
        raise RuntimeError("The exact healthy stream build requires a full commit SHA")
    checkout = directory / "stackrox"
    lib.command(["git", "init", str(checkout)])
    lib.command(["git", "-C", str(checkout), "remote", "add", "origin", "https://github.com/stackrox/stackrox.git"])
    lib.command(["git", "-C", str(checkout), "fetch", "--tags", "origin"], timeout=900)
    lib.command(["git", "-C", str(checkout), "fetch", "origin", args.revision], timeout=900)
    branch = f"alkmim/ROX-36961-t1-build-{args.stream}"
    lib.command(["git", "-C", str(checkout), "checkout", "-b", branch, args.revision])
    if lib.command(["git", "-C", str(checkout), "rev-parse", "--is-shallow-repository"]).strip() != b"false":
        raise RuntimeError("The versioned build requires full history and tags, not a shallow clone")
    if (checkout / ".git/objects/info/alternates").exists():
        raise RuntimeError("Build checkout must not share another repository's Git metadata")
    revision = lib.command(["git", "-C", str(checkout), "rev-parse", "HEAD"]).decode().strip()
    command = [
        args.runtime, "run", "--rm", "--user", f"{os.getuid()}:{os.getgid()}",
        "-v", f"{checkout}:/src", "--workdir", "/src",
        "-e", "HOME=/tmp/t1-home", "-e", "GOCACHE=/tmp/t1-gocache",
        "-e", "GOMODCACHE=/tmp/t1-gomod", IMAGE,
        "bash", "-euc", "mkdir -p \"$HOME\"; make tag; make -C scanner bin/updater",
    ]
    with (directory / "build.log").open("wb") as log:
        completed = subprocess.run(command, stdout=log, stderr=subprocess.STDOUT, timeout=7200)
    if completed.returncode:
        raise RuntimeError(f"Real stream build failed; inspect {directory / 'build.log'}")
    binary = directory / "updater"
    shutil.copy2(checkout / "scanner/bin/updater", binary)
    binary.chmod(0o755)
    module = (checkout / "go.mod").read_text()
    clair = re.search(r"^\s*github\.com/quay/claircore\s+(\S+)", module, re.MULTILINE)
    lib.write_json(directory / "updater.json", {
        "stream": args.stream, "configured_ref": refs[args.stream],
        "resolved_commit": revision, "branch": branch, "ci_image": IMAGE,
        "go_mod_sha256": lib.digest(checkout / "go.mod"),
        "claircore": clair.group(1) if clair else None,
        "updater_sha256": lib.digest(binary),
    })
    print(json.dumps({"updater": str(binary), "manifest": str(directory / "updater.json")}, indent=2))


if __name__ == "__main__":
    main()
