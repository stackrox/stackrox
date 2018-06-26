package service

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/authprovider/store"
	interceptor "bitbucket.org/stack-rox/apollo/central/interceptor/singletons"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(store.Singleton(), interceptor.AuthInterceptor())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
