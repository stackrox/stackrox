//go:build amd64

package types

import (
	"context"

	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/tecbot/gorocksdb"
	bolt "go.etcd.io/bbolt"
	"gorm.io/gorm"
)

// Databases encapsulates all the different databases we are using
// This struct helps avoid adding a new parameter when we switch DBs
type Databases struct {
	BoltDB *bolt.DB

	// TODO(cdu): deprecate this and change to use *rocksdb.RocksDB.
	RocksDB *gorocksdb.DB

	PkgRocksDB *rocksdb.RocksDB
	GormDB     *gorm.DB
	PostgresDB postgres.DB

	// Adding a context, so we can wrap migrations in a transaction if desired
	DBCtx context.Context
}
