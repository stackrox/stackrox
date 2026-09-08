"""Judge the generated responses and emit spreadsheet-friendly CSVs.

Uses lightspeed-evaluation in library mode (static): for each candidate model and
each deployment it builds a TurnData whose `query` is the full production prompt
(including the DEPLOYMENT AND RISK DATA JSON) and whose `response` is the model's
generated summary, then runs the multi-dimension GEval judge configured in
config/system.yaml.

Outputs:
  output/results_long.csv   one row per (model, deployment, metric): score + reason
  output/results_pivot.csv  one row per deployment; columns = model x metric score
                            (+ a per-model mean), for side-by-side comparison
  output/framework/...      the framework's own CSV/JSON/TXT (from the storage block)

Usage:
  python harness/run_eval.py
  python harness/run_eval.py --deployment <id>   # score only this deployment
"""

from __future__ import annotations

import argparse
import csv
import sys
from collections import defaultdict

import yaml

from common import (
    OUTPUT_DIR,
    PROMPTS_DIR,
    RESPONSES_DIR,
    SYSTEM_CONFIG,
    ensure_dirs,
    load_candidates,
)

from lightspeed_evaluation import ConfigLoader, EvaluationData, TurnData, evaluate


def turn_metric_ids() -> list[str]:
    """Read the configured turn-level metric identifiers from system.yaml."""
    with SYSTEM_CONFIG.open() as f:
        cfg = yaml.safe_load(f) or {}
    turn_level = (cfg.get("metrics_metadata") or {}).get("turn_level") or {}
    ids = list(turn_level.keys())
    if not ids:
        sys.exit("no turn_level metrics defined in system.yaml")
    return ids


def build_eval_data(dep_id: str, prompt: str, response: str,
                    metrics: list[str]) -> EvaluationData:
    return EvaluationData(
        conversation_group_id=dep_id,
        description="Deployment risk AI-summary evaluation",
        tag="risk-summary",
        turns=[
            TurnData(
                turn_id="summary",
                query=prompt,
                response=response,
                turn_metrics=metrics,
            )
        ],
    )


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("-d", "--deployment",
                        help="score only this deployment ID (default: all)")
    args = parser.parse_args()

    ensure_dirs()
    metrics = turn_metric_ids()
    candidates = load_candidates()

    config = ConfigLoader().load_system_config(str(SYSTEM_CONFIG))

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

    # rows: list of (model, dep_id, metric, score, result, reason)
    long_rows: list[tuple] = []
    # pivot[dep_id][(model, metric)] = score
    pivot: dict[str, dict[tuple[str, str], float | None]] = defaultdict(dict)

    for cand in candidates:
        name = cand["name"]
        resp_dir = RESPONSES_DIR / name
        data: list[EvaluationData] = []
        for dep_id, prompt in prompts.items():
            resp_file = resp_dir / f"{dep_id}.txt"
            if not resp_file.exists():
                print(f"  [{name}] missing response for {dep_id}, skipping", file=sys.stderr)
                continue
            response = resp_file.read_text(encoding="utf-8")
            if not response.strip():
                print(f"  [{name}] empty response for {dep_id}, skipping "
                      f"(model returned no content — see note below)", file=sys.stderr)
                continue
            data.append(build_eval_data(dep_id, prompt, response, metrics))

        if not data:
            print(f"  [{name}] no responses to judge, skipping", file=sys.stderr)
            continue

        print(f"[{name}] judging {len(data)} deployments x {len(metrics)} metrics ...")
        results = evaluate(config, data)

        for r in results:
            dep_id = getattr(r, "conversation_group_id", "")
            metric = getattr(r, "metric_identifier", "")
            score = getattr(r, "score", None)
            long_rows.append((
                name,
                dep_id,
                metric,
                score,
                getattr(r, "result", ""),
                getattr(r, "reason", ""),
            ))
            pivot[dep_id][(name, metric)] = score

    _write_long_csv(long_rows)
    _write_pivot_csv(pivot, candidates, metrics)


def _write_long_csv(rows: list[tuple]) -> None:
    path = OUTPUT_DIR / "results_long.csv"
    with path.open("w", newline="") as f:
        w = csv.writer(f)
        w.writerow(["model", "deployment_id", "metric", "score", "result", "reason"])
        w.writerows(rows)
    print(f"wrote {path} ({len(rows)} rows)")


def _write_pivot_csv(pivot, candidates, metrics) -> None:
    path = OUTPUT_DIR / "results_pivot.csv"
    names = [c["name"] for c in candidates]

    header = ["deployment_id"]
    for name in names:
        for metric in metrics:
            header.append(f"{name}::{metric}")
        header.append(f"{name}::mean")

    with path.open("w", newline="") as f:
        w = csv.writer(f)
        w.writerow(header)
        for dep_id in sorted(pivot):
            row = [dep_id]
            for name in names:
                scores = []
                for metric in metrics:
                    score = pivot[dep_id].get((name, metric))
                    row.append("" if score is None else f"{score:.3f}")
                    if score is not None:
                        scores.append(score)
                row.append(f"{sum(scores) / len(scores):.3f}" if scores else "")
            w.writerow(row)
    print(f"wrote {path}")


if __name__ == "__main__":
    main()
