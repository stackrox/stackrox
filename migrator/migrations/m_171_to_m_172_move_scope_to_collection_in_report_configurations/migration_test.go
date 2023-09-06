//go:build sql_integration

package m171tom172

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	accessScopePostgres "github.com/stackrox/rox/migrator/migrations/m_171_to_m_172_move_scope_to_collection_in_report_configurations/accessScopePostgresStore"
	collectionPostgres "github.com/stackrox/rox/migrator/migrations/m_171_to_m_172_move_scope_to_collection_in_report_configurations/collectionPostgresStore"
	reportConfigurationPostgres "github.com/stackrox/rox/migrator/migrations/m_171_to_m_172_move_scope_to_collection_in_report_configurations/reportConfigurationPostgresStore"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

const (
	id0   = "A161527B-D34F-42B8-A783-23E39B4DE15A"
	id1   = "DC04A5F8-6018-46E5-B590-87325FBF1945"
	id2   = "9C91FA2B-AE95-4C74-98A7-17AF76CC8209"
	id3   = "DE69BC7B-6331-4125-BC99-23877820DC74"
	badID = "thisisnotauuid"
)

var (
	accessScopes = []*storage.SimpleAccessScope{
		{
			Id:   id0,
			Name: id0,
			Rules: &storage.SimpleAccessScope_Rules{
				IncludedClusters: []string{"c1", "c2", "c3"},
			},
		},
		{
			Id:   id1,
			Name: id1,
			Rules: &storage.SimpleAccessScope_Rules{
				IncludedClusters: []string{"c1", "c2", "c3"},
				IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
					{
						ClusterName:   "c4",
						NamespaceName: "ns4",
					},
					{
						ClusterName:   "c5",
						NamespaceName: "ns5",
					},
				},
				ClusterLabelSelectors: []*storage.SetBasedLabelSelector{
					{
						Requirements: []*storage.SetBasedLabelSelector_Requirement{
							{
								Key:    "ck1",
								Op:     storage.SetBasedLabelSelector_IN,
								Values: []string{"ck1v1", "ck1v2"},
							},
							{
								Key:    "ck2",
								Op:     storage.SetBasedLabelSelector_IN,
								Values: []string{"ck2v1"},
							},
						},
					},
					{
						Requirements: []*storage.SetBasedLabelSelector_Requirement{
							{
								Key:    "ck3",
								Op:     storage.SetBasedLabelSelector_IN,
								Values: []string{"ck3v1", "ck3v2"},
							},
						},
					},
				},
				NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{
					{
						Requirements: []*storage.SetBasedLabelSelector_Requirement{
							{
								Key:    "nsk1",
								Op:     storage.SetBasedLabelSelector_IN,
								Values: []string{"nsk1v1", "nsk1v2"},
							},
							{
								Key:    "nsk2",
								Op:     storage.SetBasedLabelSelector_IN,
								Values: []string{"nsk2v1"},
							},
						},
					},
				},
			},
		},
		{
			Id:   id2,
			Name: id2,
			Rules: &storage.SimpleAccessScope_Rules{
				IncludedClusters: []string{"c1", "c2", "c3"},
				ClusterLabelSelectors: []*storage.SetBasedLabelSelector{
					{
						Requirements: []*storage.SetBasedLabelSelector_Requirement{
							{
								Key:    "ck1",
								Op:     storage.SetBasedLabelSelector_NOT_IN,
								Values: []string{"ck1v1", "ck1v2"},
							},
						},
					},
				},
			},
		},
		{
			Id:   id3,
			Name: id3,
			Rules: &storage.SimpleAccessScope_Rules{
				IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
					{
						ClusterName:   "c9",
						NamespaceName: "ns9",
					},
				},
			},
		},
	}

	configIDToReportConfig = map[string]*storage.ReportConfiguration{
		"config0": {
			Id:   "config0",
			Name: "migratable",
			// Report config should have the un-migrated SAC ID due to the original migration (n52_to_n53) not updating it
			ScopeId: accessScopeIDPrefix + id0,
		},
		"config1": {
			Id:      "config1",
			Name:    "migratable",
			ScopeId: accessScopeIDPrefix + id1,
		},
		"config2": {
			Id:   "config2",
			Name: "migratable",
			// This is a scope that was created after SAC migration, so it should've always been an UUID
			ScopeId: id3,
		},
		"config3": {
			Id:      "config3",
			Name:    "unmigratable: bad/old scope_id",
			ScopeId: accessScopeIDPrefix + id2,
		},
		"config4": {
			Id:      "config4",
			Name:    "unmigratable: bad/old scope_id",
			ScopeId: badID,
		},
		"config5": {
			Id:      "config5",
			Name:    "unmigratable: bad/old scope_id",
			ScopeId: accessScopeIDPrefix + badID,
		},
		"config6": {
			Id:      "config6",
			Name:    "unmigratable: complex SAC",
			ScopeId: id2,
		},
	}

	expectedCollections = map[string]*storage.ResourceCollection{
		fmt.Sprintf(embeddedCollectionTemplate, 0, id0): {
			Id:   fmt.Sprintf(embeddedCollectionTemplate, 0, id0),
			Name: fmt.Sprintf(embeddedCollectionTemplate, 0, id0),
			ResourceSelectors: []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: search.Cluster.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{Value: "c1", MatchType: storage.MatchType_EXACT},
								{Value: "c2", MatchType: storage.MatchType_EXACT},
								{Value: "c3", MatchType: storage.MatchType_EXACT},
							},
						},
					},
				},
			},
		},
		fmt.Sprintf(rootCollectionTemplate, id0): {
			Id:   id0,
			Name: fmt.Sprintf(rootCollectionTemplate, id0),
			EmbeddedCollections: []*storage.ResourceCollection_EmbeddedResourceCollection{
				{Id: fmt.Sprintf(embeddedCollectionTemplate, 0, id0)},
			},
		},
		fmt.Sprintf(embeddedCollectionTemplate, 0, id1): {
			Id:   fmt.Sprintf(embeddedCollectionTemplate, 0, id1),
			Name: fmt.Sprintf(embeddedCollectionTemplate, 0, id1),
			ResourceSelectors: []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: search.Cluster.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{Value: "c1", MatchType: storage.MatchType_EXACT},
								{Value: "c2", MatchType: storage.MatchType_EXACT},
								{Value: "c3", MatchType: storage.MatchType_EXACT},
							},
						},
					},
				},
			},
		},
		fmt.Sprintf(embeddedCollectionTemplate, 1, id1): {
			Id:   fmt.Sprintf(embeddedCollectionTemplate, 1, id1),
			Name: fmt.Sprintf(embeddedCollectionTemplate, 1, id1),
			ResourceSelectors: []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: search.Cluster.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{Value: "c4", MatchType: storage.MatchType_EXACT},
							},
						},
						{
							FieldName: search.Namespace.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{Value: "ns4", MatchType: storage.MatchType_EXACT},
							},
						},
					},
				},
			},
		},
		fmt.Sprintf(embeddedCollectionTemplate, 2, id1): {
			Id:   fmt.Sprintf(embeddedCollectionTemplate, 2, id1),
			Name: fmt.Sprintf(embeddedCollectionTemplate, 2, id1),
			ResourceSelectors: []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: search.Cluster.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{Value: "c5", MatchType: storage.MatchType_EXACT},
							},
						},
						{
							FieldName: search.Namespace.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{Value: "ns5", MatchType: storage.MatchType_EXACT},
							},
						},
					},
				},
			},
		},
		fmt.Sprintf(embeddedCollectionTemplate, 3, id1): {
			Id:   fmt.Sprintf(embeddedCollectionTemplate, 3, id1),
			Name: fmt.Sprintf(embeddedCollectionTemplate, 3, id1),
			ResourceSelectors: []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: search.ClusterLabel.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{Value: "ck1=ck1v1", MatchType: storage.MatchType_EXACT},
								{Value: "ck1=ck1v2", MatchType: storage.MatchType_EXACT},
							},
						},
						{
							FieldName: search.ClusterLabel.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{Value: "ck2=ck2v1", MatchType: storage.MatchType_EXACT},
							},
						},
					},
				},
			},
		},
		fmt.Sprintf(embeddedCollectionTemplate, 4, id1): {
			Id:   fmt.Sprintf(embeddedCollectionTemplate, 4, id1),
			Name: fmt.Sprintf(embeddedCollectionTemplate, 4, id1),
			ResourceSelectors: []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: search.ClusterLabel.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{Value: "ck3=ck3v1", MatchType: storage.MatchType_EXACT},
								{Value: "ck3=ck3v2", MatchType: storage.MatchType_EXACT},
							},
						},
					},
				},
			},
		},
		fmt.Sprintf(embeddedCollectionTemplate, 5, id1): {
			Id:   fmt.Sprintf(embeddedCollectionTemplate, 5, id1),
			Name: fmt.Sprintf(embeddedCollectionTemplate, 5, id1),
			ResourceSelectors: []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: search.NamespaceLabel.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{Value: "nsk1=nsk1v1", MatchType: storage.MatchType_EXACT},
								{Value: "nsk1=nsk1v2", MatchType: storage.MatchType_EXACT},
							},
						},
						{
							FieldName: search.NamespaceLabel.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{Value: "nsk2=nsk2v1", MatchType: storage.MatchType_EXACT},
							},
						},
					},
				},
			},
		},
		fmt.Sprintf(rootCollectionTemplate, id1): {
			Id:   id1,
			Name: fmt.Sprintf(rootCollectionTemplate, id1),
			EmbeddedCollections: []*storage.ResourceCollection_EmbeddedResourceCollection{
				{Id: fmt.Sprintf(embeddedCollectionTemplate, 0, id1)},
				{Id: fmt.Sprintf(embeddedCollectionTemplate, 1, id1)},
				{Id: fmt.Sprintf(embeddedCollectionTemplate, 2, id1)},
				{Id: fmt.Sprintf(embeddedCollectionTemplate, 3, id1)},
				{Id: fmt.Sprintf(embeddedCollectionTemplate, 4, id1)},
				{Id: fmt.Sprintf(embeddedCollectionTemplate, 5, id1)},
			},
		},
		fmt.Sprintf(embeddedCollectionTemplate, 0, id3): {
			Id:   fmt.Sprintf(embeddedCollectionTemplate, 0, id3),
			Name: fmt.Sprintf(embeddedCollectionTemplate, 0, id3),
			ResourceSelectors: []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: search.Cluster.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{Value: "c9", MatchType: storage.MatchType_EXACT},
							},
						},
						{
							FieldName: search.Namespace.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{Value: "ns9", MatchType: storage.MatchType_EXACT},
							},
						},
					},
				},
			},
		},
		fmt.Sprintf(rootCollectionTemplate, id3): {
			Id:   id3,
			Name: fmt.Sprintf(rootCollectionTemplate, id3),
			EmbeddedCollections: []*storage.ResourceCollection_EmbeddedResourceCollection{
				{Id: fmt.Sprintf(embeddedCollectionTemplate, 0, id3)},
			},
		},
	}
)

