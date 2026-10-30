#!/usr/bin/env python3
"""Runner-local W5/W6/W7 faults; never modify a GCS object."""

import argparse
import json
import os
from pathlib import Path
import zipfile

import lib


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("case", choices=("W5", "W6", "W7"))
    args = parser.parse_args()
    if os.environ.get("T1_DRY_RUN") != "true" or os.environ.get("T1_FORCE") != "true":
        raise RuntimeError("Fault injection requires force=true and dry_run=true")
    evidence = Path(os.environ["T1_EVIDENCE"])
    record = {"case": args.case, "stream": os.environ["STREAM"]}
    if args.case == "W6":
        source = Path("bundles/manual.json.zst")
        if not source.is_file() or source.stat().st_size == 0:
            raise RuntimeError("W6 requires a successfully downloaded nonempty manual source")
        record["before_sha256"] = lib.digest(source)
        record["before_size"] = source.stat().st_size
        source.write_bytes(b"")
        if source.stat().st_size != 0:
            raise RuntimeError("W6 truncation failed")
        record["after_size"] = 0
    elif args.case == "W5":
        path = Path("vulnerabilities.zip")
        before = lib.archive_manifest(path)
        name = "bundles/t1-negative-unexpected.json.zst"
        if any(e["name"] == name for e in before):
            raise RuntimeError("W5 fault already present")
        with zipfile.ZipFile(path, "a") as bundle:
            bundle.writestr(name, b"controlled runner-local fault")
        after = lib.archive_manifest(path)
        if after[:-1] != before or after[-1]["name"] != name:
            raise RuntimeError("W5 changed more than the intended entry")
        record.update(before=before, after=after)
    else:
        payload = b'{"intentionally-malformed":\n'
        try:
            json.loads(payload)
        except json.JSONDecodeError:
            pass
        else:
            raise RuntimeError("W7 fixture unexpectedly contains valid JSON")
        compressed = lib.compress(payload)
        if lib.command(["zstd", "-d", "-q", "-c"], input=compressed) != payload:
            raise RuntimeError("W7 zstd round-trip failed")
        record.update(lib.replace_entry("vulnerabilities.zip", "bundles/manual.json.zst", compressed))
    lib.write_json(evidence / "injection.json", record)
    print(f"T1 {args.case}: intended runner-local fault verified")


if __name__ == "__main__":
    main()
