#!/usr/bin/env python3
"""Integration tests for scanner-versioned-definitions-aggregate.py."""

import json
import subprocess
import tempfile
import unittest
import zipfile
from pathlib import Path

SCRIPT = Path(__file__).parent / "scanner-versioned-definitions-aggregate.py"


def create_fake_bundle(path):
    """Create a minimal valid .json.zst file."""
    subprocess.run(
        ["zstd", "-o", str(path)],
        input=b'{"test":"data"}',
        stdout=subprocess.PIPE, stderr=subprocess.PIPE,
        check=True,
    )


def create_status_json(path, updaters):
    """Create a status.json file."""
    with open(path, "w") as f:
        json.dump({"updaters": updaters}, f)


def run_script(*args):
    return subprocess.run(
        ["python3", str(SCRIPT), *args],
        stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
    )


class TestUpload(unittest.TestCase):
    def setUp(self):
        self.tmpdir = tempfile.TemporaryDirectory()
        self.root = Path(self.tmpdir.name)
        self.bundles = self.root / "bundles"
        self.bundles.mkdir()
        self.bucket = self.root / "bucket"
        self.bucket.mkdir()

        create_fake_bundle(self.bundles / "alpine.json.zst")
        create_fake_bundle(self.bundles / "nvd.json.zst")
        create_fake_bundle(self.bundles / "photon.json.zst")

        create_status_json(self.bundles / "status.json", [
            {"name": "alpine", "status": "success", "last_attempt": "2026-04-21T00:00:00Z"},
            {"name": "nvd", "status": "success", "last_attempt": "2026-04-21T00:00:00Z"},
            {"name": "photon", "status": "failed", "error": "404", "last_attempt": "2026-04-21T00:00:00Z"},
        ])

    def tearDown(self):
        self.tmpdir.cleanup()

    def _upload(self):
        return run_script(
            "--backend", "local", "--bucket", str(self.bucket),
            "upload", "--local-dir", str(self.bundles), "--version", "dev",
        )

    def test_uploads_bundles_and_status(self):
        result = self._upload()
        self.assertEqual(result.returncode, 0)

        dest = self.bucket / "dev" / "bundles"
        self.assertTrue((dest / "alpine.json.zst").exists())
        self.assertTrue((dest / "nvd.json.zst").exists())
        self.assertTrue((dest / "photon.json.zst").exists())
        self.assertTrue((dest / "status.json").exists())

    def test_enriches_status_with_timestamps(self):
        self._upload()

        with open(self.bucket / "dev" / "bundles" / "status.json") as f:
            status = json.load(f)

        for u in status["updaters"]:
            if u["status"] == "success":
                self.assertIn("last_successful_update", u)
            else:
                self.assertNotIn("last_successful_update", u)

    def test_preserves_previous_timestamps_on_failure(self):
        self._upload()

        with open(self.bucket / "dev" / "bundles" / "status.json") as f:
            first_status = json.load(f)
        alpine_ts = next(u for u in first_status["updaters"] if u["name"] == "alpine")["last_successful_update"]

        # Second upload: alpine fails (Go code would delete the bundle)
        (self.bundles / "alpine.json.zst").unlink()
        create_status_json(self.bundles / "status.json", [
            {"name": "alpine", "status": "failed", "error": "timeout", "last_attempt": "2026-04-22T00:00:00Z"},
            {"name": "nvd", "status": "success", "last_attempt": "2026-04-22T00:00:00Z"},
            {"name": "photon", "status": "success", "last_attempt": "2026-04-22T00:00:00Z"},
        ])
        self._upload()

        with open(self.bucket / "dev" / "bundles" / "status.json") as f:
            second_status = json.load(f)

        alpine = next(u for u in second_status["updaters"] if u["name"] == "alpine")
        self.assertEqual(alpine["last_successful_update"], alpine_ts)

        # The old alpine bundle should still be in the bucket (last known good)
        # but it should NOT have been overwritten by the second upload
        dest = self.bucket / "dev" / "bundles"
        self.assertTrue((dest / "alpine.json.zst").exists(), "last-known-good bundle should persist")
        self.assertTrue((dest / "nvd.json.zst").exists())
        self.assertTrue((dest / "photon.json.zst").exists())

    def test_missing_dir_fails(self):
        result = run_script(
            "--backend", "local", "--bucket", str(self.bucket),
            "upload", "--local-dir", "/nonexistent", "--version", "dev",
        )
        self.assertEqual(result.returncode, 1)

    def test_empty_dir_fails(self):
        empty = self.root / "empty"
        empty.mkdir()
        result = run_script(
            "--backend", "local", "--bucket", str(self.bucket),
            "upload", "--local-dir", str(empty), "--version", "dev",
        )
        self.assertEqual(result.returncode, 1)


