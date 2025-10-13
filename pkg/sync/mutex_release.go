//go:build release

package sync

import "sync"

// Release version: use types from `sync` package

// Mutex is an alias for sync.Mutex
type Mutex = sync.Mutex

// RWMutex is an alias for sync.RWMutex
type RWMutex = sync.RWMutex
