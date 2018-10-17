package usermapper

import (
	"sync"

	"github.com/stackrox/rox/central/role/store"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

var (
	roleMapper permissions.RoleMapper
	once       sync.Once
)

func initialize() {
	roleMapper = New(store.Singleton())
}

// Singleton returns the singleton user role mapper.
func Singleton() permissions.RoleMapper {
	once.Do(initialize)
	return roleMapper
}
