#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=scripts/lib.sh
source "${SCRIPT_DIR}/lib.sh"

UNIT_TEST_IGNORE="stackrox/rox/sensor/tests|stackrox/rox/operator/tests|stackrox/rox/central/reports/config/store/postgres|stackrox/rox/central/complianceoperator/v2/scanconfigurations/store/postgres|stackrox/rox/central/auth/store/postgres|stackrox/rox/scanner/e2etests"

# Defaults
BASE_REF="origin/master"
HEAD_REF="HEAD"
BENCHCOUNT=6
BENCHTIME="1x"
TIMEOUT="20m"
INCLUDE_DB=false
BENCH_FILTER="."
USER_PACKAGES=""
OUTPUT_DIR=""
KEEP_RESULTS=false

WORKTREE_DIR=""
WORKTREE_HEAD_DIR=""

usage() {
    cat <<EOF
Usage: $(basename "$0") [BASE_REF] [HEAD_REF] [flags]

Compare Go benchmark results between two git refs.
Detects changed packages, runs benchmarks on both refs, and compares with benchstat.

Arguments:
  BASE_REF              Ref to compare against (default: origin/master)
  HEAD_REF              Ref being evaluated (default: HEAD, i.e. current working tree)

Flags:
  --count N             Number of benchmark iterations (default: 6)
  --benchtime T         Go -benchtime flag (default: 1x)
  --timeout T           Per-test timeout (default: 20m)
  --db                  Include sql_integration benchmarks (requires Postgres)
  --packages P,...      Override auto-detection with comma-separated package list
  --bench REGEXP        Filter benchmark names (default: . meaning all)
  --output DIR          Directory for result files (default: temp dir)
  --keep                Preserve worktree and results after exit
  -h, --help            Show this help

Examples:
  $(basename "$0")                                    # current branch vs origin/master
  $(basename "$0") origin/release-4.8                  # current branch vs release-4.8
  $(basename "$0") origin/master my-branch             # explicit refs
  $(basename "$0") --db                                # include DB benchmarks
  $(basename "$0") --count 10 --benchtime 2x           # more iterations
  $(basename "$0") --packages github.com/stackrox/rox/pkg/queue
  $(basename "$0") --keep --output ./bench-results/    # keep results for later
EOF
}

# ── Argument parsing ──────────────────────────────────────────────

_positional=0
while [[ $# -gt 0 ]]; do
    case "$1" in
        --count)     BENCHCOUNT="$2"; shift 2 ;;
        --benchtime) BENCHTIME="$2"; shift 2 ;;
        --timeout)   TIMEOUT="$2"; shift 2 ;;
        --db)        INCLUDE_DB=true; shift ;;
        --no-db)     INCLUDE_DB=false; shift ;;
        --packages)  USER_PACKAGES="$2"; shift 2 ;;
        --bench)     BENCH_FILTER="$2"; shift 2 ;;
        --output)    OUTPUT_DIR="$2"; shift 2 ;;
        --keep)      KEEP_RESULTS=true; shift ;;
        -h|--help)   usage; exit 0 ;;
        -*)          die "Unknown flag: $1. Use --help for usage." ;;
        *)
            case "$_positional" in
                0) BASE_REF="$1" ;;
                1) HEAD_REF="$1" ;;
                *) die "Unexpected argument: $1" ;;
            esac
            _positional=$((_positional + 1))
            shift
            ;;
    esac
done

# ── Safety checks ────────────────────────────────────────────────

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

git rev-parse --verify "${BASE_REF}^{commit}" >/dev/null 2>&1 \
    || die "Invalid base ref: $BASE_REF"
git rev-parse --verify "${HEAD_REF}^{commit}" >/dev/null 2>&1 \
    || die "Invalid head ref: $HEAD_REF"

if ! command -v benchstat &>/dev/null; then
    info "Installing benchstat..."
    go install golang.org/x/perf/cmd/benchstat@latest
fi

