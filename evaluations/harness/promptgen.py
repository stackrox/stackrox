"""Reconstruct the exact production prompt for each cached deployment.

Shells out to the Go promptgen binary (evaluations/bin/promptgen), which reuses
the production prompt/sanitization code (central/deployment/service/aisummary),
so the prompt is byte-identical to what Central sends. Reads data/cache/{id}.json
and writes data/prompts/{id}.txt.

Build the binary first:
  go build -o evaluations/bin/promptgen ./evaluations/cmd/promptgen

Usage:
  python harness/promptgen.py
"""

from __future__ import annotations

import subprocess
import sys

from common import CACHE_DIR, PROMPTS_DIR, PROMPTGEN_BIN, ensure_dirs


def build_prompt(cache_json: bytes) -> str:
    """Run the Go promptgen binary on a cached API response, return the prompt."""
    result = subprocess.run(
        [str(PROMPTGEN_BIN)],
        input=cache_json,
        capture_output=True,
        check=True,
    )
    return result.stdout.decode("utf-8")


def main() -> None:
    ensure_dirs()
    if not PROMPTGEN_BIN.exists():
        sys.exit(
            f"promptgen binary not found at {PROMPTGEN_BIN}\n"
            "Build it: go build -o evaluations/bin/promptgen ./evaluations/cmd/promptgen"
        )

    cache_files = sorted(CACHE_DIR.glob("*.json"))
    if not cache_files:
        sys.exit(f"no cached deployments in {CACHE_DIR}; run fetch.py first")

    count = 0
    for cache_file in cache_files:
        dep_id = cache_file.stem
        try:
            prompt = build_prompt(cache_file.read_bytes())
        except subprocess.CalledProcessError as exc:
            print(f"  FAILED {dep_id}: {exc.stderr.decode('utf-8').strip()}", file=sys.stderr)
            continue
        (PROMPTS_DIR / f"{dep_id}.txt").write_text(prompt, encoding="utf-8")
        count += 1

    print(f"reconstructed {count} prompts -> {PROMPTS_DIR}")


if __name__ == "__main__":
    main()
