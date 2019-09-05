package datastore

import (
	"context"
	"sort"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	pkgRisk "github.com/stackrox/rox/pkg/risk"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

const (
	paginationQueryLimit = 1000
)

var (
	riskEntitiesRequiringAggregation = map[storage.RiskEntityType]bool{
		storage.RiskEntityType_DEPLOYMENT: true,
		storage.RiskEntityType_NAMESPACE:  true,
		storage.RiskEntityType_CLUSTER:    true,
	}
)

// RiskAggregator aggregates dependee risks for depending risk
type RiskAggregator struct {
	risks DataStore
}

// NewRiskAggregator initializes new RiskAggregator
func NewRiskAggregator(risks DataStore) *RiskAggregator {
	return &RiskAggregator{
		risks: risks,
	}
}

// GetDependencies gets all dependent risks for given entity specified by it's ID and type.
func (a *RiskAggregator) GetDependencies(ctx context.Context, entityID string, entityType storage.RiskEntityType) []*storage.Risk {
	if entityID == "" || entityType == storage.RiskEntityType_UNKNOWN {
		log.Errorf("cannot determine risk dependencies: invalid arguments")
		return nil
	}
	// Dependencies are only maintained for deployments within risk proto.
	// For namespace and cluster risk, we perform search on deployment risks
	// since namespace/cluster to deployment relation can be built from information available in risk entity.
	var risks []*storage.Risk
	switch entityType {
	case storage.RiskEntityType_DEPLOYMENT:
		risks = append(risks, a.getDeploymentRiskDependencies(ctx, entityID)...)
	case storage.RiskEntityType_NAMESPACE:
		risks = append(risks, a.getNamespaceRiskDependencies(ctx, entityID)...)
	case storage.RiskEntityType_CLUSTER:
		risks = append(risks, a.getClusterRiskDependencies(ctx, entityID)...)
	default:
		logging.Errorf("dependency query not supported for %s type risk entity", entityType)
		return nil
	}

	return risks
}

func (a *RiskAggregator) getDeploymentRiskDependencies(ctx context.Context, deploymentID string) []*storage.Risk {
	riskID, err := pkgRisk.GetID(deploymentID, storage.RiskEntityType_DEPLOYMENT)
	if err != nil {
		log.Error(err)
		return nil
	}
	if riskID == "" {
		return nil
	}
	dependentRiskIDs := a.risks.GetDependentRiskIDs(riskID)
	if len(dependentRiskIDs) == 0 {
		return nil
	}
	q := search.NewQueryBuilder().AddDocIDs(dependentRiskIDs...).ProtoQuery()
	dependentRisks := a.paginatedRiskSearch(ctx, q, false)
	return dependentRisks
}

func (a *RiskAggregator) getNamespaceRiskDependencies(ctx context.Context, namespaceID string) []*storage.Risk {
	deploymentRiskQuery := search.NewQueryBuilder().
		AddExactMatches(search.RiskEntityType, storage.RiskEntityType_DEPLOYMENT.String()).
		AddExactMatches(search.NamespaceID, namespaceID).ProtoQuery()

	return a.queryAndFilterDeploymentRisks(ctx, deploymentRiskQuery, true)
}

func (a *RiskAggregator) getClusterRiskDependencies(ctx context.Context, clusterID string) []*storage.Risk {
	// Currently no namespace specific risks are stored hence we get deployment risks
	deploymentRiskQuery := search.NewQueryBuilder().
		AddExactMatches(search.RiskEntityType, storage.RiskEntityType_DEPLOYMENT.String()).
		AddExactMatches(search.ClusterID, clusterID).ProtoQuery()

	return a.queryAndFilterDeploymentRisks(ctx, deploymentRiskQuery, true)
}

func (a *RiskAggregator) queryAndFilterDeploymentRisks(ctx context.Context, q *v1.Query, scrubFactors bool) []*storage.Risk {
	deploymentRisks := a.paginatedRiskSearch(ctx, q, scrubFactors)
	deploymentRisks = filterRisksOnDeployment(ctx, deploymentRisks...)

	var aggregatedDeploymentRisks []*storage.Risk
	for _, deploymentRisk := range deploymentRisks {
		dependentRisks := a.GetDependencies(ctx, deploymentRisk.GetEntity().GetId(), deploymentRisk.GetEntity().GetType())
		aggregateRisks(deploymentRisk, dependentRisks...)
		aggregatedDeploymentRisks = append(aggregatedDeploymentRisks, deploymentRisk)
	}
	scrubRiskFactors(aggregatedDeploymentRisks...)
	return aggregatedDeploymentRisks
}

func (a *RiskAggregator) paginatedRiskSearch(ctx context.Context, q *v1.Query, scrubFactors bool) (risks []*storage.Risk) {
	q.Pagination = &v1.QueryPagination{
		Limit:  paginationQueryLimit,
		Offset: 0,
		SortOptions: []*v1.QuerySortOption{
			{
				Field:    search.RiskScore.String(),
				Reversed: true,
			},
		},
	}
	for {
		pagedRisks, err := a.risks.SearchRawRisks(ctx, q)
		if err != nil {
			log.Error(err)
			return nil
		}
		if len(pagedRisks) == 0 {
			return
		}
		if scrubFactors {
			scrubRiskFactors(pagedRisks...)
		}
		risks = append(risks, pagedRisks...)
		q.Pagination.Offset = paginationQueryLimit
	}
}

func filterRisksOnDeployment(ctx context.Context, risks ...*storage.Risk) []*storage.Risk {
	filtered := risks[:0]
	for _, risk := range risks {
		scopeKeys := sac.KeyForNSScopedObj(risk.GetEntity())
		if ok, err := entityTypeToSACResourceHelper[storage.RiskEntityType_DEPLOYMENT].ReadAllowed(ctx, scopeKeys...); err != nil || !ok {
			continue
		}
		filtered = append(filtered, risk)
	}

	return filtered
}

// AggregationRequired returns true if aggregation of dependee risks is required
func (a *RiskAggregator) AggregationRequired(entityType storage.RiskEntityType) bool {
	return riskEntitiesRequiringAggregation[entityType]
}

func aggregateRisks(risk *storage.Risk, subRisks ...*storage.Risk) {
	groupedRiskResults := make(map[string]*storage.Risk_Result)
	for _, subRisk := range subRisks {
		risk.AggregateScore += subRisk.GetAggregateScore()
		for _, riskResult := range subRisk.GetResults() {
			riskResultName := riskResult.GetName()
			if _, ok := groupedRiskResults[riskResultName]; !ok {
				groupedRiskResults[riskResultName] = riskResult
				continue
			}
			groupedRiskResults[riskResultName].Factors = append(groupedRiskResults[riskResultName].Factors, riskResult.GetFactors()...)
			groupedRiskResults[riskResultName].Score += riskResult.GetScore()
		}
	}

	for _, v := range groupedRiskResults {
		risk.Results = append(risk.Results, v)
	}

	sort.Slice(risk.Results, func(i, j int) bool {
		return pkgRisk.AllIndicatorMap[risk.GetResults()[i].GetName()].DisplayPriority <
			pkgRisk.AllIndicatorMap[risk.GetResults()[j].GetName()].DisplayPriority
	})
}

func scrubRiskFactors(risks ...*storage.Risk) {
	for _, risk := range risks {
		risk.Results = nil
	}
}
