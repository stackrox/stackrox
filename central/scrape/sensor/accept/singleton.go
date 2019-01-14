package accept

import (
	"sync"
)

var (
	once sync.Once

	updater Accepter
)

// SingletonAccepter returns the singleton instance of Accepter.
func SingletonAccepter() Accepter {
	once.Do(func() {
		updater = NewAccepter()
	})
	return updater
}
