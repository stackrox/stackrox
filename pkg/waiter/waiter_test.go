package waiter

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringWaiter(t *testing.T) {
	wm := NewManager[string]()
	wm.Start(context.Background())

	w, err := wm.NewWaiter()
	assert.NoError(t, err)

	want := "hello"
	require.NoError(t, wm.Send(w.ID(), want, nil))

	got, err := w.Wait(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, got, want)
}

func TestStructWaiter(t *testing.T) {
	type widget struct {
		msg string
	}

	wm := NewManager[widget]()
	wm.Start(context.Background())

	w, err := wm.NewWaiter()
	assert.NoError(t, err)

	want := widget{msg: "hello"}
	require.NoError(t, wm.Send(w.ID(), want, nil))

	got, err := w.Wait(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, got.msg, want.msg)
}

func TestPointerWaiter(t *testing.T) {
	type widget struct {
		msg string
	}

	wm := NewManager[*widget]()
	wm.Start(context.Background())

	w, err := wm.NewWaiter()
	assert.NoError(t, err)

	want := &widget{msg: "hello"}
	require.NoError(t, wm.Send(w.ID(), want, nil))

	got, err := w.Wait(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, got, want)
	if got != want {
		t.Errorf("%p != %p", got, want)
	}
}

func TestWaitCancel(t *testing.T) {
	t.Parallel()
	wm := NewManager[string]()
	wm.Start(context.Background())

	w, err := wm.NewWaiter()
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	got, err := w.Wait(ctx)
	assert.ErrorContains(t, err, "context canceled")
	assert.Zero(t, got)

	// allow time for manager to cleanup canceled waiters
	time.Sleep(100 * time.Millisecond)
	assert.Len(t, wm.waiters, 0)
}

func TestWaitClose(t *testing.T) {
	t.Parallel()
	wm := NewManager[string]()
	wm.Start(context.Background())

	w, err := wm.NewWaiter()
	assert.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := w.Wait(context.Background())
		assert.ErrorIs(t, err, ErrWaiterClosed)
	}()

	assert.Len(t, wm.waiters, 1)
	w.Close()
	wg.Wait()

	// allow time for manager to cleanup closed waiters
	time.Sleep(200 * time.Millisecond)
	assert.Len(t, wm.waiters, 0)
}

func TestSimpleIDGen(t *testing.T) {
	want := "1"
	idgen := IDGeneratorFuncs{
		GenIDFunc: func() (string, error) {
			return want, nil
		},
	}

	wm := NewManager[string](WithIDGenerator(idgen))
	w, err := wm.NewWaiter()
	require.NoError(t, err)
	assert.Equal(t, want, w.ID())
}

func TestZeroMaxCollisions(t *testing.T) {
	wm := NewManager[string](WithMaxCollisions(0))
	wm.Start(context.Background())

	_, err := wm.NewWaiter()
	assert.NoError(t, err)
}

func TestCloseManager(t *testing.T) {
	t.Parallel()
	wm := NewManager[string]()
	ctx, cancel := context.WithCancel(context.Background())
	wm.Start(ctx)

	w, err := wm.NewWaiter()
	require.NoError(t, err)

	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	_, err = w.Wait(context.Background())
	require.ErrorIs(t, err, ErrWaiterClosed)

	err = wm.Send("fake", "", nil)
	require.ErrorIs(t, err, ErrManagerShutdown)
}

func TestCloseManagerMany(t *testing.T) {
	t.Parallel()
	wm := NewManager[string]()
	ctx, cancel := context.WithCancel(context.Background())
	wm.Start(ctx)

	wg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		w, err := wm.NewWaiter()
		require.NoError(t, err)

		go func() {
			_, err := w.Wait(context.Background())
			assert.Error(t, err)
			wg.Done()
		}()
	}

	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	wg.Wait()
}

func TestSendToClosedWaiter(t *testing.T) {
	t.Parallel()
	wm := NewManager[string]()
	wm.Start(context.Background())

	w, err := wm.NewWaiter()
	require.NoError(t, err)

	w.Close()
	_, err = w.Wait(context.Background())
	require.Error(t, err, ErrWaiterClosed)

	err = wm.Send(w.ID(), "data", nil)
	require.NoError(t, err)
}

func TestNewWaiterOnShutdownManager(t *testing.T) {
	t.Parallel()
	wm := NewManager[string]()
	ctx, cancel := context.WithCancel(context.Background())
	wm.Start(ctx)

	_, err := wm.NewWaiter()
	require.NoError(t, err)

	cancel()
	// allow some time for cancel to be read
	time.Sleep(100 * time.Millisecond)

	_, err = wm.NewWaiter()
	require.ErrorIs(t, err, ErrManagerShutdown)
}

// --

func printAlloc() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	v := m.Alloc / 1024
	fmt.Printf("%d KB\n", v)
}

func printAndGetAlloc() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	v := m.Alloc / 1024
	fmt.Printf("%d KB\n", v)
	return v

}

func doSends(wm Manager[int32], n int) {
	ids := []string{}
	for i := 0; i < n; i++ {
		w, _ := wm.NewWaiter()
		ids = append(ids, w.ID())
	}

	for _, id := range ids {
		_ = wm.Send(id, 1, nil)
	}
}

func TestLots(t *testing.T) {
	// t.Skip()
	n := 20_000
	wm := NewManager[int32]()
	wm.Start(context.Background())
	printAlloc()

	max := uint64(0)
	for i := 0; i < 2000; i++ {
		doSends(wm, n)
		runtime.GC()
		v := printAndGetAlloc()
		if v > max {
			max = v
		}
	}

	t.Logf("max: %v KB", max)
	assert.Less(t, max, uint64(2300))
}