class TestAggregate(unittest.TestCase):
    def setUp(self):
        self.tmpdir = tempfile.TemporaryDirectory()
        self.root = Path(self.tmpdir.name)
        self.bundles = self.root / "bundles"
        self.bundles.mkdir()
        self.bucket = self.root / "bucket"
        self.bucket.mkdir()
        self.output = self.root / "output"
        self.output.mkdir()

        create_fake_bundle(self.bundles / "alpine.json.zst")
        create_fake_bundle(self.bundles / "nvd.json.zst")
        create_fake_bundle(self.bundles / "photon.json.zst")

        create_status_json(self.bundles / "status.json", [
            {"name": "alpine", "status": "success", "last_attempt": "2026-04-21T00:00:00Z"},
            {"name": "nvd", "status": "success", "last_attempt": "2026-04-21T00:00:00Z"},
            {"name": "photon", "status": "failed", "error": "404", "last_attempt": "2026-04-21T00:00:00Z"},
        ])

        # Upload first so there's data to aggregate
        run_script(
            "--backend", "local", "--bucket", str(self.bucket),
            "upload", "--local-dir", str(self.bundles), "--version", "dev",
        )

    def tearDown(self):
        self.tmpdir.cleanup()

    def _aggregate(self, *extra_args):
        return run_script(
            "--backend", "local", "--bucket", str(self.bucket),
            "aggregate", "--version", "dev", "--output-dir", str(self.output),
            *extra_args,
        )

    def test_creates_zip_with_bundles(self):
        result = self._aggregate()
        self.assertEqual(result.returncode, 0)

        zip_path = self.output / "vulnerabilities.zip"
        self.assertTrue(zip_path.exists())

        with zipfile.ZipFile(zip_path) as zf:
            names = zf.namelist()
            self.assertIn("alpine.json.zst", names)
            self.assertIn("nvd.json.zst", names)
            self.assertIn("photon.json.zst", names)

    def test_includes_status_json_in_zip(self):
        self._aggregate()

        with zipfile.ZipFile(self.output / "vulnerabilities.zip") as zf:
            self.assertIn("status.json", zf.namelist())
            status = json.loads(zf.read("status.json"))
            self.assertEqual(len(status["updaters"]), 3)

    def test_uploads_zip_to_bucket(self):
        self._aggregate()
        self.assertTrue((self.bucket / "dev" / "vulnerabilities.zip").exists())

    def test_dry_run_does_not_upload(self):
        result = self._aggregate("--dry-run")
        self.assertEqual(result.returncode, 0)
        self.assertTrue((self.output / "vulnerabilities.zip").exists())
        self.assertFalse((self.bucket / "dev" / "vulnerabilities.zip").exists())

    def test_no_bundles_fails(self):
        empty_bucket = self.root / "empty-bucket"
        empty_bucket.mkdir()
        result = run_script(
            "--backend", "local", "--bucket", str(empty_bucket),
            "aggregate", "--version", "dev", "--output-dir", str(self.output),
        )
        self.assertEqual(result.returncode, 1)


if __name__ == "__main__":
    unittest.main()
