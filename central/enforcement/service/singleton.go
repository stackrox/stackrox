package service

import (
	"sync"

	"github.com/stackrox/rox/central/sensorevent/service/streamer"
)

var (
	once sync.Once
	as   Service
)

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(func() {
		as = New(streamer.ManagerSingleton())
	})
	return as
}
