package m170tom171

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v74"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	policyCategoryEdgePostgresStore "github.com/stackrox/rox/migrator/migrations/m_170_to_m_171_create_policy_categories_and_edges/policycategoryedgepostgresstore"
	policyCategoryPostgresStore "github.com/stackrox/rox/migrator/migrations/m_170_to_m_171_create_policy_categories_and_edges/policycategorypostgresstore"
	policyPostgresStore "github.com/stackrox/rox/migrator/migrations/m_170_to_m_171_create_policy_categories_and_edges/policypostgresstore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"gorm.io/gorm"
)

var (
	startSeqNum = 170

	migration = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)}, // 171
		Run: func(databases *types.Databases) error {
			err := CreatePolicyCategoryEdges(databases.GormDB, databases.PostgresDB)
			if err != nil {
				return errors.Wrap(err, "updating policy categories schema")
			}
			return nil
		},
	}

	batchSize = 500
	log       = loghelper.LogWrapper{}
)

// CreatePolicyCategoryEdges reads policies and creates categories and policy <-> category edges
func CreatePolicyCategoryEdges(gormDB *gorm.DB, db *pgxpool.Pool) error {
	pgutils.CreateTableFromModel(context.Background(), gormDB, frozenSchema.CreateTablePolicyCategoryEdgesStmt)

	ctx := sac.WithAllAccess(context.Background())
	policyStore := policyPostgresStore.New(db)
	categoriesStore := policyCategoryPostgresStore.New(db)
	edgeStore := policyCategoryEdgePostgresStore.New(db)

	categoryCount, err := categoriesStore.Count(ctx)
	if err != nil {
		return err
	}
	categoryNameToIDMap := make(map[string]string, categoryCount)

	// read all categories and get category name to id map
	if err = categoriesStore.Walk(ctx, func(category *storage.PolicyCategory) error {
		categoryNameToIDMap[category.Name] = category.Id
		return nil
	}); err != nil {
		return err
	}

	var policyCount int
	policyCount, err = policyStore.Count(ctx)
	if err != nil {
		return err
	}
	policyToCategoryIDsMap := make(map[string][]string, policyCount)

	policiesToUpdate := make([]*storage.Policy, 0, policyCount)
	// read all policies, create policy id -> category ids edge map for each policy
	err = policyStore.Walk(ctx, func(p *storage.Policy) error {
		policyToCategoryIDsMap[p.Id] = make([]string, 0)
		for _, c := range p.Categories {
			if categoryNameToIDMap[strings.Title(c)] != "" {
				// category exists, can only be a default category
				policyToCategoryIDsMap[p.Id] = append(policyToCategoryIDsMap[p.Id], categoryNameToIDMap[c])
			} else {
				// non default category (since default categories are populated in earlier migration)
				id := uuid.NewV4().String()
				if err := categoriesStore.Upsert(ctx, &storage.PolicyCategory{
					Id:        id,
					Name:      strings.Title(c),
					IsDefault: false,
				}); err != nil {
					return err
				}
				policyToCategoryIDsMap[p.Id] = append(policyToCategoryIDsMap[p.Id], id)
				categoryNameToIDMap[c] = id
			}
		}
		// policies will be upserted without category info
		p.Categories = []string{}
		policiesToUpdate = append(policiesToUpdate, p)

		return nil
	})

	if err != nil {
		return err
	}

	// insert policy category edges
	for policyID, categoryIDs := range policyToCategoryIDsMap {
		edges := make([]*storage.PolicyCategoryEdge, 0, len(policyToCategoryIDsMap[policyID]))
		for _, categoryID := range categoryIDs {
			edges = append(edges, &storage.PolicyCategoryEdge{
				Id:         uuid.NewV4().String(),
				PolicyId:   policyID,
				CategoryId: categoryID,
			})
		}
		if err := edgeStore.UpsertMany(ctx, edges); err != nil {
			return err
		}
	}

	// upsert policies with blank categories
	if err = policyStore.UpsertMany(ctx, policiesToUpdate); err != nil {
		return err
	}
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
