package m193tom194

import (
	"context"
	"embed"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/m_193_to_m_194_policy_updates_for_4_3/schema"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	"github.com/stackrox/rox/migrator/types"
	"gorm.io/gorm"
)

var (
	migration = types.Migration{
		StartingSeqNum: 193,
		VersionAfter:   &storage.Version{SeqNum: 194},
		Run: func(databases *types.Databases) error {
			err := updatePolicies(databases.GormDB)
			if err != nil {
				return errors.Wrap(err, "updating policies")
			}
			return nil
		},
	}

	//go:embed policies_before_and_after
	policyDiffFS embed.FS

	// Update only if the existing name, description and policy sections haven't changed.
	fieldsToCompare = []policymigrationhelper.FieldComparator{
		policymigrationhelper.NameComparator,
		policymigrationhelper.DescriptionComparator,
		policymigrationhelper.PolicySectionComparator,
	}

	// Update the nftables policy only if the existing name, description, rationale, remediation and policy sections haven't changed.
	nftablesFieldsToCompare = []policymigrationhelper.FieldComparator{
		policymigrationhelper.NameComparator,
		policymigrationhelper.DescriptionComparator,
		policymigrationhelper.RationaleComparator,
		policymigrationhelper.RemediationComparator,
		policymigrationhelper.PolicySectionComparator,
	}

	policyDiffs = []policymigrationhelper.PolicyDiff{
		{
			FieldsToCompare: nftablesFieldsToCompare,
			PolicyFileName:  "exec-iptables-root.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "process_uid_zero.json",
		},
	}
)

func updatePolicies(db *gorm.DB) error {

	return policymigrationhelper.MigratePoliciesWithDiffsAndStoreV2(
		policyDiffFS,
		policyDiffs,
		// Get policy with specified id
		func(ctx context.Context, id string) (*storage.Policy, bool, error) {
			var foundPolicy schema.Policies
			result := db.WithContext(ctx).Table(schema.PoliciesTableName).Where(&schema.Policies{ID: id}).First(&foundPolicy)
			if result.Error != nil {
				if errors.Is(result.Error, gorm.ErrRecordNotFound) {
					return nil, false, nil
				}
				return nil, false, result.Error
			}
			storagePolicy, err := schema.ConvertPolicyToProto(&foundPolicy)
			if err != nil {
				return nil, false, err
			}
			return storagePolicy, true, nil
		},
		// Upsert policy. Technically it should be just an update and not create because in theory policy has been verified to exist
		func(ctx context.Context, policy *storage.Policy) error {
			dbPolicy, err := schema.ConvertPolicyFromProto(policy)
			if err != nil {
				return err
			}
			result := db.WithContext(ctx).Table(schema.PoliciesTableName).Save(dbPolicy)
			if result.RowsAffected != 1 {
				return errors.Errorf("failed to save policy with id %s", policy.GetId())
			}
			return result.Error
		},
	)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
