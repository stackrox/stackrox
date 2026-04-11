// Package mocks provides test utilities for k8scfgwatch.
package mocks

import (
	"testing"

	"k8s.io/apimachinery/pkg/watch"
	k8sTest "k8s.io/client-go/testing"
)

type testWatchReactor struct {
	watcher watch.Interface
	err     error
}

func (w *testWatchReactor) Handles(_ k8sTest.Action) bool {
	return true
}

func (w *testWatchReactor) React(_ k8sTest.Action) (bool, watch.Interface, error) {
	return true, w.watcher, w.err
}

// NewTestWatchReactor creates a new test watch reactor for testing.
func NewTestWatchReactor(_ *testing.T, watcher watch.Interface) k8sTest.WatchReactor {
	return &testWatchReactor{watcher: watcher}
}
