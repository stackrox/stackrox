#!/usr/bin/env python3
"""One N3/N4 case inside an isolated CI container and fresh PostgreSQL service."""

import argparse
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import time

import lib


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, required=True)
    parser.add_argument("--case", required=True)
    parser.add_argument("--stream", required=True)
    parser.add_argument("--timeout", type=int, default=3600)
    args = parser.parse_args()
    lib.isolated_database()
    root = Path("/work")
    evidence = root / "evidence"
    evidence.mkdir(exist_ok=False)
    record = {"case": args.case, "stream": args.stream, "status": "Failed"}
    env = lib.sandbox(args.repo, root)
    env.update(STREAM=args.stream, FAILED_SOURCES="")
    os.chdir(root)
    Path("/tmp/opencode").mkdir(exist_ok=True)
    lib.extract(args.repo, root / "original-blocks")
    script = root / "original-blocks/Validate-bundle-import.sh"
    checker = root / "original-blocks/metadata-check.sh"
    shutil.copy2("/inputs/vulnerabilities.zip", root / "vulnerabilities.zip")

    def run(path, label, timeout=None):
        completed = subprocess.run(["bash", str(path)], env=env, capture_output=True, text=True, timeout=timeout or args.timeout)
        text = completed.stdout + completed.stderr
        (evidence / f"{label}.log").write_text(text)
        return completed.returncode, text

    def healthy():
        code, text = run(script, "real-import")
        if code:
            raise RuntimeError(f"Healthy import failed ({code}): {text[-4000:]}")
        counts = lib.database_counts()
        if counts["vulnerability_links"] <= 0:
            raise RuntimeError("Healthy fixture imported no linked vulnerabilities")
        record["counts"] = counts

    try:
        deadline = time.monotonic() + 90
        while True:
            try:
                if lib.sql("SELECT 1;") == "1":
                    break
            except RuntimeError:
                if time.monotonic() >= deadline:
                    raise
                time.sleep(1)
        expected = [f"bundles/{s}.json.zst" for s in lib.sources(root, args.stream, env)]
        manifest = lib.archive_manifest(root / "vulnerabilities.zip")
        if sorted(e["name"] for e in manifest) != sorted(expected):
            raise RuntimeError("Positive bundle does not match the real configured stream")
        record["input_sha256"] = lib.digest(root / "vulnerabilities.zip")
        if args.case in ("control", "manual-control"):
            if args.case == "manual-control":
                record["mutation"] = lib.replace_entry("vulnerabilities.zip", "bundles/manual.json.zst", lib.compress(lib.manual_record()))
            healthy()
            if args.case == "manual-control":
                linked = lib.sql("SELECT count(x.vuln) FROM public.update_operation o LEFT JOIN public.uo_vuln x ON x.uo=o.id WHERE o.fingerprint='t1-offline-manual-v1';")
                if linked != "1":
                    raise RuntimeError("The hand-built semantic manual control did not import exactly one linked record")
        elif args.case == "N3a":
            record["mutation"] = lib.replace_entry("vulnerabilities.zip", "bundles/manual.json.zst", lib.compress(b'{"malformed":\n'))
            code, text = run(script, "real-import")
            record["import_exit"] = code
            if code != 1 or not lib.manual_json_error(text):
                raise RuntimeError("Malformed data did not produce the intended real importer diagnostic")
        elif args.case == "N3b":
            lib.sql("ALTER DATABASE matcher ALLOW_CONNECTIONS false;", database="postgres")
            try:
                code, text = run(script, "real-import", max(args.timeout, 900))
            finally:
                lib.sql("ALTER DATABASE matcher ALLOW_CONNECTIONS true;", database="postgres")
            record["import_exit"] = code
            if code != 1 or not re.search(r"database|connection|SQLSTATE", text, re.IGNORECASE):
                raise RuntimeError("Unavailable database did not produce a real connection failure")
        elif args.case == "N3c":
            record["mutation"] = lib.replace_entry("vulnerabilities.zip", "bundles/manual.json.zst", lib.compress(lib.manual_record(zero=True)))
            code, text = run(script, "real-import")
            linked = int(lib.sql("SELECT count(x.vuln) FROM public.update_operation o LEFT JOIN public.uo_vuln x ON x.uo=o.id WHERE o.fingerprint='t1-zero-supported-records';"))
            record.update(import_exit=code, zero_source_linked_records=linked)
            record["source_timestamp"] = lib.sql("SELECT update_timestamp FROM public.last_vuln_update WHERE key='bundles/manual.json.zst';")
            if code == 0 and linked == 0:
                raise RuntimeError("DETECTION GAP: actual import/check block accepted a source with zero supported imported records")
            if code != 1 or linked != 0 or not re.search(r"zero.*record|no.*record|non.?zero", text, re.IGNORECASE):
                raise RuntimeError("No source-specific zero-import rejection was demonstrated")
        elif args.case in ("N3d-control", "N3d"):
            mode = "release" if args.case == "N3d-control" else "fail"
            completed = subprocess.run([
                sys.executable, str(Path(__file__).with_name("mid-import.py")),
                "--mode", mode, "--script", str(script), "--evidence", str(evidence),
                "--timeout", str(args.timeout),
            ], env=env, capture_output=True, text=True, timeout=args.timeout + 400)
            (evidence / "supervisor.log").write_text(completed.stdout + completed.stderr)
            observed = json.loads((evidence / "mid-import.json").read_text())
            record["mid_import"] = observed
            expected_code = 0 if mode == "release" else 1
            if completed.returncode != expected_code or observed["status"] != "Passed" or observed.get("import_exit") != expected_code:
                raise RuntimeError("SQL-gated import did not meet its progress/control/failure requirements")
        elif args.case in ("N4-missing", "N4-epoch", "N4-go-zero"):
            healthy()
            record["before_timestamp"] = lib.sql("SELECT update_timestamp FROM public.last_vuln_update WHERE key='bundles/manual.json.zst';")
            if not record["before_timestamp"]:
                raise RuntimeError("Metadata control lacks the expected manual row")
            if args.case == "N4-missing":
                lib.sql("DELETE FROM public.last_vuln_update WHERE key='bundles/manual.json.zst';")
            else:
                value = "1970-01-01 00:00:00" if args.case == "N4-epoch" else "0001-01-01 00:00:00"
                lib.sql(f"UPDATE public.last_vuln_update SET update_timestamp=TIMESTAMP '{value}' WHERE key='bundles/manual.json.zst';")
            record["after_timestamp"] = lib.sql("SELECT update_timestamp FROM public.last_vuln_update WHERE key='bundles/manual.json.zst';")
            code, text = run(checker, "real-metadata-check")
            record["checker_exit"] = code
            if code == 0 and args.case == "N4-go-zero":
                raise RuntimeError("DETECTION GAP: actual metadata checker accepted Go's year-0001 zero timestamp")
            if code != 1 or "manual: missing or zero timestamp" not in text:
                raise RuntimeError("The real checker did not reject the intended metadata mutation")
        else:
            raise RuntimeError(f"Unknown case: {args.case}")
        record["status"] = "Passed"
    except Exception as error:
        record["error"] = str(error)
    lib.write_json(evidence / "result.json", record)
    print(json.dumps(record, indent=2))
    return 0 if record["status"] == "Passed" else 1


if __name__ == "__main__":
    sys.exit(main())
