# NetworkConn Binary Hash Migration Plan

## 🎯 **Objective**

Convert `NetworkConn` from string-based keys to `BinaryHash`-based keys for memory optimization, achieving the same 75% memory reduction as implemented for endpoints.

## 📊 **Current State Analysis**

### Key Components Using String Keys:
1. **`NetworkConn.Key()`** - Returns `string` for connection identification
2. **`connectionsDeduper`** - Uses `*set.StringSet` for tracking open connections
3. **`closedConnTimestamps`** - Uses `map[string]closedConnEntry` for afterglow tracking
4. **`categorizeUpdate()`** - Takes `connKey string` parameter
5. **Multiple helper functions** - Accept string keys for connection processing

### Files Requiring Changes:
- `sensor/common/networkflow/manager/indicator/indicator.go`
- `sensor/common/networkflow/manager/indicator/key.go`
- `sensor/common/networkflow/updatecomputer/transition_based.go`
- `sensor/common/networkflow/manager/indicator/key_test.go`

## 🛠️ **Implementation Plan**

### **Phase 1: Foundation Setup**

#### Step 1.1: Add BinaryHashSet type alias to transition_based.go
```go
// Add near the top of the file, after imports
// BinaryHashSet is a set of BinaryHash values for memory-efficient connection tracking.
type BinaryHashSet = set.Set[indicator.BinaryHash]
```

**Rationale:** Keep changes localized to the network flow package, avoid modifying the generic set package. This follows the principle of minimal external dependencies and keeps the migration self-contained.

#### Step 1.2: Add NetworkConn.BinaryKey() method to indicator.go
```go
// Add after NetworkConn.Key() method
// BinaryKey generates a binary hash for memory-efficient storage in dedupers
func (i *NetworkConn) BinaryKey() BinaryHash {
	return i.binaryKeyHash()
}
```

#### Step 1.3: Add NetworkConn.binaryKeyHash() method to key.go
```go
// Add after NetworkConn.keyHash() method
// binaryKeyHash produces a binary hash that uniquely identifies a given NetworkConn indicator.
// This is a memory-optimized implementation using direct hash generation without string conversion.
func (i *NetworkConn) binaryKeyHash() BinaryHash {
	h := fnv.New64a()
	hashStrings(h, i.SrcEntity.ID, i.DstEntity.ID)
	hashPortAndProtocol(h, i.DstPort, i.Protocol)

	var result [8]byte
	binary.BigEndian.PutUint64(result[:], h.Sum64())
	return result
}
```

### **Phase 2: Core Data Structure Migration**

#### Step 2.1: Update TransitionBased struct fields in transition_based.go
```go
// Change from:
connectionsDeduperMutex sync.RWMutex
connectionsDeduper      *set.StringSet

// To:
connectionsDeduperMutex sync.RWMutex
connectionsDeduper      *BinaryHashSet
```

```go
// Change from:
closedConnTimestamps       map[string]closedConnEntry

// To:
closedConnTimestamps       map[indicator.BinaryHash]closedConnEntry
```

#### Step 2.2: Update NewTransitionBased() constructor
```go
// Change from:
connectionsDeduper: newStringSetPtr(),

// To:
connectionsDeduper: newBinaryHashSetPtr(),
```

#### Step 2.3: Add helper function for BinaryHashSet pointer
```go
// Add after newStringSetPtr()
func newBinaryHashSetPtr() *BinaryHashSet {
	s := set.NewSet[indicator.BinaryHash]()
	return &s
}
```

#### Step 2.4: Update required imports
```go
// Add import for BinaryHash type
"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
```

### **Phase 3: Function Signature Updates**

#### Step 3.1: Update categorizeUpdate() function signature
```go
// Change from:
func categorizeUpdate(prevTS, currTS timestamp.MicroTS, prevTsFound bool, key string,
	deduper *set.StringSet, deduperMutex *sync.RWMutex) (update bool, tt TransitionType)

// To:
func categorizeUpdate(prevTS, currTS timestamp.MicroTS, prevTsFound bool, key indicator.BinaryHash,
	deduper *BinaryHashSet, deduperMutex *sync.RWMutex) (update bool, tt TransitionType)
```

#### Step 3.2: Update lookupPrevTimestamp() function signature
```go
// Change from:
func (c *TransitionBased) lookupPrevTimestamp(connKey string) (found bool, prevTS timestamp.MicroTS)

// To:
func (c *TransitionBased) lookupPrevTimestamp(connKey indicator.BinaryHash) (found bool, prevTS timestamp.MicroTS)
```

#### Step 3.3: Update storeClosedConnectionTimestamp() function signature
```go
// Change from:
func (c *TransitionBased) storeClosedConnectionTimestamp(
	connKey string, closedTS timestamp.MicroTS, closedConnRememberDuration time.Duration)

// To:
func (c *TransitionBased) storeClosedConnectionTimestamp(
	connKey indicator.BinaryHash, closedTS timestamp.MicroTS, closedConnRememberDuration time.Duration)
```

