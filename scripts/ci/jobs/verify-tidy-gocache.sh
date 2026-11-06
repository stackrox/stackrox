#!/usr/bin/env bash
# EXPERIMENT (do not merge): verify that a restored GOCACHE does not reintroduce
# the #20369 failure mode in `go mod tidy`.
#
# #20369 (May 2026) removed GOCACHE from check-generated-files because Go's
# module index caches parsed imports keyed by (filename, mtime, size) — not
# content — and the then-current fixed-mtime scheme let a restored GOCACHE
# serve stale import data to `go mod tidy`, which re-added the removed
# dependency github.com/quay/zlog. The mtime scheme was later replaced with
# content-derived (git blob hash) mtimes (#21719), which should close that
# hole. This script tests it directly, replicating the original scenario:
#
#   1. Populate the module index from the pristine tree (the "before" state).
#   2. Remove the only usage of quay/zlog/v2 from the tree and commit it
#      locally (so the blob-hash mtime scheme assigns the new content a new
#      mtime, exactly as a fresh CI checkout of such a commit would).
#   3. Run `go mod tidy` twice on the modified tree: once with the stale
#      GOCACHE (populated in step 1) and once with a fresh GOCACHE.
#   4. Require: identical results, the dependency is correctly dropped, and
#      nothing else changed. Any difference is the #20369 bug resurfacing.

set -euo pipefail
export SHELLOPTS

ZLOG_FILE=scanner/internal/logging/logging.go
STALE_GC="${GOCACHE:?GOCACHE must be set}"
WORK=$(mktemp -d)

step() { echo "::group::verify-tidy: $1"; }
endstep() { echo "::endgroup::"; }

# Content-derived mtime stabilization, verbatim from
# .github/actions/cache-go-dependencies (the scheme under test, from #21719).
# The module index and import cache key entries by (filename, mtime, size);
# this scheme derives mtime from the git blob hash so that content changes
# always re-key (same content -> same mtime, different content -> different
# mtime). Applying it here makes the test faithful: a pass means the scheme
# works, not that random checkout mtimes happened to differ.
stabilize() {
    git ls-files -s | perl -lne '/^120000 \S+ \S+\t(.+)$/ and print $1' \
        | xargs -r touch -t 200101010000
    git ls-files | sed 's|/[^/]*$||' | sort -u | xargs touch -t 200101010000
    git ls-files -s | perl -lne '
        /^100\S+ (\S+) \S+\t(.+)$/ or next;
        my $v = 0;
        $v = ($v * 16 + hex $_) % 31536000 for split //, $1;
        my $ts = 978307200 + $v;
        utime $ts, $ts, $2;'
    echo "stabilized mtimes (blob-hash scheme)"
}

fail() {
    echo "::error::verify-tidy-gocache FAILED: $1"
    exit 1
}

# --- sanity: the expected precondition --------------------------------------
grep -q 'quay/zlog' "$ZLOG_FILE" || fail "precondition: $ZLOG_FILE no longer imports zlog; pick another single-usage dependency"
grep -q 'quay/zlog/v2' go.mod || fail "precondition: go.mod no longer requires zlog"

# --- 1. populate the module index from the pristine tree --------------------
step "populate module index from pristine tree (before state)"
stabilize
go mod tidy
git diff --exit-code HEAD -- go.mod go.sum || fail "precondition: tree is not tidy to begin with"
endstep

# --- 2. remove the only usage of the dependency and commit ------------------
step "remove zlog usage and commit (new blob hash, new content-derived mtime)"
sed -i '/quay\/zlog/d' "$ZLOG_FILE"
! grep -q 'quay/zlog' "$ZLOG_FILE" || fail "sed did not remove the zlog import"
git add "$ZLOG_FILE"
git -c user.email=verify@test -c user.name=verify commit -q -m "verify: remove zlog usage"
stabilize
endstep

# --- 3a. tidy with the stale GOCACHE ----------------------------------------
step "go mod tidy with STALE GOCACHE"
GOCACHE="$STALE_GC" go mod tidy
cp go.mod "$WORK/go.mod.stale"
cp go.sum "$WORK/go.sum.stale"
git checkout -q HEAD -- go.mod go.sum
endstep

# --- 3b. tidy with a fresh GOCACHE ------------------------------------------
step "go mod tidy with FRESH GOCACHE"
FRESH_GC=$(mktemp -d)
GOCACHE="$FRESH_GC" go mod tidy
cp go.mod "$WORK/go.mod.fresh"
cp go.sum "$WORK/go.sum.fresh"
endstep

# --- 4. assertions -----------------------------------------------------------
step "compare results"

# (a) the dependency was correctly dropped by the fresh run (test sensitivity:
#     if this fails, the test construction is broken and any PASS is vacuous)
grep -q 'quay/zlog/v2' "$WORK/go.mod.fresh" && fail "fresh tidy did not drop zlog — test construction broken"
! grep -q 'quay/zlog/v2' "$WORK/go.mod.stale" || fail "stale-cache tidy kept zlog required — this is the #20369 failure mode"

# (b) identical results: the stale cache did not alter tidy behavior
diff "$WORK/go.mod.stale" "$WORK/go.mod.fresh" || fail "stale and fresh tidy produced different go.mod"
diff "$WORK/go.sum.stale" "$WORK/go.sum.fresh" || fail "stale and fresh tidy produced different go.sum"

# (c) the change is scoped: only zlog-related lines differ from the committed state
git diff HEAD -- go.mod | grep -E '^[+-]' | grep -v '^[+-][+-]' | grep -qv 'zlog' \
    && fail "tidy changed go.mod beyond dropping zlog" || true
echo "PASS: stale-cache tidy and fresh tidy agree; zlog correctly dropped; no other changes"
endstep
