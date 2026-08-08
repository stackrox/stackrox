package client

import "github.com/stackrox/rox/pkg/sync"

var (
	once sync.Once
	c    Client
)

func initialize() {
	c = NewClient()
}

// Singleton provides the Client instance.
func Singleton() Client {
	once.Do(initialize)
	return c
}
