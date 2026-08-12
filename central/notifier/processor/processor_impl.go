package processor

import (
	"context"
	"fmt"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/logging"
	pkgNotifier "github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"

	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
)

var (
	// Replacing with a background context such that outside context cancellation
	// does not affect long-running go routines.
	ctxBackground = context.Background()
	log           = logging.LoggerForModule()
)

// Processor takes in alerts and sends the notifications tied to that alert
type processorImpl struct {
	ns                  pkgNotifier.Set
	reporter            integrationhealth.Reporter
	collectionDataStore collectionDS.DataStore
	collectionResolver  collectionDS.QueryResolver
}

func (p *processorImpl) HasNotifiers() bool {
	return p.ns.HasNotifiers()
}

func (p *processorImpl) HasEnabledAuditNotifiers() bool {
	return p.ns.HasEnabledAuditNotifiers()
}

// RemoveNotifier removes the in memory copy of the specified notifier
func (p *processorImpl) RemoveNotifier(ctx context.Context, id string) {
	p.ns.RemoveNotifier(ctx, id)
}

// GetNotifier gets the in memory copy of the specified notifier id
func (p *processorImpl) GetNotifier(ctx context.Context, id string) (notifier notifiers.Notifier) {
	return p.ns.GetNotifier(ctx, id)
}

// UpdateNotifier updates or adds the passed notifier into memory
func (p *processorImpl) UpdateNotifier(ctx context.Context, notifier notifiers.Notifier) {
	p.ns.UpsertNotifier(ctx, notifier)
}

// ProcessAlert pushes the alert into a channel to be processed
func (p *processorImpl) ProcessAlert(ctx context.Context, alert *storage.Alert) {
	policy := alert.GetPolicy()
	unscopedNotifiers := policy.GetNotifiers()
	scopedMappings := policy.GetNotifierToCollectionMappings()

	if len(unscopedNotifiers) == 0 && len(scopedMappings) == 0 {
		return
	}

	alertNotifiers := set.NewStringSet(unscopedNotifiers...)

	// For scoped mappings, check if the alert entity matches the collection
	for _, mapping := range scopedMappings {
		if p.alertMatchesCollection(ctx, alert, mapping.GetCollectionId()) {
			alertNotifiers.Add(mapping.GetNotifierId())
		}
	}

	if alertNotifiers.Cardinality() == 0 {
		return
	}

	p.ns.ForEach(ctx, func(ctx context.Context, notifier notifiers.Notifier, failures pkgNotifier.AlertSet) {
		if alertNotifiers.Contains(notifier.ProtoNotifier().GetId()) {
			go func() {
				err := pkgNotifier.TryToAlert(ctx, notifier, alert)
				if err != nil {
					p.UpdateNotifierHealthStatus(notifier, storage.IntegrationHealth_UNHEALTHY, err.Error())
					failures.Add(alert)
				} else {
					p.UpdateNotifierHealthStatus(notifier, storage.IntegrationHealth_HEALTHY, "")
				}
			}()
		}
	})
}

func (p *processorImpl) alertMatchesCollection(ctx context.Context, alert *storage.Alert, collectionID string) bool {
	if p.collectionDataStore == nil || p.collectionResolver == nil || collectionID == "" {
		return false
	}

	collection, exists, err := p.collectionDataStore.Get(ctx, collectionID)
	if err != nil || !exists {
		log.Warnf("Could not resolve collection %s for scoped notification: %v", collectionID, err)
		return false
	}

	collectionQuery, err := p.collectionResolver.ResolveCollectionQuery(ctx, collection)
	if err != nil {
		log.Warnf("Could not resolve collection query for %s: %v", collectionID, err)
		return false
	}

	return alertMatchesQuery(alert, collectionQuery)
}

func alertMatchesQuery(alert *storage.Alert, query *v1.Query) bool {
	deployment := alert.GetDeployment()
	if deployment == nil {
		return false
	}

	return matchesQueryRecursive(deployment, query)
}

func matchesQueryRecursive(deployment *storage.Alert_Deployment, query *v1.Query) bool {
	if query == nil {
		return true
	}

	switch q := query.GetQuery().(type) {
	case *v1.Query_BaseQuery:
		return matchesBaseQuery(deployment, q.BaseQuery)
	case *v1.Query_Conjunction:
		for _, subQuery := range q.Conjunction.GetQueries() {
			if !matchesQueryRecursive(deployment, subQuery) {
				return false
			}
		}
		return true
	case *v1.Query_Disjunction:
		for _, subQuery := range q.Disjunction.GetQueries() {
			if matchesQueryRecursive(deployment, subQuery) {
				return true
			}
		}
		return false
	case *v1.Query_BooleanQuery:
		for _, mustQuery := range q.BooleanQuery.GetMust().GetQueries() {
			if !matchesQueryRecursive(deployment, mustQuery) {
				return false
			}
		}
		for _, mustNotQuery := range q.BooleanQuery.GetMustNot().GetQueries() {
			if matchesQueryRecursive(deployment, mustNotQuery) {
				return false
			}
		}
		return true
	}
	return false
}

