// Frozen pre-PR#21423 schema copied from release-4.11.

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
			{
				GormModel: (*RoleBindingsSubjects)(nil),
				Children:  []*postgres.CreateStmts{},
			},
		},
	}
)

const (
	// RoleBindingsTableName specifies the name of the table in postgres.
	RoleBindingsTableName = "role_bindings"
	// RoleBindingsSubjectsTableName specifies the name of the table in postgres.
	RoleBindingsSubjectsTableName = "role_bindings_subjects"
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

// RoleBindingsSubjects holds the Gorm model for Postgres table `role_bindings_subjects`.
type RoleBindingsSubjects struct {
	RoleBindingsID  string              `gorm:"column:role_bindings_id;type:uuid;primaryKey"`
	Idx             int                 `gorm:"column:idx;type:integer;primaryKey;index:rolebindingssubjects_idx,type:btree"`
	Kind            storage.SubjectKind `gorm:"column:kind;type:integer"`
	Name            string              `gorm:"column:name;type:varchar"`
	RoleBindingsRef RoleBindings        `gorm:"foreignKey:role_bindings_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}
