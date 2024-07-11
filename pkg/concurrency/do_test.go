package concurrency

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDoInWaitable(t *testing.T) {
	// 1. Action which finishes _after_ cancellation should return an error with context.Canceled.
	neverFinishAction := func() {
		c := make(chan int)
		<-c
	}
	ctx, cancel := context.WithCancel(context.Background())
	AfterFunc(10*time.Millisecond, func() {
		cancel()
	}, context.Background())
	err := DoInWaitable(ctx, neverFinishAction)
	assert.ErrorIs(t, err, context.Canceled)

	// 2. Action which finishes _before_ cancellation should return no error and the action should have
	// been finished.
	var finished bool
	finishAction := func() {
		finished = true
	}
	ctx, cancel = context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err = DoInWaitable(ctx, finishAction)
	assert.NoError(t, err)
	assert.True(t, finished)
}
