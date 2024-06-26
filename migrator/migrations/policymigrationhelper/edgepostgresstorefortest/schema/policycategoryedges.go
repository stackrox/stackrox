// Code generated by pg-bindings generator. DO NOT EDIT.

package schema

import (
	"reflect"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper/categorypostgresstorefortest/schema"
	schema2 "github.com/stackrox/rox/migrator/migrations/policymigrationhelper/policypostgresstorefortest/schema"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
)

var (
	// CreateTablePolicyCategoryEdgesStmt holds the create statement for table `policy_category_edges`.
	CreateTablePolicyCategoryEdgesStmt = &postgres.CreateStmts{
		GormModel: (*PolicyCategoryEdges)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	// PolicyCategoryEdgesSchema is the go schema for table `policy_category_edges`.
	PolicyCategoryEdgesSchema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.PolicyCategoryEdge)(nil)), "policy_category_edges")
		schema.SetOptionsMap(search.Walk(v1.SearchCategory_POLICY_CATEGORY_EDGE, "policycategoryedge", (*storage.PolicyCategoryEdge)(nil)))
		return schema
	}()
)

const (
	// PolicyCategoryEdgesTableName specifies the name of the table in postgres.
	PolicyCategoryEdgesTableName = "policy_category_edges"
)

// PolicyCategoryEdges holds the Gorm model for Postgres table `policy_category_edges`.
type PolicyCategoryEdges struct {
	ID                  string           `gorm:"column:id;type:varchar;primaryKey"`
	PolicyID            string           `gorm:"column:policyid;type:varchar"`
	CategoryID          string           `gorm:"column:categoryid;type:varchar"`
	Serialized          []byte                  `gorm:"column:serialized;type:bytea"`
	PoliciesRef         schema2.Policies        `gorm:"foreignKey:policyid;references:id;belongsTo;constraint:OnDelete:CASCADE"`
	PolicyCategoriesRef schema.PolicyCategories `gorm:"foreignKey:categoryid;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}
