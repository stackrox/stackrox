#!/usr/bin/env python3
"""
Bundle management script for vulnerability updates:
- Upload overwrites previous bundles (failed updaters don't produce files, so old ones persist)
- Aggregate downloads all bundles from GCS, creates final zip, and uploads it

GCS structure:
    gs://bucket/v2/bundles/
    ├── alpine.json.zst
    ├── nvd.json.zst
    ├── photon.json.zst
    └── status.json

Usage:
    python scanner-versioned-definitions-aggregate.py upload --local-dir ./bundles --version v2
    python scanner-versioned-definitions-aggregate.py aggregate --version v2 --output-dir ./out
    python scanner-versioned-definitions-aggregate.py aggregate --version v2 --output-dir ./out --dry-run

Local testing (no GCS):
    python scanner-versioned-definitions-aggregate.py upload --local-dir ./bundles --version v2 --backend local --bucket /tmp/test-bucket
    python scanner-versioned-definitions-aggregate.py aggregate --version v2 --output-dir ./out --backend local --bucket /tmp/test-bucket
"""

import argparse
import glob as globmod
import json
import shutil
import subprocess
import sys
import tempfile
import zipfile
from datetime import datetime, timezone
from pathlib import Path


DEFAULT_BUCKET = "gs://definitions.stackrox.io/v4/vulnerability-bundles"


class GCSBackend:
    """Storage backend using gsutil for Google Cloud Storage."""

    def copy(self, src, dest) -> bool:
        result = subprocess.run(
            ["gsutil", "cp", str(src), str(dest)],
            capture_output=True, text=True,
        )
        return result.returncode == 0

    def copy_many(self, srcs, dest) -> bool:
        result = subprocess.run(
            ["gsutil", "-m", "cp", *[str(s) for s in srcs], str(dest)],
            capture_output=True, text=True,
        )
        return result.returncode == 0


class LocalBackend:
    """Storage backend using local filesystem (for testing)."""

    def copy(self, src, dest) -> bool:
        src, dest = Path(str(src)), Path(str(dest))
        if not src.exists():
            return False
        if dest.is_dir() or str(dest).endswith("/"):
            dest = Path(str(dest).rstrip("/"))
            dest.mkdir(parents=True, exist_ok=True)
            dest = dest / src.name
        else:
            dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(src, dest)
        return True

    def copy_many(self, srcs, dest) -> bool:
        dest = Path(str(dest).rstrip("/"))
        dest.mkdir(parents=True, exist_ok=True)
        for src in srcs:
            src_str = str(src)
            if "*" in src_str:
                for f in globmod.glob(src_str):
                    shutil.copy2(f, dest / Path(f).name)
            else:
                shutil.copy2(src_str, dest / Path(src_str).name)
        return True


def download_previous_status(backend, bucket: str, version: str) -> dict | None:
    """Download previous status.json, return None if not found."""
    status_path = f"{bucket}/{version}/bundles/status.json"

    with tempfile.NamedTemporaryFile(mode='w+', suffix='.json', delete=False) as tmp:
        if not backend.copy(status_path, tmp.name):
            return None
        try:
            return json.load(open(tmp.name))
        except (json.JSONDecodeError, FileNotFoundError):
            return None


def enrich_status_with_timestamps(status_file: Path, backend, bucket: str, version: str):
    """Add last_successful_update timestamps to status.json."""
    with open(status_file) as f:
        status = json.load(f)

    previous = download_previous_status(backend, bucket, version)
    previous_updaters = {u["name"]: u for u in previous.get("updaters", [])} if previous else {}

    now = datetime.now(timezone.utc).isoformat()
    for updater in status.get("updaters", []):
        if updater["status"] == "success":
            updater["last_successful_update"] = now
        else:
            prev = previous_updaters.get(updater["name"])
            if prev and "last_successful_update" in prev:
                updater["last_successful_update"] = prev["last_successful_update"]

    with open(status_file, "w") as f:
        json.dump(status, f, indent=2)


