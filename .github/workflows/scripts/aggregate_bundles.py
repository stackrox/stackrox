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
    python aggregate_bundles.py upload --local-dir ./bundles --version v2
    python aggregate_bundles.py aggregate --version v2 --output-dir ./out
    python aggregate_bundles.py aggregate --version v2 --output-dir ./out --dry-run
"""

import argparse
import subprocess
import sys
import tempfile
import zipfile
from pathlib import Path


GCS_BUCKET = "gs://definitions.stackrox.io/v4/vulnerability-bundles"


def gsutil_copy(src, dest) -> subprocess.CompletedProcess:
    """Copy a single file to/from GCS."""
    return subprocess.run(["gsutil", "cp", src, dest], capture_output=True, text=True)


def gsutil_copy_parallel(srcs: list, dest) -> subprocess.CompletedProcess:
    """Copy multiple files to/from GCS using parallel threads."""
    return subprocess.run(["gsutil", "-m", "cp", *srcs, dest], capture_output=True, text=True)


def cmd_upload(args: argparse.Namespace) -> int:
    """Upload individual bundles to GCS (overwrites existing)."""
    local_dir = Path(args.local_dir)
    gcs_dest = f"{GCS_BUCKET}/{args.version}/bundles"

    if not local_dir.exists():
        print(f"Error: directory does not exist: {local_dir}")
        return 1

    bundles = list(local_dir.glob("*.json.zst"))
    if not bundles:
        print(f"No bundles found in {local_dir}")
        return 1

    print(f"Uploading {len(bundles)} bundles to {gcs_dest}/")
    result = gsutil_copy_parallel(bundles, f"{gcs_dest}/")
    if result.returncode != 0:
        print(f"Error: {result.stderr}")
        return 1

    status_file = local_dir / "status.json"
    if status_file.exists():
        print("Uploading status.json")
        result = gsutil_copy(status_file, f"{gcs_dest}/")
        if result.returncode != 0:
            print(f"Warning: failed to upload status.json: {result.stderr}")

    print("Done")
    return 0


def cmd_aggregate(args: argparse.Namespace) -> int:
    """Download all bundles from GCS, create final zip, and upload."""
    version = args.version
    output_dir = Path(args.output_dir)
    gcs_bundles = f"{GCS_BUCKET}/{version}/bundles"
    gcs_dest = f"{GCS_BUCKET}/{version}"

    output_dir.mkdir(parents=True, exist_ok=True)
    output_zip = output_dir / "vulnerabilities.zip"

    print(f"Downloading bundles from {gcs_bundles}/")

    with tempfile.TemporaryDirectory() as tmpdir:
        tmppath = Path(tmpdir)

        result = gsutil_copy_parallel([f"{gcs_bundles}/*.json.zst"], f"{tmppath}/")
        if result.returncode != 0:
            print(f"Error downloading bundles: {result.stderr}")
            return 1

        bundles = list(tmppath.glob("*.json.zst"))
        if not bundles:
            print("No bundles found in GCS")
            return 1

        print(f"Creating {output_zip} with {len(bundles)} bundles")
        with zipfile.ZipFile(output_zip, "w", zipfile.ZIP_DEFLATED) as zf:
            for bundle in bundles:
                zf.write(bundle, bundle.name)

    if args.dry_run:
        print(f"Dry run: would upload {output_zip} to {gcs_dest}/")
        return 0

    print(f"Uploading {output_zip} to {gcs_dest}/")
    result = gsutil_copy(output_zip, f"{gcs_dest}/")
    if result.returncode != 0:
        print(f"Error uploading: {result.stderr}")
        return 1

    print("Done")
    return 0


def main():
    parser = argparse.ArgumentParser(description="Manage vulnerability bundles in GCS")
    subparsers = parser.add_subparsers(dest="command", required=True)

    upload_parser = subparsers.add_parser("upload", help="Upload individual bundles to GCS")
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
