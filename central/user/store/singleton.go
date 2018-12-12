package store

import (
	"sync"
)

var (
	as   Store
	once sync.Once
)

// Singleton returns the singleton user role mapper.
func Singleton() Store {
	once.Do(func() {
		as = New()
	})
	return as
}
