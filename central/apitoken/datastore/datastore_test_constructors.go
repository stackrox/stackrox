package datastore

import (
	"testing"

	"github.com/stackrox/rox/pkg/postgres"
)

func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	return newPostgres(pool)
}
