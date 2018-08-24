package blevesearch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
	var cases = []struct {
		value    string
		duration time.Duration
		valid    bool
	}{
		{
			value:    "1",
			duration: 24 * 60 * 60 * time.Second,
			valid:    true,
		},
		{
			value:    "1d",
			duration: 24 * 60 * 60 * time.Second,
			valid:    true,
		},
		{
			value:    "lol",
			duration: time.Second,
			valid:    false,
		},
	}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			duration, valid := parseDuration(c.value)
			require.Equal(t, c.valid, valid)
			assert.Equal(t, c.duration, duration)
		})
	}
}
