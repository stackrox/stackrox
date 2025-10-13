package schema

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

var (
	// CreateTableGroupsStmt holds the create statement for table `groups`.
	CreateTableGroupsStmt = &postgres.CreateStmts{
		GormModel: (*Groups)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	// GroupsSchema is the go schema for table `groups`.
	GroupsSchema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.Group)(nil)), "groups")
		return schema
	}()
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
