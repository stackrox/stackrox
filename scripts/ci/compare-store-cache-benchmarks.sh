#!/usr/bin/env bash
set -euo pipefail

# Compare selected read benchmarks using one machine and one Postgres server.
# The server must be dedicated to this comparison while measurements run.
if [[ $# -lt 1 || $# -gt 2 ]]; then
    echo "Usage: $0 BASE_REF [CANDIDATE_REF]" >&2
    exit 1
fi

repo_root=$(git rev-parse --show-toplevel)
base_commit=$(git rev-parse --verify --end-of-options "${1}^{commit}")
candidate_commit=$(git rev-parse --verify --end-of-options "${2:-HEAD}^{commit}")
rounds=${BENCH_ROUNDS:-10}
bench_time=${BENCHTIME:-1s}
benchstat_bin=${BENCHSTAT_BIN:-benchstat}
if [[ ! "$rounds" =~ ^[1-9][0-9]*$ ]]; then
    echo "BENCH_ROUNDS must be a positive integer" >&2
    exit 1
fi
command -v "$benchstat_bin" >/dev/null

output_dir=${BENCH_OUTPUT_DIR:-$(mktemp -d "${TMPDIR:-/tmp}/store-cache-results.XXXXXX")}
mkdir -p "$output_dir"
output_dir=$(cd "$output_dir" && pwd)
if [[ -e "$output_dir/master.txt" || -e "$output_dir/pr.txt" ]]; then
    echo "Use a fresh BENCH_OUTPUT_DIR to avoid mixing measurements" >&2
    exit 1
fi
work_dir=$(mktemp -d "${TMPDIR:-/tmp}/store-cache-bench.XXXXXX")
cleanup() {
    for variant in master pr; do
        if [[ -d "$work_dir/$variant" ]]; then
            git -C "$repo_root" worktree remove --force "$work_dir/$variant"
        fi
    done
    rm -rf "$work_dir"
}
trap cleanup EXIT

git -C "$repo_root" worktree add --detach "$work_dir/master" "$base_commit"
git -C "$repo_root" worktree add --detach "$work_dir/pr" "$candidate_commit"

# Build both variants before timing anything, with the same toolchain/settings.
go_version=$(go env GOVERSION)
export GOTOOLCHAIN="$go_version"
export GOMAXPROCS="${GOMAXPROCS:-4}"
export CGO_ENABLED="${CGO_ENABLED:-1}"
export ROX_POSTGRES_TEST_KEEP_DB=false

# package;shared benchmark source;benchmark selector
benchmarks=(
    'central/cluster/datastore;datastore_sac_bench_test.go;^BenchmarkGetClustersForSAC$'
    'central/namespace/datastore;datastore_sac_bench_test.go;^BenchmarkGetNamespacesForSAC$'
    'central/deployment/datastore;datastore_bench_test.go;^BenchmarkSearchAllDeployments$'
    'central/pod/datastore;datastore_bench_test.go;^BenchmarkSearchAllPods$'
    'central/image/service;service_impl_benchmark_test.go;^BenchmarkService_Export$'
    'central/reports/scheduler/v2/reportgenerator;report_gen_bench_flat_data_model_test.go;^Benchmark(FlatDataModelReportGenerator|EntityScopeReportGenerator)$'
)

for specification in "${benchmarks[@]}"; do
    IFS=';' read -r package source_file selector <<< "$specification"
    # Use the candidate's benchmark definitions on both revisions, including
    # authorization benchmarks that do not exist on older master revisions.
    git -C "$repo_root" show "$candidate_commit:$package/$source_file" > "$work_dir/common-benchmark.go"
    for variant in master pr; do
        cp "$work_dir/common-benchmark.go" "$work_dir/$variant/$package/$source_file"
    done
done

python3 - "$work_dir" <<'PY'
import pathlib
import re
import sys

work = pathlib.Path(sys.argv[1])
for variant in ("master", "pr"):
    checkout = work / variant
    # Keep each revision's schema template separate when pgtest runs in CI.
    helper = checkout / "pkg/postgres/pgtest/postgres.go"
    helper.write_text(helper.read_text().replace(
        'templateDBName = "test_template"',
        f'templateDBName = "bench_{variant}_{work.name.rsplit(".", 1)[-1].lower()}"',
    ))
    for entity, method in (("cluster", "GetClustersForSAC"), ("namespace", "GetNamespacesForSAC")):
        package = checkout / f"central/{entity}/datastore"
        if re.search(rf"{method}\(\s*\)", (package / "datastore.go").read_text()):
            benchmark = package / "datastore_sac_bench_test.go"
            # Only adapt the old method signature; fixtures and timing stay shared.
            benchmark.write_text(benchmark.read_text()
                                 .replace(f"{method}(ctx)", f"{method}()")
                                 .replace("\t\t\tctx := context.Background()\n", ""))
PY

{
    echo "Master: $base_commit"
    echo "Candidate: $candidate_commit"
    echo "Toolchain: $go_version"
    echo "GOMAXPROCS: $GOMAXPROCS"
    echo "GOEXPERIMENT: ${GOEXPERIMENT:-}"
    echo "Rounds: $rounds; benchtime: $bench_time"
    uname -a
    if command -v lscpu >/dev/null; then lscpu; fi
} > "$output_dir/environment.txt"

for variant in master pr; do
    mkdir -p "$work_dir/bin/$variant"
    for specification in "${benchmarks[@]}"; do
        IFS=';' read -r package source_file selector <<< "$specification"
        echo "Building $variant $package"
        (
            cd "$work_dir/$variant"
            go test -p 2 -tags test,sql_integration -c \
                -o "$work_dir/bin/$variant/${package//\//_}.test" "./$package"
        ) >> "$output_dir/build-$variant.log" 2>&1
    done
done

run_benchmark() {
    local variant=$1 package=$2 selector=$3 destination=$4 duration=$5
    (
        cd "$work_dir/$variant/$package"
        "$work_dir/bin/$variant/${package//\//_}.test" \
            -test.run='^$' -test.bench="$selector" -test.benchtime="$duration" \
            -test.benchmem -test.count=1 -test.timeout=10m
    ) >> "$destination" 2>&1
}

for specification in "${benchmarks[@]}"; do
    IFS=';' read -r package source_file selector <<< "$specification"
    echo "Warming up $package"
    for variant in master pr; do
        run_benchmark "$variant" "$package" "$selector" "$output_dir/warmup-$variant.log" 1x
    done
    for ((round = 1; round <= rounds; round++)); do
        order=(master pr)
        if ((round % 2 == 0)); then order=(pr master); fi
        echo "Measuring $package, round $round/$rounds: ${order[*]}"
        for variant in "${order[@]}"; do
            run_benchmark "$variant" "$package" "$selector" "$output_dir/$variant.txt" "$bench_time"
        done
    done
done

python3 - "$output_dir" "$rounds" <<'PY'
import collections
import pathlib
import re
import sys

output = pathlib.Path(sys.argv[1])
rounds = int(sys.argv[2])
observations = {}
for variant in ("master", "pr"):
    counts = collections.Counter()
    package = None
    for line in (output / f"{variant}.txt").read_text().splitlines():
        if line.startswith("pkg: "):
            package = line[5:]
        match = re.match(r"^(Benchmark\S+)\s+\d+\s+[\d.]+\s+ns/op", line)
        if match:
            counts[(package, match[1])] += 1
    if len({package for package, _ in counts}) != 6 or any(n != rounds for n in counts.values()):
        raise SystemExit(f"Missing benchmark results or incorrect sample count for {variant}: {counts}")
    observations[variant] = counts.keys()
if observations["master"] != observations["pr"]:
    raise SystemExit("Master and PR benchmark cases do not match")
PY

# Avoid unwieldy absolute paths in benchstat's column headings.
(
    cd "$output_dir"
    "$benchstat_bin" master.txt pr.txt > comparison.txt
)
{
    echo '## Store cache benchmark comparison'
    echo
    echo "Master: \`$base_commit\`; candidate: \`$candidate_commit\`."
    echo
    echo "$rounds alternating rounds at $bench_time per benchmark, on the same runner and PostgreSQL instance."
    echo 'Each process creates fresh test data. Fixtures have the same shape and size; generated IDs may differ.'
    echo 'Results describe these read workloads; they do not measure retained cache memory or production concurrency.'
    echo
    echo '```text'
    cat "$output_dir/comparison.txt"
    echo '```'
} > "$output_dir/summary.md"
echo "Results: $output_dir"
