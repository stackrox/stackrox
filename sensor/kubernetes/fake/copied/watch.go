/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fake

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog"
)

const (
	// DefaultChanSize is the maximum size number of items in the channel before panic
	DefaultChanSize int32 = 100000
)

var (
	events int32
)

// RaceFreeFakeWatcher lets you test anything that consumes a watch.Interface; threadsafe.
type RaceFreeFakeWatcher struct {
	result  chan watch.Event
	Stopped bool
	sync.Mutex
}

// NewRaceFreeFake returns a new race watcher
func NewRaceFreeFake() *RaceFreeFakeWatcher {
	return &RaceFreeFakeWatcher{
		result: make(chan watch.Event, DefaultChanSize),
	}
}

// Stop implements Interface.Stop().
func (f *RaceFreeFakeWatcher) Stop() {
	f.Lock()
	defer f.Unlock()
	if !f.Stopped {
		klog.V(4).Infof("Stopping fake watcher.")
		close(f.result)
		f.Stopped = true
	}
}

// IsStopped signals if the watcher is stopped
func (f *RaceFreeFakeWatcher) IsStopped() bool {
	f.Lock()
	defer f.Unlock()
	return f.Stopped
}

// Reset prepares the watcher to be reused.
func (f *RaceFreeFakeWatcher) Reset() {
	f.Lock()
	defer f.Unlock()
	f.Stopped = false
	f.result = make(chan watch.Event, DefaultChanSize)
}

// ResultChan returns the events
func (f *RaceFreeFakeWatcher) ResultChan() <-chan watch.Event {
	f.Lock()
	defer f.Unlock()
	return f.result
}

// Add sends an add event.
func (f *RaceFreeFakeWatcher) Add(obj runtime.Object) {
	f.Lock()
	defer f.Unlock()
	if !f.Stopped {
		select {
		case f.result <- watch.Event{Type: watch.Added, Object: obj}:
			f.printChanSize()
			return
		default:
			panic(errors.New("channel full"))
		}
	}
}

func (f *RaceFreeFakeWatcher) printChanSize() {
	prev := atomic.AddInt32(&events, 1)
	if prev%100 == 0 && prev != 0 {
		fmt.Println("Current chan size is", len(f.result))
	}
}

// Modify sends a modify event.
func (f *RaceFreeFakeWatcher) Modify(obj runtime.Object) {
	f.Lock()
	defer f.Unlock()
	if !f.Stopped {
		select {
		case f.result <- watch.Event{Type: watch.Modified, Object: obj}:
			f.printChanSize()
			return
		default:
			panic(errors.New("channel full"))
		}
	}
}

// Delete sends a delete event.
func (f *RaceFreeFakeWatcher) Delete(lastValue runtime.Object) {
	f.Lock()
	defer f.Unlock()
	if !f.Stopped {
		select {
		case f.result <- watch.Event{Type: watch.Deleted, Object: lastValue}:
			f.printChanSize()
			return
		default:
			panic(errors.New("channel full"))
		}
	}
}

// Error sends an Error event.
func (f *RaceFreeFakeWatcher) Error(errValue runtime.Object) {
	f.Lock()
	defer f.Unlock()
	if !f.Stopped {
		select {
		case f.result <- watch.Event{Type: watch.Error, Object: errValue}:
			f.printChanSize()
			return
		default:
			panic(errors.New("channel full"))
		}
	}
}

// Action sends an event of the requested type, for table-based testing.
func (f *RaceFreeFakeWatcher) Action(action watch.EventType, obj runtime.Object) {
	f.Lock()
	defer f.Unlock()
	if !f.Stopped {
		select {
		case f.result <- watch.Event{Type: action, Object: obj}:
			f.printChanSize()
			return
		default:
			panic(errors.New("channel full"))
		}
	}
}
