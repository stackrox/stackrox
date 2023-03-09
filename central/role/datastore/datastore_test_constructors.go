package datastore

import (
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	permissionSetPostgresStore "github.com/stackrox/rox/central/role/store/permissionset/postgres"
	permissionSetRocksDBStore "github.com/stackrox/rox/central/role/store/permissionset/rocksdb"
	rolePostgresStore "github.com/stackrox/rox/central/role/store/role/postgres"
	roleRocksDBStore "github.com/stackrox/rox/central/role/store/role/rocksdb"
	accessScopePostgresStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/postgres"
	accessScopeRocksDBStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/rocksdb"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
)

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool *pgxpool.Pool) (DataStore, error) {
	permissionStore := permissionSetPostgresStore.New(pool)
	roleStore := rolePostgresStore.New(pool)
	scopeStore := accessScopePostgresStore.New(pool)

	return New(roleStore, permissionStore, scopeStore), nil
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(_ *testing.T, rocksengine *rocksdbBase.RocksDB) (DataStore, error) {
	permissionStore, err := permissionSetRocksDBStore.New(rocksengine)
	if err != nil {
		return nil, err
	}
	roleStore, err := roleRocksDBStore.New(rocksengine)
	if err != nil {
		return nil, err
	}
	scopeStore, err := accessScopeRocksDBStore.New(rocksengine)
	if err != nil {
		return nil, err
	}

	return New(roleStore, permissionStore, scopeStore), nil
}