type reportConfigsMigrationTestSuite struct {
	suite.Suite

	db                *pghelper.TestPostgres
	reportConfigStore reportConfigurationPostgres.Store
	accessScopeStore  accessScopePostgres.Store
	collectionStore   collectionPostgres.Store
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(reportConfigsMigrationTestSuite))
}

func (s *reportConfigsMigrationTestSuite) SetupTest() {
	s.db = pghelper.ForT(s.T(), false)
	s.reportConfigStore = reportConfigurationPostgres.New(s.db.DB)
	s.accessScopeStore = accessScopePostgres.New(s.db.DB)
	s.collectionStore = collectionPostgres.New(s.db.DB)

	schema.ApplySchemaForTable(context.Background(), s.db.GetGormDB(), schema.ReportConfigurationsTableName)
	schema.ApplySchemaForTable(context.Background(), s.db.GetGormDB(), schema.SimpleAccessScopesTableName)
}

func (s *reportConfigsMigrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
	scopeIDToConfigs = make(map[string][]*storage.ReportConfiguration)
}

func (s *reportConfigsMigrationTestSuite) TestMigration() {
	ctx := sac.WithAllAccess(context.Background())
	s.NoError(s.accessScopeStore.UpsertMany(ctx, accessScopes))
	s.NoError(s.reportConfigStore.UpsertMany(ctx, configSliceFromMap(configIDToReportConfig)))

	// mock idGenerator func so we can generate predictable Ids. This will help us guess what embedded collection ids will
	// be present in the generated root collections. We can use that to build expected collection objects for matching in tests.
	idGenerator = func(collectionName string) string {
		return collectionName
	}

	dbs := &types.Databases{
		PostgresDB: s.db.DB,
		GormDB:     s.db.GetGormDB(),
	}

	s.NoError(migration.Run(dbs))

	// check all expected collections were generated
	err := s.collectionStore.Walk(ctx, func(collection *storage.ResourceCollection) error {
		expectedCollection, found := expectedCollections[collection.GetName()]
		s.True(found)
		s.Equal(expectedCollection.GetId(), collection.GetId())
		s.Equal(expectedCollection.GetName(), collection.GetName())
		s.Equal(expectedCollection.GetResourceSelectors(), collection.GetResourceSelectors())
		s.Equal(expectedCollection.GetEmbeddedCollections(), collection.GetEmbeddedCollections())
		return nil
	})
	s.NoError(err)

	// check all migratable reports have migrated and unmigratable reports remain the same
	err = s.reportConfigStore.Walk(ctx, func(config *storage.ReportConfiguration) error {
		collection, found, err := s.collectionStore.Get(ctx, config.GetScopeId())
		s.NoError(err)
		if config.GetName() == "migratable" {
			s.True(found)

			// The scopeId in the report should be updated to remove the prefix and just be a UUID (same as in n52_to_n53).
			// The generated root collection, original scope and this now converted scopeId should all be the same.
			// So, we can use the same config.scopeID to get both the access scope and the collection
			scope, scopeFound, err := s.accessScopeStore.Get(ctx, config.GetScopeId())
			s.NoError(err)
			s.True(scopeFound)

			s.Equal(fmt.Sprintf(rootCollectionTemplate, scope.GetName()), collection.GetName())
		} else {
			s.False(found)
			// If migration fails, scopeID is not updated at all
			_, scopeFound, err := s.accessScopeStore.Get(ctx, config.GetScopeId())
			if config.GetName() == "unmigratable: bad/old scope_id" {
				// migration failed due to 1)bad scopeID or 2)old style scopeID that can be converted to uuid, but the SAC is complex
				s.Error(err)
				s.False(scopeFound)
			} else {
				// report config already has uuid type scopeID but collection generation fails due to complex SAC
				s.NoError(err)
				s.True(scopeFound)
			}
		}
		return nil
	})
	s.NoError(err)
}

func (s *reportConfigsMigrationTestSuite) TestMigrationOnCleanDB() {
	dbs := &types.Databases{
		PostgresDB: s.db.DB,
		GormDB:     s.db.GetGormDB(),
	}
	s.NoError(migration.Run(dbs))
}

func configSliceFromMap(reportConfigsMap map[string]*storage.ReportConfiguration) []*storage.ReportConfiguration {
	configs := make([]*storage.ReportConfiguration, 0, len(reportConfigsMap))
	for _, config := range reportConfigsMap {
		configs = append(configs, config)
	}
	return configs
}
