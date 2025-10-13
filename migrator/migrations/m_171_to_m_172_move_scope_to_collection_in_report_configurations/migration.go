package m171tom172

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v74"
	accessScopePostgres "github.com/stackrox/rox/migrator/migrations/m_171_to_m_172_move_scope_to_collection_in_report_configurations/accessScopePostgresStore"
	collectionPostgres "github.com/stackrox/rox/migrator/migrations/m_171_to_m_172_move_scope_to_collection_in_report_configurations/collectionPostgresStore"
	reportConfigurationPostgres "github.com/stackrox/rox/migrator/migrations/m_171_to_m_172_move_scope_to_collection_in_report_configurations/reportConfigurationPostgresStore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"gorm.io/gorm"
)

const (
	startSeqNum = 171

	accessScopeIDPrefix = "io.stackrox.authz.accessscope."

	embeddedCollectionTemplate = "System-generated embedded collection %d for scope <%s>"
	rootCollectionTemplate     = "System-generated root collection for scope <%s>"
)

var (
	log = logging.LoggerForModule()

	migration = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)}, // 172
		Run: func(databases *types.Databases) error {
			err := moveScopesInReportsToCollections(databases.GormDB, databases.PostgresDB)
			if err != nil {
				return errors.Wrap(err, "error converting scopes to collections in reportConfigurations")
			}
			return nil
		},
	}

	scopeIDToConfigs = make(map[string][]*storage.ReportConfiguration)
	idGenerator      func(collectionName string) string

	// Copied from migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/migration.go
	accessScopeIDMapping = map[string]string{
		"denyall":      "ffffffff-ffff-fff4-f5ff-fffffffffffe",
		"unrestricted": "ffffffff-ffff-fff4-f5ff-ffffffffffff",
	}
)

func buildEmbeddedCollection(scopeName string, index int, rules []*storage.SelectorRule) *storage.ResourceCollection {
	timeNow := protoconv.ConvertTimeToTimestamp(time.Now())
	colName := fmt.Sprintf(embeddedCollectionTemplate, index, scopeName)
	return &storage.ResourceCollection{
		Id:          idGenerator(colName),
		Name:        colName,
		CreatedAt:   timeNow,
		LastUpdated: timeNow,
		ResourceSelectors: []*storage.ResourceSelector{
			{
				Rules: rules,
			},
		},
	}
}

func labelSelectorsToCollections(scopeName string, index int, labelSelectors []*storage.SetBasedLabelSelector, fieldName string) ([]*storage.ResourceCollection, error) {
	collections := make([]*storage.ResourceCollection, 0, len(labelSelectors))
	for _, labelSelector := range labelSelectors {
		selectorRules := make([]*storage.SelectorRule, 0, len(labelSelector.GetRequirements()))
		for _, requirement := range labelSelector.GetRequirements() {
			if requirement.GetOp() != storage.SetBasedLabelSelector_IN {
				return nil, errors.Errorf("Unsupported operator %s in scope's label selectors. Only operator 'IN' is supported.", requirement.GetOp())
			}
			ruleValues := make([]*storage.RuleValue, 0, len(requirement.GetValues()))
			for _, val := range requirement.GetValues() {
				ruleValues = append(ruleValues, &storage.RuleValue{
					Value:     fmt.Sprintf("%s=%s", requirement.GetKey(), val),
					MatchType: storage.MatchType_EXACT,
				})
			}
			selectorRules = append(selectorRules, &storage.SelectorRule{
				FieldName: fieldName,
				Operator:  storage.BooleanOperator_OR,
				Values:    ruleValues,
			})
		}
		col := buildEmbeddedCollection(scopeName, index, selectorRules)
		collections = append(collections, col)
		index = index + 1
	}
	return collections, nil
}

