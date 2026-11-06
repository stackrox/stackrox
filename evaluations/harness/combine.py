"""Combine prompts and per-model responses into a single spreadsheet-friendly CSV.

For every deployment prompt (data/prompts/{id}.txt) and every candidate model in
config/models.yaml, this emits one row pairing the prompt with that model's
generated response (data/responses/{model}/{id}.txt).

Unlike output/results_{long,pivot}.csv (which hold judge *scores*), this captures
the raw prompt and response *text* side by side for eyeball comparison.

Output (long shape, one row per model):
  output/combined.csv   columns: deployment_id, prompt, model, response

Usage:
  python harness/combine.py
  python harness/combine.py --deployment <id>   # only this deployment
  python harness/combine.py --output <path>     # custom CSV path
"""

from __future__ import annotations

import argparse
import csv
import sys
from pathlib import Path

from common import (
    OUTPUT_DIR,
    PROMPTS_DIR,
    RESPONSES_DIR,
    ensure_dirs,
    load_candidates,
)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("-d", "--deployment",
                        help="combine only this deployment ID (default: all)")
    parser.add_argument("-o", "--output", type=Path,
                        default=OUTPUT_DIR / "combined.csv",
                        help="output CSV path (default: output/combined.csv)")
    args = parser.parse_args()

    ensure_dirs()
    candidates = load_candidates()

    prompt_files = sorted(PROMPTS_DIR.glob("*.txt"))
    if not prompt_files:
        sys.exit(f"no prompts in {PROMPTS_DIR}; run promptgen.py first")
    prompts = {p.stem: p.read_text(encoding="utf-8") for p in prompt_files}

    if args.deployment:
        if args.deployment not in prompts:
            sys.exit(f"no prompt for {args.deployment!r} in {PROMPTS_DIR}; "
                     "run promptgen.py first")
        prompts = {args.deployment: prompts[args.deployment]}
        print(f"scoped to single deployment {args.deployment}")

    # Rows ordered by deployment_id, then by candidate order from models.yaml.
    rows: list[tuple[str, str, str, str]] = []
    for dep_id in sorted(prompts):
        prompt = prompts[dep_id]
        for cand in candidates:
            name = cand["name"]
            resp_file = RESPONSES_DIR / name / f"{dep_id}.txt"
            if not resp_file.exists():
                print(f"  [{name}] missing response for {dep_id}, skipping",
                      file=sys.stderr)
                continue
            response = resp_file.read_text(encoding="utf-8")
            rows.append((dep_id, prompt, name, response))

    args.output.parent.mkdir(parents=True, exist_ok=True)
    with args.output.open("w", newline="") as f:
        w = csv.writer(f)
        w.writerow(["deployment_id", "prompt", "model", "response"])
        w.writerows(rows)
    print(f"wrote {args.output} ({len(rows)} rows)")


if __name__ == "__main__":
    main()
