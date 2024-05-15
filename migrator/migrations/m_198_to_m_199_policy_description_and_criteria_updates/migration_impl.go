package m198tom199

import (
	"context"
	"embed"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_198_to_m_199_policy_description_and_criteria_updates/schema"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	"github.com/stackrox/rox/migrator/types"
	"gorm.io/gorm"
)

var (

	//go:embed policies_before_and_after
	policyDiffFS embed.FS

	// Update only if the existing name, description and policy sections haven't changed.
	fieldsToCompare = []policymigrationhelper.FieldComparator{
		policymigrationhelper.NameComparator,
		policymigrationhelper.DescriptionComparator,
		policymigrationhelper.PolicySectionComparator,
	}

	policyDiffs = []policymigrationhelper.PolicyDiff{
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "exec-adduser.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "exec-systemctl.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "no_resources_specified.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "root_user.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "sensitive_files.json",
		},

		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "automount_service_account_token.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "host_pids.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "host_network.json",
		},
	}
)

func migrate(database *types.Databases) error {

	db := database.GormDB
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
		func(ctx context.Context) (map[string]string, error) {
			return nil, nil
		},
		func(ctx context.Context, edge *storage.PolicyCategoryEdge) error {
			return nil
		},
		func(ctx context.Context, edge *storage.PolicyCategoryEdge) error {
			return nil
		},
	)
}
