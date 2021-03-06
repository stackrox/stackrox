// Code generated by pg-bindings generator. DO NOT EDIT.

package schema

import (
	"reflect"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
)

var (
	// CreateTableRoleBindingsStmt holds the create statement for table `role_bindings`.
	CreateTableRoleBindingsStmt = &postgres.CreateStmts{
		Table: `
               create table if not exists role_bindings (
                   Id varchar,
                   Name varchar,
                   Namespace varchar,
                   ClusterId varchar,
                   ClusterName varchar,
                   ClusterRole bool,
                   Labels jsonb,
                   Annotations jsonb,
                   RoleId varchar,
                   serialized bytea,
                   PRIMARY KEY(Id)
               )
               `,
		GormModel: (*RoleBindings)(nil),
		Indexes:   []string{},
		Children: []*postgres.CreateStmts{
			&postgres.CreateStmts{
				Table: `
               create table if not exists role_bindings_subjects (
                   role_bindings_Id varchar,
                   idx integer,
                   Kind integer,
                   Name varchar,
                   PRIMARY KEY(role_bindings_Id, idx),
                   CONSTRAINT fk_parent_table_0 FOREIGN KEY (role_bindings_Id) REFERENCES role_bindings(Id) ON DELETE CASCADE
               )
               `,
				GormModel: (*RoleBindingsSubjects)(nil),
				Indexes: []string{
					"create index if not exists roleBindingsSubjects_idx on role_bindings_subjects using btree(idx)",
				},
				Children: []*postgres.CreateStmts{},
			},
		},
	}

	// RoleBindingsSchema is the go schema for table `role_bindings`.
	RoleBindingsSchema = func() *walker.Schema {
		schema := GetSchemaForTable("role_bindings")
		if schema != nil {
			return schema
		}
		schema = walker.Walk(reflect.TypeOf((*storage.K8SRoleBinding)(nil)), "role_bindings")
		schema.SetOptionsMap(search.Walk(v1.SearchCategory_ROLEBINDINGS, "k8srolebinding", (*storage.K8SRoleBinding)(nil)))
		RegisterTable(schema, CreateTableRoleBindingsStmt)
		return schema
	}()
)

const (
	RoleBindingsTableName         = "role_bindings"
	RoleBindingsSubjectsTableName = "role_bindings_subjects"
)

// RoleBindings holds the Gorm model for Postgres table `role_bindings`.
type RoleBindings struct {
	Id          string            `gorm:"column:id;type:varchar;primaryKey"`
	Name        string            `gorm:"column:name;type:varchar"`
	Namespace   string            `gorm:"column:namespace;type:varchar"`
	ClusterId   string            `gorm:"column:clusterid;type:varchar"`
	ClusterName string            `gorm:"column:clustername;type:varchar"`
	ClusterRole bool              `gorm:"column:clusterrole;type:bool"`
	Labels      map[string]string `gorm:"column:labels;type:jsonb"`
	Annotations map[string]string `gorm:"column:annotations;type:jsonb"`
	RoleId      string            `gorm:"column:roleid;type:varchar"`
	Serialized  []byte            `gorm:"column:serialized;type:bytea"`
}

// RoleBindingsSubjects holds the Gorm model for Postgres table `role_bindings_subjects`.
type RoleBindingsSubjects struct {
	RoleBindingsId  string              `gorm:"column:role_bindings_id;type:varchar;primaryKey"`
	Idx             int                 `gorm:"column:idx;type:integer;primaryKey;index:rolebindingssubjects_idx,type:btree"`
	Kind            storage.SubjectKind `gorm:"column:kind;type:integer"`
	Name            string              `gorm:"column:name;type:varchar"`
	RoleBindingsRef RoleBindings        `gorm:"foreignKey:role_bindings_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}
