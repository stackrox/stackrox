#!/usr/bin/env python3
"""Record the validated candidate without copying credentials into evidence."""

import base64
import hashlib
import os
from pathlib import Path

import lib


path = Path("vulnerabilities.zip")
sha = hashlib.sha256()
md5 = hashlib.md5(usedforsecurity=False)
with path.open("rb") as source:
    for chunk in iter(lambda: source.read(1024 * 1024), b""):
        sha.update(chunk)
        md5.update(chunk)
lib.write_json(Path(os.environ["T1_EVIDENCE"]) / "candidate.json", {
    "size": path.stat().st_size, "sha256": sha.hexdigest(),
    "md5_base64": base64.b64encode(md5.digest()).decode(),
})
