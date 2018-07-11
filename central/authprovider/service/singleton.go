package service

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/authprovider/cachedstore"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(cachedstore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
