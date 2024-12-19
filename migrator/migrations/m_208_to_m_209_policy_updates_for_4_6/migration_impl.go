package m208tom209

import (
	"context"
	"embed"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_208_to_m_209_policy_updates_for_4_6/conversion"
	"github.com/stackrox/rox/migrator/migrations/m_208_to_m_209_policy_updates_for_4_6/schema"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	"github.com/stackrox/rox/migrator/types"
	"gorm.io/gorm"
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
			PolicyFileName:  "impersonated_secret_access.json",
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
			storagePolicy, err := conversion.ConvertPolicyToProto(&foundPolicy)
			if err != nil {
				return nil, false, err
			}
			return storagePolicy, true, nil
		},
		// Upsert policy. Technically it should be just an update and not create because in theory policy has been verified to exist
		func(ctx context.Context, policy *storage.Policy) error {
			dbPolicy, err := conversion.ConvertPolicyFromProto(policy)
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
			var results []*schema.PolicyCategories
			db.WithContext(ctx).Table(schema.PolicyCategoriesTableName).Find(&results)

			categories := make(map[string]string, 0)
			for _, r := range results {
				c, err := conversion.ConvertPolicyCategoryToProto(r)
				if err != nil {
					return nil, err
				}
				categories[c.Name] = c.Id
			}
			return categories, nil
		},
		func(ctx context.Context, edge *storage.PolicyCategoryEdge) error {
			dbEdge, err := conversion.ConvertPolicyCategoryEdgeFromProto(edge)
			if err != nil {
				return err
			}
			result := db.WithContext(ctx).Table(schema.PolicyCategoryEdgesTableName).Save(dbEdge)
			if result.RowsAffected != 1 {
				return errors.Errorf("failed to save edge for policy id %s, category id %s: %q",
					edge.GetPolicyId(), edge.GetCategoryId(), result.Error)
			}
			return result.Error
		},
		func(ctx context.Context, edge *storage.PolicyCategoryEdge) error {
			dbEdge, err := conversion.ConvertPolicyCategoryEdgeFromProto(edge)
			if err != nil {
				return err
			}
			result := db.WithContext(ctx).Table(schema.PolicyCategoryEdgesTableName).Where(&schema.PolicyCategoryEdges{
				PolicyID:   edge.GetPolicyId(),
				CategoryID: edge.GetCategoryId(),
			}).Delete(dbEdge)
			if result.Error != nil {
				if errors.Is(result.Error, gorm.ErrRecordNotFound) {
					return nil
				}
				return errors.Errorf("failed to remove edge for policy id %s, category id %s", edge.GetPolicyId(), edge.GetCategoryId())

			}
			return nil
		},
	)
}
