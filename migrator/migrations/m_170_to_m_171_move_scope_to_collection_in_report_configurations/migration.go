package m170Tom171

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	accessScopePostgres "github.com/stackrox/rox/migrator/migrations/m_170_to_m_171_move_scope_to_collection_in_report_configurations/accessScopePostgresStore"
	collectionPostgres "github.com/stackrox/rox/migrator/migrations/m_170_to_m_171_move_scope_to_collection_in_report_configurations/collectionPostgresStore"
	reportConfigurationPostgres "github.com/stackrox/rox/migrator/migrations/m_170_to_m_171_move_scope_to_collection_in_report_configurations/reportConfigurationPostgresStore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	startSeqNum = 170

	scopeAttachedToConfigsTemplate = "This scope is attached to following report configurations [%s]; "

	manualResolutionTemplate = "Resolution : Please manually create a collection and attach it to the listed report configs. " +
		"These reports will stop working until a collection is attached to them."

	embeddedCollectionTemplate = "System-generated embedded collection %d for scope <%s>"
	rootCollectionTemplate     = "System-generated root collection for scope <%s>"
)

var (
	log = logging.LoggerForModule()

	migration = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)}, // 171
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
				return nil, errors.Errorf("Unsupported operator %s in scope's label selectors. Only operator 'IN' is supported", requirement.GetOp())
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
		return errors.Wrapf(err, "Failed to fetch scope with id %s. "+scopeAttachedToConfigsTemplate,
			scopeID, getJoinedConfigsForScopeID(scopeID))
	}
	if !found {
		log.Errorf("Scope with id %s not found. "+scopeAttachedToConfigsTemplate+manualResolutionTemplate,
			scopeID, strings.Join(scopeIDToConfigNames[scopeID], ", "))
		return nil
	}

	collectionsToEmbed, err := getCollectionsToEmbed(scope)
	if err != nil {
		if strings.Contains(err.Error(), "Unsupported operator") {
			log.Errorf("Failed to create collections for scope <%s>; Reason : %s; "+
				scopeAttachedToConfigsTemplate+manualResolutionTemplate,
				scope.GetName(), err.Error(), getJoinedConfigsForScopeID(scopeID))
			return nil
		}
		return err
	}
	err = collectionStore.UpsertMany(ctx, collectionsToEmbed)
	if err != nil {
		return err
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
	return collectionStore.Upsert(ctx, rootCollection)
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

func getJoinedConfigsForScopeID(scopeID string) string {
	return strings.Join(scopeIDToConfigNames[scopeID], ", ")
}

func init() {
	idGenerator = func(collectionName string) string { return uuid.NewV4().String() }
	migrations.MustRegisterMigration(migration)
}
