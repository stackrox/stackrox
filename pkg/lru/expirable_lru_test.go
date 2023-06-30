package lru

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/sync"
)

func getRand(tb testing.TB) int64 {
	out, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		tb.Fatal(err)
	}
	return out.Int64()
}

func BenchmarkExpirableLRU_Rand_NoExpire(b *testing.B) {
	l := NewExpirableLRU[int64, int64](8192, nil, 0)

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		trace[i] = getRand(b) % 32768
	}

	b.ResetTimer()

	var hit, miss int
	for i := 0; i < 2*b.N; i++ {
		if i%2 == 0 {
			l.Add(trace[i], trace[i])
		} else {
			if _, ok := l.Get(trace[i]); ok {
				hit++
			} else {
				miss++
			}
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
}

func BenchmarkExpirableLRU_Freq_NoExpire(b *testing.B) {
	l := NewExpirableLRU[int64, int64](8192, nil, 0)

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		if i%2 == 0 {
			trace[i] = getRand(b) % 16384
		} else {
			trace[i] = getRand(b) % 32768
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.Add(trace[i], trace[i])
	}
	var hit, miss int
	for i := 0; i < b.N; i++ {
		if _, ok := l.Get(trace[i]); ok {
			hit++
		} else {
			miss++
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(hit+miss))
}

func BenchmarkExpirableLRU_Rand_WithExpire(b *testing.B) {
	l := NewExpirableLRU[int64, int64](8192, nil, time.Millisecond*10)
	defer l.Close()

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		trace[i] = getRand(b) % 32768
	}

	b.ResetTimer()

	var hit, miss int
	for i := 0; i < 2*b.N; i++ {
		if i%2 == 0 {
			l.Add(trace[i], trace[i])
		} else {
			if _, ok := l.Get(trace[i]); ok {
				hit++
			} else {
				miss++
			}
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(hit+miss))
}

func BenchmarkExpirableLRU_Freq_WithExpire(b *testing.B) {
	l := NewExpirableLRU[int64, int64](8192, nil, time.Millisecond*10)
	defer l.Close()

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		if i%2 == 0 {
			trace[i] = getRand(b) % 16384
		} else {
			trace[i] = getRand(b) % 32768
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.Add(trace[i], trace[i])
	}
	var hit, miss int
	for i := 0; i < b.N; i++ {
		if _, ok := l.Get(trace[i]); ok {
			hit++
		} else {
			miss++
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(hit+miss))
}

func TestExpirableLRUInterface(t *testing.T) {
	var _ Cache[int, int] = &expirableLRU[int, int]{}
}

func TestExpirableLRUNoPurge(t *testing.T) {
	lc := NewExpirableLRU[string, string](10, nil, 0)

	lc.Add("key1", "val1")
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	v, ok := lc.Peek("key1")
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

	if !lc.Contains("key1") {
		t.Fatalf("should contain key1")
	}
	if lc.Contains("key2") {
		t.Fatalf("should not contain key2")
	}

	v, ok = lc.Peek("key2")
	if v != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}

	if !reflect.DeepEqual(lc.Keys(), []string{"key1"}) {
		t.Fatalf("value differs from expected")
	}

	if lc.Resize(0) != 0 {
		t.Fatalf("evicted count differs from expected")
	}
	if lc.Resize(2) != 0 {
		t.Fatalf("evicted count differs from expected")
	}
	lc.Add("key2", "val2")
	if lc.Resize(1) != 1 {
		t.Fatalf("evicted count differs from expected")
	}
}

func TestExpirableMultipleClose(t *testing.T) {
	lc := NewExpirableLRU[string, string](10, nil, 0)
	lc.Close()
	// should not panic
	lc.Close()
}

func TestExpirableLRUWithPurge(t *testing.T) {
	var evicted []string
	onExpire := func(key string, val string) { evicted = append(evicted, key, val) }
	ttl := 150 * time.Millisecond
	lc := NewTestExpirableLRU[string, string](t, 10, onExpire, ttl)
	defer lc.Close()

	k, v, ok := lc.GetOldest()
	if k != "" {
		t.Fatalf("should be empty")
	}
	if v != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}

	lc.Add("key1", "val1")

	// Ensure the expiration cleanup loop goes over all timeout buckets,
	// but does not touch the data as it has not expired yet
	lc.TriggerExpiration(t)

	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	v, ok = lc.Get("key1")
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

	// Change the cached item expiry date and ensure the cleanup loop goes over all timeout buckets,
	// as the expiration should now be in the past, the item should be subjected to the expiration
	// callback
	lc.ExpireItem(t, "key1")
	lc.TriggerExpiration(t)

	v, ok = lc.Get("key1")
	if ok {
		t.Fatalf("should be false")
	}
	if v != "" {
		t.Fatalf("should be nil")
	}

	if lc.Len() != 0 {
		t.Fatalf("length differs from expected")
	}
	if !reflect.DeepEqual(evicted, []string{"key1", "val1"}) {
		t.Fatalf("value differs from expected")
	}

	// add new entry
	lc.Add("key2", "val2")
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	k, v, ok = lc.GetOldest()
	if k != "key2" {
		t.Fatalf("value differs from expected")
	}
	if v != "val2" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

	// DeleteExpired, nothing deleted
	lc.underlying.deleteExpired()
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}
	if !reflect.DeepEqual(evicted, []string{"key1", "val1"}) {
		t.Fatalf("value differs from expected")
	}

	// Purge, cache should be clean
	lc.Purge()
	if lc.Len() != 0 {
		t.Fatalf("length differs from expected")
	}
	if !reflect.DeepEqual(evicted, []string{"key1", "val1", "key2", "val2"}) {
		t.Fatalf("value differs from expected")
	}
}

