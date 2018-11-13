package store

import (
	"sync"
)

var (
	storage Store
	once    sync.Once
)

// Singleton returns the singleton user role mapper.
func Singleton() Store {
	once.Do(func() {
		storage = New()
	})
	return storage
}
