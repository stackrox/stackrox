"""Shared helpers for disposable T1 tests, never production validation."""

from contextlib import contextmanager
from functools import partial
import hashlib
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import threading
import zipfile

BASE = "82a24f57b7ca65ab93d5cdca4f7f98fa3db6a9bb"
WORKFLOW = ".github/workflows/scanner-updater-bundle.yaml"
CONFIG = "scanner/updater/ci/source-config.yaml"
HELPER = ".github/workflows/scripts/scanner-updater-config.sh"
VALIDATOR = ".github/workflows/scripts/scanner-updater-validate-source.sh"
BATS = "scripts/scanner-updater-validate-source.bats"
SCRIPT_MARKER = "# Verify all expected sources were imported with non-zero timestamps."


def command(args, *, cwd=None, env=None, timeout=120, input=None):
    result = subprocess.run(
        args, cwd=cwd, env=env, input=input, capture_output=True, timeout=timeout,
    )
    if result.returncode:
        raise RuntimeError(
            f"Command failed ({result.returncode}): {args!r}\n"
            + result.stdout.decode(errors="replace")
            + result.stderr.decode(errors="replace")
        )
    return result.stdout


def original(repo, path):
    return command(["git", "-C", str(repo), "show", f"{BASE}:{path}"])


def workflow(repo):
    return document(repo, WORKFLOW)


def document(repo, path):
    return json.loads(command(["yq", "eval", "-o=json", ".", "-"], input=original(repo, path)))


def block(repo, name):
    matches = [s for s in workflow(repo)["jobs"]["bundle"]["steps"] if s.get("name") == name]
    if len(matches) != 1 or "run" not in matches[0]:
        raise RuntimeError(f"Expected one original run block: {name}")
    return matches[0]["run"]


def digest(path):
    h = hashlib.sha256()
    with Path(path).open("rb") as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def write_json(path, value):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n")


def sandbox(repo, root):
    root = Path(root)
    root.mkdir(parents=True, exist_ok=True)
    for relative in (CONFIG, HELPER, VALIDATOR, BATS):
        target = root / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(original(repo, relative))
        if relative.endswith(".sh"):
            target.chmod(0o755)
    env = os.environ.copy()
    env.update(SOURCE_CONFIG=CONFIG, GITHUB_OUTPUT=str(root / "github-output"))
    return env


def sources(root, stream, env):
    value = command(["bash", HELPER, "sources", stream], cwd=root, env=env).decode()
    result = value.splitlines()
    if not result or any(not s or "/" in s or s in ("null", "..") for s in result):
        raise RuntimeError(f"Invalid real source configuration for {stream}: {result}")
    return result


def compress(payload):
    return command(["zstd", "-q", "-c"], input=payload)


def archive_manifest(path):
    result = []
    with zipfile.ZipFile(path) as bundle:
        for entry in bundle.infolist():
            h = hashlib.sha256()
            with bundle.open(entry) as data:
                for chunk in iter(lambda: data.read(1024 * 1024), b""):
                    h.update(chunk)
            result.append({"name": entry.filename, "size": entry.file_size, "sha256": h.hexdigest()})
    return result


def replace_entry(path, entry_name, payload):
    """Repack one entry, retaining all other names, payloads, and ZIP timestamps."""
    path = Path(path)
    before = archive_manifest(path)
    if sum(e["name"] == entry_name for e in before) != 1:
        raise RuntimeError(f"Expected exactly one {entry_name}")
    fd, temporary = tempfile.mkstemp(prefix="t1-repack-", suffix=".zip", dir=path.parent)
    os.close(fd)
    try:
        with zipfile.ZipFile(path) as source, zipfile.ZipFile(temporary, "w", allowZip64=True) as target:
            for entry in source.infolist():
                if entry.filename == entry_name:
                    target.writestr(entry, payload)
                else:
                    with source.open(entry) as data, target.open(entry, "w", force_zip64=True) as dest:
                        shutil.copyfileobj(data, dest, length=1024 * 1024)
        after = archive_manifest(temporary)
        changed = [a["name"] for a, b in zip(before, after) if a != b]
        if len(before) != len(after) or changed != [entry_name]:
            raise RuntimeError(f"Unexpected fixture mutation: {changed}")
        os.replace(temporary, path)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)
    return {"before": before, "after": after, "changed": changed}


def manual_record(*, zero=False):
    value = json.loads(Path(__file__).with_name("manual-record.json").read_text())
    if zero:
        value["Fingerprint"] = "t1-zero-supported-records"
        value["Vuln"]["package"] = {}
    return (json.dumps(value) + "\n").encode()


def manual_json_error(text):
    pattern = re.compile(r"invalid character|unexpected EOF|unexpected end.*JSON|cannot unmarshal|invalid JSON|JSON syntax", re.IGNORECASE)
    return any("bundles/manual.json.zst" in line and pattern.search(line) for line in text.splitlines())


def extract(repo, dest):
    dest = Path(dest)
    dest.mkdir(parents=True, exist_ok=True)
    provenance = {"implementation_commit": BASE, "blocks": {}}
    for name in ("Validate source content", "Validate ZIP entries", "Validate bundle import"):
        text = block(repo, name)
        target = dest / (name.replace(" ", "-") + ".sh")
        target.write_text(text)
        provenance["blocks"][name] = digest(target)
    text = block(repo, "Validate bundle import")
    if text.count(SCRIPT_MARKER) != 1:
        raise RuntimeError("Original metadata-check boundary changed")
    checker = "set -euo pipefail\nsc=./.github/workflows/scripts/scanner-updater-config.sh\n"
    checker += text[text.index(SCRIPT_MARKER):]
    (dest / "metadata-check.sh").write_text(checker)
    provenance["metadata_check_sha256"] = digest(dest / "metadata-check.sh")
    write_json(dest / "original-blocks.json", provenance)


def isolated_database():
    if os.environ.get("T1_ISOLATED_DATABASE") != "true":
        raise RuntimeError("SQL mutations require T1_ISOLATED_DATABASE=true in a disposable test container")
    if os.environ.get("PGHOST", "postgres") != "postgres":
        raise RuntimeError("Only the isolated postgres service hostname is allowed")


def sql(query, *, database="matcher", app="t1-observer"):
    isolated_database()
    env = os.environ.copy()
    env.update(PGPASSWORD="test", PGAPPNAME=app, PGCONNECT_TIMEOUT="5")
    return command([
        "psql", "-X", "-h", "postgres", "-U", "test", "-d", database,
        "-v", "ON_ERROR_STOP=1", "-At", "-c", query,
    ], env=env, timeout=20).decode().strip()


def database_counts():
    return json.loads(sql("""
        SELECT json_build_object(
          'vulnerabilities', (SELECT count(*) FROM public.vuln),
          'vulnerability_links', (SELECT count(*) FROM public.uo_vuln),
          'enrichments', (SELECT count(*) FROM public.enrichment),
          'enrichment_links', (SELECT count(*) FROM public.uo_enrich));
    """))


@contextmanager
def serve(directory):
    server = ThreadingHTTPServer(("127.0.0.1", 0), partial(SimpleHTTPRequestHandler, directory=str(directory)))
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        yield f"http://127.0.0.1:{server.server_port}"
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=10)
