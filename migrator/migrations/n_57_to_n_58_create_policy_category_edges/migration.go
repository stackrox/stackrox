package n57ton58

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	"github.com/stackrox/rox/migrator/types"
	pkgMigrations "github.com/stackrox/rox/pkg/migrations"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/uuid"
	"gorm.io/gorm"
)

var (
	migration = types.Migration{
		StartingSeqNum: pkgMigrations.CurrentDBVersionSeqNumWithoutPostgres() + 57,
		VersionAfter:   storage.Version{SeqNum: int32(pkgMigrations.CurrentDBVersionSeqNumWithoutPostgres()) + 58},
		Run: func(databases *types.Databases) error {
			CreatePolicyCategoryEdges(databases.GormDB)
			return nil
		},
	}
	batchSize = 500
	log       = loghelper.LogWrapper{}
)

type PolicyIDAndCategories struct {
	Id         string   `gorm:"column:id;type:varchar;primaryKey"`
	Categories []string `gorm:"column:categories;type:varchar"`
}

type Categories struct {
	Id   string `gorm:"column:id;type:varchar;primaryKey"`
	Name string `gorm:"column:name;type:varchar"`
}

func CreatePolicyCategoryEdges(gormDB *gorm.DB) error {
	policyTable := gormDB.Table(pkgSchema.PoliciesSchema.Table).Model(pkgSchema.CreateTablePoliciesStmt.GormModel)
	categoriesTable := gormDB.Table(pkgSchema.PolicyCategoriesSchema.Table).Model(pkgSchema.CreateTablePolicyCategoriesStmt.GormModel)

	categoriesBuf := make([]Categories, batchSize)
	policyBuf := make([]PolicyIDAndCategories, batchSize)

	var categoryCount int64
	if err := categoriesTable.Count(&categoryCount).Error; err != nil {
		return err
	}
	categoryNameToIDMap := make(map[string]string, categoryCount)

	//read all categories and get category name to id map
	result := categoriesTable.FindInBatches(&categoriesBuf, 2, func(_ *gorm.DB, batch int) error {
		for _, c := range categoriesBuf {
			categoryNameToIDMap[c.Name] = c.Id
		}
		return nil
	})
	if result.Error != nil {
		return result.Error
	}

	var policyCount int64
	if err := policyTable.Count(&policyCount).Error; err != nil {
		return err
	}
	policyToCategoryIDsMap := make(map[string][]string, policyCount)

	//read all policies, create policy id -> category ids edge map for each policy
	result = policyTable.FindInBatches(&policyBuf, batchSize, func(_ *gorm.DB, batch int) error {
		for _, p := range policyBuf {
			policyToCategoryIDsMap[p.Id] = make([]string, 0)
			for _, c := range p.Categories {
				if categoryNameToIDMap[strings.Title(c)] != "" {
					// category exists, can only be a default category
					policyToCategoryIDsMap[p.Id] = append(policyToCategoryIDsMap[p.Id], categoryNameToIDMap[c])
				} else {
					// non default category (since default categories are populated at postgres init)
					id := uuid.NewV4().String()
					if err := categoriesTable.Create(&storage.PolicyCategory{
						Id:        id,
						Name:      strings.Title(c),
						IsDefault: false,
					}).Error; err != nil {
						return err
					}
					policyToCategoryIDsMap[p.Id] = append(policyToCategoryIDsMap[p.Id], id)
					categoryNameToIDMap[c] = id
				}
			}
		}
		return nil
	})
	if result.Error != nil {
		return result.Error
	}
	edgesTable := gormDB.Table(pkgSchema.PolicyCategoryEdgesSchema.Table).Model(pkgSchema.CreateTablePolicyCategoryEdgesStmt.GormModel)
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
		if err := edgesTable.CreateInBatches(edges, len(policyToCategoryIDsMap[policyID])).Error; err != nil {
			return err
		}
	}

	//blank out policy categories for each policy and write those back
	result = gormDB.Model(pkgSchema.CreateTablePoliciesStmt.GormModel).Session(&gorm.Session{AllowGlobalUpdate: true}).
		Update("Categories", []string{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
