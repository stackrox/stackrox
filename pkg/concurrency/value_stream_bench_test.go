package concurrency

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/require"
)

func BenchmarkValueStreamWrite(b *testing.B) {
	b.StopTimer()

	vs := NewValueStream(0)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		vs.Push(i)
	}
}

func BenchmarkBufChanWrite(b *testing.B) {
	b.StopTimer()

	c := make(chan int, b.N)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		c <- i
	}
}

func BenchmarkBuf1ChanWrite(b *testing.B) {
	b.StopTimer()

	c := make(chan int, 1)

	// Read from channel in a tight loop
	go func() {
		ok := true
		for ok {
			_, ok = <-c
		}
	}()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		c <- i
	}
	b.StopTimer()
	close(c)
}

func BenchmarkUnbufChanWrite(b *testing.B) {
	b.StopTimer()

	c := make(chan int)

	// Read from channel in a tight loop
	go func() {
		ok := true
		for ok {
			_, ok = <-c
		}
	}()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		c <- i
	}
	b.StopTimer()
	close(c)
}

func BenchmarkSliceAppend(b *testing.B) {
	b.StopTimer()

	var slice []int

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		slice = append(slice, i) //nolint:staticcheck // SA4010 slice append without reading is intended
	}
}

func BenchmarkSliceAppendWithMutex(b *testing.B) {
	b.StopTimer()

	var slice []int
	var mutex sync.Mutex

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		mutex.Lock()
		slice = append(slice, i) //nolint:staticcheck // SA4010 slice append without reading is intended
		mutex.Unlock()
	}
}

func BenchmarkValueStreamRead(b *testing.B) {
	b.StopTimer()

	vs := NewValueStream(0)
	it := vs.Iterator(true)

	for i := 0; i < b.N; i++ {
		vs.Push(i)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(b.N)*time.Millisecond)
	defer cancel()

	b.StartTimer()

	var err error
	for i := 0; i < b.N && err == nil; i++ {
		it, err = it.Next(ctx)
	}
	require.NoError(b, err)
}

func BenchmarkValueStreamReadAsync(b *testing.B) {
	b.StopTimer()

	vs := NewValueStream(0)
	it := vs.Iterator(true)

	go func(n int) {
		for i := 0; i < n; i++ {
			vs.Push(i)
		}
	}(b.N)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(b.N)*time.Millisecond)
	defer cancel()

	b.StartTimer()

	var err error
	for i := 0; i < b.N && err == nil; i++ {
		it, err = it.Next(ctx)
	}
	require.NoError(b, err)
}

func BenchmarkBufChanRead(b *testing.B) {
	b.StopTimer()

	c := make(chan int, b.N)
	for i := 0; i < b.N; i++ {
		c <- i
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(b.N)*time.Millisecond)
	defer cancel()

	b.StartTimer()

	var err error
	for i := 0; i < b.N && err == nil; i++ {
		select {
		case <-c:
		case <-ctx.Done():
			err = ctx.Err()
		}
	}
	require.NoError(b, err)
}

func BenchmarkBuf1ChanRead(b *testing.B) {
	b.StopTimer()

	c := make(chan int, 1)

	// Write to channel in a tight loop
	go func(n int) {
		for i := 0; i < n; i++ {
			c <- i
		}
	}(b.N)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(b.N)*time.Millisecond)
	defer cancel()

	b.StartTimer()

	var err error
	for i := 0; i < b.N && err == nil; i++ {
		select {
		case <-c:
		case <-ctx.Done():
			err = ctx.Err()
		}
	}
}

func BenchmarkUnbufChanRead(b *testing.B) {
	b.StopTimer()

	c := make(chan int)

	// Write to channel in a tight loop
	go func(n int) {
		for i := 0; i < n; i++ {
			c <- i
		}
	}(b.N)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(b.N)*time.Millisecond)
	defer cancel()

	b.StartTimer()

	var err error
	for i := 0; i < b.N && err == nil; i++ {
		select {
		case <-c:
		case <-ctx.Done():
			err = ctx.Err()
		}
	}
}

func BenchmarkSliceRead(b *testing.B) {
	b.StopTimer()

	slice := make([]int, 0, b.N)
	for i := 0; i < b.N; i++ {
		slice = append(slice, i)
	}

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		slice = slice[1:]
	}
	require.Empty(b, slice)
}

func BenchmarkSliceReadWithMutex(b *testing.B) {
	b.StopTimer()

	slice := make([]int, 0, b.N)
	for i := 0; i < b.N; i++ {
		slice = append(slice, i)
	}
	var mutex sync.Mutex

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		mutex.Lock()
		slice = slice[1:]
		mutex.Unlock()
	}
	require.Empty(b, slice)
}