func getCollectionsToEmbed(scope *storage.SimpleAccessScope) ([]*storage.ResourceCollection, error) {
	collectionsToEmbed := make([]*storage.ResourceCollection, 0)

	index := 0
	if includedClusters := scope.GetRules().GetIncludedClusters(); len(includedClusters) > 0 {
		ruleVals := make([]*storage.RuleValue, 0, len(includedClusters))
		for _, cluster := range includedClusters {
			ruleVals = append(ruleVals, &storage.RuleValue{
				Value:     cluster,
				MatchType: storage.MatchType_EXACT,
			})
		}
		col := buildEmbeddedCollection(scope.GetName(), index, []*storage.SelectorRule{
			{
				FieldName: search.Cluster.String(),
				Operator:  storage.BooleanOperator_OR,
				Values:    ruleVals,
			},
		})
		collectionsToEmbed = append(collectionsToEmbed, col)
		index = index + 1
	}

	if includedNamespaces := scope.GetRules().GetIncludedNamespaces(); len(includedNamespaces) > 0 {
		for _, namespace := range includedNamespaces {
			col := buildEmbeddedCollection(scope.GetName(), index, []*storage.SelectorRule{
				{
					FieldName: search.Cluster.String(),
					Operator:  storage.BooleanOperator_OR,
					Values: []*storage.RuleValue{
						{
							Value:     namespace.GetClusterName(),
							MatchType: storage.MatchType_EXACT,
						},
					},
				},
				{
					FieldName: search.Namespace.String(),
					Operator:  storage.BooleanOperator_OR,
					Values: []*storage.RuleValue{
						{
							Value:     namespace.GetNamespaceName(),
							MatchType: storage.MatchType_EXACT,
						},
					},
				},
			})
			collectionsToEmbed = append(collectionsToEmbed, col)
			index = index + 1
		}
	}

	if clusterLabelSelectors := scope.GetRules().GetClusterLabelSelectors(); len(clusterLabelSelectors) > 0 {
		labelCollections, err := labelSelectorsToCollections(scope.GetName(), index, clusterLabelSelectors, search.ClusterLabel.String())
		if err != nil {
			return nil, err
		}
		index = index + len(labelCollections)
		collectionsToEmbed = append(collectionsToEmbed, labelCollections...)
	}

	if namespaceLabelSelectors := scope.GetRules().GetNamespaceLabelSelectors(); len(namespaceLabelSelectors) > 0 {
		labelCollections, err := labelSelectorsToCollections(scope.GetName(), index, namespaceLabelSelectors, search.NamespaceLabel.String())
		if err != nil {
			return nil, err
		}
		collectionsToEmbed = append(collectionsToEmbed, labelCollections...)
	}
	return collectionsToEmbed, nil
}

// Creates embedded and root collections for the access scope with ID scopeID.
// Adds the embedded and root collections to the collection store.
func createCollectionsForScope(ctx context.Context, scopeID string,
	accessScopeStore accessScopePostgres.Store, collectionStore collectionPostgres.Store) (string, bool) {
	newScopeID, err := getNewAccessScopeID(scopeID)
	if err != nil {
		log.Error(errorWithResolutionMsg(errors.Wrapf(err, "Report configuration had an invalid scope id %q.", scopeID), scopeID))
		return "", false
	}
	scope, found, err := accessScopeStore.Get(ctx, newScopeID)
	if err != nil {
		log.Error(errorWithResolutionMsg(errors.Wrapf(err, "Failed to fetch scope with id %q. ", scopeID), scopeID))
		return "", false
	}
	if !found {
		log.Error(errorWithResolutionMsg(errors.Errorf("Scope with id %q not found.", scopeID), scopeID))
		return "", false
	}

	collectionsToEmbed, err := getCollectionsToEmbed(scope)
	if err != nil {
		log.Error(errorWithResolutionMsg(errors.Wrapf(err, "Failed to create collections for scope <%q>", scope.GetName()), scopeID))
		return "", false
	}
	err = collectionStore.UpsertMany(ctx, collectionsToEmbed)
	if err != nil {
		log.Error(errorWithResolutionMsg(errors.Wrapf(err, "Failed to create collections for scope <%q>", scope.GetName()), scopeID))
		return "", false
	}
	embeddedCollections := make([]*storage.ResourceCollection_EmbeddedResourceCollection, 0, len(collectionsToEmbed))
	for _, collection := range collectionsToEmbed {
		embeddedCollections = append(embeddedCollections, &storage.ResourceCollection_EmbeddedResourceCollection{
			Id: collection.GetId(),
		})
	}
	timeNow := protoconv.ConvertTimeToTimestamp(time.Now())
	rootCollection := &storage.ResourceCollection{
		Id:                  newScopeID,
		Name:                fmt.Sprintf(rootCollectionTemplate, scope.GetName()),
		CreatedAt:           timeNow,
		LastUpdated:         timeNow,
		EmbeddedCollections: embeddedCollections,
	}
	if err := collectionStore.Upsert(ctx, rootCollection); err != nil {
		log.Error(errorWithResolutionMsg(errors.Wrapf(err, "Failed to create collections for scope <%q>", scope.GetName()), scopeID))
		return "", false
	}
	return newScopeID, true
}

