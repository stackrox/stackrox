#!/usr/bin/env python3
"""Run real imports and the corrected checker for healthy/N3c/N4 test cases."""

import json
import os
from pathlib import Path
import signal
import subprocess
import sys

import lib

CORRECTED_CASES = {"healthy", "N3c", "N4-go-zero"}
DB_CONN = "host=postgres user=test password=test dbname=matcher sslmode=disable"


def psql_update_returned_exactly_one_key(output, key):
    """Require the one RETURNING row and its matching psql command tag."""
    lines = output.splitlines()
    return lines == [key, "UPDATE 1"]


def terminate_process(process):
    if process is not None and process.poll() is None:
        os.killpg(process.pid, signal.SIGTERM)
        try:
            process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            os.killpg(process.pid, signal.SIGKILL)
            process.wait(timeout=10)


def run_corrected_import(evidence):
    case = os.environ["T1_CASE"]
    stream = os.environ["STREAM"]
    if os.environ.get("T1_DRY_RUN") != "true" or os.environ.get("T1_FORCE") != "true":
        raise RuntimeError("Corrected Part C tests require force=true and dry_run=true")
    log = evidence / "real-import.log"
    validator_log = evidence / "validator.log"
    updater = Path("/tmp/updater")
    validator = Path("/tmp/t1-tests/scanner-updater-validate-import.py")
    run_manifest = json.loads((evidence / "run.json").read_text())
    if run_manifest.get("case") != case or run_manifest.get("stream") != stream:
        raise RuntimeError("Corrected Part C run manifest does not match its case/stream")
    result = {
        "case": case,
        "stream": stream,
        "part_c_checker": "scanner-updater-validate-import.py",
        "validator_sha256": lib.digest(validator),
        "updater_sha256": lib.digest(updater),
        "updater_exit": None,
        "import_exit": None,
        "checker_exit": None,
        "timed_out": False,
    }
    process = None
    try:
        with lib.serve(Path("/tmp")) as url, log.open("wb") as output:
            process = subprocess.Popen(
                [
                    str(updater), "import",
                    "--db-conn", DB_CONN,
                    "--vulns-url", f"{url}/vulnerabilities.zip",
                ],
                stdout=output,
                stderr=subprocess.STDOUT,
                start_new_session=True,
            )
            try:
                result["updater_exit"] = process.wait(timeout=3600)
            except subprocess.TimeoutExpired:
                result["timed_out"] = True
                terminate_process(process)
                result["updater_exit"] = process.returncode
        result["import_exit"] = result["updater_exit"]
        print(log.read_text(errors="replace"), end="")
        if result["updater_exit"] != 0:
            result["failure_stage"] = "updater-import"
            return result["updater_exit"]
        if result["timed_out"]:
            result["failure_stage"] = "updater-timeout"
            return 1

        if case == "N4-go-zero":
            key = "bundles/manual.json.zst"
            before = lib.sql(
                "SELECT update_timestamp FROM public.last_vuln_update "
                "WHERE key='bundles/manual.json.zst';"
            )
            changed = lib.sql(
                "UPDATE public.last_vuln_update "
                "SET update_timestamp=TIMESTAMPTZ '0001-01-01 00:00:00+00' "
                "WHERE key='bundles/manual.json.zst' RETURNING key;"
            )
            after = lib.sql(
                "SELECT update_timestamp FROM public.last_vuln_update "
                "WHERE key='bundles/manual.json.zst';"
            )
            changed_lines = changed.splitlines()
            if (
                not before
                or not psql_update_returned_exactly_one_key(changed, key)
                or not after.startswith("0001-01-01")
            ):
                raise RuntimeError("N4-go-zero metadata mutation did not affect exactly the manual row")
            mutation = {
                "case": case,
                "stream": stream,
                "key": key,
                "before_timestamp": before,
                "after_timestamp": after,
                "rows_updated": changed_lines,
            }
            lib.write_json(evidence / "injection.json", mutation)
            result["metadata_mutation"] = mutation

        sources = run_manifest.get("configured_sources")
        if not isinstance(sources, list) or not sources or len(sources) != len(set(sources)):
            raise RuntimeError("Corrected Part C run has no valid configured source manifest")
        result["expected_sources"] = sources
        validation = subprocess.run(
            [
                sys.executable,
                str(validator),
                "--log", str(log),
                "--db-conn", DB_CONN,
                *sources,
            ],
            capture_output=True,
            text=True,
            timeout=300,
            check=False,
        )
        result["checker_exit"] = validation.returncode
        validator_log.write_text(validation.stdout + validation.stderr)
        print(validation.stdout, end="")
        print(validation.stderr, end="", file=sys.stderr)
        result["database_counts"] = lib.database_counts()
        if validation.returncode:
            result["failure_stage"] = "post-import-validation"
        return validation.returncode
    except subprocess.TimeoutExpired:
        result["timed_out"] = True
        result["failure_stage"] = "post-import-validation-timeout"
        return 1
    finally:
        terminate_process(process)
        lib.write_json(evidence / "import-result.json", result)


def run_legacy_import(evidence):
    script = Path("/tmp/t1-tests/Validate-bundle-import.sh")
    log = evidence / "real-import.log"
    result = {"script_sha256": lib.digest(script), "import_exit": None, "timed_out": False}
    process = None
    try:
        with log.open("wb") as output:
            process = subprocess.Popen(
                ["bash", str(script)],
                stdout=output,
                stderr=subprocess.STDOUT,
                start_new_session=True,
            )
            result["import_exit"] = process.wait(timeout=3600)
        return result["import_exit"]
    except subprocess.TimeoutExpired:
        result["timed_out"] = True
        raise
    finally:
        terminate_process(process)
        lib.write_json(evidence / "import-result.json", result)
        if log.exists():
            print(log.read_text(errors="replace"), end="")


def main():
    evidence = Path(os.environ["T1_EVIDENCE"])
    if os.environ.get("T1_CASE") in CORRECTED_CASES:
        return run_corrected_import(evidence)
    return run_legacy_import(evidence)


if __name__ == "__main__":
    sys.exit(main())