func matchesBaseQuery(deployment *storage.Alert_Deployment, baseQuery *v1.BaseQuery) bool {
	matchFieldQuery, ok := baseQuery.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
	if !ok {
		return true
	}

	fieldName := matchFieldQuery.MatchFieldQuery.GetField()
	value := matchFieldQuery.MatchFieldQuery.GetValue()

	switch search.FieldLabel(fieldName) {
	case search.Cluster:
		return matchString(deployment.GetClusterName(), value) || matchString(deployment.GetClusterId(), value)
	case search.Namespace:
		return matchString(deployment.GetNamespace(), value)
	case search.DeploymentName:
		return matchString(deployment.GetName(), value)
	}
	return true
}

func matchString(actual, pattern string) bool {
	if pattern == "" {
		return true
	}
	// Strip exact-match quotes added by search.ExactMatchString
	if len(pattern) >= 2 && pattern[0] == '"' && pattern[len(pattern)-1] == '"' {
		pattern = pattern[1 : len(pattern)-1]
	}
	if actual == pattern {
		return true
	}
	if len(pattern) > 2 && pattern[:2] == "r/" {
		return false
	}
	return false
}

// ProcessAuditMessage sends the audit message with all applicable notifiers.
func (p *processorImpl) ProcessAuditMessage(ctx context.Context, msg *v1.Audit_Message) {
	p.ns.ForEach(ctx, func(_ context.Context, notifier notifiers.Notifier, _ pkgNotifier.AlertSet) {
		go p.tryToSendAudit(ctxBackground, notifier, msg)
	})
}

func (p *processorImpl) UpdateNotifierHealthStatus(notifier notifiers.Notifier, healthStatus storage.IntegrationHealth_Status, errMessage string) {
	p.reporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
		Id:            notifier.ProtoNotifier().GetId(),
		Name:          notifier.ProtoNotifier().GetId(),
		Type:          storage.IntegrationHealth_NOTIFIER,
		Status:        healthStatus,
		LastTimestamp: protocompat.TimestampNow(),
		ErrorMessage:  errMessage,
	})
}

func (p *processorImpl) tryToSendAudit(ctx context.Context, notifier notifiers.Notifier, msg *v1.Audit_Message) {
	auditNotifier, ok := notifier.(notifiers.AuditNotifier)
	if ok {
		if err := auditNotifier.SendAuditMessage(ctx, msg); err != nil {
			protoNotifier := notifier.ProtoNotifier()
			log.Errorf("Unable to send audit msg to %s (%s): %v", protoNotifier.GetName(), protoNotifier.GetType(), err)
			p.UpdateNotifierHealthStatus(notifier, storage.IntegrationHealth_UNHEALTHY, fmt.Sprintf("Unable to send audit msg: %v", err))
		}
		p.UpdateNotifierHealthStatus(notifier, storage.IntegrationHealth_HEALTHY, "")
	}
}

// Used for testing.
func (p *processorImpl) processAlertSync(ctx context.Context, alert *storage.Alert) {
	alertNotifiers := set.NewStringSet(alert.GetPolicy().GetNotifiers()...)
	p.ns.ForEach(ctx, func(ctx context.Context, notifier notifiers.Notifier, failures pkgNotifier.AlertSet) {
		if alertNotifiers.Contains(notifier.ProtoNotifier().GetId()) {
			err := pkgNotifier.TryToAlert(ctx, notifier, alert)
			if err != nil {
				failures.Add(alert)
			}
		}
	})
}

// New returns a new Processor
func New(ns pkgNotifier.Set, reporter integrationhealth.Reporter) pkgNotifier.Processor {
	return &processorImpl{
		ns:       ns,
		reporter: reporter,
	}
}

// NewWithCollections returns a new Processor with collection-scoped notification support
func NewWithCollections(ns pkgNotifier.Set, reporter integrationhealth.Reporter, collectionDS collectionDS.DataStore, collectionResolver collectionDS.QueryResolver) pkgNotifier.Processor {
	return &processorImpl{
		ns:                  ns,
		reporter:            reporter,
		collectionDataStore: collectionDS,
		collectionResolver:  collectionResolver,
	}
}
