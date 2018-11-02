package expiringcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExpiringCache(t *testing.T) {
	c := NewExpiringCacheOrPanic(2, time.Nanosecond)

	c.Add("hello", "goodbye")
	time.Sleep(5 * time.Nanosecond)
	assert.Nil(t, c.Get("hello"))

	c = NewExpiringCacheOrPanic(2, time.Hour)
	c.Add("hello", "goodbye")
	assert.Equal(t, "goodbye", c.Get("hello"))

	c.Purge()
	assert.Nil(t, c.Get("hello"))
}
