package m199tom200

import (
	"context"
	"embed"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/m_199_to_m_200_policy_updates_for_4_5/schema"
	"github.com/stackrox/rox/migrator/migrations/policycategorymigrationhelper"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	"github.com/stackrox/rox/migrator/types"
	"gorm.io/gorm"
)

var (
	migration = types.Migration{
		StartingSeqNum: 199,
		VersionAfter:   &storage.Version{SeqNum: 200},
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
	//go:embed categories_add_and_remove
	categoryFS embed.FS

	// Update only if the existing name, description and policy sections haven't changed.
	fieldsToCompare = []policymigrationhelper.FieldComparator{
		policymigrationhelper.NameComparator,
		policymigrationhelper.DescriptionComparator,
		policymigrationhelper.PolicySectionComparator,
		policymigrationhelper.SeverityComparator,
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
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "access_central_secret.json",
		},
		{
			FieldsToCompare: nftablesFieldsToCompare,
			PolicyFileName:  "exec-iptables-root.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "deployment_has_ingress_network_policy.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "network_baseline.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "no_scans.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "scan_age.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "image_age_days.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "owner_team_label.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "owner_team_annotation.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "email_annotation.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "latest_tag.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "process_uid_zero.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "privileged.json",
		},
		{
			FieldsToCompare: fieldsToCompare,
			PolicyFileName:  "rapid-reset.json",
		},
	}
)

func updatePolicies(db *gorm.DB) error {

	err := policycategorymigrationhelper.AddNewCategoriesToDB(categoryFS,
		func(ctx context.Context, category *storage.PolicyCategory) error {
			dbCategory, err := schema.ConvertPolicyCategoryFromProto(category)
			if err != nil {
				return err
			}
			result := db.WithContext(ctx).Table(schema.PolicyCategoriesTableName).Save(dbCategory)
			if result.RowsAffected != 1 {
				return errors.Errorf("failed to save policy category with id %s: %q", category.GetId(), result.Error)
			}
			return result.Error
		})

	if err != nil {
		return err
	}
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
			var results []*schema.PolicyCategories
			db.WithContext(ctx).Table(schema.PolicyCategoriesTableName).Find(&results)

			categories := make(map[string]string, 0)
			for _, r := range results {
				c, err := schema.ConvertPolicyCategoryToProto(r)
				if err != nil {
					return nil, err
				}
				categories[c.GetName()] = c.GetId()
			}
			return categories, nil
		},
		func(ctx context.Context, edge *storage.PolicyCategoryEdge) error {
			dbEdge, err := schema.ConvertPolicyCategoryEdgeFromProto(edge)
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
			dbEdge, err := schema.ConvertPolicyCategoryEdgeFromProto(edge)
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

func init() {
	migrations.MustRegisterMigration(migration)
}
