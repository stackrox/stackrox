package datastore

import (
	"testing"

	groupFilter "github.com/stackrox/rox/central/group/datastore/filter"
	permissionSetPostgresStore "github.com/stackrox/rox/central/role/store/permissionset/postgres"
	rolePostgresStore "github.com/stackrox/rox/central/role/store/role/postgres"
	accessScopePostgresStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/postgres"
	"github.com/stackrox/rox/pkg/postgres"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) (DataStore, error) {
	permissionStore := permissionSetPostgresStore.New(pool)
	roleStore := rolePostgresStore.New(pool)
	scopeStore := accessScopePostgresStore.New(pool)

	return New(roleStore, permissionStore, scopeStore, groupFilter.GetFiltered), nil
}