if "$INCLUDE_DB"; then
    pg_host="${POSTGRES_HOST:-localhost}"
    pg_port="${POSTGRES_PORT:-5432}"
    if command -v pg_isready &>/dev/null; then
        pg_isready -h "$pg_host" -p "$pg_port" -q 2>/dev/null \
            || die "Postgres is not reachable at ${pg_host}:${pg_port}. Start Postgres or drop --db."
    else
        (echo >/dev/tcp/"$pg_host"/"$pg_port") 2>/dev/null \
            || die "Postgres is not reachable at ${pg_host}:${pg_port}. Start Postgres or drop --db."
    fi
fi

# ── Resolve refs for display ─────────────────────────────────────

BASE_SHA="$(git rev-parse --short "$BASE_REF")"
HEAD_SHA="$(git rev-parse --short "$HEAD_REF")"
info "Comparing $BASE_REF ($BASE_SHA) vs $HEAD_REF ($HEAD_SHA)"

# ── Detect changed packages ─────────────────────────────────────

go_tags="test"
if "$INCLUDE_DB"; then go_tags="test,sql_integration"; fi

if [[ -n "$USER_PACKAGES" ]]; then
    IFS=',' read -ra BENCH_PKGS <<< "$USER_PACKAGES"
    info "Using user-specified packages: ${BENCH_PKGS[*]}"
