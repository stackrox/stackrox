// Code generated by pg-bindings generator. DO NOT EDIT.

package schema

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	// CreateTableClusterInitBundlesStmt holds the create statement for table `cluster_init_bundles`.
	CreateTableClusterInitBundlesStmt = &postgres.CreateStmts{
		GormModel: (*ClusterInitBundles)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	// ClusterInitBundlesSchema is the go schema for table `cluster_init_bundles`.
	ClusterInitBundlesSchema = func() *walker.Schema {
		schema := GetSchemaForTable("cluster_init_bundles")
		if schema != nil {
			return schema
		}
		schema = walker.Walk(reflect.TypeOf((*storage.InitBundleMeta)(nil)), "cluster_init_bundles")
		schema.ScopingResource = resources.InitBundleMeta
		RegisterTable(schema, CreateTableClusterInitBundlesStmt)
		return schema
	}()
)

const (
	// ClusterInitBundlesTableName specifies the name of the table in postgres.
	ClusterInitBundlesTableName = "cluster_init_bundles"
)

// ClusterInitBundles holds the Gorm model for Postgres table `cluster_init_bundles`.
type ClusterInitBundles struct {
	ID         string `gorm:"column:id;type:varchar;primaryKey"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
