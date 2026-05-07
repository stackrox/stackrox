# No-Serialized Store Benchmark Results

**Date:** 2026-05-07
**Hardware:** Apple M3 Pro, PostgreSQL 15 (localhost)
**Branch:** `dashrews/no-serialize-experiment-and-design-34568`
**Methodology:** Go `testing.B` with `-benchmem`, `-count=3`. Postgres tests with `EXPLAIN ANALYZE`. All runs on localhost with exclusive access.

---

## Tier 1: Go-Side Micro-Benchmarks

### Single-Object Operations

| Operation | Serialized (bytea) | No-Serialized (inlined bytea) | Arrays | JSONB | Winner |
|-----------|-------------------|-------------------------------|--------|-------|--------|
| **Upsert x1** | 105 µs, 6.2 KB, 153 allocs | 101 µs, 6.1 KB, 150 allocs | 104 µs, 6.2 KB, 153 allocs | 125 µs, 9.6 KB, 189 allocs | **No-serialized** (-4%) |
| **Get x1** | 65 µs, 38 KB, 98 allocs | 63 µs, 40 KB, 155 allocs | 71 µs, 40.6 KB, 157 allocs | 54 µs, 8 KB, 175 allocs | **JSONB** (-17%) |

### Batch Operations

| Operation | Serialized | No-Serialized | Arrays | JSONB | Winner |
|-----------|-----------|---------------|--------|-------|--------|
| **UpsertMany x10** | 1.1 ms, 69 KB | 0.7 ms, 65 KB | 1.2 ms, 72 KB | 1.3 ms, 104 KB | **No-serialized** (-36%) |
| **UpsertMany x100** | 5.3 ms, 584 KB | 3.5 ms, 520 KB | 8.1 ms, 640 KB | 7.7 ms, 1065 KB | **No-serialized** (-34%) |
| **UpsertMany x500** | 13.4 ms, 2.9 MB | 8.3 ms, 2.2 MB | 24.6 ms, 3.2 MB | 14.4 ms, 5.9 MB | **No-serialized** (-38%) |
| **GetMany x10** | 73 µs, 19 KB | 69 µs, 18 KB | 75 µs, 19 KB | 121 µs, 34 KB | **No-serialized** (-5%) |
| **GetMany x100** | 220 µs, 149 KB | 187 µs, 143 KB | 280 µs, 155 KB | 765 µs, 300 KB | **No-serialized** (-15%) |
| **GetMany x500** | 1.7 ms, 964 KB | 1.3 ms, 933 KB | 2.6 ms, 1011 KB | 3.8 ms, 1510 KB | **No-serialized** (-24%) |

### Allocation Summary (UpsertMany x500)

| Variant | Allocs/op | vs. Serialized |
|---------|-----------|---------------|
| Serialized | 50,189 | baseline |
| **No-serialized** | **18,217** | **-64%** |
| Arrays | 30,450 | -39% |
| JSONB | 71,073 | +42% |

---

## Tier 2: Postgres-Side Analysis

### Storage Size

| Metric | No-Serialized (measured) | Serialized (from PR 19669) | Delta |
|--------|-------------------------|---------------------------|-------|
| **5K rows — Total** | 2,008 KB | 3,688 KB | **-46%** |
| **5K rows — Table** | 1,328 KB | 2,632 KB | -50% |
| **5K rows — Index** | 680 KB | 1,048 KB | -35% |
| **5K rows — Toast** | 8 KB | 8 KB | same |
| **Avg row size** | 256 B | 512 B | **-50%** |
| **100K rows — Total** | 39,160 KB | ~73,760 KB (est.) | **-47%** |

### EXPLAIN ANALYZE (5K rows)

| Query Pattern | No-Serialized | Serialized (from PR 19669) | Notes |
|---------------|--------------|---------------------------|-------|
| **PK lookup** | 0.008 ms, 2 buffers | 0.005 ms, 3 buffers | Both fast; no-serialized fewer buffers |
| **IN-list (500 IDs)** | 0.280 ms, 162 buffers | 0.328 ms, 334 buffers | **No-serialized 15% faster, 52% fewer buffers** |
| **Full scan (5K rows)** | 0.312 ms, 162 buffers | 0.448 ms, 334 buffers | **No-serialized 30% faster, 52% fewer buffers** |

