package postgres

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCacheNotificationQueueBounds(t *testing.T) {
	for name, keys := range map[string][]string{
		"key count": func() []string {
			var keys []string
			for i := range cachePendingKeys + 1 {
				keys = append(keys, fmt.Sprint(i))
			}
			return keys
		}(),
		"byte count": {strings.Repeat("x", cachePendingBytes+1)},
	} {
		t.Run(name, func(t *testing.T) {
			var invalidations int
			r := &cacheRegistration{keys: make(map[string]struct{}), wake: make(chan struct{}, 1), invalidate: func() { invalidations++ }}
			r.enqueue(keys, false)
			r.enqueue([]string{"later"}, false)
			require.Equal(t, 1, invalidations)
			require.Empty(t, r.keys)
			require.Zero(t, r.bytes)
			batch, full := r.take()
			require.Empty(t, batch)
			require.True(t, full)
			// Events during a full scan belong to the next batch.
			r.enqueue([]string{"during scan", "during scan"}, false)
			batch, full = r.take()
			require.False(t, full)
			require.Equal(t, []string{"during scan"}, batch)
		})
	}
}
