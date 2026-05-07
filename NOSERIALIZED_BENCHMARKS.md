# NoSerialized Store: Inlined bytea Benchmark Results

**Date:** 2026-04-03
**Platform:** Apple M-series, PostgreSQL 15, Go 1.24, 12 cores
**Data:** 5,000 rows seeded, 5 iterations per benchmark

## PostgreSQL Storage (5,000 rows)

```
                                              total       table       indexes
  Bytea (serialized)                          3,696 kB    2,704 kB    992 kB
  Jsonb                                       5,168 kB    4,480 kB    688 kB
  NoSerialized (inlined bytea)                2,040 kB    1,328 kB    712 kB
```

| Store | Avg Row | Blob/Inlined Col | % of Row |
|-------|---------|------------------|----------|
| Bytea | 512 B | 304 B (serialized) | 59% |
| Jsonb | 871 B | 663 B (serialized) | 76% |
| **NoSerialized** | **256 B** | **30 B** (lineageinfo) | **12%** |

NoSerialized uses **45% less total disk** than bytea and **61% less** than jsonb.

## PostgreSQL EXPLAIN ANALYZE (server-side only)

### Single Row by PK (Index Scan)

| Store | Execution Time | Planning Time | Buffers |
|-------|---------------|---------------|---------|
| Bytea | 0.005 ms | 0.012 ms | 3 shared hit |
| Jsonb | 0.003 ms | 0.007 ms | 3 shared hit |
| **NoSerialized** | **0.004 ms** | **0.009 ms** | **3 shared hit** |

All three are effectively identical for single-row PK lookup.

### 500 Rows by PK List (Seq Scan)

| Store | Execution Time | Planning Time | Buffers |
|-------|---------------|---------------|---------|
| Bytea | 0.328 ms | 0.060 ms | 334 shared hit |
| Jsonb | 0.369 ms | 0.046 ms | 556 shared hit |
| **NoSerialized** | **0.262 ms** | **0.049 ms** | **162 shared hit** |

NoSerialized is **20% faster** than bytea and **29% faster** than jsonb, using **52% fewer buffer hits**.

### Full Table Scan (Walk, 5,000 rows)

| Store | Execution Time | Buffers |
|-------|---------------|---------|
| Bytea | 0.448 ms | 334 shared hit |
| Jsonb | 0.511 ms | 556 shared hit |
| **NoSerialized** | **0.221 ms** | **162 shared hit** |

NoSerialized is **51% faster** than bytea on full scans — half the I/O because rows are half the size.

## Go-side Benchmarks (median of 5 runs)

### Single Upsert (end-to-end)

| Store | Latency | Memory | Allocs |
|-------|---------|--------|--------|
| Bytea | 361 µs | 7,052 B | 153 |
| Jsonb | 320 µs | 9,594 B | 189 |
| **NoSerialized** | **296 µs** | **6,132 B** | **160** |

### UpsertMany (InsertBatch path)

| Batch | Bytea | Jsonb | NoSerialized | NoSer vs Bytea |
|-------|-------|-------|--------------|----------------|
| 10 | 653 µs | 1,347 µs | 949 µs | +45% |
| 100 | 3,013 µs | 6,092 µs | 2,367 µs | **-21%** |
| 500 | 7,836 µs | 15,865 µs | 7,370 µs | **-6%** |
| 1000 | 13,632 µs | — | 13,334 µs | **-2%** |

At batch sizes >= 100, NoSerialized matches or beats bytea. Zero per-row UPDATE fallback.

### UpsertMany Memory (batch=500)

| Store | Memory | Allocs |
|-------|--------|--------|
| Bytea | 2.2 MB | 50,189 |
| Jsonb | 5.9 MB | 71,073 |
| **NoSerialized** | **2.2 MB** | **18,217** |

NoSerialized uses **64% fewer allocations** than bytea at the same memory footprint.

### Write Strategies at 1,000 Objects

| Strategy | Latency | Allocs |
|----------|---------|--------|
| PerRow (1 INSERT/obj) | 17,976 µs | 87,047 |
| **BulkUnnest (single unnest)** | **12,757 µs** | **31,169** |

BulkUnnest is **29% faster** with **64% fewer allocations**.

### Single Get

| Store | Latency | Memory | Allocs |
|-------|---------|--------|--------|
| **Bytea** | **52 µs** | **6,254 B** | **98** |
| Jsonb | 57 µs | 7,926 B | 175 |
| NoSerialized | 69 µs | 36,872 B | 158 |

Bytea wins on reads due to single-column unmarshal. NoSerialized is 1.3x slower.

### GetMany (batch reads)

| Batch | Bytea | Jsonb | NoSerialized | NoSer vs Jsonb |
|-------|-------|-------|--------------|----------------|
| 10 | 57 µs | 135 µs | 80 µs | **-41%** |
| 100 | 196 µs | 707 µs | 272 µs | **-62%** |
| 500 | 781 µs | 3,214 µs | 1,128 µs | **-65%** |

NoSerialized is **2.5-3x faster than jsonb** on batch reads. Gap to bytea is ~1.4x.

### Walk (2,000 rows full scan)

| Store | Latency | Allocs |
|-------|---------|--------|
| Bytea | 1,222 µs | 14,153 |
| NoSerialized | 2,143 µs | 56,226 |

### Count (no data transfer)

| Store | Latency |
|-------|---------|
| Bytea | 75 µs |
| Jsonb | 76 µs |
| NoSerialized | 72 µs |

All identical — confirms the difference is in data transfer/deserialization, not query planning.

## Summary

| Dimension | Winner | Details |
|-----------|--------|---------|
| **Storage size** | NoSerialized | 45% less than bytea, 61% less than jsonb |
| **DB-side scans** | NoSerialized | 51% faster full scan, 52% fewer buffer hits |
| **Single writes** | NoSerialized | 18% faster than bytea |
| **Batch writes (>=100)** | NoSerialized ≈ Bytea | Within 6%, 64% fewer allocs |
| **Single reads** | Bytea | 1.3x faster (single-column unmarshal) |
| **Batch reads** | Bytea > NoSerialized >> Jsonb | NoSerialized 2.5x faster than jsonb |
| **Schema evolution** | Bytea/Jsonb | Blob absorbs proto changes |
| **SQL queryability** | NoSerialized | Every scalar field is a column |

### Key Optimizations Applied
1. **Inlined repeated messages as bytea** — eliminated child table and FetchChildren round trip
2. **Removed deprecated `lineage` field** — eliminated the last non-unnestable column
3. **Pure unnest bulk INSERT** — single statement for all rows, zero per-row fallback
4. **Direct proto field scanning** — scan DB columns directly into proto struct fields
