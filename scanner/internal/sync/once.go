package sync

// The following implementation is strongly based on
// https://pkg.go.dev/golang.org/x/sync/singleflight
// which is governed by a BSD-style license.

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
)

// errGoexit indicates the runtime.Goexit was called in
// the user given function.
var errGoexit = errors.New("runtime.Goexit was called")

// A panicError is an arbitrary value recovered from a panic
// with the stack trace during the execution of given function.
type panicError struct {
	value interface{}
	stack []byte
}

// Error implements error interface.
func (p *panicError) Error() string {
	return fmt.Sprintf("%v\n\n%s", p.value, p.stack)
}

func newPanicError(v interface{}) error {
	stack := debug.Stack()

	// The first line of the stack trace is of the form "goroutine N [status]:"
	// but by the time the panic reaches Do the goroutine may no longer exist
	// and its status will have changed. Trim out the misleading line.
	if line := bytes.IndexByte(stack[:], '\n'); line >= 0 {
		stack = stack[line+1:]
	}
	return &panicError{value: v, stack: stack}
}

// call is an in-flight or completed Do call.
type call struct {
	// Blocks duplicate calls from returning
	// until the first call completes.
	wg sync.WaitGroup

	// These fields are written once before the WaitGroup is done
	// and are only read after the WaitGroup is done.
	v   any
	err error
}

// Result holds the results of Do, so they can be passed
// on a channel.
type Result struct {
	V   any
	Err error
}

// KeyedOnce is an object which will perform exactly one action
// per key until forgotten.
//
// Semantically, KeyedOnce works as if each key is associated with its own
// sync/once. See https://pkg.go.dev/sync#Once for more details about sync/once.
//
// A KeyedOnce must not be copied after first use.
type KeyedOnce[K comparable] struct {
	mu sync.Mutex  // protects m.
	m  map[K]*call // lazily initialized.
}

// Do calls the function fn for the given key if and only if
// Do is being called for the first time for this key.
//
// If Do is called multiple times for the same key,
// only the first call will invoke fn,
// even if fn has a different value in each invocation.
//
// After the first call to Do for some key, fn will only be called
// once the key is forgotten.
func (k *KeyedOnce[K]) Do(key K, fn func() (any, error)) (any, error) {
	k.mu.Lock()
	if k.m == nil {
		k.m = make(map[K]*call)
	}
	if c, exists := k.m[key]; exists {
		k.mu.Unlock()
		c.wg.Wait()
		if e, ok := c.err.(*panicError); ok {
			panic(e)
		} else if c.err == errGoexit {
			runtime.Goexit()
		}
		return c.v, c.err
	}
	c := &call{}
	c.wg.Add(1)
	k.m[key] = c
	k.mu.Unlock()

	c.doCall(fn)
	return c.v, c.err
}

// DoChan is like Do but returns a channel that will receive the
// results when they are ready.
func (k *KeyedOnce[K]) DoChan(key K, fn func() (any, error)) <-chan Result {
	ch := make(chan Result, 1)

	k.mu.Lock()
	if k.m == nil {
		k.m = make(map[K]*call)
	}
	if c, ok := k.m[key]; ok {
		k.mu.Unlock()
		go func() {
			c.wg.Wait()
			if e, ok := c.err.(*panicError); ok {
				panic(e)
			}
			ch <- Result{
				V:   c.v,
				Err: c.err,
			}
			close(ch)
		}()
		return ch
	}
	c := &call{}
	c.wg.Add(1)
	k.m[key] = c
	k.mu.Unlock()

	go func() {
		c.doCall(fn)
		ch <- Result{
			V:   c.v,
			Err: c.err,
		}
		close(ch)
	}()

	return ch
}

// doCall handles the single call for a key.
func (c *call) doCall(fn func() (any, error)) {
	normalReturn := false
	recovered := false

	// use double-defer to distinguish panic from runtime.Goexit,
	// more details see https://golang.org/cl/134395
	defer func() {
		// the given function invoked runtime.Goexit
		if !normalReturn && !recovered {
			c.err = errGoexit
		}

		// Allow duplicate calls to return the output
		// of the first call.
		c.wg.Done()

		if e, ok := c.err.(*panicError); ok {
			panic(e)
		} else if c.err == errGoexit {
			// Already in the process of goexit, no need to call again
		} else {
			// Normal return
			// Do nothing
		}
	}()

	func() {
		defer func() {
			if !normalReturn {
				// Ideally, we would wait to take a stack trace until we've determined
				// whether this is a panic or a runtime.Goexit.
				//
				// Unfortunately, the only way we can distinguish the two is to see
				// whether the recover stopped the goroutine from terminating, and by
				// the time we know that, the part of the stack trace relevant to the
				// panic has been discarded.
				if r := recover(); r != nil {
					c.err = newPanicError(r)
				}
			}
		}()

		c.v, c.err = fn()
		normalReturn = true
	}()

	if !normalReturn {
		recovered = true
	}
}

// Forget forgets about a key.
//
// Future calls to Do for this key will call the
// function rather than waiting for an earlier call to complete.
func (k *KeyedOnce[K]) Forget(key K) {
	k.mu.Lock()
	delete(k.m, key)
	k.mu.Unlock()
}
