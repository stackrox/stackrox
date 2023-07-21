package datastore

import (
	"context"

	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/central/risk/datastore/internal/search"
	"github.com/stackrox/rox/central/risk/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

var (
	deploymentExtensionSAC = sac.ForResource(resources.DeploymentExtension)
)

type datastoreImpl struct {
	storage            store.Store
	searcher           search.Searcher
	entityTypeToRanker map[string]*ranking.Ranker
}

func (d *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return d.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (d *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.searcher.Count(ctx, q)
}

func (d *datastoreImpl) SearchRawRisks(ctx context.Context, q *v1.Query) ([]*storage.Risk, error) {
	return d.searcher.SearchRawRisks(ctx, q)
}

// TODO: if subject is namespace or cluster, compute risk based on all visible child subjects
func (d *datastoreImpl) GetRisk(ctx context.Context, subjectID string, subjectType storage.RiskSubjectType) (*storage.Risk, bool, error) {
	if allowed, err := deploymentExtensionSAC.ReadAllowed(ctx); err != nil || !allowed {
		return nil, false, err
	}
	return d.getRiskForSubject(ctx, subjectID, subjectType)
}

func (d *datastoreImpl) GetRiskForDeployment(ctx context.Context, deployment *storage.Deployment) (*storage.Risk, bool, error) {
	if allowed, err := deploymentExtensionSAC.ReadAllowed(ctx, sac.KeyForNSScopedObj(deployment)...); err != nil || !allowed {
		return nil, false, err
	}
	return d.getRiskForSubject(ctx, deployment.GetId(), storage.RiskSubjectType_DEPLOYMENT)
}

func (d *datastoreImpl) getRiskForSubject(ctx context.Context, subjectID string, subjectType storage.RiskSubjectType) (*storage.Risk, bool, error) {
	id, err := GetID(subjectID, subjectType)
	if err != nil {
		return nil, false, err
	}

	risk, exists, err := d.getRisk(ctx, id)
	if err != nil || !exists {
		return nil, false, err
	}

	return risk, true, nil
}

func (d *datastoreImpl) GetRiskByIndicators(ctx context.Context, subjectID string, subjectType storage.RiskSubjectType, _ []string) (*storage.Risk, error) {
	if allowed, err := deploymentExtensionSAC.ReadAllowed(ctx); err != nil || !allowed {
		return nil, err
	}
	risk, found, err := d.GetRisk(ctx, subjectID, subjectType)
	if err != nil || !found {
		return nil, err
	}

	overallScore := float32(1.0)
	riskResults := make([]*storage.Risk_Result, 0, len(risk.GetResults()))
	for _, result := range risk.GetResults() {
		overallScore *= result.GetScore()
		riskResults = append(riskResults, result)
	}

	return &storage.Risk{
		Id:      risk.GetId(),
		Subject: risk.GetSubject(),
		Results: riskResults,
		Score:   overallScore,
	}, nil
}

func (d *datastoreImpl) UpsertRisk(ctx context.Context, risk *storage.Risk) error {
	if allowed, err := deploymentExtensionSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !allowed {
		return sac.ErrResourceAccessDenied
	}

	id, err := GetID(risk.GetSubject().GetId(), risk.GetSubject().GetType())
	if err != nil {
		return err
	}

	risk.Id = id
	if err := d.storage.Upsert(ctx, risk); err != nil {
		return err
	}
	upsertRankerRecord(d.getRanker(risk.GetSubject().GetType()), risk.GetSubject().GetId(), risk.GetScore())
	return nil
}

func (d *datastoreImpl) RemoveRisk(ctx context.Context, subjectID string, subjectType storage.RiskSubjectType) error {
	if allowed, err := deploymentExtensionSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !allowed {
		return sac.ErrResourceAccessDenied
	}

	id, err := GetID(subjectID, subjectType)
	if err != nil {
		return err
	}

	risk, exists, err := d.getRisk(ctx, id)
	if err != nil || !exists {
		return err
	}

	if err := d.storage.Delete(ctx, id); err != nil {
		return err
	}
	removeRankerRecord(d.getRanker(risk.GetSubject().GetType()), risk.GetSubject().GetId())
	return nil
}

func (d *datastoreImpl) getRisk(ctx context.Context, id string) (*storage.Risk, bool, error) {
	risk, exists, err := d.storage.Get(ctx, id)
	if err != nil || !exists {
		return nil, false, err
	}
	return risk, true, nil
}

func (d *datastoreImpl) getRanker(subjectType storage.RiskSubjectType) *ranking.Ranker {
	ranker, found := d.entityTypeToRanker[subjectType.String()]
	if !found {
		logging.Panicf("ranker not implemented for type %v", subjectType.String())
	}
	return ranker
}

func upsertRankerRecord(ranker *ranking.Ranker, id string, score float32) {
	ranker.Add(id, score)
}

func removeRankerRecord(ranker *ranking.Ranker, id string) {
	ranker.Remove(id)
}
