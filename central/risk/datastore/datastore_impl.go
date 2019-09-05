package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/central/risk/datastore/internal/index"
	"github.com/stackrox/rox/central/risk/datastore/internal/search"
	"github.com/stackrox/rox/central/risk/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	pkgRisk "github.com/stackrox/rox/pkg/risk"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	entityTypeToSACResourceHelper = map[storage.RiskEntityType]sac.ForResourceHelper{
		storage.RiskEntityType_DEPLOYMENT:     sac.ForResource(resources.Deployment),
		storage.RiskEntityType_SERVICEACCOUNT: sac.ForResource(resources.ServiceAccount),
	}
	riskSAC = sac.ForResource(resources.Risk)
)

func (d *datastoreImpl) initRankers() {
	d.entityTypeToRanker = map[storage.RiskEntityType]*ranking.Ranker{
		storage.RiskEntityType_DEPLOYMENT:     ranking.DeploymentRanker(),
		storage.RiskEntityType_IMAGE:          ranking.ImageRanker(),
		storage.RiskEntityType_SERVICEACCOUNT: ranking.ServiceAccRanker(),
	}
}

type datastoreImpl struct {
	riskLock sync.Mutex

	storage            store.Store
	indexer            index.Indexer
	searcher           search.Searcher
	entityTypeToRanker map[storage.RiskEntityType]*ranking.Ranker
	aggregator         *RiskAggregator

	parentToChildRiskMap map[string]set.StringSet
	childToParentRiskMap map[string]set.StringSet
}

func (d *datastoreImpl) buildIndex() error {
	risks, err := d.storage.ListRisks()
	if err != nil {
		return err
	}
	return d.indexer.AddRisks(risks)
}

func (d *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return d.searcher.Search(ctx, q)
}

func (d *datastoreImpl) SearchRawRisks(ctx context.Context, q *v1.Query) ([]*storage.Risk, error) {
	risks, err := d.searcher.SearchRawRisks(ctx, q)
	if err != nil {
		return nil, err
	}
	for _, risk := range risks {
		if d.aggregator.AggregationRequired(risk.GetEntity().GetType()) {
			dependencies := d.aggregator.GetDependencies(ctx, risk.GetEntity().GetId(), risk.GetEntity().GetType())
			if risk == nil || len(dependencies) == 0 {
				continue
			}

			aggregateRisks(risk, dependencies...)
		}
	}

	return risks, nil
}

func (d *datastoreImpl) GetRisk(ctx context.Context, entityID string, entityType storage.RiskEntityType, aggregateRisk bool) (*storage.Risk, bool, error) {
	id, err := pkgRisk.GetID(entityID, entityType)
	if err != nil {
		return nil, false, err
	}

	risk, exists, err := d.getRisk(id)
	if err != nil {
		return nil, false, err
	}
	if !exists && !d.aggregator.AggregationRequired(entityType) {
		return nil, false, nil
	}
	if !exists && !aggregateRisk {
		return nil, false, nil
	}

	if aggregateRisk && d.aggregator.AggregationRequired(entityType) {
		dependencies := d.aggregator.GetDependencies(ctx, entityID, entityType)
		if risk == nil && len(dependencies) == 0 {
			return nil, false, nil
		}
		if risk == nil {
			risk = &storage.Risk{
				Id:             id,
				Score:          float32(0.0),
				AggregateScore: float32(0.0),
				Entity: &storage.RiskEntityMeta{
					Id:   entityID,
					Type: entityType,
				},
			}
		}
		aggregateRisks(risk, dependencies...)
	}

	if allowed, err := riskReadAllowed(ctx, risk); err != nil || !allowed {
		return nil, false, err
	}

	if allowed, err := d.riskEntityReadAllowed(ctx, risk); err != nil || !allowed {
		return nil, false, err
	}

	return risk, true, nil
}

