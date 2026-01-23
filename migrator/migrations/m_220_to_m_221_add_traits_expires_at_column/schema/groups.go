package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableGroupsStmt holds the create statement for table `groups`.
	CreateTableGroupsStmt = &postgres.CreateStmts{
		GormModel: (*Groups)(nil),
		Children:  []*postgres.CreateStmts{},
	}
)

// Groups holds the Gorm model for Postgres table `groups`.
type Groups struct {
	PropsID             string `gorm:"column:props_id;type:varchar;primaryKey"`
	PropsAuthProviderID string `gorm:"column:props_authproviderid;type:varchar;uniqueIndex:groups_unique_indicator"`
	PropsKey            string `gorm:"column:props_key;type:varchar;uniqueIndex:groups_unique_indicator"`
	PropsValue          string `gorm:"column:props_value;type:varchar;uniqueIndex:groups_unique_indicator"`
	RoleName            string `gorm:"column:rolename;type:varchar;uniqueIndex:groups_unique_indicator"`
	Serialized          []byte `gorm:"column:serialized;type:bytea"`
}
