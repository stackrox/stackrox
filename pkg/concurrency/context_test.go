package concurrency

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsContext_ErrReturnsNilWhileNotDone(t *testing.T) {
	sig := NewSignal()
	errSig := NewErrorSignal()
	ch := make(WaitableChan)
	ctx := context.Background()
	for _, w := range []Waitable{
		&sig,
		&errSig,
		ch,
		ctx,
	} {
		t.Run(fmt.Sprintf("AsContext(%T)", w), func(t *testing.T) {
			wCtx := AsContext(w)
			err := wCtx.Err()
			select {
			case <-w.Done():
				require.Fail(t, "waitable was done")
			default:
			}
			assert.Nil(t, err, "waitable should return a nil Err() while not done")
		})
	}
}
