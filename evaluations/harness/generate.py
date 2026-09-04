"""Generate candidate responses by sending each prompt to each candidate model.

Calls the models DIRECTLY via LiteLLM (bypassing OpenShift Lightspeed), so we can
compare raw models on the identical production prompt. Reads data/prompts/{id}.txt
and writes data/responses/{model_name}/{id}.txt. Existing responses are skipped
unless --force, so adding a new model only generates the missing ones.

API keys are read from the environment (OPENAI_API_KEY, ANTHROPIC_API_KEY, ...).

Usage:
  python harness/generate.py
  python harness/generate.py --force
  python harness/generate.py --only gpt-5.4-mini
"""

from __future__ import annotations

import argparse
import sys

import litellm

from common import PROMPTS_DIR, RESPONSES_DIR, ensure_dirs, load_candidates


def generate(model: str, prompt: str, temperature: float, max_tokens: int,
             api_base: str | None, api_key: str | None) -> str:
    kwargs: dict = {
        "model": model,
        "messages": [{"role": "user", "content": prompt}],
        "temperature": temperature,
        "max_tokens": max_tokens,
    }
    if api_base:
        kwargs["api_base"] = api_base
    if api_key:
        kwargs["api_key"] = api_key
    resp = litellm.completion(**kwargs)
    return resp.choices[0].message.content or ""


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--force", action="store_true", help="regenerate existing responses")
    parser.add_argument("--only", help="generate only for this candidate name")
    args = parser.parse_args()

    ensure_dirs()
    litellm.drop_params = True  # tolerate provider-unsupported params
    litellm.suppress_debug_info = True  # hide the "Provider List / Give Feedback" banners

    prompt_files = sorted(PROMPTS_DIR.glob("*.txt"))
    if not prompt_files:
        sys.exit(f"no prompts in {PROMPTS_DIR}; run promptgen.py first")

    candidates = load_candidates()
    if args.only:
        candidates = [c for c in candidates if c.get("name") == args.only]
        if not candidates:
            sys.exit(f"no candidate named {args.only!r} in models.yaml")

    import os

    total, failed = 0, 0
    for cand in candidates:
        name = cand["name"]
        model = cand["model"]
        temperature = float(cand.get("temperature", 0.0))
        max_tokens = int(cand.get("max_tokens", 1024))
        api_base = cand.get("api_base")
        api_key = os.environ.get(cand["api_key_env"]) if cand.get("api_key_env") else None

        out_dir = RESPONSES_DIR / name
        out_dir.mkdir(parents=True, exist_ok=True)
        print(f"[{name}] model={model}")

        for prompt_file in prompt_files:
            dep_id = prompt_file.stem
            out = out_dir / f"{dep_id}.txt"
            if out.exists() and not args.force:
                continue
            try:
                response = generate(
                    model, prompt_file.read_text(encoding="utf-8"),
                    temperature, max_tokens, api_base, api_key,
                )
            except Exception as exc:  # noqa: BLE001 - surface any provider error
                print(f"  FAILED {dep_id}: {exc}", file=sys.stderr)
                failed += 1
                continue
            out.write_text(response, encoding="utf-8")
            total += 1
            print(f"  generated {dep_id}")

    print(f"generated={total} failed={failed} -> {RESPONSES_DIR}")
    if failed:
        sys.exit(1)


if __name__ == "__main__":
    main()
