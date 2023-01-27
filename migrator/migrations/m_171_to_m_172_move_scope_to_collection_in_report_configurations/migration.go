package m171Tom172

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	accessScopePostgres "github.com/stackrox/rox/migrator/migrations/m_171_to_m_172_move_scope_to_collection_in_report_configurations/accessScopePostgresStore"
	collectionPostgres "github.com/stackrox/rox/migrator/migrations/m_171_to_m_172_move_scope_to_collection_in_report_configurations/collectionPostgresStore"
	reportConfigurationPostgres "github.com/stackrox/rox/migrator/migrations/m_171_to_m_172_move_scope_to_collection_in_report_configurations/reportConfigurationPostgresStore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	startSeqNum = 171

	embeddedCollectionTemplate = "System-generated embedded collection %d for scope <%s>"
	rootCollectionTemplate     = "System-generated root collection for scope <%s>"
)

var (
	log = logging.LoggerForModule()

	migration = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)}, // 172
		Run: func(databases *types.Databases) error {
			err := moveScopesInReportsToCollections(databases.PostgresDB)
			if err != nil {
				return errors.Wrap(err, "error converting scopes to collections in reportConfigurations")
			}
			return nil
		},
	}

	scopeIDToConfigNames = make(map[string][]string)
	idGenerator          func(collectionName string) string
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
	accessScopeStore accessScopePostgres.Store, collectionStore collectionPostgres.Store) error {
	scope, found, err := accessScopeStore.Get(ctx, scopeID)
	if err != nil {
		log.Error(errorWithResolutionMsg(errors.Wrapf(err, "Failed to fetch scope with id %s. ", scopeID), scopeID))
		return nil
	}
	if !found {
		log.Error(errorWithResolutionMsg(errors.Errorf("Scope with id %s not found.", scopeID), scopeID))
		return nil
	}

	collectionsToEmbed, err := getCollectionsToEmbed(scope)
	if err != nil {
		log.Error(errorWithResolutionMsg(errors.Wrapf(err, "Failed create collections for scope <%s>", scope.GetName()), scopeID))
		return nil
	}
	err = collectionStore.UpsertMany(ctx, collectionsToEmbed)
	if err != nil {
		log.Error(errorWithResolutionMsg(errors.Wrapf(err, "Failed create collections for scope <%s>", scope.GetName()), scopeID))
		return nil
	}
	embeddedCollections := make([]*storage.ResourceCollection_EmbeddedResourceCollection, 0, len(collectionsToEmbed))
	for _, collection := range collectionsToEmbed {
		embeddedCollections = append(embeddedCollections, &storage.ResourceCollection_EmbeddedResourceCollection{
			Id: collection.GetId(),
		})
	}
	timeNow := protoconv.ConvertTimeToTimestamp(time.Now())
	rootCollection := &storage.ResourceCollection{
		Id:                  scopeID,
		Name:                fmt.Sprintf(rootCollectionTemplate, scope.GetName()),
		CreatedAt:           timeNow,
		LastUpdated:         timeNow,
		EmbeddedCollections: embeddedCollections,
	}
	if err := collectionStore.Upsert(ctx, rootCollection); err != nil {
		log.Error(errorWithResolutionMsg(errors.Wrapf(err, "Failed create collections for scope <%s>", scope.GetName()), scopeID))
	}
	return nil
}

func moveScopesInReportsToCollections(db *pgxpool.Pool) error {
	ctx := context.Background()
	reportConfigStore := reportConfigurationPostgres.New(db)
	accessScopeStore := accessScopePostgres.New(db)
	collectionStore := collectionPostgres.New(db)

	err := reportConfigStore.Walk(ctx, func(reportConfig *storage.ReportConfiguration) error {
		scopeIDToConfigNames[reportConfig.GetScopeId()] = append(scopeIDToConfigNames[reportConfig.GetScopeId()], reportConfig.GetName())
		return nil
	})
	if err != nil {
		return err
	}

	for scopeID := range scopeIDToConfigNames {
		err = createCollectionsForScope(ctx, scopeID, accessScopeStore, collectionStore)
		if err != nil {
			return err
		}
	}
	return err
}

func errorWithResolutionMsg(err error, scopeID string) string {
	return err.Error() + "\n" +
		" The scope is attached to the following report configurations: " +
		"[" + strings.Join(scopeIDToConfigNames[scopeID], ", ") + "]; " +
		"Please manually create an equivalent collection and attach it to the listed report configurations. " +
		"Note that reports will not function correctly until a collection is attached."
}

func init() {
	idGenerator = func(collectionName string) string { return uuid.NewV4().String() }
	migrations.MustRegisterMigration(migration)
}
