package gorm

import (
	"testing"

	"github.com/stackrox/rox/pkg/postgres/pgtest/conn"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
)

// SetupAndGetMockConfig creates a gorm config for testing
func SetupAndGetMockConfig(t *testing.T) Config {
	once.Do(func() {
		source := conn.GetConnectionString(t)
		source = pgutils.PgxpoolDsnToPgxDsn(source)
		gConfig = &gormConfig{source: source, password: "MockPass"}
	})
	return gConfig
}
