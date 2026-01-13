package concurrency

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/require"
)

func BenchmarkValueStreamWrite(b *testing.B) {

	vs := NewValueStream(0)

	for b.Loop() {
		vs.Push(1)
	}
}

func BenchmarkBufChanWrite(b *testing.B) {

	c := make(chan struct{}, b.N)

	for b.Loop() {
		c <- struct{}{}
	}
}

func BenchmarkBuf1ChanWrite(b *testing.B) {

	c := make(chan struct{}, 1)
	b.Cleanup(func() {
		close(c)
	})

	// Read from channel in a tight loop
	go func() {
		ok := true
		for ok {
			_, ok = <-c
		}
	}()

	for b.Loop() {
		c <- struct{}{}
	}
}

func BenchmarkUnbufChanWrite(b *testing.B) {

	c := make(chan struct{})
	b.Cleanup(func() {
		close(c)
	})

	// Read from channel in a tight loop
	go func() {
		ok := true
		for ok {
			_, ok = <-c
		}
	}()

	for b.Loop() {
		c <- struct{}{}
	}
}

func BenchmarkSliceAppend(b *testing.B) {

	var slice []struct{}

	for b.Loop() {
		slice = append(slice, struct{}{}) //nolint:staticcheck // SA4010 slice append without reading is intended
	}
}

func BenchmarkSliceAppendWithMutex(b *testing.B) {

	var slice []struct{}
	var mutex sync.Mutex

	for b.Loop() {
		WithLock(&mutex, func() {
			slice = append(slice, struct{}{})
		})
	}
}

func BenchmarkValueStreamRead(b *testing.B) {

	vs := NewValueStream(0)
	it := vs.Iterator(true)

	for i := 0; i < b.N; i++ {
		vs.Push(i)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(b.N)*time.Millisecond)
	defer cancel()

	var err error
	for b.Loop() && err == nil {
		it, err = it.Next(ctx)
	}
	require.NoError(b, err)
}

func BenchmarkValueStreamReadAsync(b *testing.B) {

	vs := NewValueStream(0)
	it := vs.Iterator(true)

	go func(n int) {
		for i := 0; i < n; i++ {
			vs.Push(i)
		}
	}(b.N)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(b.N)*time.Millisecond)
	defer cancel()

	var err error
	for b.Loop() && err == nil {
		it, err = it.Next(ctx)
	}
	require.NoError(b, err)
}

func BenchmarkBufChanRead(b *testing.B) {

	c := make(chan int, b.N)
	for i := 0; i < b.N; i++ {
		c <- i
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(b.N)*time.Millisecond)
	defer cancel()

	var err error
	for b.Loop() && err == nil {
		select {
		case <-c:
		case <-ctx.Done():
			err = ctx.Err()
		}
	}
	require.NoError(b, err)
}

func BenchmarkBuf1ChanRead(b *testing.B) {

	c := make(chan int, 1)

	// Write to channel in a tight loop
	go func(n int) {
		for i := 0; i < n; i++ {
			c <- i
		}
	}(b.N)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(b.N)*time.Millisecond)
	defer cancel()

	var err error
	for b.Loop() && err == nil {
		select {
		case <-c:
		case <-ctx.Done():
			err = ctx.Err()
		}
	}
}

func BenchmarkUnbufChanRead(b *testing.B) {

	c := make(chan int)

	// Write to channel in a tight loop
	go func(n int) {
		for i := 0; i < n; i++ {
			c <- i
		}
	}(b.N)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(b.N)*time.Millisecond)
	defer cancel()

	var err error
	for b.Loop() && err == nil {
		select {
		case <-c:
		case <-ctx.Done():
			err = ctx.Err()
		}
	}
}

func BenchmarkSliceRead(b *testing.B) {

	slice := make([]int, 0, b.N)
	for i := 0; i < b.N; i++ {
		slice = append(slice, i)
	}

	for b.Loop() {
		slice = slice[1:]
	}
	require.Empty(b, slice)
}

func BenchmarkSliceReadWithMutex(b *testing.B) {

	slice := make([]int, 0, b.N)
	for i := 0; i < b.N; i++ {
		slice = append(slice, i)
	}
	var mutex sync.Mutex

	for b.Loop() {
		WithLock(&mutex, func() {
			slice = slice[1:]
		})
	}
	require.Empty(b, slice)
}
