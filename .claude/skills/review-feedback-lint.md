# Review Feedback → Static Analysis

**Purpose:** Mine historical pull-request review feedback and convert recurring or important engineering feedback into static-analysis rules (ESLint for UI, `go/analysis` for Go).

**Not a rule factory:** The goal is to discover cases where humans repeatedly identify a statically detectable problem during review, and convert that knowledge into a reliable rule. A run may produce zero candidates—that's valid and desirable when no strong opportunity exists.

---

## Usage

```bash
# From repository root
/review-feedback-lint --since YYYY-MM-DD --until YYYY-MM-DD --scope ui|go|all
```

**Examples:**
```bash
/review-feedback-lint --since 2026-08-01 --until 2026-09-01 --scope ui
/review-feedback-lint --since 2026-08-01 --until 2026-09-01 --scope go
```

Scopes:
- `ui` matches paths under `ui/`.
- `go` matches Go/backend review files (`*.go`, `*.proto`, `go.mod`, `go.sum`) regardless of top-level directory.
- `all` keeps all paths except deterministic noise filters.

---

## Design Overview

### Lifecycle

```
PR review corpus (GitHub)
    ↓ deterministic filtering
compact review records
    ↓ optional human/LLM acceptance analysis
review feedback with explicit triage status
    ↓ LLM or manual invariant extraction
candidate engineering patterns
    ↓ deterministic: existing-rule search
new rule candidates
    ↓ LLM: lintability gate + ranking
top N candidates
    ↓ language-specific implementation
working rules + tests + validation
    ↓
human-reviewed proposal
```

### Token Efficiency

- **Deterministic first:** Use code for filtering, file operations, and analysis whenever possible
- **Batched LLM:** Send compact structured records, not full PRs or diffs
- **Cached intermediates:** Store results at each phase to avoid re-analysis on iteration

### Storage

- **Corpus cache root:** `~/.cache/stackrox/review-lint/` by default; override with `REVIEW_LINT_CACHE_ROOT`.
  - `corpus.sqlite3` - Persistent source of truth for PRs, files, review threads, comments, filter runs, comment triage state, and per-thread invariant extraction cache entries across all date ranges
- **Dated export directories:** `~/.cache/stackrox/review-lint/YYYY-MM-DD_to_YYYY-MM-DD/`
  - `raw-prs.jsonl` - Compatibility export of full PR metadata from GitHub for that range
  - `filtered-threads.jsonl` - Compatibility export of substantive review threads after filtering for that range
  - `accepted-feedback.jsonl` - High-confidence accepted changes
  - `candidate-invariants-{repo}-{scope}-{status}.jsonl` - Extracted patterns for a specific repo/scope/status selection
  - `ranked-candidates.json` - Final shortlist with scores

**Comment triage statuses:**
- `undecided` - default; eligible for future analysis runs
- `considered` - reviewed by a human/agent but not yet rejected or incorporated
- `rejected` - not useful for lint-rule mining; exclude from normal follow-up runs
- `incorporated` - already represented by a generated/enabled lint rule or analyzer

Use `.claude/skills/review-feedback-lint/corpus-status.py` to inspect or update statuses.

---

## Implementation Phases

### Phase 1: GitHub Corpus Collection

**Script:** `.claude/skills/review-feedback-lint/collect-prs.py`

**Input:**
- Date range (--since, --until)
- Scope filter (--scope ui|go|all); `go` is semantic, not a path prefix
- Repository (default: stackrox/stackrox)

**Output:** `~/.cache/stackrox/review-lint/corpus.sqlite3` plus per-range `raw-prs.jsonl` compatibility export.

**Per-PR record:**
```json
{
  "pr": {
    "number": 12345,
    "title": "...",
    "url": "...",
    "author": "...",
    "merged_at": "...",
    "base_sha": "...",
    "merge_sha": "...",
    "files_changed": ["ui/apps/platform/src/..."]
  },
  "threads": [
    {
      "id": "...",
      "path": "ui/apps/platform/src/...",
      "line": 42,
      "resolved": true,
      "comments": [
        {
          "id": "...",
          "author": "reviewer",
          "body": "...",
          "created_at": "...",
          "diff_hunk": "..."
        }
      ]
    }
  ]
}
```

**GitHub API usage:**
- `gh api search/issues` pages through merged PR candidates in batches.
- `gh pr view {number} --json ...` fetches PR metadata only for PRs not already present in SQLite, unless `--force` is passed.
- `gh api repos/stackrox/stackrox/pulls/{number}/files` pages through the complete changed-file list before scope filtering.
- `gh api repos/stackrox/stackrox/pulls/{number}/comments` pages through review comments for each newly fetched PR.

