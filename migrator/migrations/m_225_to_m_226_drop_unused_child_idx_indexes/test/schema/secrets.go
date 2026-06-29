// Frozen pre-PR#21423 schema copied from release-4.11.

package schema

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableSecretsStmt holds the create statement for table `secrets`.
	CreateTableSecretsStmt = &postgres.CreateStmts{
		GormModel: (*Secrets)(nil),
		Children: []*postgres.CreateStmts{
			{
				GormModel: (*SecretsFiles)(nil),
				Children: []*postgres.CreateStmts{
					{
						GormModel: (*SecretsFilesRegistries)(nil),
						Children:  []*postgres.CreateStmts{},
					},
				},
			},
		},
	}
)

const (
	// SecretsTableName specifies the name of the table in postgres.
	SecretsTableName = "secrets"
	// SecretsFilesTableName specifies the name of the table in postgres.
	SecretsFilesTableName = "secrets_files"
	// SecretsFilesRegistriesTableName specifies the name of the table in postgres.
	SecretsFilesRegistriesTableName = "secrets_files_registries"
)

// Secrets holds the Gorm model for Postgres table `secrets`.
type Secrets struct {
	ID          string     `gorm:"column:id;type:uuid;primaryKey"`
	Name        string     `gorm:"column:name;type:varchar"`
	ClusterID   string     `gorm:"column:clusterid;type:uuid;index:secrets_sac_filter,type:btree"`
	ClusterName string     `gorm:"column:clustername;type:varchar"`
	Namespace   string     `gorm:"column:namespace;type:varchar;index:secrets_sac_filter,type:btree"`
	CreatedAt   *time.Time `gorm:"column:createdat;type:timestamp"`
	Serialized  []byte     `gorm:"column:serialized;type:bytea"`
}

// SecretsFiles holds the Gorm model for Postgres table `secrets_files`.
type SecretsFiles struct {
	SecretsID   string             `gorm:"column:secrets_id;type:uuid;primaryKey"`
	Idx         int                `gorm:"column:idx;type:integer;primaryKey;index:secretsfiles_idx,type:btree"`
	Type        storage.SecretType `gorm:"column:type;type:integer"`
	CertEndDate *time.Time         `gorm:"column:cert_enddate;type:timestamp"`
	SecretsRef  Secrets            `gorm:"foreignKey:secrets_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

// SecretsFilesRegistries holds the Gorm model for Postgres table `secrets_files_registries`.
type SecretsFilesRegistries struct {
	SecretsID       string       `gorm:"column:secrets_id;type:uuid;primaryKey"`
	SecretsFilesIdx int          `gorm:"column:secrets_files_idx;type:integer;primaryKey"`
	Idx             int          `gorm:"column:idx;type:integer;primaryKey;index:secretsfilesregistries_idx,type:btree"`
	Name            string       `gorm:"column:name;type:varchar"`
	SecretsFilesRef SecretsFiles `gorm:"foreignKey:secrets_id,secrets_files_idx;references:secrets_id,idx;belongsTo;constraint:OnDelete:CASCADE"`
}
