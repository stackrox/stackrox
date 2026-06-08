package m226tom227

import (
	"context"
	"embed"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	migration = types.Migration{
		StartingSeqNum: 226,
		VersionAfter:   &storage.Version{SeqNum: 227},
		Run: func(databases *types.Databases) error {
			return updatePolicies(databases.PostgresDB)
		},
	}

	//go:embed policies_before_and_after
	policyDiffFS embed.FS

	policyDiffs = []policymigrationhelper.PolicyDiff{
		{
			FieldsToCompare: []policymigrationhelper.FieldComparator{
				policymigrationhelper.PolicySectionComparator,
			},
			PolicyFileName: "access_kubeadmin_secret.json",
		},
	}
)

func updatePolicies(db postgres.DB) error {
	get := func(ctx context.Context, id string) (*storage.Policy, bool, error) {
		var serialized []byte
		err := db.QueryRow(ctx, "SELECT serialized FROM policies WHERE id = $1", id).Scan(&serialized)
		if err != nil {
			return nil, false, err
		}
		policy := &storage.Policy{}
		if err := policy.UnmarshalVT(serialized); err != nil {
			return nil, false, err
		}
		return policy, true, nil
	}

	upsert := func(ctx context.Context, policy *storage.Policy) error {
		serialized, err := policy.MarshalVT()
		if err != nil {
			return fmt.Errorf("marshal policy %s: %w", policy.GetId(), err)
		}
		_, err = db.Exec(ctx, "UPDATE policies SET serialized = $1 WHERE id = $2", serialized, policy.GetId())
		return err
	}

	getCategories := func(ctx context.Context) (map[string]string, error) {
		rows, err := db.Query(ctx, "SELECT id, name FROM policy_categories")
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		categories := make(map[string]string)
		for rows.Next() {
			var id, name string
			if err := rows.Scan(&id, &name); err != nil {
				return nil, err
			}
			categories[name] = id
		}
		return categories, rows.Err()
	}

	upsertEdge := func(_ context.Context, _ *storage.PolicyCategoryEdge) error {
		return nil
	}

	removeEdge := func(_ context.Context, _ *storage.PolicyCategoryEdge) error {
		return nil
	}

	return policymigrationhelper.MigratePoliciesWithDiffsAndStoreV2(
		policyDiffFS, policyDiffs, get, upsert, getCategories, upsertEdge, removeEdge,
	)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