func moveScopesInReportsToCollections(gormDB *gorm.DB, db postgres.DB) error {
	ctx := sac.WithAllAccess(context.Background())
	pgutils.CreateTableFromModel(ctx, gormDB, frozenSchema.CreateTableCollectionsStmt)
	reportConfigStore := reportConfigurationPostgres.New(db)
	accessScopeStore := accessScopePostgres.New(db)
	collectionStore := collectionPostgres.New(db)

	err := reportConfigStore.Walk(ctx, func(reportConfig *storage.ReportConfiguration) error {
		scopeIDToConfigs[reportConfig.GetScopeId()] = append(scopeIDToConfigs[reportConfig.GetScopeId()], reportConfig)
		return nil
	})
	if err != nil {
		return err
	}

	for scopeID := range scopeIDToConfigs {
		newScopeID, created := createCollectionsForScope(ctx, scopeID, accessScopeStore, collectionStore)
		if created && newScopeID != scopeID {
			// Update each of the reports with the new scope id.
			// This is required since the scope id may have changed between RocksDB and Postgres.
			// See migration n52_to_n53
			configs := scopeIDToConfigs[scopeID]
			for _, config := range configs {
				config.ScopeId = newScopeID
				// Do it one at a time even though it's not as performant so that at least some reports will get migrated
				// even if any fail. It should be rare though.
				if err := reportConfigStore.Upsert(ctx, config); err != nil {
					log.Error(errors.Wrapf(err, "Failed to attach collection with id %s to report configuration '%s'. "+
						"Please manually edit the report configuration to use this collection. "+
						"Note that reports will not function correctly until a collection is attached.",
						newScopeID, config.GetName()))
				}
			}
		}
	}
	return err
}

func errorWithResolutionMsg(err error, scopeID string) string {
	var configNames []string
	for _, config := range scopeIDToConfigs[scopeID] {
		configNames = append(configNames, config.GetName())
	}
	return err.Error() + "\n" +
		" The scope is attached to the following report configurations: " +
		"[" + strings.Join(configNames, ", ") + "]; " +
		"Please manually create an equivalent collection and edit the listed report configurations to use this collection. " +
		"Note that reports will not function correctly until a collection is attached."
}

// Copied and slightly modified func getRoleAccessScopeID from migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/migration.go
func getNewAccessScopeID(scopeID string) (string, error) {
	accessScopeID := strings.TrimPrefix(scopeID, accessScopeIDPrefix)
	if replacement, found := accessScopeIDMapping[accessScopeID]; found {
		accessScopeID = replacement
	}
	_, accessIDParseErr := uuid.FromString(accessScopeID)
	if accessIDParseErr != nil {
		return "", accessIDParseErr
	}
	return accessScopeID, nil
}

func init() {
	idGenerator = func(collectionName string) string { return uuid.NewV4().String() }
	migrations.MustRegisterMigration(migration)
}