The collector stores each PR as it is fetched and links already-known PRs into overlapping collection runs without re-fetching their review comments. A small delay is inserted between paginated API calls; tune with `--api-delay-seconds`, `--pr-batch-size`, `--file-page-size`, and `--comment-page-size` if needed. The collector fails if a query reaches the 1000-PR search cap. Split the date range instead of accepting a potentially incomplete corpus.

### Phase 2: Deterministic Corpus Reduction

**Script:** `.claude/skills/review-feedback-lint/filter-corpus.py`

**Filters out:**
- Bot comments (dependabot, renovate, github-actions, etc.); CodeRabbit comments are kept after trimming boilerplate because they can contain actionable review feedback
- Generated file paths (package-lock.json, generated/, mocks/, etc.)
- Pure acknowledgements ("done", "thanks", "LGTM" with no substance)
- Non-code discussion (title/description suggestions, release notes)
- Formatting-only when covered by prettier/eslint-plugin-prettier
- Threads in paths outside requested scope

**Output:** shared `corpus.sqlite3` filter-run records plus per-range `filtered-threads.jsonl` compatibility export.

**Compact record format:**
```json
{
  "pr_number": 12345,
  "pr_url": "...",
  "thread_id": "...",
  "file_path": "ui/apps/platform/src/Components/Example.tsx",
  "line": 42,
  "initial_comment": {
    "author": "reviewer",
    "body": "...",
    "diff_hunk": "..."
  },
  "replies": [
    {"author": "...", "body": "..."}
  ],
  "code_changed_after": true,
  "resolution_status": "resolved"
}
```

### Phase 3: Acceptance Analysis (Human/LLM, optional)

The shipped scripts do not prove that feedback was accepted. They preserve triage state and lightweight heuristics (`code_changed_after`) for prioritization, but a human or stronger follow-up analysis should validate acceptance before a comment is treated as rule evidence.

**For each filtered thread:** Determine if the review feedback was accepted.

**Evidence:**
- Code near the location changed afterward
- Author replies: "fixed", "good catch", "done"
- Reviewer acknowledgement post-change
- Final approval

**Negative evidence:**
- Explicit disagreement
- Discussion concluding original is intentional
- Resolution without implementation

**Output:** `accepted-feedback.jsonl`
```json
{
  "thread": { /* from filtered */ },
  "accepted": true,
  "confidence": 0.9,
  "reason": "...",
  "review_principle": "...",
  "before_code": "...",
  "after_code": "..."
}
```

**Discard:** Low-confidence rejected feedback. Retain uncertain only as supporting evidence.

### Phase 4: Invariant Extraction (LLM)

**Cluster equivalent principles** across threads. Current automated clustering is intentionally conservative lexical clustering, so recurrence counts are useful hints but not proof that every semantically equivalent comment was grouped.

Per-thread LLM analysis results are cached in SQLite by repository, thread ID, scope, model, and a hash of the prompt-relevant review text so repeat runs do not re-spend API calls for unchanged threads.

**Extract underlying engineering invariant:**
```
Review comment: "This subscription needs cleanup on unmount."
Invariant: Calls to subscribe() in useEffect must return cleanup.
```

**Output:** `candidate-invariants-{repo}-{scope}-{status}.jsonl` to avoid collisions between scope, status, and repository selections.
```json
{
  "id": "react-subscription-cleanup",
  "principle": "...",
  "scope": "ui",
  "category": ["correctness", "API-misuse"],
  "severity": "high",
  "generalizability": "high",
  "static_analysis_feasibility": "high",
  "false_positive_risk": "low",
  "required_analysis": ["syntax", "imports"],
  "historical_examples": [
    {"pr": 12345, "thread": "...", "thread_id": "..."}
  ]
}
```

### Phase 5: Existing-Rule Search (Deterministic)

**Mandatory before generating any rule.**

**For each candidate:**
1. Search current ESLint/golangci-lint config
2. Search existing StackRox custom rules
3. Determine if already handled or overlaps

**Prefer:**
1. Enable/configure existing rule
2. Extend existing StackRox rule
3. Add simple custom rule
4. (last resort) Add complex custom rule

**Update candidate:** Add `existing_enforcement` field describing findings.

### Phase 6: Lintability Gate (LLM)

**Classify each invariant:**
- ✅ GOOD STATIC-ANALYSIS CANDIDATE
- ⚠️ POSSIBLE BUT NOT WORTH COMPLEXITY
- ❌ NOT SUITABLE FOR STATIC ANALYSIS
- 🔍 ALREADY COVERED

