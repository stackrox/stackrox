#!/usr/bin/env python3
"""SQL-only AFTER INSERT gate: release control or sustained mid-import outage."""

import argparse
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import sys
import tempfile
import time
import zipfile

import lib


CONNECTION = "host=postgres user=test password=test dbname=matcher sslmode=disable"


def stop(process):
    if process is not None and process.poll() is None:
        os.killpg(process.pid, signal.SIGTERM)
        try:
            process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            os.killpg(process.pid, signal.SIGKILL)
            process.wait(timeout=10)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--mode", choices=("release", "fail"), required=True)
    parser.add_argument("--script", type=Path, required=True)
    parser.add_argument("--evidence", type=Path, required=True)
    parser.add_argument("--timeout", type=int, default=3600)
    parser.add_argument("--progress-timeout", type=int, default=900)
    args = parser.parse_args()
    lib.isolated_database()
    evidence = args.evidence.resolve()
    evidence.mkdir(parents=True, exist_ok=True)
    result = {"mode": args.mode, "fault_applied": False, "status": "Failed"}
    child = gate = None
    disabled = False
    log = evidence / "real-import.log"
    try:
        if lib.sql("SELECT to_regclass('public.vuln') IS NULL;") != "t":
            raise RuntimeError("Gate setup requires a fresh, isolated matcher database")
        with tempfile.TemporaryDirectory(prefix="t1-bootstrap-", dir="/tmp/opencode") as directory:
            with zipfile.ZipFile(Path(directory) / "empty.zip", "w"):
                pass
            with lib.serve(directory) as url:
                output = lib.command([
                    "/tmp/updater", "import", "--db-conn", CONNECTION,
                    "--vulns-url", url + "/empty.zip",
                ], timeout=300)
                (evidence / "schema-bootstrap.log").write_bytes(output)
        counts = lib.database_counts()
        if any(counts.values()):
            raise RuntimeError("Schema initialization imported unexpected data")
        result["before_counts"] = counts
        lib.sql("""
            CREATE FUNCTION public.t1_after_insert_gate() RETURNS trigger LANGUAGE plpgsql AS $$
            BEGIN
              PERFORM pg_advisory_xact_lock(36961, 1);
              RETURN NEW;
            END $$;
            CREATE TRIGGER t1_after_insert_gate AFTER INSERT ON public.vuln
              FOR EACH ROW EXECUTE FUNCTION public.t1_after_insert_gate();
        """)
        gate_env = os.environ.copy()
        gate_env.update(PGPASSWORD="test", PGAPPNAME="t1-gate", PGCONNECT_TIMEOUT="5")
        with (evidence / "gate.log").open("wb") as gate_log:
            gate = subprocess.Popen([
                "psql", "-X", "-h", "postgres", "-U", "test", "-d", "matcher",
                "-v", "ON_ERROR_STOP=1", "-At", "-c",
                "SELECT pg_advisory_lock(36961, 1); SELECT pg_sleep(7200);",
            ], env=gate_env, stdout=gate_log, stderr=subprocess.STDOUT, start_new_session=True)
            deadline = time.monotonic() + 30
            gate_pid = ""
            while time.monotonic() < deadline:
                if gate.poll() is not None:
                    raise RuntimeError("Gate connection exited before acquiring its lock")
                gate_pid = lib.sql("""
                    SELECT a.pid FROM pg_stat_activity a JOIN pg_locks l ON l.pid=a.pid
                    WHERE a.datname='matcher' AND a.application_name='t1-gate'
                      AND l.locktype='advisory' AND l.classid=36961 AND l.objid=1
                      AND l.objsubid=2 AND l.granted;
                """)
                if gate_pid:
                    break
                time.sleep(0.2)
            if not gate_pid.isdigit():
                raise RuntimeError("Gate lock was not observed within its setup deadline")
            result["gate_pid"] = int(gate_pid)
            with log.open("wb") as output:
                child = subprocess.Popen(
                    ["bash", str(args.script.resolve())], stdout=output,
                    stderr=subprocess.STDOUT, start_new_session=True,
                )
                started = time.monotonic()
                deadline = started + args.progress_timeout
                writer = None
                while time.monotonic() < deadline:
                    if child.poll() is not None:
                        raise RuntimeError("Importer exited before an AFTER INSERT wait was observed")
                    observed = lib.sql(f"""
                        SELECT row_to_json(s) FROM (
                          SELECT a.pid, a.application_name, a.backend_xid::text,
                                 a.wait_event_type, a.wait_event, a.query,
                                 pg_blocking_pids(a.pid) AS blockers,
                                 clock_timestamp() AS observed_at
                          FROM pg_stat_activity a JOIN pg_locks l ON l.pid=a.pid
                          WHERE a.datname='matcher' AND a.usename='test'
                            AND a.application_name NOT LIKE 't1-%'
                            AND a.backend_xid IS NOT NULL AND NOT l.granted
                            AND l.locktype='advisory' AND l.classid=36961
                            AND l.objid=1 AND l.objsubid=2
                            AND {int(gate_pid)}=ANY(pg_blocking_pids(a.pid))
                          LIMIT 1
                        ) s;
                    """)
                    if observed:
                        writer = json.loads(observed)
                        break
                    time.sleep(0.2)
                if writer is None or child.poll() is not None:
                    raise RuntimeError("No live importer AFTER INSERT progress proof before deadline")
                result["writer"] = writer
                result["progress_proof"] = "Importer blocked inside the test-owned AFTER INSERT trigger; row write started but is not claimed committed"
                lib.write_json(evidence / "mid-import.json", result)
                if args.mode == "fail":
                    lib.sql("ALTER DATABASE matcher ALLOW_CONNECTIONS false;", database="postgres")
                    disabled = True
                    if lib.sql("SELECT datallowconn FROM pg_database WHERE datname='matcher';", database="postgres") != "f":
                        raise RuntimeError("Reconnect prohibition was not committed")
                    killed = json.loads(lib.sql("""
                        SELECT coalesce(json_agg(s), '[]'::json) FROM (
                          SELECT pid, application_name, pg_terminate_backend(pid, 5000) AS terminated
                          FROM pg_stat_activity
                          WHERE datname='matcher' AND usename='test'
                            AND backend_type='client backend'
                            AND application_name NOT LIKE 't1-%'
                        ) s;
                    """, database="postgres"))
                    if not any(row["pid"] == writer["pid"] and row["terminated"] for row in killed):
                        raise RuntimeError("The captured importer writer was not terminated")
                    result.update(fault_applied=True, terminated=killed)
                else:
                    if lib.sql(f"SELECT pg_terminate_backend({int(gate_pid)}, 5000);", database="postgres") != "t":
                        raise RuntimeError("Healthy control could not release the gate")
                lib.write_json(evidence / "mid-import.json", result)
                remaining = args.timeout - (time.monotonic() - started)
                if remaining <= 0:
                    raise RuntimeError("Import deadline expired before fault/control observation")
                returncode = child.wait(timeout=remaining)
                result["import_exit"] = returncode
            text = log.read_text(errors="replace")
            print(text, end="")
            if args.mode == "fail":
                if returncode != 1 or not re.search(r"database|connection|SQLSTATE|context canceled|conn closed", text, re.IGNORECASE):
                    raise RuntimeError("The real importer did not report the expected database failure")
            else:
                if returncode:
                    raise RuntimeError("Healthy gated import failed")
                result["after_counts"] = lib.database_counts()
                if result["after_counts"]["vulnerability_links"] <= 0:
                    raise RuntimeError("Healthy gated control imported no linked vulnerabilities")
            result["status"] = "Passed"
            lib.write_json(evidence / "mid-import.json", result)
            return returncode
    except Exception as error:
        result["error"] = str(error)
        lib.write_json(evidence / "mid-import.json", result)
        if log.exists():
            print(log.read_text(errors="replace"), end="")
        raise
    finally:
        stop(child)
        if disabled:
            lib.sql("ALTER DATABASE matcher ALLOW_CONNECTIONS true;", database="postgres")
        if gate is not None:
            stop(gate)
        if lib.sql("SELECT to_regclass('public.vuln') IS NOT NULL;") == "t":
            lib.sql("DROP TRIGGER IF EXISTS t1_after_insert_gate ON public.vuln; DROP FUNCTION IF EXISTS public.t1_after_insert_gate();")


if __name__ == "__main__":
    sys.exit(main())
