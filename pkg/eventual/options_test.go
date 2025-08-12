package eventual

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type nothing struct{}

func TestWithType(t *testing.T) {
	var n int
	f1 := func(context.Context, bool) {
		n = 1
	}
	f2 := func(context.Context, bool) {
		n = 2
	}
	var opts options[string]

	WithType[string]().
		WithContext(context.WithValue(context.Background(), nothing{}, "value")).
		WithContextCallback(f1).
		WithContextCallback(f2).
		WithTimeout(time.Minute)(&opts)

	assert.Equal(t, "value", opts.context.Value(nothing{}))

	assert.NotNil(t, opts.context)
	opts.contextCancel()
	<-opts.context.Done()

	assert.NotNil(t, opts.contextCancel)

	if assert.Len(t, opts.contextCallbacks, 2) {
		opts.contextCallbacks[0](nil, false)
		assert.Equal(t, 1, n)
		opts.contextCallbacks[1](nil, false)
		assert.Equal(t, 2, n)
	}
}