func (d *datastoreImpl) GetRiskByIndicators(ctx context.Context, entityID string, entityType storage.RiskEntityType, riskIndicatorNames []string) (*storage.Risk, error) {
	risk, found, err := d.GetRisk(ctx, entityID, entityType, true)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Errorf("risk for %s %s not found", entityType.String(), entityID)
	}

	overallScore := float32(1.0)
	riskResults := make([]*storage.Risk_Result, 0, len(risk.GetResults()))
	for _, result := range risk.GetResults() {
		overallScore *= result.GetScore()
		riskResults = append(riskResults, result)
	}

	return &storage.Risk{
		Id:      risk.GetId(),
		Entity:  risk.GetEntity(),
		Results: riskResults,
		Score:   overallScore,
	}, nil
}

func (d *datastoreImpl) UpsertRisk(ctx context.Context, risk *storage.Risk) error {
	if ok, err := riskSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}
	riskEntityType := risk.GetEntity().GetType()
	riskEntityID := risk.GetEntity().GetId()
	id, err := pkgRisk.GetID(riskEntityID, riskEntityType)
	if err != nil {
		return err
	}
	risk.Id = id

	// Aggregate scores from all dependent entities
	risk.AggregateScore = risk.GetScore()
	dependentRisks := d.GetDependentRiskIDs(risk.GetId())
	for _, dependentRisk := range dependentRisks {
		entityType, entityID, err := pkgRisk.GetIDParts(dependentRisk)
		if err != nil {
			return err
		}
		risk.AggregateScore += d.getRanker(entityType).GetScoreForID(entityID)
	}

	// Update depending entities
	currentRiskScore := d.getRanker(riskEntityType).GetScoreForID(riskEntityID)
	for _, dependingRiskID := range d.GetDependingRiskIDs(id) {
		entityType, entityID, err := pkgRisk.GetIDParts(dependingRiskID)
		if err != nil {
			return err
		}

		ranker := d.getRanker(entityType)
		oldScore := ranker.GetScoreForID(entityID)
		newScore := oldScore - currentRiskScore + risk.GetAggregateScore()
		ranker.Add(entityID, newScore)

		dependingRisk, exists, err := d.getRisk(dependingRiskID)
		if err != nil {
			return err
		}
		if !exists {
			dependingRisk = &storage.Risk{
				Id:             dependingRiskID,
				Score:          float32(0.0),
				AggregateScore: float32(0.0),
				Entity: &storage.RiskEntityMeta{
					Id:   entityID,
					Type: entityType,
				},
			}
		}
		dependingRisk.AggregateScore = newScore
		if err := d.storage.UpsertRisk(dependingRisk); err != nil {
			return err
		}
		if err := d.indexer.AddRisk(dependingRisk); err != nil {
			return err
		}
	}

	d.getRanker(riskEntityType).Add(riskEntityID, risk.GetAggregateScore())
	if err := d.storage.UpsertRisk(risk); err != nil {
		return err
	}
	return d.indexer.AddRisk(risk)
}

func (d *datastoreImpl) RemoveRisk(ctx context.Context, entityID string, entityType storage.RiskEntityType) error {
	if ok, err := riskSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	id, err := pkgRisk.GetID(entityID, entityType)
	if err != nil {
		return err
	}

	risk, exists, err := d.getRisk(id)
	if err != nil || !exists {
		return err
	}

	// Update depending entities
	var dependingRisks []*storage.Risk
	dependingIds := d.GetDependingRiskIDs(id)
	if len(dependingIds) != 0 {
		riskQuery := pkgSearch.NewQueryBuilder().AddDocIDs(d.GetDependingRiskIDs(id)...).ProtoQuery()
		dependingRisks, err = d.searcher.SearchRawRisks(ctx, riskQuery)
		if err != nil {
			return err
		}
	}
	for _, dependingRisk := range dependingRisks {
		entityType, entityID, err := pkgRisk.GetIDParts(dependingRisk.GetId())
		if err != nil {
			return err
		}
		ranker := d.getRanker(entityType)
		oldScore := ranker.GetScoreForID(entityID)
		ranker.Add(entityID, oldScore-risk.GetAggregateScore())
		if err := d.storage.UpsertRisk(dependingRisk); err != nil {
			return err
		}
		if err := d.indexer.AddRisk(risk); err != nil {
			return err
		}
	}

	d.getRanker(entityType).Remove(entityID)
	d.RemoveRiskDependencies(risk.GetId())
	if err := d.storage.DeleteRisk(id); err != nil {
		return err
	}
	return d.indexer.DeleteRisk(id)
}