**Good candidates:**
- Forbidden APIs/imports
- Project-specific API misuse
- Missing required props
- Lifecycle patterns detectable via AST/types
- Security-sensitive constructs

**Poor candidates:**
- Subjective abstraction preferences
- Broad architectural judgment
- Checks with frequent legitimate exceptions

### Phase 7: Candidate Ranking

**Rank by:**
- Engineering severity
- Acceptance confidence
- Generalizability
- Implementation simplicity
- False-positive risk
- Runtime cost

**Output:** `ranked-candidates.json` (top N, default 3)

---

## Language-Specific Implementation

### ESLint (UI)

**Plugin location:** `ui/apps/platform/eslint-plugins/`

**Rule structure:**
```javascript
'rule-name': {
  // Initially generated by review-feedback lint agent.
  // Derived from review feedback; see PRs #123, #456.
  meta: {
    type: 'problem',
    docs: { description: '...' },
    schema: []
  },
  create(context) {
    return {
      JSXOpeningElement(node) { /* ... */ }
    };
  }
}
```

**Tests:** Use ESLint `RuleTester` (see reference in `eslint-plugins/`)

**Performance:** `npm run lint:profile -- <paths>` from `ui/apps/platform/` runs ESLint with `TIMING=1` and `--no-cache` so rule timing is visible. For full UI impact, run `npm run lint:profile -- .`.

**Directory exclusions:** Add to `ignores:` arrays in `eslint.config.js` if legacy violations exist

### Go Analyzers

**Location:** `tools/roxvet/analyzers/{name}/analyzer.go`

**Structure:**
```go
package name

// Initially generated by review-feedback lint agent.
// Derived from review feedback; see PRs #123, #456.
var Analyzer = &analysis.Analyzer{
  Name: "name",
  Doc: "...",
  Requires: []*analysis.Analyzer{inspect.Analyzer},
  Run: run,
}

func run(pass *analysis.Pass) (interface{}, error) {
  // ...
}
```

**Tests:** `testdata/src/{package}/` with `// want "diagnostic"` comments

**Registration:** Import in `tools/roxvet/roxvet.go`, add to `unitchecker.Main()`

---

## Validation Requirements

### Historical Validation

**Mandatory:** For every motivating example, demonstrate:
```
bad (reviewed) version → FAIL
good (accepted) version → PASS
```

Create fixtures representing before/after when exact replay is impractical.

### Current Repository Validation

**Run rule across all current source.**

**No violations:** Good preventative candidate.

**Small number:** Strengthens the case; report them.

**Large number:** Determine if rule is too broad, has false positives, or reveals genuine legacy debt. If legacy, use directory exclusions (document every exclusion).

### Performance Validation

**ESLint:** `npm run lint:profile -- <paths>` from `ui/apps/platform/` - inspect the `TIMING=1` rule table and reject rules contributing >5% of total runtime without clear justification.

**Go:** Standard benchmark profiling.

---

## Output: Candidate Report

**Format:** Markdown report describing each candidate.

**Sections:**
- Engineering invariant
- Why this candidate exists (PRs)
- Acceptance evidence
- Existing-rule analysis
- Proposed enforcement
- Historical validation results
- Current repository findings
- False-positive evaluation
- Performance impact
- Recommendation: ACCEPT / REVISE / REJECT

**Valid outcome:** "No lint rules are recommended from this corpus."

---

## Success Criteria

- High-confidence rules only
- Low false-positive rates
- Meaningful correctness/security/architecture value
- Low runtime overhead
- Understandable provenance
- Rules that coding agents can react to deterministically

**Do not optimize for number of rules generated.**

---

## Helper Scripts

Supporting scripts in `.claude/skills/review-feedback-lint/`:

- `collect-prs.py` - GitHub PR corpus collection into SQLite
- `filter-corpus.py` - Deterministic noise reduction recorded in SQLite
- `corpus-status.py` - Inspect and update comment triage status (`undecided`, `considered`, `rejected`, `incorporated`)
- `corpus_db.py` - Shared SQLite schema and persistence helpers
- `analyze-acceptance.py` - LLM-based acceptance classification
- `extract-invariants.py` - LLM-based pattern extraction
- `search-existing.py` - Search current lint configurations
- `gate-lintability.py` - LLM-based candidate filtering
- `rank-candidates.py` - Score and rank candidates
- `generate-eslint-rule.py` - Scaffold ESLint rule implementation
- `generate-go-analyzer.py` - Scaffold Go analyzer implementation
- `validate-historical.py` - Run before/after test cases
- `validate-current.py` - Scan current codebase
- `validate-performance.py` - Measure rule performance

Each script operates on standardized JSON formats and supports `--help`.