### WAL Volume

| Metric | No-Serialized (measured) |
|--------|-------------------------|
| WAL per 10K upserts | 6,556 KB |
| WAL per row | 671 B |
| WAL/row ratio | 2.6x avg row size |

---

## Tier 3: Production-Scale Benchmarks

### Operations on 100K-Row Table

| Operation | No-Serialized | Notes |
|-----------|--------------|-------|
| **GetMany 500 from 100K** | 1.3-2.3 ms/op, 933 KB, 16K allocs | Consistent with Tier 1 ratios |
| **UpsertMany 500 into 100K** | 7.3-9.8 ms/op, 2.6 MB, 26K allocs | Slight overhead vs empty table |

### Scale Impact

| Table Size | UpsertMany x500 | GetMany x500 |
|-----------|-----------------|-------------|
| Empty | 8.3 ms | 1.3 ms |
| 100K rows | 9.8 ms (+18%) | 2.3 ms (+77%) |

The GetMany degradation at 100K rows is expected — index lookups become less efficient as the table grows. The upsert degradation is minimal since it uses unnest-based bulk insert.

---

## Variant Comparison Summary

### When to Use Each Approach

| Scenario | Best Choice | Why |
|----------|------------|-----|
| **High-volume writes** | **No-serialized (inlined bytea)** | 38% faster batch upserts, 64% fewer allocations |
| **Bulk reads (100+ rows)** | **No-serialized (inlined bytea)** | 15-24% faster, smallest rows (fewer buffer hits) |
| **Single-object reads** | JSONB | 17% faster, 79% less memory (single scan) |
| **Storage efficiency** | **No-serialized (inlined bytea)** | 46% less disk, 50% smaller rows |
| **SQL debugging/ad-hoc queries** | JSONB | Human-readable, JSON operators |
| **Repeated field queryability** | Arrays or Child tables | Per-element SQL access |
| **Schema migration simplicity** | **No-serialized** | ALTER TABLE, no re-serialization |

### Repeated Field Strategy Comparison

| Strategy | UpsertMany x500 | GetMany x500 | Use When |
|----------|-----------------|-------------|----------|
| **Inlined bytea** | 8.3 ms (fastest) | 1.3 ms (fastest) | Field always read/written whole, never queried |
| Postgres arrays | 24.6 ms (3x slower) | 2.6 ms (2x slower) | Repeated scalars needing `ANY()` queries |
| Child tables | (tests pass, no benchmark yet) | (tests pass, no benchmark yet) | Per-element JOINs and filtering needed |

### Overall Recommendation

**No-serialized with inlined bytea** is the clear winner for ProcessIndicator-class entities:
- Faster across all batch operations (writes 38% faster, reads 15-24% faster)
- 64% fewer Go-side allocations (reduced GC pressure)
- 46% less storage, 52% fewer Postgres buffer hits
- Simpler schema migrations
- Only trade-off: single-object reads are comparable (not slower, but JSONB edges ahead by 17%)

**JSONB** is best positioned as a debugging/operational tool, not a production optimization. Its batch read performance (2.9x slower at 500 rows) disqualifies it for high-volume workloads.

**Arrays** underperform on both reads and writes for ProcessIndicator. They may be justified for entities where repeated scalar field queryability is a hard requirement.

---

## Entity Decision Framework Application

Using the scoring framework from the design spec, with benchmark data:

| Entity | Score | Recommendation | Rationale |
|--------|-------|---------------|-----------|
| ProcessIndicator | 20 | **No-serialized (inlined bytea)** | Highest write volume, benefits most from 38% upsert improvement |
| NetworkFlow | 21 | **No-serialized (inlined bytea)** | Similar profile to PI, flat schema |
| CVE/Vuln | 18 | **No-serialized (evaluate)** | High read volume benefits from 15-24% GetMany improvement |
| Alert | 15 | Evaluate case-by-case | Complex proto with deep nesting — needs per-entity benchmarks |
| Deployment | 12 | Keep serialized for now | Complex proto, moderate volume, many child fields |
| Policy | 11 | Keep serialized | Low volume, no performance pressure |