else
    changed_dirs="$(
        git diff --name-only "${BASE_REF}...${HEAD_REF}" -- '*.go' \
        | xargs -n1 dirname 2>/dev/null \
        | sort -u
    )" || true

    if [[ -z "$changed_dirs" ]]; then
        info "No Go files changed between $BASE_REF and $HEAD_REF."
        exit 0
    fi

    changed_pkgs="$(
        echo "$changed_dirs" \
        | sed 's@^@./@' \
        | xargs go list -tags "$go_tags" -e 2>/dev/null \
        | grep -Ev "$UNIT_TEST_IGNORE"
    )" || true

    if [[ -z "$changed_pkgs" ]]; then
        info "No valid Go packages in the changed files."
        exit 0
    fi

    # Find which changed packages actually have benchmarks
    BENCH_PKGS_PURE=()
    BENCH_PKGS_DB=()

    while IFS= read -r pkg; do
        pkg_dir="$(go list -tags "$go_tags" -f '{{.Dir}}' "$pkg" 2>/dev/null)" || continue
        if ! grep -qrl 'testing\.B' "$pkg_dir" --include='*_test.go' 2>/dev/null; then
            continue
        fi
        if grep -ql '//go:build.*sql_integration' "$pkg_dir"/*_test.go 2>/dev/null; then
            BENCH_PKGS_DB+=("$pkg")
        else
            BENCH_PKGS_PURE+=("$pkg")
        fi
    done <<< "$changed_pkgs"

    BENCH_PKGS=("${BENCH_PKGS_PURE[@]+"${BENCH_PKGS_PURE[@]}"}")
    if "$INCLUDE_DB"; then
        BENCH_PKGS+=("${BENCH_PKGS_DB[@]+"${BENCH_PKGS_DB[@]}"}")
    fi

    if [[ ${#BENCH_PKGS[@]} -eq 0 ]]; then
        info "No benchmarks found in changed packages."
        if [[ ${#BENCH_PKGS_DB[@]} -gt 0 ]]; then
            info "${#BENCH_PKGS_DB[@]} package(s) have sql_integration benchmarks. Re-run with --db to include them."
        fi
        exit 0
    fi
fi

info "Found benchmarks in ${#BENCH_PKGS[@]} package(s):"
for pkg in "${BENCH_PKGS[@]}"; do
    info "  $pkg"
done

# ── Setup results directory ──────────────────────────────────────

if [[ -n "$OUTPUT_DIR" ]]; then
    mkdir -p "$OUTPUT_DIR"
    RESULTS_DIR="$(cd "$OUTPUT_DIR" && pwd)"
else
    RESULTS_DIR="$(mktemp -d "${TMPDIR:-/tmp}/bench-compare-XXXXXX")"
fi

BASE_RESULTS="$RESULTS_DIR/base.txt"
HEAD_RESULTS="$RESULTS_DIR/head.txt"
COMPARISON="$RESULTS_DIR/comparison.txt"

# ── Cleanup trap ─────────────────────────────────────────────────

cleanup() {
    local exit_code=$?
    if [[ -n "$WORKTREE_DIR" ]]; then
        git worktree remove --force "$WORKTREE_DIR" 2>/dev/null || true
    fi
    if [[ -n "$WORKTREE_HEAD_DIR" ]]; then
        git worktree remove --force "$WORKTREE_HEAD_DIR" 2>/dev/null || true
    fi
    if ! "$KEEP_RESULTS" && [[ -z "$OUTPUT_DIR" ]]; then
        rm -rf "$RESULTS_DIR"
    fi
    exit "$exit_code"
}
trap cleanup EXIT

# ── Benchmark runner ─────────────────────────────────────────────

run_benchmarks() {
    local work_dir="$1"
    local output_file="$2"
    local label="$3"

    # Filter packages that exist in this ref
    local valid_pkgs=()
    for pkg in "${BENCH_PKGS[@]}"; do
        if (cd "$work_dir" && go list -tags "$go_tags" -e "$pkg" >/dev/null 2>&1); then
            valid_pkgs+=("$pkg")
        else
            warn "[$label] Package $pkg does not exist in this ref, skipping."
        fi
    done

    if [[ ${#valid_pkgs[@]} -eq 0 ]]; then
        warn "[$label] No valid benchmark packages found."
        touch "$output_file"
        return
    fi

    local parallel_flag=""
    if "$INCLUDE_DB"; then parallel_flag="-p 1"; fi

    info "[$label] Running ${#valid_pkgs[@]} package(s) with -count=$BENCHCOUNT -benchtime=$BENCHTIME..."

    (
        cd "$work_dir"
        export CGO_ENABLED=1
        export GOEXPERIMENT=cgocheck2
        export MUTEX_WATCHDOG_TIMEOUT_SECS=30

        # shellcheck disable=SC2086
        go test \
            $parallel_flag \
            -tags "$go_tags" \
            -run='^$' \
            -bench="$BENCH_FILTER" \
            -benchtime="$BENCHTIME" \
            -benchmem \
            -timeout "$TIMEOUT" \
            -count "$BENCHCOUNT" \
            "${valid_pkgs[@]}" \
            2>&1 | tee "$output_file"
    ) || warn "[$label] Some benchmarks failed. Results may be partial."
}

# ── Run benchmarks on base ref ───────────────────────────────────

WORKTREE_DIR="$(mktemp -d "${TMPDIR:-/tmp}/bench-base-XXXXXX")"
info "Creating worktree for $BASE_REF at $WORKTREE_DIR..."
git worktree add --detach "$WORKTREE_DIR" "$BASE_REF" 2>/dev/null

run_benchmarks "$WORKTREE_DIR" "$BASE_RESULTS" "base"

# ── Run benchmarks on head ref ───────────────────────────────────

if [[ "$HEAD_REF" == "HEAD" ]]; then
    run_benchmarks "$REPO_ROOT" "$HEAD_RESULTS" "head"
else
    WORKTREE_HEAD_DIR="$(mktemp -d "${TMPDIR:-/tmp}/bench-head-XXXXXX")"
    info "Creating worktree for $HEAD_REF at $WORKTREE_HEAD_DIR..."
    git worktree add --detach "$WORKTREE_HEAD_DIR" "$HEAD_REF" 2>/dev/null
    run_benchmarks "$WORKTREE_HEAD_DIR" "$HEAD_RESULTS" "head"
fi

# ── Compare with benchstat ───────────────────────────────────────

echo ""
echo "============================================"
echo "  Benchmark Comparison"
echo "  base: $BASE_REF ($BASE_SHA)"
echo "  head: $HEAD_REF ($HEAD_SHA)"
echo "============================================"
echo ""

if [[ ! -s "$BASE_RESULTS" ]] || [[ ! -s "$HEAD_RESULTS" ]]; then
    die "One or both benchmark result files are empty. Cannot compare."
fi

benchstat "$BASE_RESULTS" "$HEAD_RESULTS" | tee "$COMPARISON"

echo ""
if "$KEEP_RESULTS"; then
    info "Results preserved:"
    info "  Base:       $BASE_RESULTS"
    info "  Head:       $HEAD_RESULTS"
    info "  Comparison: $COMPARISON"
fi
