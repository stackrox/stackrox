package service

import (
	"sync"

	"github.com/stackrox/rox/central/auth/userpass"
)

var (
	once sync.Once

	as Service
)

// Singleton provides the instance of the Service interface to register.
func Singleton(pass *userpass.Issuer) Service {
	once.Do(func() {
		as = New(pass)
	})
	return as
}
