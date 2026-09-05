"""Shared paths and helpers for the deployment risk AI-summary eval harness."""

from __future__ import annotations

import csv
from pathlib import Path

# evaluations/harness/common.py -> evaluations/ -> repo root
EVAL_DIR = Path(__file__).resolve().parents[1]
REPO_ROOT = EVAL_DIR.parent

CONFIG_DIR = EVAL_DIR / "config"
SYSTEM_CONFIG = CONFIG_DIR / "system.yaml"
MODELS_CONFIG = CONFIG_DIR / "models.yaml"

DATA_DIR = EVAL_DIR / "data"
IDS_CSV = DATA_DIR / "deployment_ids.csv"
CACHE_DIR = DATA_DIR / "cache"          # raw {id}.json from the API
PROMPTS_DIR = DATA_DIR / "prompts"      # reconstructed {id}.txt prompts
RESPONSES_DIR = DATA_DIR / "responses"  # {model}/{id}.txt generated responses

OUTPUT_DIR = EVAL_DIR / "output"

PROMPTGEN_BIN = EVAL_DIR / "bin" / "promptgen"


def read_deployment_ids(path: Path = IDS_CSV) -> list[str]:
    """Read deployment IDs from a CSV.

    Accepts either a header column named ``id`` or a plain one-ID-per-line file.
    Blank lines and lines starting with ``#`` are ignored.
    """
    ids: list[str] = []
    with path.open(newline="") as f:
        sample = f.readline()
        f.seek(0)
        if "id" in sample.lower() and "," in sample or "id" == sample.strip().lower():
            reader = csv.DictReader(f)
            if reader.fieldnames and "id" in reader.fieldnames:
                for row in reader:
                    value = (row.get("id") or "").strip()
                    if value and not value.startswith("#"):
                        ids.append(value)
                return _dedup(ids)
            f.seek(0)
        for line in f:
            value = line.strip()
            if value and not value.startswith("#") and value.lower() != "id":
                ids.append(value)
    return _dedup(ids)


def _dedup(ids: list[str]) -> list[str]:
    seen: set[str] = set()
    out: list[str] = []
    for i in ids:
        if i not in seen:
            seen.add(i)
            out.append(i)
    return out


def load_candidates(path: Path = MODELS_CONFIG) -> list[dict]:
    """Load the candidate model list from models.yaml."""
    import yaml

    with path.open() as f:
        cfg = yaml.safe_load(f) or {}
    candidates = cfg.get("candidates") or []
    if not candidates:
        raise ValueError(f"no candidates defined in {path}")
    return candidates


def ensure_dirs() -> None:
    for d in (CACHE_DIR, PROMPTS_DIR, RESPONSES_DIR, OUTPUT_DIR):
        d.mkdir(parents=True, exist_ok=True)