def cmd_upload(args: argparse.Namespace) -> int:
    """Upload individual bundles (overwrites existing)."""
    local_dir = Path(args.local_dir)
    bucket = args.bucket
    backend = LocalBackend() if args.backend == "local" else GCSBackend()
    dest = f"{bucket}/{args.version}/bundles"

    if not local_dir.exists():
        print(f"Error: directory does not exist: {local_dir}")
        return 1

    bundles = list(local_dir.glob("*.json.zst"))
    if not bundles:
        print(f"No bundles found in {local_dir}")
        return 1

    print(f"Uploading {len(bundles)} bundles to {dest}/")
    if not backend.copy_many(bundles, f"{dest}/"):
        print("Error uploading bundles")
        return 1

    status_file = local_dir / "status.json"
    if status_file.exists():
        print("Enriching status.json with timestamps")
        enrich_status_with_timestamps(status_file, backend, bucket, args.version)

        print("Uploading status.json")
        if not backend.copy(status_file, f"{dest}/"):
            print("Warning: failed to upload status.json")

    print("Done")
    return 0


def cmd_aggregate(args: argparse.Namespace) -> int:
    """Download all bundles, create final zip, and upload."""
    version = args.version
    output_dir = Path(args.output_dir)
    bucket = args.bucket
    backend = LocalBackend() if args.backend == "local" else GCSBackend()
    bundles_path = f"{bucket}/{version}/bundles"
    dest = f"{bucket}/{version}"

    output_dir.mkdir(parents=True, exist_ok=True)
    output_zip = output_dir / "vulnerabilities.zip"

    print(f"Downloading bundles from {bundles_path}/")

    with tempfile.TemporaryDirectory() as tmpdir:
        tmppath = Path(tmpdir)

        if not backend.copy_many([f"{bundles_path}/*.json.zst"], f"{tmppath}/"):
            print("Error downloading bundles")
            return 1

        bundles = list(tmppath.glob("*.json.zst"))
        if not bundles:
            print("No bundles found")
            return 1

        # Download status.json if available
        has_status = backend.copy(f"{bundles_path}/status.json", f"{tmppath}/status.json")

        print(f"Creating {output_zip} with {len(bundles)} bundles")
        with zipfile.ZipFile(output_zip, "w", zipfile.ZIP_DEFLATED) as zf:
            for bundle in bundles:
                zf.write(bundle, bundle.name)
            if has_status:
                zf.write(tmppath / "status.json", "status.json")

    if args.dry_run:
        print(f"Dry run: would upload {output_zip} to {dest}/")
        return 0

    print(f"Uploading {output_zip} to {dest}/")
    if not backend.copy(output_zip, f"{dest}/"):
        print("Error uploading")
        return 1

    print("Done")
    return 0


def main():
    parser = argparse.ArgumentParser(description="Manage vulnerability bundles")
    parser.add_argument("--bucket", default=DEFAULT_BUCKET, help="Storage bucket path")
    parser.add_argument("--backend", choices=["gcs", "local"], default="gcs",
                        help="Storage backend (default: gcs)")
    subparsers = parser.add_subparsers(dest="command", required=True)

    upload_parser = subparsers.add_parser("upload", help="Upload individual bundles")
    upload_parser.add_argument("--local-dir", required=True, help="Directory with bundles")
    upload_parser.add_argument("--version", required=True, help="Version stream (e.g., v2)")

    agg_parser = subparsers.add_parser("aggregate", help="Download bundles, create zip, upload")
    agg_parser.add_argument("--version", required=True, help="Version stream (e.g., v2)")
    agg_parser.add_argument("--output-dir", required=True, help="Local directory for output zip")
    agg_parser.add_argument("--dry-run", action="store_true", help="Create zip but don't upload")

    args = parser.parse_args()

    if args.command == "upload":
        return cmd_upload(args)
    elif args.command == "aggregate":
        return cmd_aggregate(args)


if __name__ == "__main__":
    sys.exit(main())
