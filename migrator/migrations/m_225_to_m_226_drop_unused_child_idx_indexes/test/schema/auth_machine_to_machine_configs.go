// Frozen pre-migration GORM schema for auth_machine_to_machine_configs.
// Reproduces old index tags so AutoMigrate creates the _idx indexes that the migration drops.

package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableAuthMachineToMachineConfigsStmt holds the create statement for table `auth_machine_to_machine_configs`.
	CreateTableAuthMachineToMachineConfigsStmt = &postgres.CreateStmts{
		GormModel: (*AuthMachineToMachineConfigs)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*AuthMachineToMachineConfigsMappings)(nil), Children: []*postgres.CreateStmts{}},
		},
	}
)

// AuthMachineToMachineConfigs holds the Gorm model for Postgres table `auth_machine_to_machine_configs`.
type AuthMachineToMachineConfigs struct {
	ID         string `gorm:"column:id;type:uuid;primaryKey"`
	Issuer     string `gorm:"column:issuer;type:varchar;unique"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}

// TableName returns the table name for GORM.
func (AuthMachineToMachineConfigs) TableName() string {
	return "auth_machine_to_machine_configs"
}

// AuthMachineToMachineConfigsMappings holds the Gorm model for Postgres table `auth_machine_to_machine_configs_mappings`.
type AuthMachineToMachineConfigsMappings struct {
	AuthMachineToMachineConfigsID  string                      `gorm:"column:auth_machine_to_machine_configs_id;type:uuid;primaryKey"`
	Idx                            int                         `gorm:"column:idx;type:integer;primaryKey;index:authmachinetomachineconfigsmappings_idx,type:btree"`
	Role                           string                      `gorm:"column:role;type:varchar"`
	AuthMachineToMachineConfigsRef AuthMachineToMachineConfigs `gorm:"foreignKey:auth_machine_to_machine_configs_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name for GORM.
func (AuthMachineToMachineConfigsMappings) TableName() string {
	return "auth_machine_to_machine_configs_mappings"
}
