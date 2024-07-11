package concurrency

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlag(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	f := &Flag{}
	a.False(f.Get())
	f.Set(false)
	a.False(f.Get())
	f.Set(true)
	a.True(f.Get())
	a.False(f.Toggle())
	a.False(f.Get())
	a.False(f.TestAndSet(false))
	a.False(f.Get())
	a.False(f.TestAndSet(true))
	a.True(f.Get())
	a.True(f.TestAndSet(false))
	a.False(f.Get())
}