### **Phase 4: Method Call Updates**

#### Step 4.1: Update ComputeUpdatedConns() method calls
```go
// Change from:
key := conn.Key()

// To:
key := conn.BinaryKey()
```

### **Phase 4: Memory Calculation Updates**

#### Step 4.1: Update calculateConnectionsDeduperByteSize() method
```go
// Change the calculation logic from string-based to binary hash-based:
func (c *TransitionBased) calculateConnectionsDeduperByteSize() uintptr {
	baseSize := concurrency.WithRLock1(&c.connectionsDeduperMutex, func() uintptr {
		return uintptr(8) + // map reference
			uintptr(c.connectionsDeduper.Cardinality())*8 // 8 bytes per BinaryHash entry
	})
	// Conservative 1.8x multiplier for Go map overhead (same as endpoints deduper)
	return baseSize * 18 / 10
}
```

### **Phase 5: Test Updates**

#### Step 5.1: Update key_test.go for NetworkConn
```go
// Add BinaryKey() tests similar to endpoints:
func TestNetworkConnBinaryKey(t *testing.T) {
	// Test that different connections have different binary keys
	// Test that identical connections have identical binary keys
	// Test binary key length is exactly 8 bytes
}
```

#### Step 5.2: Update any string key assertions to use BinaryKey()
- Search for NetworkConn.Key() calls in tests
- Replace with NetworkConn.BinaryKey() calls
- Update expected values from hex strings to [8]byte arrays

### **Phase 6: Integration and Validation**

#### Step 6.1: Address PeriodicCleanup() method
```go
// Update to use BinaryHash keys:
concurrency.WithLock(&c.closedConnMutex, func() {
	for key, entry := range c.closedConnTimestamps {
		if now.After(entry.expiresAt.GoTime()) {
			delete(c.closedConnTimestamps, key)
		}
	}
})
```

#### Step 6.2: Address any remaining string-based operations
- Search for hardcoded string operations on connection keys
- Update logging/debugging code that might use string keys
- Ensure all map operations use BinaryHash consistently

## 🚨 **Critical Considerations**

### **Memory Impact**
- **Current**: ~32 bytes per connection key (16-char hex string + Go string overhead)
- **After**: 8 bytes per connection key (BinaryHash array)
- **Savings**: ~75% memory reduction for connection tracking

### **Performance Implications**
- **Hash computation**: Slightly faster (no string conversion)
- **Map operations**: Faster (smaller keys, better cache locality)
- **Memory allocations**: Significantly reduced

### **Backward Compatibility**
- This is an internal optimization - no external API changes
- All existing functionality preserved
- No changes to protobuf messages or external interfaces

### **Error-Prone Areas**
1. **Connection lookup logic** - Most complex change due to inability to reconstruct NetworkConn from hash
2. **Test expectations** - All string-based test expectations need binary equivalents
3. **Debug logging** - Any code that logs connection keys needs hex conversion for readability

## 🧪 **Testing Strategy**

### **Unit Tests**
1. **Key generation**: Verify BinaryKey() produces correct hashes
2. **Deduplication**: Ensure binary deduper works identically to string version
3. **Memory calculations**: Validate new byte size calculations
4. **Edge cases**: Test hash collisions (extremely rare but possible)

### **Integration Tests**
1. **End-to-end flow**: Full connection tracking lifecycle with binary keys
2. **Performance benchmarks**: Compare memory usage before/after
3. **Concurrent access**: Ensure thread safety with binary keys

### **Validation Checklist**
- [ ] All tests pass
- [ ] Memory usage reduced by ~75% for connection tracking
- [ ] No functional regressions
- [ ] Performance improved or unchanged
- [ ] Code style consistent with existing patterns

## 📚 **Context: Existing BinaryHash Implementation**

The endpoint deduplication already uses this pattern successfully:

```go
// From indicator.go
type BinaryHash [8]byte

// From transition_based.go  
endpointsDeduper map[indicator.BinaryHash]indicator.BinaryHash

// From testing.go
type EndpointDeduperAssertion func(map[indicator.BinaryHash]indicator.BinaryHash)
```

This migration will create consistency across all deduplication mechanisms in the network flow manager.

## 🎯 **Success Criteria**

1. ✅ All tests pass after implementation
2. ✅ Memory usage for connection tracking reduced by ~75%
3. ✅ Performance maintained or improved
4. ✅ Code follows existing BinaryHash patterns
5. ✅ No functional regressions in connection tracking
6. ✅ Clean, idiomatic Go code with proper error handling

---

**Estimated Implementation Time**: 4-6 hours
**Risk Level**: Medium (due to connection lookup complexity)
**Dependencies**: Requires existing BinaryHash infrastructure (already implemented)
