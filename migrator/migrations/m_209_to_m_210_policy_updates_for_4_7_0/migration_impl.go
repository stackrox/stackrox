package m209tom210

import (
	"embed"

	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	"github.com/stackrox/rox/migrator/types"
)

var (
	//go:embed policies_before_and_after
	policyDiffFS embed.FS

	// We want to migrate only if the existing policy sections,name and description haven't changed.
	fieldsToCompare = []policymigrationhelper.FieldComparator{
		policymigrationhelper.DescriptionComparator,
		policymigrationhelper.PolicySectionComparator,
		policymigrationhelper.NameComparator,
	}

	policyDiffs = []policymigrationhelper.PolicyDiff{
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "cvss_6_privileged.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "cvss_7.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "severity_high_privileged.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "severity_important.json",
		},
	}
)

func migrate(database *types.Databases) error {
	_ = database // TODO(dont-merge): remove this line, it is there to make the compiler happy while the migration code is being written.
	// Use databases.DBCtx to take advantage of the transaction wrapping present in the migration initiator

	// TODO(dont-merge): Migration code comes here
	// TODO(dont-merge): When using gorm, make sure you use a separate handle for the updates and the query.  Such as:
	// TODO(dont-merge): db = db.WithContext(database.DBCtx).Table(schema.ListeningEndpointsTableName)
	// TODO(dont-merge): query := db.WithContext(database.DBCtx).Table(schema.ListeningEndpointsTableName).Select("serialized")
	// TODO(dont-merge): See README for more details

	return nil
}

// TODO(dont-merge): Write the additional code to support the migration

// TODO(dont-merge): remove any pending TODO
