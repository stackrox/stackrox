#!/usr/bin/env bash
# Experiment: Does the mtime value on source files affect Go test cache lookup speed?
#
# Methodology:
# 1. For each mtime timestamp:
#    a. Clean GOCACHE
#    b. Set all relevant source files to that mtime
#    c. Run tests once (cold - seeds the cache)
#    d. Run tests 3 more times (warm - measures cache hit speed)
# 2. Compare warm run times across different mtime values
#
# The hypothesis: different mtime values might produce different cache lookup
# times due to:
# - Time formatting speed differences in fmt.Sprintf("%v", modTime)
# - Filesystem stat() return speed differences
# - SHA256 hash input pattern differences (unlikely)

set -euo pipefail

cd /Users/house/dev/stack/stackrox

# Test packages - chosen for ~75s cold run time
TEST_PKGS=(
  ./pkg/concurrency/...
  ./pkg/booleanpolicy/...
  ./pkg/stringutils/...
  ./pkg/sliceutils/...
  ./pkg/renderer/...
  ./pkg/ioutils/...
  ./sensor/kubernetes/listener/resources/...
)

# Timestamps to test (touch -t format: YYYYMMDDhhmm)
# Categories:
# 1. All-zeros dates (minimal numeric values)
# 2. Unix epoch and near-epoch
# 3. Round dates
# 4. Max-digit dates (lots of 9s)
# 5. Current-ish dates
# 6. Random/interesting patterns
TIMESTAMPS=(
  "200101010000"   # Baseline - what we currently use
  "197001010000"   # Unix epoch (internal value = 0 on unix)
  "198001010000"   # 10 years after epoch
  "200001010000"   # Y2K (lots of zeros)
  "199912312359"   # Pre-Y2K max (lots of 9s)
  "202301150830"   # Recent date with varied digits
  "202512312359"   # Future date, max month/day/hour/min
  "197001020000"   # Unix epoch + 1 day (86400 seconds internally)
  "200707070707"   # Repeating 7s
  "201111111111"   # Repeating 1s
  "199901010000"   # 1999 start
  "202601010000"   # Future round date
)

RESULTS_FILE="/Users/house/dev/stack/stackrox/.claude/mtime-experiment-results.csv"
WARM_RUNS=3

echo "timestamp,run_type,run_number,wall_seconds,cached_packages,total_packages" > "$RESULTS_FILE"

# Collect all source files that affect the test packages
echo "Collecting source file list..."
SOURCE_FILES=()
while IFS= read -r f; do
  SOURCE_FILES+=("$f")
done < <(find pkg/concurrency pkg/booleanpolicy pkg/stringutils pkg/sliceutils pkg/renderer pkg/ioutils sensor/kubernetes/listener/resources -name '*.go' 2>/dev/null)
echo "Found ${#SOURCE_FILES[@]} Go source files"

# Also include go.mod and go.sum as they affect cache
SOURCE_FILES+=("go.mod" "go.sum")

run_tests() {
  local label="$1"
  local ts="$2"
  local run_type="$3"  # cold or warm
  local run_num="$4"

  local start end elapsed cached total
  start=$(python3 -c 'import time; print(time.time())')

  # Run tests: cold uses -count=1 to force execution and seed cache,
  # warm omits -count=1 to allow test cache hits (what we're measuring)
  local output
  if [[ "$run_type" == "cold" ]]; then
    output=$(go test -count=1 "${TEST_PKGS[@]}" 2>&1) || true
  else
    output=$(go test "${TEST_PKGS[@]}" 2>&1) || true
  fi

  end=$(python3 -c 'import time; print(time.time())')
  elapsed=$(python3 -c "print(f'{$end - $start:.3f}')")

  # Count cached vs total packages
  cached=$(echo "$output" | grep -c '(cached)' || true)
  total=$(echo "$output" | grep -cE '^ok\s' || true)

  echo "$ts,$run_type,$run_num,$elapsed,$cached,$total" >> "$RESULTS_FILE"
  printf "  %-6s #%d: %7ss  (cached: %d/%d)\n" "$run_type" "$run_num" "$elapsed" "$cached" "$total"
}

echo ""
echo "=== Go Test Cache mtime Experiment ==="
echo "=== Test packages: ${#TEST_PKGS[@]} package patterns ==="
echo "=== Warm runs per timestamp: $WARM_RUNS ==="
echo "=== Results: $RESULTS_FILE ==="
echo ""

for ts in "${TIMESTAMPS[@]}"; do
  echo "--- Testing mtime: $ts ---"

  # Step 1: Clean GOCACHE to ensure cold start
  go clean -testcache

  # Step 2: Set mtimes on all source files
  for f in "${SOURCE_FILES[@]}"; do
    touch -t "$ts" "$f" 2>/dev/null || true
  done

  # Step 3: Cold run (seeds cache)
  run_tests "$ts" "$ts" "cold" 1

  # Step 4: Warm runs (tests cache hit speed)
  for i in $(seq 1 $WARM_RUNS); do
    run_tests "$ts" "$ts" "warm" "$i"
  done

  echo ""
done

# Restore mtimes to current time
echo "Restoring file mtimes to current time..."
for f in "${SOURCE_FILES[@]}"; do
  touch "$f" 2>/dev/null || true
done

echo ""
echo "=== Experiment Complete ==="
echo "Results saved to: $RESULTS_FILE"
echo ""

# Print summary
echo "=== Summary: Average Warm Run Times ==="
echo ""
printf "%-16s  %10s  %10s  %10s  %10s\n" "Timestamp" "Warm Avg" "Warm Min" "Warm Max" "Cold"
echo "----------------  ----------  ----------  ----------  ----------"

python3 << 'PYEOF'
import csv
from collections import defaultdict

results = defaultdict(lambda: {"warm": [], "cold": []})
with open("/Users/house/dev/stack/stackrox/.claude/mtime-experiment-results.csv") as f:
    reader = csv.DictReader(f)
    for row in reader:
        ts = row["timestamp"]
        rt = row["run_type"]
        results[ts][rt].append(float(row["wall_seconds"]))

rows = []
for ts, data in sorted(results.items()):
    warm = data["warm"]
    cold = data["cold"]
    if warm:
        avg = sum(warm) / len(warm)
        mn = min(warm)
        mx = max(warm)
    else:
        avg = mn = mx = 0
    cold_time = cold[0] if cold else 0
    rows.append((ts, avg, mn, mx, cold_time))

# Sort by average warm time
rows.sort(key=lambda r: r[1])

for ts, avg, mn, mx, cold_time in rows:
    print(f"{ts:<16}  {avg:>10.3f}  {mn:>10.3f}  {mx:>10.3f}  {cold_time:>10.3f}")

print()
fastest = rows[0]
slowest = rows[-1]
diff = slowest[1] - fastest[1]
pct = (diff / fastest[1]) * 100 if fastest[1] > 0 else 0
print(f"Range: {diff:.3f}s ({pct:.1f}%) between fastest and slowest avg warm run")
print(f"Fastest: {fastest[0]} ({fastest[1]:.3f}s avg)")
print(f"Slowest: {slowest[0]} ({slowest[1]:.3f}s avg)")
PYEOF
