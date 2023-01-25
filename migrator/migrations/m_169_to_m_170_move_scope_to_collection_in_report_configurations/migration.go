package m169Tom170

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	accessScopePostgres "github.com/stackrox/rox/migrator/migrations/m_169_to_m_170_move_scope_to_collection_in_report_configurations/accessScopePostgresStore"
	collectionPostgres "github.com/stackrox/rox/migrator/migrations/m_169_to_m_170_move_scope_to_collection_in_report_configurations/collectionPostgresStore"
	reportConfigurationPostgres "github.com/stackrox/rox/migrator/migrations/m_169_to_m_170_move_scope_to_collection_in_report_configurations/reportConfigurationPostgresStore"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	batchSize = 500

	startSeqNum = 169

	skippedMigrationMessageTemplate = "Failed to migrate report configuration '%s': Attached scope <%s> has label selector/s " +
		"with a different operator than 'IN'; Resolution: %s"

	manualInterventionInstruction = "Please manually create a collection and add it to the report configuration. " +
		"The report will stop working until a collection is added to it."

	embeddedCollectionTemplate = "System-generated embedded collection %d for scope <%s>"
	rootCollectionTemplate     = "System-generated root collection for scope <%s>"
)

var (
	log = logging.LoggerForModule()

	migration = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)}, // 170
		Run: func(databases *types.Databases) error {
			err := moveScopeIDToCollectionIDInReports(databases.PostgresDB)
			if err != nil {
				return errors.Wrap(err, "error converting scopes to collections in reportConfigurations")
			}
			return nil
		},
	}

	scopeIDToCollectionID = make(map[string]string)
	skippedScopes         = set.NewStringSet()
	idGenerator           func(collectionName string) string
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

func labelSelectorsToCollections(scopeName string, index int, labelSelectors []*storage.SetBasedLabelSelector, fieldName string) ([]*storage.ResourceCollection, bool) {
	collections := make([]*storage.ResourceCollection, 0, len(labelSelectors))
	for _, labelSelector := range labelSelectors {
		selectorRules := make([]*storage.SelectorRule, 0, len(labelSelector.GetRequirements()))
		for _, requirement := range labelSelector.GetRequirements() {
			if requirement.GetOp() != storage.SetBasedLabelSelector_IN {
				return nil, false
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
	return collections, true
}

func createCollectionsToEmbedFromScope(scope *storage.SimpleAccessScope) ([]*storage.ResourceCollection, bool) {
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
		labelCollections, success := labelSelectorsToCollections(scope.GetName(), index, clusterLabelSelectors, search.ClusterLabel.String())
		if !success {
			return nil, false
		}
		index = index + len(labelCollections)
		collectionsToEmbed = append(collectionsToEmbed, labelCollections...)
	}

	if namespaceLabelSelectors := scope.GetRules().GetNamespaceLabelSelectors(); len(namespaceLabelSelectors) > 0 {
		labelCollections, success := labelSelectorsToCollections(scope.GetName(), index, namespaceLabelSelectors, search.NamespaceLabel.String())
		if !success {
			return nil, false
		}
		collectionsToEmbed = append(collectionsToEmbed, labelCollections...)
	}
	return collectionsToEmbed, true
}

func moveScopeIDToCollectionIDInReports(db *pgxpool.Pool) error {
	ctx := context.Background()
	reportConfigStore := reportConfigurationPostgres.New(db)
	accessScopeStore := accessScopePostgres.New(db)
	collectionStore := collectionPostgres.New(db)

	reportConfigsToUpsert := make([]*storage.ReportConfiguration, 0, batchSize)
	err := reportConfigStore.Walk(ctx, func(reportConfig *storage.ReportConfiguration) error {
		scopeID := reportConfig.GetScopeId()
		scope, found, err := accessScopeStore.Get(ctx, scopeID)
		if err != nil {
			return errors.Wrapf(err, "error migrating report configuration '%s': failed to fetch scope with id %s", reportConfig.GetName(), scopeID)
		}
		if !found {
			log.Errorf("Failed to migrate report configuration '%s': Scope with id %s not found; Resolution: %s",
				reportConfig.GetName(), scopeID, manualInterventionInstruction)
		}
		if _, exists := scopeIDToCollectionID[scopeID]; !exists && !skippedScopes.Contains(scopeID) {
			collectionsToEmbed, success := createCollectionsToEmbedFromScope(scope)
			if !success {
				skippedScopes.Add(scopeID)
				log.Warnf(skippedMigrationMessageTemplate, reportConfig.GetName(), scope.GetName(), manualInterventionInstruction)
				return nil
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
			rootColName := fmt.Sprintf(rootCollectionTemplate, scope.GetName())
			rootCollection := &storage.ResourceCollection{
				Id:                  idGenerator(rootColName),
				Name:                rootColName,
				CreatedAt:           timeNow,
				LastUpdated:         timeNow,
				EmbeddedCollections: embeddedCollections,
			}
			err = collectionStore.Upsert(ctx, rootCollection)
			if err != nil {
				return err
			}
			scopeIDToCollectionID[reportConfig.GetScopeId()] = rootCollection.GetId()
			reportConfig.ScopeId = rootCollection.GetId()
			reportConfigsToUpsert = append(reportConfigsToUpsert, reportConfig)
			if len(reportConfigsToUpsert) >= batchSize {
				err = reportConfigStore.UpsertMany(ctx, reportConfigsToUpsert)
				if err != nil {
					return err
				}
				reportConfigsToUpsert = reportConfigsToUpsert[:0]
			}
		} else if skippedScopes.Contains(scopeID) {
			log.Warnf(skippedMigrationMessageTemplate, reportConfig.GetName(), scope.GetName(), manualInterventionInstruction)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(reportConfigsToUpsert) > 0 {
		err = reportConfigStore.UpsertMany(ctx, reportConfigsToUpsert)
	}
	return err
}

func init() {
	idGenerator = func(collectionName string) string { return uuid.NewV4().String() }
	migrations.MustRegisterMigration(migration)
}
