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
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

var (
	riskSAC = sac.ForResource(resources.Risk)
)

type datastoreImpl struct {
	storage          store.Store
	indexer          index.Indexer
	searcher         search.Searcher
	deploymentRanker *ranking.Ranker
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
	return d.searcher.SearchRawRisks(ctx, q)
}

// TODO: if subject is namespace or cluster, compute risk based on all visible child subjects
func (d *datastoreImpl) GetRisk(ctx context.Context, subjectID string, subjectType storage.RiskSubjectType) (*storage.Risk, error) {
	id, err := GetID(subjectID, subjectType)
	if err != nil {
		return nil, err
	}

	risk, exists, err := d.getRisk(id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Errorf("risk %s not found", id)
	}

	if allowed, err := allowed(ctx, storage.Access_READ_ACCESS, risk); err != nil || !allowed {
		return nil, err
	}

	return risk, nil
}

func (d *datastoreImpl) GetRiskByIndicators(ctx context.Context, subjectID string, subjectType storage.RiskSubjectType, riskIndicatorNames []string) (*storage.Risk, error) {
	risk, err := d.GetRisk(ctx, subjectID, subjectType)
	if err != nil {
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
	if allowed, err := allowed(ctx, storage.Access_READ_WRITE_ACCESS, risk); err != nil {
		return err
	} else if !allowed {
		return errors.New("permission denied")
	}

	id, err := GetID(risk.GetSubject().GetId(), risk.GetSubject().GetType())
	if err != nil {
		return err
	}

	risk.Id = id
	if err := d.storage.UpsertRisk(risk); err != nil {
		return err
	}
	upsertRankerRecord(d.getRanker(risk.GetSubject().GetType()), risk.GetSubject().GetId(), risk.GetScore())
	return d.indexer.AddRisk(risk)
}

func (d *datastoreImpl) RemoveRisk(ctx context.Context, subjectID string, subjectType storage.RiskSubjectType) error {
	id, err := GetID(subjectID, subjectType)
	if err != nil {
		return err
	}

	risk, exists, err := d.getRisk(id)
	if err != nil || !exists {
		return err
	}

	if allowed, err := allowed(ctx, storage.Access_READ_WRITE_ACCESS, risk); err != nil {
		return err
	} else if !allowed {
		return errors.New("permission denied")
	}

	if err := d.storage.DeleteRisk(id); err != nil {
		return err
	}
	removeRankerRecord(d.getRanker(risk.GetSubject().GetType()), risk.GetSubject().GetId())
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

func allowed(ctx context.Context, sm storage.Access, risk *storage.Risk) (bool, error) {
	subject := risk.GetSubject()
	if subject.GetType() == storage.RiskSubjectType_UNKNOWN {
		return false, errors.Errorf("cannot determine scope: risk subject type %s", subject.GetType().String())
	}
	scopeKeys := sac.KeyForNSScopedObj(subject)
	if ok, err := riskSAC.ScopeChecker(ctx, sm, scopeKeys...).Allowed(ctx); err != nil || !ok {
		return false, err
	}

	return true, nil
}

func (d *datastoreImpl) getRanker(subjectType storage.RiskSubjectType) *ranking.Ranker {
	switch subjectType {
	case storage.RiskSubjectType_DEPLOYMENT:
		return d.deploymentRanker
	default:
		logging.Panicf("ranker not implemented for type %v", subjectType.String())
	}
	return nil
}

func upsertRankerRecord(ranker *ranking.Ranker, id string, score float32) {
	ranker.Add(id, score)
}

func removeRankerRecord(ranker *ranking.Ranker, id string) {
	ranker.Remove(id)
}
