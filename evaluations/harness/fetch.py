"""Fetch deployment+risk data from a live ACS/StackRox instance.

For each deployment ID in data/deployment_ids.csv, calls
GET {ROX_ENDPOINT}/v1/deploymentswithrisk/{id} and caches the raw JSON to
data/cache/{id}.json. This is the exact pair of objects
(storage.Deployment + storage.Risk) that GetDeploymentRiskAISummary consumes.

Env:
  ROX_ENDPOINT     e.g. https://central.example.com  (or host:port)
  ROX_API_TOKEN    API token with View(Deployment) permission
  ROX_INSECURE     set to "1"/"true" to skip TLS verification (dev only)

Usage:
  python harness/fetch.py            # fetch all IDs, skip already-cached
  python harness/fetch.py --force    # re-fetch even if cached
"""

from __future__ import annotations

import argparse
import os
import sys

import requests

from common import CACHE_DIR, ensure_dirs, read_deployment_ids


def _endpoint() -> str:
    ep = os.environ.get("ROX_ENDPOINT")
    if not ep:
        sys.exit("ROX_ENDPOINT is not set")
    if not ep.startswith("http"):
        ep = "https://" + ep
    return ep.rstrip("/")


def _token() -> str:
    tok = os.environ.get("ROX_API_TOKEN")
    if not tok:
        sys.exit("ROX_API_TOKEN is not set")
    return tok


def _verify() -> bool:
    return os.environ.get("ROX_INSECURE", "").lower() not in ("1", "true", "yes")


def fetch_one(session: requests.Session, endpoint: str, dep_id: str, verify: bool) -> bytes:
    url = f"{endpoint}/v1/deploymentswithrisk/{dep_id}"
    resp = session.get(url, timeout=60, verify=verify)
    resp.raise_for_status()
    return resp.content


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--force", action="store_true", help="re-fetch cached IDs")
    args = parser.parse_args()

    ensure_dirs()
    endpoint = _endpoint()
    verify = _verify()
    if not verify:
        requests.packages.urllib3.disable_warnings()  # type: ignore[attr-defined]

    ids = read_deployment_ids()
    if not ids:
        sys.exit("no deployment IDs found in data/deployment_ids.csv")

    session = requests.Session()
    session.headers["Authorization"] = f"Bearer {_token()}"

    ok, skipped, failed = 0, 0, 0
    for dep_id in ids:
        out = CACHE_DIR / f"{dep_id}.json"
        if out.exists() and not args.force:
            skipped += 1
            continue
        try:
            content = fetch_one(session, endpoint, dep_id, verify)
        except requests.HTTPError as exc:
            print(f"  FAILED {dep_id}: {exc}", file=sys.stderr)
            failed += 1
            continue
        out.write_bytes(content)
        ok += 1
        print(f"  cached {dep_id}")

    print(f"fetched={ok} skipped={skipped} failed={failed} -> {CACHE_DIR}")
    if failed:
        sys.exit(1)


if __name__ == "__main__":
    main()
