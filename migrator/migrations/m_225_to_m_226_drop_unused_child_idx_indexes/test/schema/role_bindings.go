// Frozen pre-migration GORM schema for role_bindings.
// Reproduces old index tags so AutoMigrate creates the _idx indexes that the migration drops.

package schema

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableRoleBindingsStmt holds the create statement for table `role_bindings`.
	CreateTableRoleBindingsStmt = &postgres.CreateStmts{
		GormModel: (*RoleBindings)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*RoleBindingsSubjects)(nil), Children: []*postgres.CreateStmts{}},
		},
	}
)

// RoleBindings holds the Gorm model for Postgres table `role_bindings`.
type RoleBindings struct {
	ID          string            `gorm:"column:id;type:uuid;primaryKey"`
	Name        string            `gorm:"column:name;type:varchar"`
	Namespace   string            `gorm:"column:namespace;type:varchar;index:rolebindings_sac_filter,type:btree"`
	ClusterID   string            `gorm:"column:clusterid;type:uuid;index:rolebindings_sac_filter,type:btree"`
	ClusterName string            `gorm:"column:clustername;type:varchar"`
	ClusterRole bool              `gorm:"column:clusterrole;type:bool"`
	Labels      map[string]string `gorm:"column:labels;type:jsonb"`
	Annotations map[string]string `gorm:"column:annotations;type:jsonb"`
	RoleID      string            `gorm:"column:roleid;type:uuid"`
	Serialized  []byte            `gorm:"column:serialized;type:bytea"`
}

// TableName returns the table name for GORM.
func (RoleBindings) TableName() string { return "role_bindings" }

// RoleBindingsSubjects holds the Gorm model for Postgres table `role_bindings_subjects`.
type RoleBindingsSubjects struct {
	RoleBindingsID  string              `gorm:"column:role_bindings_id;type:uuid;primaryKey"`
	Idx             int                 `gorm:"column:idx;type:integer;primaryKey;index:rolebindingssubjects_idx,type:btree"`
	Kind            storage.SubjectKind `gorm:"column:kind;type:integer"`
	Name            string              `gorm:"column:name;type:varchar"`
	RoleBindingsRef RoleBindings        `gorm:"foreignKey:role_bindings_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name for GORM.
func (RoleBindingsSubjects) TableName() string { return "role_bindings_subjects" }
