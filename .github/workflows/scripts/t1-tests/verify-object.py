#!/usr/bin/env python3
"""Check exact authenticated GCS status; 403 never means an absent artifact."""

import hashlib
import json
import os
from pathlib import Path
import subprocess
from urllib.error import HTTPError
from urllib.parse import quote, urlencode
from urllib.request import Request, urlopen

import lib


def main():
    evidence = Path(os.environ["T1_EVIDENCE"])
    run = json.loads((evidence / "run.json").read_text())
    case = run["case"]
    positive = case in ("healthy", "gate-control")
    outcomes = {key: os.environ.get(f"T1_{key.upper()}_OUTCOME", "") for key in ("content", "zip", "import", "upload_auth", "upload", "fault")}
    report = {"case": case, "stream": run["stream"], "outcomes": outcomes, "status": "Failed"}
    try:
        if positive:
            if any(outcomes[k] != "success" for k in ("content", "zip", "import", "upload_auth", "upload")):
                raise RuntimeError("The healthy pipeline did not reach a successful real upload")
        else:
            gate = {"W5": "zip", "W6": "content", "W7": "import", "W9": "import"}[case]
            if outcomes[gate] != "failure" or outcomes["upload"] != "skipped" or outcomes["upload_auth"] != "skipped":
                raise RuntimeError("The intended validation gate did not fail and block upload")
            if case != "W9":
                injection = json.loads((evidence / "injection.json").read_text())
                if outcomes["fault"] != "success" or injection["case"] != case:
                    raise RuntimeError("The intended fixture injection was not verified")
            if case == "W7":
                imported = json.loads((evidence / "import-result.json").read_text())
                text = (evidence / "real-import.log").read_text(errors="replace")
                if imported["import_exit"] != 1 or imported["timed_out"] or not lib.manual_json_error(text):
                    raise RuntimeError("W7 lacks a real malformed-JSON diagnostic attributable to the manual bundle")
        if case in ("W9", "gate-control"):
            middle = json.loads((evidence / "mid-import.json").read_text())
            if middle["status"] != "Passed" or middle["mode"] != ("fail" if case == "W9" else "release"):
                raise RuntimeError("Missing verified mid-import progress/control evidence")
            if case == "W9" and (not middle["fault_applied"] or middle["import_exit"] != 1):
                raise RuntimeError("W9 did not observe a real importer failure after its SQL fault")
        prefix = os.environ["GCS_PREFIX"]
        object_name = f"{prefix}/test-runs/{run['run_id']}/{run['run_attempt']}/{run['stream']}/vulnerabilities.zip"
        report["uri"] = "gs://scanner-v4-test/" + object_name
        token = subprocess.check_output(["gcloud", "auth", "print-access-token"], text=True).strip()
        if not token:
            raise RuntimeError("No test-account access token")
        url = "https://storage.googleapis.com/storage/v1/b/scanner-v4-test/o/" + quote(object_name, safe="")
        request = Request(url, headers={"Authorization": "Bearer " + token})
        try:
            with urlopen(request, timeout=60) as response:
                metadata = json.load(response)
                report["http_status"] = response.status
        except HTTPError as error:
            report["http_status"] = error.code
            report["http_error_body"] = error.read(65536).decode(errors="replace")
            if positive or error.code != 404:
                raise RuntimeError(f"Object verification returned HTTP {error.code}, not the required result") from error
            metadata = None
        if positive:
            candidate = json.loads((evidence / "candidate.json").read_text())
            if metadata is None or int(metadata["size"]) != candidate["size"]:
                raise RuntimeError("Uploaded object size does not match the candidate")
            if metadata.get("md5Hash") and metadata["md5Hash"] != candidate["md5_base64"]:
                raise RuntimeError("Uploaded object MD5 does not match the candidate")
            readback = Request(url + "?" + urlencode({"alt": "media", "generation": metadata["generation"]}), headers={"Authorization": "Bearer " + token})
            h = hashlib.sha256()
            size = 0
            with urlopen(readback, timeout=120) as response:
                if response.status != 200:
                    raise RuntimeError("Generation-matched read-back did not return HTTP 200")
                for chunk in iter(lambda: response.read(1024 * 1024), b""):
                    size += len(chunk)
                    h.update(chunk)
            if h.hexdigest() != candidate["sha256"] or size != candidate["size"]:
                raise RuntimeError("Generation-matched read-back differs from the validated candidate")
            report.update(metadata=metadata, readback_sha256=h.hexdigest(), readback_size=size)
        elif metadata is not None:
            raise RuntimeError("Negative run unexpectedly created a test object")
        report["status"] = "Passed"
    except Exception as error:
        report["error"] = str(error)
        raise
    finally:
        lib.write_json(evidence / "object-check.json", report)
    print(json.dumps(report, indent=2))


if __name__ == "__main__":
    main()
