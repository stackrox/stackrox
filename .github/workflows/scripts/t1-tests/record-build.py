#!/usr/bin/env python3
"""Record the actual stream checkout, dependency pin, and built binary hash."""

import os
from pathlib import Path
import re

import lib


module = Path("go.mod").read_text()
clair = re.search(r"^\s*github\.com/quay/claircore\s+(\S+)", module, re.MULTILINE)
lib.write_json(Path(os.environ["T1_EVIDENCE"]) / "updater-build.json", {
    "stream": os.environ["STREAM"],
    "configured_ref": os.environ["T1_RESOLVED_REF"],
    "resolved_commit": lib.command(["git", "rev-parse", "HEAD"]).decode().strip(),
    "go_mod_sha256": lib.digest("go.mod"),
    "claircore": clair.group(1) if clair else None,
    "updater_sha256": lib.digest("/tmp/updater"),
})