func (d *datastoreImpl) getRisk(id string) (*storage.Risk, bool, error) {
	risk, err := d.storage.GetRisk(id)
	if err != nil {
		return nil, false, err
	}

	if risk == nil {
		return nil, false, nil
	}

	return risk, true, nil
}

func (d *datastoreImpl) riskEntityReadAllowed(ctx context.Context, risk *storage.Risk) (bool, error) {
	entityType := risk.GetEntity().GetType()
	scopeKeys := sac.KeyForNSScopedObj(risk.GetEntity())

	if entityType == storage.RiskEntityType_UNKNOWN {
		return false, errors.Errorf("cannot determine scope: risk entity type %s", entityType.String())
	}
	// Currently namespace and cluster risk is mere aggregation of deployment risk.
	// The required scoped based filtering of deployment risks is performed during aggregation.
	if entityType == storage.RiskEntityType_CLUSTER || entityType == storage.RiskEntityType_NAMESPACE {
		return true, nil
	}
	// Image scope derived from deployment.
	if entityType == storage.RiskEntityType_IMAGE {
		riskIDs := d.GetDependingRiskIDs(risk.GetId())
		riskQuery := pkgSearch.NewQueryBuilder().
			AddExactMatches(pkgSearch.RiskEntityType, storage.RiskEntityType_DEPLOYMENT.String()).
			AddDocIDs(riskIDs...).ProtoQuery()
		risks, err := d.searcher.SearchRawRisks(ctx, riskQuery)
		if err != nil {
			return false, err
		}
		for _, risk := range risks {
			if allowed, err := d.riskEntityReadAllowed(ctx, risk); err == nil && allowed {
				return true, nil
			}
		}
		return false, nil
	}
	resourceHelper, ok := entityTypeToSACResourceHelper[entityType]
	if !ok {
		return false, errors.Errorf("cannot determine scope: no sac resource helper for risk entity type %s", entityType.String())
	}
	allowed, err := resourceHelper.ReadAllowed(ctx, scopeKeys...)
	if err != nil || !allowed {
		return false, err
	}
	return true, nil
}

func riskReadAllowed(ctx context.Context, risk *storage.Risk) (bool, error) {
	if ok, err := riskSAC.ReadAllowed(ctx); err != nil || !ok {
		return false, err
	}
	return true, nil
}

func (d *datastoreImpl) getRanker(entityType storage.RiskEntityType) *ranking.Ranker {
	ranker, ok := d.entityTypeToRanker[entityType]
	if !ok {
		logging.Panicf("ranker not implemented for type %v", entityType.String())
	}
	return ranker
}

func (d *datastoreImpl) GetDependingRiskIDs(riskID string) []string {
	d.riskLock.Lock()
	defer d.riskLock.Unlock()
	return d.childToParentRiskMap[riskID].AsSlice()
}

func (d *datastoreImpl) GetDependentRiskIDs(riskID string) []string {
	d.riskLock.Lock()
	defer d.riskLock.Unlock()
	return d.parentToChildRiskMap[riskID].AsSlice()
}

func (d *datastoreImpl) AddRiskDependencies(parentRiskID string, dependentIDs ...string) {
	d.riskLock.Lock()
	defer d.riskLock.Unlock()

	if _, ok := d.parentToChildRiskMap[parentRiskID]; !ok {
		d.parentToChildRiskMap[parentRiskID] = set.NewStringSet()
	}
	d.parentToChildRiskMap[parentRiskID].AddAll(dependentIDs...)

	for _, dependentID := range dependentIDs {
		if _, ok := d.childToParentRiskMap[dependentID]; !ok {
			d.childToParentRiskMap[dependentID] = set.NewStringSet()
		}
		d.childToParentRiskMap[dependentID].Add(parentRiskID)
	}
}

func (d *datastoreImpl) RemoveRiskDependencies(riskID string) {
	d.riskLock.Lock()
	defer d.riskLock.Unlock()

	parentIDs := d.childToParentRiskMap[riskID].AsSlice()

	for _, parentID := range parentIDs {
		d.parentToChildRiskMap[parentID].Remove(riskID)
	}
	delete(d.childToParentRiskMap, riskID)
}
