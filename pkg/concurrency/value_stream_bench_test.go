package concurrency

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/require"
)

const benchStreamSize = 10000

func BenchmarkValueStreamWrite(b *testing.B) {
	vs := NewValueStream(0)

	for b.Loop() {
		vs.Push(1)
	}
}

func BenchmarkBufChanWrite(b *testing.B) {
	c := make(chan struct{}, benchStreamSize)
	b.Cleanup(func() { close(c) })

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

func BenchmarkBuf1ChanWrite(b *testing.B) {
	c := make(chan struct{}, 1)
	b.Cleanup(func() { close(c) })

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
	b.Cleanup(func() { close(c) })

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

	for i := 0; i < benchStreamSize; i++ {
		vs.Push(i)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	i := 0
	for b.Loop() && err == nil {
		it, err = it.Next(ctx)
		i++
		if i >= benchStreamSize {
			it = vs.Iterator(true)
			for j := 0; j < benchStreamSize; j++ {
				vs.Push(j)
			}
			i = 0
		}
	}
	require.NoError(b, err)
}

func BenchmarkValueStreamReadAsync(b *testing.B) {
	vs := NewValueStream(0)
	it := vs.Iterator(true)

	done := make(chan struct{})
	b.Cleanup(func() { close(done) })

	go func() {
		i := 0
		for {
			select {
			case <-done:
				return
			default:
				vs.Push(i)
				i++
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	for b.Loop() && err == nil {
		it, err = it.Next(ctx)
	}
	require.NoError(b, err)
}

func BenchmarkBufChanRead(b *testing.B) {
	c := make(chan int, benchStreamSize)

	done := make(chan struct{})
	b.Cleanup(func() { close(done) })

	go func() {
		i := 0
		for {
			select {
			case <-done:
				return
			case c <- i:
				i++
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	done := make(chan struct{})
	b.Cleanup(func() { close(done) })

	go func() {
		i := 0
		for {
			select {
			case <-done:
				return
			case c <- i:
				i++
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

func BenchmarkUnbufChanRead(b *testing.B) {
	c := make(chan int)

	done := make(chan struct{})
	b.Cleanup(func() { close(done) })

	go func() {
		i := 0
		for {
			select {
			case <-done:
				return
			case c <- i:
				i++
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

func BenchmarkSliceRead(b *testing.B) {
	for b.Loop() {
		b.StopTimer()
		slice := make([]int, benchStreamSize)
		b.StartTimer()

		for len(slice) > 0 {
			slice = slice[1:]
		}
	}
}

func BenchmarkSliceReadWithMutex(b *testing.B) {
	var mutex sync.Mutex

	for b.Loop() {
		b.StopTimer()
		slice := make([]int, benchStreamSize)
		b.StartTimer()

		for len(slice) > 0 {
			WithLock(&mutex, func() {
				slice = slice[1:]
			})
		}
	}
}
