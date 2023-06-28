//go:build !amd64

package types

import (
	"github.com/stackrox/rox/pkg/postgres"
	"gorm.io/gorm"
)

// Databases encapsulates all the different databases we are using
// This struct helps avoid adding a new parameter when we switch DBs
type Databases struct {
	GormDB     *gorm.DB
	PostgresDB postgres.DB
}
