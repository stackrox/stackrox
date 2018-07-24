package store

import "sync"

var (
	store Store
	once  sync.Once
)

func initialize() {
	store = New()
}

// Singleton returns the singleton providing access to the roles store.
func Singleton() Store {
	once.Do(initialize)
	return store
}
