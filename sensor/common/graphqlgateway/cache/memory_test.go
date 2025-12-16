package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCache_BasicOperations(t *testing.T) {
	cache := NewMemoryCache(1 * time.Minute)
	defer cache.Close()

	ctx := context.Background()
	key := NewCacheKey("user1", "default", "nginx")
	token := "test-token-12345"

	t.Run("Get on empty cache returns miss", func(t *testing.T) {
		val, ok := cache.Get(ctx, key)
		assert.False(t, ok)
		assert.Empty(t, val)
		assert.Equal(t, int64(1), cache.GetMisses())
	})

	t.Run("Set and Get returns token", func(t *testing.T) {
		cache.Set(ctx, key, token, 5*time.Minute)
		val, ok := cache.Get(ctx, key)
		assert.True(t, ok)
		assert.Equal(t, token, val)
		assert.Equal(t, int64(1), cache.GetHits())
	})

	t.Run("Size returns correct count", func(t *testing.T) {
		assert.Equal(t, 1, cache.Size())
	})

	t.Run("Invalidate removes entry", func(t *testing.T) {
		cache.Invalidate(key)
		val, ok := cache.Get(ctx, key)
		assert.False(t, ok)
		assert.Empty(t, val)
		assert.Equal(t, 0, cache.Size())
	})
}

func TestMemoryCache_TTLExpiration(t *testing.T) {
	cache := NewMemoryCache(100 * time.Millisecond)
	defer cache.Close()

	ctx := context.Background()
	key := NewCacheKey("user1", "default", "")

	t.Run("Entry expires after TTL", func(t *testing.T) {
		cache.Set(ctx, key, "short-lived-token", 200*time.Millisecond)

		// Should be available immediately
		val, ok := cache.Get(ctx, key)
		assert.True(t, ok)
		assert.Equal(t, "short-lived-token", val)

		// Wait for expiration
		time.Sleep(250 * time.Millisecond)

		// Should be expired
		val, ok = cache.Get(ctx, key)
		assert.False(t, ok)
		assert.Empty(t, val)
	})
}

func TestMemoryCache_BackgroundCleanup(t *testing.T) {
	// Short cleanup interval for testing
	cache := NewMemoryCache(100 * time.Millisecond)
	defer cache.Close()

	ctx := context.Background()

	// Add multiple entries with short TTLs
	for i := 0; i < 10; i++ {
		key := NewCacheKey("user"+string(rune(i)), "default", "")
		cache.Set(ctx, key, "token", 50*time.Millisecond)
	}

	assert.Equal(t, 10, cache.Size())

	// Wait for cleanup to run (cleanup interval + TTL + buffer)
	time.Sleep(200 * time.Millisecond)

	// All entries should be cleaned up
	assert.Equal(t, 0, cache.Size())
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	cache := NewMemoryCache(1 * time.Minute)
	defer cache.Close()

	ctx := context.Background()
	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := NewCacheKey("user", "ns", "deploy")
				token := "token-" + string(rune(id)) + "-" + string(rune(j))
				cache.Set(ctx, key, token, 5*time.Minute)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := NewCacheKey("user", "ns", "deploy")
				cache.Get(ctx, key)
			}
		}()
	}

	wg.Wait()

	// Cache should still be functional
	assert.Greater(t, cache.GetHits()+cache.GetMisses(), int64(0))
}

func TestMemoryCache_Clear(t *testing.T) {
	cache := NewMemoryCache(1 * time.Minute)
	defer cache.Close()

	ctx := context.Background()

	// Add multiple entries
	cache.Set(ctx, NewCacheKey("user1", "ns1", ""), "token1", 5*time.Minute)
	cache.Set(ctx, NewCacheKey("user2", "ns2", ""), "token2", 5*time.Minute)
	cache.Set(ctx, NewCacheKey("user3", "ns3", ""), "token3", 5*time.Minute)

	assert.Equal(t, 3, cache.Size())

	cache.Clear()

	assert.Equal(t, 0, cache.Size())
}

