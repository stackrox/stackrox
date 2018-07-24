package usermapper

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/role/store"
	"bitbucket.org/stack-rox/apollo/pkg/auth/tokenbased"
)

var (
	roleMapper tokenbased.RoleMapper
	once       sync.Once
)

func initialize() {
	roleMapper = New(store.Singleton())
}

// Singleton returns the singleton user role mapper.
func Singleton() tokenbased.RoleMapper {
	once.Do(initialize)
	return roleMapper
}
