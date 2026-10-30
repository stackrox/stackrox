#!/usr/bin/env python3
"""N1/N2: exercise pinned original scripts/blocks with the real configuration."""

import argparse
import json
from pathlib import Path
import shutil
import subprocess
import tempfile
import warnings
import zipfile

import lib


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, required=True)
    parser.add_argument("--evidence", type=Path, required=True)
    parser.add_argument("--stream", action="append", help="Repeat to select streams; default: all configured streams")
    args = parser.parse_args()
    repo, evidence = args.repo.resolve(), args.evidence.resolve()
    evidence.mkdir(parents=True, exist_ok=False)
    for tool in ("bash", "bats", "git", "zstd", "jq", "yq", "zipinfo"):
        if not shutil.which(tool):
            raise RuntimeError(f"Missing prerequisite: {tool}")
    lib.extract(repo, evidence / "original-blocks")
    configured = lib.document(repo, lib.CONFIG)["bundles"]
    streams = args.stream or list(configured)
    if any(stream not in configured for stream in streams):
        raise RuntimeError("Unknown stream")
    results = []

    def execute(case, command, root, env, *, error=None):
        result = subprocess.run(command, cwd=root, env=env, capture_output=True, text=True, timeout=60)
        text = result.stdout + result.stderr
        (evidence / f"{case}.log").write_text(text)
        good = result.returncode == 0 if error is None else result.returncode != 0 and error in text
        results.append({"case": case, "exit": result.returncode, "expected_error": error, "status": "Passed" if good else "Failed"})
        lib.write_json(evidence / "results.json", results)
        if not good:
            raise RuntimeError(f"{case}: expected diagnostic/exit not observed; see {evidence}")

    with tempfile.TemporaryDirectory(prefix="t1-local-", dir="/tmp/opencode") as temporary:
        existing = Path(temporary) / "existing-unit-tests"
        existing_env = lib.sandbox(repo, existing)
        execute("N1-existing-bats", ["bats", lib.BATS], existing, existing_env)
        for stream in streams:
            root = Path(temporary) / stream
            env = lib.sandbox(repo, root)
            env.update(STREAM=stream, FAILED_SOURCES="")
            expected = lib.sources(root, stream, env)
            if "manual" not in expected:
                raise RuntimeError("The planned manual-source fault is not configured")
            (root / "bundles").mkdir()
            healthy = lib.compress(b'{"fixture":true}\n')
            for source in expected:
                (root / "bundles" / f"{source}.json.zst").write_bytes(healthy)
            lib.write_json(evidence / f"{stream}-sources.json", expected)
            source = root / "bundles/manual.json.zst"
            content = lib.block(repo, "Validate source content")
            zip_check = lib.block(repo, "Validate ZIP entries")
            execute(f"N1-{stream}-script-control", ["bash", lib.VALIDATOR, str(source)], root, env)
            execute(f"N1-{stream}-block-control", ["bash", "-c", content], root, env)
            cases = {
                "missing": (None, "missing or empty source file"),
                "empty": (b"", "missing or empty source file"),
                "corrupt-zstd": (b"not a zstd frame", "invalid or unreadable vulnerability source"),
                "truncated-zstd": (healthy[:len(healthy) // 2], "invalid or unreadable vulnerability source"),
                "malformed-json": (lib.compress(b'{"broken":\n'), "invalid or unreadable vulnerability source"),
                "non-object": (lib.compress(b'42\n'), "invalid or unreadable vulnerability source"),
                "zero-objects": (lib.compress(b''), "vulnerability source contains zero records"),
            }
            for name, (payload, error) in cases.items():
                if payload is None:
                    source.unlink()
                else:
                    source.write_bytes(payload)
                if name in ("empty", "truncated-zstd"):
                    execute(f"N1-{stream}-{name}-script", ["bash", lib.VALIDATOR, str(source)], root, env, error=error)
                execute(f"N1-{stream}-{name}-block", ["bash", "-c", content], root, env, error="manual: invalid vulnerability source")
                source.write_bytes(healthy)
            for item in expected:
                (root / "bundles" / f"{item}.json.zst").write_bytes(b"")
            execute(f"N1-{stream}-all-empty", ["bash", "-c", content], root, env, error=f"{len(expected)} invalid source(s)")

            names = [f"bundles/{s}.json.zst" for s in expected]
            def fixture(entries):
                with warnings.catch_warnings():
                    warnings.simplefilter("ignore", UserWarning)
                    with zipfile.ZipFile(root / "vulnerabilities.zip", "w") as bundle:
                        for entry in entries:
                            info = zipfile.ZipInfo(entry, date_time=(2026, 1, 1, 0, 0, 0))
                            bundle.writestr(info, healthy)
                manifest = lib.archive_manifest(root / "vulnerabilities.zip")
                if [entry["name"] for entry in manifest] != entries:
                    raise RuntimeError("Fixture manifest does not match the intended mutation")
                return manifest

            manifest = fixture(names)
            lib.write_json(evidence / f"N2-{stream}-control-manifest.json", manifest)
            execute(f"N2-{stream}-control", ["bash", "-c", zip_check], root, env)
            for name, entries, error in (
                ("duplicate", names + [names[0]], f"duplicate ZIP entry: {expected[0]}"),
                ("unexpected", names + ["bundles/t1-negative-unexpected.json.zst"], "unexpected ZIP entry: t1-negative-unexpected"),
                ("missing", [n for n in names if n != "bundles/manual.json.zst"], "missing ZIP entry without prior exclusion: manual"),
            ):
                lib.write_json(evidence / f"N2-{stream}-{name}-manifest.json", fixture(entries))
                execute(f"N2-{stream}-{name}", ["bash", "-c", zip_check], root, env, error=error)
            (root / "vulnerabilities.zip").write_bytes(b"not a ZIP archive")
            execute(f"N2-{stream}-unreadable", ["bash", "-c", zip_check], root, env, error="End-of-central-directory signature not found")
    print(json.dumps(results, indent=2))


if __name__ == "__main__":
    main()