func TestExpirableLRUWithPurgeEnforcedBySize(t *testing.T) {
	lc := NewExpirableLRU[string, string](10, nil, time.Hour)
	defer lc.Close()

	for i := 0; i < 100; i++ {
		i := i
		lc.Add(fmt.Sprintf("key%d", i), fmt.Sprintf("val%d", i))
		v, ok := lc.Get(fmt.Sprintf("key%d", i))
		if v != fmt.Sprintf("val%d", i) {
			t.Fatalf("value differs from expected")
		}
		if !ok {
			t.Fatalf("should be true")
		}
		if lc.Len() > 20 {
			t.Fatalf("length should be less than 20")
		}
	}

	if lc.Len() != 10 {
		t.Fatalf("length differs from expected")
	}
}

func TestExpirableLRUConcurrency(t *testing.T) {
	lc := NewExpirableLRU[string, string](0, nil, 0)
	wg := sync.WaitGroup{}
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func(i int) {
			lc.Add(fmt.Sprintf("key-%d", i/10), fmt.Sprintf("val-%d", i/10))
			wg.Done()
		}(i)
	}
	wg.Wait()
	if lc.Len() != 100 {
		t.Fatalf("length differs from expected")
	}
}

func TestExpirableLRUInvalidateAndEvict(t *testing.T) {
	var evicted int
	lc := NewExpirableLRU(-1, func(_, _ string) { evicted++ }, 0)

	lc.Add("key1", "val1")
	lc.Add("key2", "val2")

	val, ok := lc.Get("key1")
	if !ok {
		t.Fatalf("should be true")
	}
	if val != "val1" {
		t.Fatalf("value differs from expected")
	}
	if evicted != 0 {
		t.Fatalf("value differs from expected")
	}

	lc.Remove("key1")
	if evicted != 1 {
		t.Fatalf("value differs from expected")
	}
	val, ok = lc.Get("key1")
	if val != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}
}

func TestLoadingExpired(t *testing.T) {
	ttl := 5 * time.Millisecond
	lc := NewTestExpirableLRU[string, string](t, 0, nil, ttl)
	defer lc.Close()

	lc.Add("key1", "val1")
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	v, ok := lc.Peek("key1")
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

	v, ok = lc.Get("key1")
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

	lc.ExpireItem(t, "key1")
	lc.TriggerExpiration(t)
	if lc.Len() != 0 {
		t.Fatalf("length differs from expected")
	}

	v, ok = lc.Peek("key1")
	if v != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}

	v, ok = lc.Get("key1")
	if v != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}
}

func TestExpirableLRURemoveOldest(t *testing.T) {
	lc := NewExpirableLRU[string, string](2, nil, 0)

	k, v, ok := lc.RemoveOldest()
	if k != "" {
		t.Fatalf("should be empty")
	}
	if v != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}

	ok = lc.Remove("non_existent")
	if ok {
		t.Fatalf("should be false")
	}

	lc.Add("key1", "val1")
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	v, ok = lc.Get("key1")
	if !ok {
		t.Fatalf("should be true")
	}
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}

	if !reflect.DeepEqual(lc.Keys(), []string{"key1"}) {
		t.Fatalf("value differs from expected")
	}
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	lc.Add("key2", "val2")
	if !reflect.DeepEqual(lc.Keys(), []string{"key1", "key2"}) {
		t.Fatalf("value differs from expected")
	}
	if lc.Len() != 2 {
		t.Fatalf("length differs from expected")
	}

	k, v, ok = lc.RemoveOldest()
	if k != "key1" {
		t.Fatalf("value differs from expected")
	}
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

	if !reflect.DeepEqual(lc.Keys(), []string{"key2"}) {
		t.Fatalf("value differs from expected")
	}
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}
}
