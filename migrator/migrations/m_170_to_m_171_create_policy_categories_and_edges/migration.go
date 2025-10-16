package m170tom171

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v74"
	policyCategoryEdgePostgresStore "github.com/stackrox/rox/migrator/migrations/m_170_to_m_171_create_policy_categories_and_edges/policycategoryedgepostgresstore"
	policyCategoryPostgresStore "github.com/stackrox/rox/migrator/migrations/m_170_to_m_171_create_policy_categories_and_edges/policycategorypostgresstore"
	policyPostgresStore "github.com/stackrox/rox/migrator/migrations/m_170_to_m_171_create_policy_categories_and_edges/policypostgresstore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
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

	defaultCategories = []*storage.PolicyCategory{
		storage.PolicyCategory_builder{
			Name:      "Anomalous Activity",
			Id:        "1cf56ef4-2669-4bcd-928c-cae178e5873f",
			IsDefault: true,
		}.Build(),
		storage.PolicyCategory_builder{
			Name:      "Cryptocurrency Mining",
			Id:        "a1245e73-00b8-422c-a2c5-cac95d87cc4e",
			IsDefault: true,
		}.Build(),
		storage.PolicyCategory_builder{
			Name:      "DevOps Best Practices",
			Id:        "3274122b-a016-441c-9efb-a50fc98b2280",
			IsDefault: true,
		}.Build(),
		storage.PolicyCategory_builder{
			Name:      "Docker CIS",
			Id:        "d2bbe19e-3009-4a0e-a701-a0b621b319a0",
			IsDefault: true,
		}.Build(),
		storage.PolicyCategory_builder{
			Name:      "Kubernetes",
			Id:        "c57c15d2-8c8f-449d-9904-92a4aa325d66",
			IsDefault: true,
		}.Build(),
		storage.PolicyCategory_builder{
			Name:      "Kubernetes Events",
			Id:        "19e04fdf-d7ed-465a-9d37-fa5320aa0c64",
			IsDefault: true,
		}.Build(),
		storage.PolicyCategory_builder{
			Name:      "Network Tools",
			Id:        "9d924f5d-6679-4449-8154-795449c8e754",
			IsDefault: true,
		}.Build(),
		storage.PolicyCategory_builder{
			Name:      "Package Management",
			Id:        "c489b821-27c4-47cb-a461-69796f1aa24e",
			IsDefault: true,
		}.Build(),
		storage.PolicyCategory_builder{
			Name:      "Privileges",
			Id:        "f732f1a5-1515-4e9e-9179-3ab2aefe9ad9",
			IsDefault: true,
		}.Build(),
		storage.PolicyCategory_builder{
			Name:      "Security Best Practices",
			Id:        "99cfb323-c9d3-4e0c-af64-4d0101659866",
			IsDefault: true,
		}.Build(),
		storage.PolicyCategory_builder{
			Name:      "System Modification",
			Id:        "12a75c7e-7651-4e38-ad1d-baed20539aa2",
			IsDefault: true,
		}.Build(),
		storage.PolicyCategory_builder{
			Name:      "Vulnerability Management",
			Id:        "88979ffe-f1b6-48f9-8ef0-e18751196ba6",
			IsDefault: true,
		}.Build(),
	}
)

// CreatePolicyCategoryEdges reads policies and creates categories and policy <-> category edges
func CreatePolicyCategoryEdges(gormDB *gorm.DB, db postgres.DB) error {
	pgutils.CreateTableFromModel(context.Background(), gormDB, frozenSchema.CreateTablePolicyCategoryEdgesStmt)

	ctx := sac.WithAllAccess(context.Background())
	policyStore := policyPostgresStore.New(db)
	categoriesStore := policyCategoryPostgresStore.New(db)
	edgeStore := policyCategoryEdgePostgresStore.New(db)

	// Add default categories
	if err := categoriesStore.UpsertMany(ctx, defaultCategories); err != nil {
		return err
	}

	categoryCount, err := categoriesStore.Count(ctx)
	if err != nil {
		return err
	}
	categoryNameToIDMap := make(map[string]string, categoryCount)

	// read all categories and get category name to id map
	if err = categoriesStore.Walk(ctx, func(category *storage.PolicyCategory) error {
		categoryNameToIDMap[category.GetName()] = category.GetId()
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
		policyToCategoryIDsMap[p.GetId()] = make([]string, 0)
		categorySet := set.NewStringSet()
		for _, c := range p.GetCategories() {
			if strings.TrimSpace(c) == "" {
				continue
			}
			categoryName := strings.Title(c)
			// Ensure there are no duplicate categories for the same policy
			if !categorySet.Add(categoryName) {
				continue
			}
			if categoryID, exists := categoryNameToIDMap[categoryName]; exists {
				// category exists
				policyToCategoryIDsMap[p.GetId()] = append(policyToCategoryIDsMap[p.GetId()], categoryID)
			} else {
				// category does not exist, has to be a non default category
				id := uuid.NewV4().String()
				pc := &storage.PolicyCategory{}
				pc.SetId(id)
				pc.SetName(categoryName)
				pc.SetIsDefault(false)
				if err := categoriesStore.Upsert(ctx, pc); err != nil {
					return err
				}
				policyToCategoryIDsMap[p.GetId()] = append(policyToCategoryIDsMap[p.GetId()], id)
				categoryNameToIDMap[categoryName] = id
			}
		}
		// policies will be upserted without category info
		p.SetCategories([]string{})
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
			pce := &storage.PolicyCategoryEdge{}
			pce.SetId(uuid.NewV4().String())
			pce.SetPolicyId(policyID)
			pce.SetCategoryId(categoryID)
			edges = append(edges, pce)
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