func TestMemoryCache_Metrics(t *testing.T) {
	cache := NewMemoryCache(1 * time.Minute)
	defer cache.Close()

	ctx := context.Background()
	key := NewCacheKey("user1", "default", "")

	// Initial state
	assert.Equal(t, int64(0), cache.GetHits())
	assert.Equal(t, int64(0), cache.GetMisses())
	assert.Equal(t, 0.0, cache.GetHitRate())

	// Miss
	cache.Get(ctx, key)
	assert.Equal(t, int64(0), cache.GetHits())
	assert.Equal(t, int64(1), cache.GetMisses())
	assert.Equal(t, 0.0, cache.GetHitRate())

	// Hit
	cache.Set(ctx, key, "token", 5*time.Minute)
	cache.Get(ctx, key)
	assert.Equal(t, int64(1), cache.GetHits())
	assert.Equal(t, int64(1), cache.GetMisses())
	assert.Equal(t, 50.0, cache.GetHitRate())

	// More hits
	cache.Get(ctx, key)
	cache.Get(ctx, key)
	assert.Equal(t, int64(3), cache.GetHits())
	assert.Equal(t, int64(1), cache.GetMisses())
	assert.Equal(t, 75.0, cache.GetHitRate())
}

func TestMemoryCache_ReplaceEntry(t *testing.T) {
	cache := NewMemoryCache(1 * time.Minute)
	defer cache.Close()

	ctx := context.Background()
	key := NewCacheKey("user1", "default", "nginx")

	// Set initial token
	cache.Set(ctx, key, "token1", 5*time.Minute)
	val, ok := cache.Get(ctx, key)
	require.True(t, ok)
	assert.Equal(t, "token1", val)

	// Replace with new token
	cache.Set(ctx, key, "token2", 5*time.Minute)
	val, ok = cache.Get(ctx, key)
	require.True(t, ok)
	assert.Equal(t, "token2", val)

	// Size should still be 1
	assert.Equal(t, 1, cache.Size())
}

func TestCacheKey_Generation(t *testing.T) {
	tests := []struct {
		name   string
		key1   CacheKey
		key2   CacheKey
		equal  bool
	}{
		{
			name:  "identical keys should produce same hash",
			key1:  NewCacheKey("user1", "default", "nginx"),
			key2:  NewCacheKey("user1", "default", "nginx"),
			equal: true,
		},
		{
			name:  "different users should produce different hash",
			key1:  NewCacheKey("user1", "default", "nginx"),
			key2:  NewCacheKey("user2", "default", "nginx"),
			equal: false,
		},
		{
			name:  "different namespaces should produce different hash",
			key1:  NewCacheKey("user1", "default", "nginx"),
			key2:  NewCacheKey("user1", "production", "nginx"),
			equal: false,
		},
		{
			name:  "different deployments should produce different hash",
			key1:  NewCacheKey("user1", "default", "nginx"),
			key2:  NewCacheKey("user1", "default", "postgres"),
			equal: false,
		},
		{
			name:  "empty namespace vs specified should differ",
			key1:  NewCacheKey("user1", "", ""),
			key2:  NewCacheKey("user1", "default", ""),
			equal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := tt.key1.Key()
			hash2 := tt.key2.Key()

			if tt.equal {
				assert.Equal(t, hash1, hash2)
			} else {
				assert.NotEqual(t, hash1, hash2)
			}

			// All hashes should be 64 characters (SHA-256 hex)
			assert.Len(t, hash1, 64)
			assert.Len(t, hash2, 64)
		})
	}
}

func TestCacheKey_String(t *testing.T) {
	tests := []struct {
		name     string
		key      CacheKey
		contains string
	}{
		{
			name:     "deployment scope",
			key:      NewCacheKey("user1", "default", "nginx"),
			contains: "deployment=nginx",
		},
		{
			name:     "namespace scope",
			key:      NewCacheKey("user1", "default", ""),
			contains: "all deployments",
		},
		{
			name:     "cluster scope",
			key:      NewCacheKey("user1", "", ""),
			contains: "all namespaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.key.String()
			assert.Contains(t, str, tt.contains)
			assert.Contains(t, str, "user1")
		})
	}
}
