# Process Filter Optimization: String Keys → uint64 Hashes

## Summary

Optimized the process filter by replacing string map keys with uint64 hashes using xxhash, resulting in **~37% faster map lookups** and reduced memory overhead.

## Changes

### Before
```go
containersInDeployment map[string]map[string]*level
children map[string]*level
```

### After
```go
h hash.Hash64  // xxhash instance
containersInDeployment map[uint64]map[uint64]*level
children map[uint64]*level
```

## Performance Results

### Map Lookup Comparison
```
BenchmarkHashVsString/uint64-8    75,199,090 ops    15.56 ns/op    0 B/op    0 allocs/op
BenchmarkHashVsString/string-8    50,181,097 ops    24.78 ns/op    0 B/op    0 allocs/op
```

**Improvement: ~37% faster lookups** (15.56 ns vs 24.78 ns)

### Real-World Impact
```
BenchmarkFilter-8    1,000,000 ops    1,268 ns/op    43 B/op    0 allocs/op
                     9.0 MiB max heap    70,502 objects    2% accept rate
```

Typical workload (90% add, 10% delete) with diverse indicators:
- **1.3 μs per operation** (includes hash computation and map operations)
- **Minimal allocations**: 43 B/op with 0 allocs/op (due to filter reuse)
- **Efficient memory**: 9 MiB heap for sustained operations

## Benefits

1. **Faster Lookups**: uint64 map keys are ~37% faster than string keys
2. **Reduced Memory**: 8 bytes vs variable-length strings
3. **Better Cache Locality**: Fixed-size keys fit better in CPU cache
4. **No String Copies**: Hash computed once, no string copying on map operations

## Trade-offs

**Hash Collisions**: Theoretical risk with 64-bit hash, but:
- xxhash has excellent distribution
- Collision probability ~10^-19 for typical workloads
- Filter is ephemeral (not persistent storage)
- Collisions would cause false positives (over-filtering, safe direction)

## Why Not Bloom Filter?

Bloom filters were considered but rejected because:
1. **Deletion Required**: Filter removes dead containers via `UpdateByPod` and `Delete`
2. **Exact Counts Needed**: Tracks exact hit counts per path (`level.hits >= maxExactPathMatches`)
3. **Hierarchical Structure**: Tree-based argument tracking requires navigation
4. **Fan-out Enforcement**: Needs exact count of unique children per level

## Files Modified

- `pkg/process/filter/filter.go` - Core filter implementation with hash-based maps
- `pkg/process/filter/filter_test.go` - Unit tests (updated for uint64 keys)
- `pkg/process/filter/filter_bench_test.go` - Streamlined benchmarks

## Benchmarks

- `BenchmarkFilter` - Comprehensive test covering typical usage (add/delete mix)
- `BenchmarkHashVsString` - Direct uint64 vs string key comparison

## Future Optimizations

1. **String Interning**: Reuse common argument strings
2. **Compact level struct**: Use uint32 for `hits` if limits allow
3. **Lazy Cleanup**: Batch cleanup operations
4. **Metrics**: Add Prometheus metrics for filter hit/miss rates
