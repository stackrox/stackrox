# Deployment Risk AI-Summary Prompt Evaluation

Offline harness to evaluate and compare LLMs / prompt variants for
`GetDeploymentRiskAISummary` (`central/deployment/service/service_impl.go`).

It reconstructs the **exact** production prompt for real deployments, sends it to
multiple candidate models **directly** (via LiteLLM, bypassing OpenShift
Lightspeed), and scores each response with a multi-dimension **LLM-as-judge**
using [lightspeed-evaluation](https://github.com/lightspeed-core/lightspeed-evaluation)
in library mode. Results are written as spreadsheet-friendly CSV.

## Why this design

- **Zero prompt drift.** The prompt + data sanitization live in one Go package,
  `central/deployment/service/aisummary`, used by *both* production and the
  `promptgen` CLI here. The evaluated prompt is byte-identical to what Central sends.
- **Model comparison.** Production only talks to OLS (which hides the model). To
  compare models we call them directly; `config/models.yaml` is the switch point.
- **No ground truth needed.** Judging uses GEval metrics with custom criteria over
  `[query, response]` — the `query` includes the full deployment/risk data, so the
  judge can check factual grounding without a reference answer.

## Layout

```
config/system.yaml     judge model, GEval metrics, output/CSV settings
config/models.yaml     candidate models to compare (edit freely)
cmd/promptgen/         Go CLI: API JSON -> exact production prompt
harness/fetch.py       ACS API -> data/cache/{id}.json
harness/promptgen.py   cached JSON -> data/prompts/{id}.txt (via the Go binary)
harness/generate.py    prompts -> data/responses/{model}/{id}.txt (LiteLLM)
harness/run_eval.py    judge responses -> output/results_{long,pivot}.csv
data/deployment_ids.csv  input: deployment IDs to evaluate
```

## Setup

```bash
# 1. Build the Go prompt reconstructor (uses production code, zero drift)
go build -o evaluations/bin/promptgen ./evaluations/cmd/promptgen

# 2. Python environment with uv (creates a project-local .venv and installs deps)
#    The harness pins Python 3.12 (.python-version) and caps <3.14: lightspeed-
#    evaluation's deps (DeepEval/LangChain + a Pydantic V1 shim) aren't 3.14-ready.
#    uv fetches 3.12 automatically, so a system Python of 3.13/3.14 is fine.
cd evaluations && uv sync

# 3. Credentials
export ROX_ENDPOINT=https://central.example.com   # your ACS
export ROX_API_TOKEN=...                           # needs View(Deployment)
export ROX_INSECURE=1                              # only for self-signed dev certs
export OPENAI_API_KEY=...                           # judge + candidate keys as needed
# export ANTHROPIC_API_KEY=...                       # Anthropic via direct API key
```

### Anthropic (or Gemini) via Vertex AI — no API key

If you reach Anthropic models through GCP Vertex AI instead of a direct API key,
authenticate with Application Default Credentials and point LiteLLM at your project
and region. No `ANTHROPIC_API_KEY` is needed.

```bash
gcloud auth application-default login        # one-time; sets up ADC
gcloud auth application-default set-quota-project "$ANTHROPIC_VERTEX_PROJECT_ID"

# LiteLLM's Anthropic-on-Vertex path reads ANTHROPIC_VERTEX_PROJECT_ID / CLOUD_ML_REGION.
# These also map to the VERTEXAI_* names LiteLLM uses for other Vertex providers:
export VERTEXAI_PROJECT="$ANTHROPIC_VERTEX_PROJECT_ID"
export VERTEXAI_LOCATION="$CLOUD_ML_REGION"
```

Then use the `vertex_ai/` model prefix (candidate) or `provider: vertex_ai` (judge) —
see [Choosing models](#choosing-models). This flows through the same LiteLLM path as
API-key providers, so no code changes are required.

## Choosing models

Two independent choices: the **candidates** being compared (`config/models.yaml`) and
the **judge** that scores them (`config/system.yaml`). Both go through LiteLLM.

**Candidates** — `config/models.yaml`, `model:` is a LiteLLM model id:

```yaml
candidates:
  - name: gpt-5.4-mini
    model: openai/gpt-5.4-mini
  - name: claude-vertex                       # Anthropic via Vertex AI (ADC, no key)
    model: vertex_ai/claude-sonnet-4-5@20250929
```

**Judge** — `config/system.yaml`, `llm_pool.models.judge`:

```yaml
llm_pool:
  models:
    judge:
      provider: vertex_ai                     # or openai / anthropic
      model: claude-sonnet-4-5@20250929
```

Vertex model ids use an `@version` suffix (e.g. `@20250929`) and must be enabled in
your region. Find the exact dated ids and their supported regions on the model's
**Model Garden** page in the GCP console (`gcloud ai models list` won't show them —
it lists your project's Model Registry, not Model Garden).

Tip: to avoid family bias in scoring, pick a judge from a different model family than
the candidates you care most about.

## Run

`uv run` uses the project `.venv` automatically (no manual activation needed):

```bash
cd evaluations

# Put deployment IDs (one per row, "id" header) in data/deployment_ids.csv, then:
uv run python harness/fetch.py        # GET /v1/deploymentswithrisk/{id} -> data/cache/
uv run python harness/promptgen.py    # reconstruct exact prompts -> data/prompts/
uv run python harness/generate.py     # candidate responses -> data/responses/
uv run python harness/run_eval.py     # judge + write output/results_{long,pivot}.csv
```

Re-runs are cheap: fetch/generate skip work that already exists. Add a model to
`config/models.yaml` and re-run `generate.py` + `run_eval.py` to compare it — only
the new model is generated.

Useful flags:

```bash
uv run python harness/generate.py --only <model-name>   # generate just one candidate
uv run python harness/generate.py --force               # overwrite existing responses
uv run python harness/run_eval.py --deployment <id>     # score just one deployment
```

`--deployment` is handy when you add a single new ID to `data/deployment_ids.csv`
and only want to score that one rather than re-judging the whole set.

## Output

- `output/results_long.csv` — one row per (model, deployment, metric): `score` + judge `reason`.
- `output/results_pivot.csv` — one row per deployment; columns are `model::metric`
  scores plus a per-model mean, for side-by-side comparison in a spreadsheet.
- `output/framework/` — the framework's own CSV/JSON/TXT (from the `storage` block).

## Iterating on the prompt

Edit the prompt/sanitizer in `central/deployment/service/aisummary`, rebuild the Go
binary, then re-run `promptgen.py` + `generate.py --force` + `run_eval.py`. Changes
flow to production automatically because both share that package.

## Caveats

- LLM-judge scores are noisy and family-biased. Treat them as *relative* signal,
  read the `reason` strings, and keep the judge at temperature 0. For neutral runs,
  prefer a judge from a family not in the candidate set.
- This measures the raw model on our prompt, not the full OLS pipeline (OLS may add
  its own system prompt / RAG). That is intentional for model/prompt comparison.

## Troubleshooting

- **A candidate produces empty responses (reasoning models, e.g. gpt-5.x).** For
  reasoning models `max_tokens` is a *combined* budget for hidden reasoning plus
  visible output; the default is too small, so the model spends it all on reasoning
  and returns empty `content` (larger deployments fail first). Raise `max_tokens` in
  `config/models.yaml` (the sample entries use `16384`) and force-regenerate that
  candidate: `uv run python harness/generate.py --only <name> --force`. `run_eval.py`
  skips any response that is still empty instead of crashing.

- **`OSError: [Errno 24] Too many open files: '.deepeval/.deepeval'` during
  `run_eval.py`.** DeepEval (under lightspeed-evaluation's GEval) reopens its
  telemetry file on every metric and exhausts the low default fd limit. In the shell
  you run the eval from:
  ```bash
  ulimit -n 8192                          # raise the fd limit (macOS default is ~256)
  export DEEPEVAL_TELEMETRY_OPT_OUT=YES   # stop DeepEval touching .deepeval each metric
  ```
  (`ulimit` only affects the current shell, so set it right before `uv run`.)
