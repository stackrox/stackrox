package service

import (
	"context"
	"sort"

	"github.com/gogo/protobuf/proto"
	deployemntDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type itemRange struct {
	offset int
	limit  int
}

type splitQueries struct {
	deploymentDatastore deployemntDS.DataStore
	riskDatastore       riskDS.DataStore

	deploymentQuery      *v1.Query
	deploymentPagination *v1.QueryPagination

	riskQuery      *v1.Query
	riskPagination *v1.QueryPagination

	riskSortOrder map[string]int
	riskItemRange *itemRange
}

func newSplitQueryExecutor(q *v1.Query, ranker *ranking.Ranker, deploymentDatastore deployemntDS.DataStore, riskDatastore riskDS.DataStore) *splitQueries {
	deploymentQuery := filterDeploymentQuery(q)
	deploymentPagination := filterDeploymentPagination(q)

	// Create the risk query.
	riskQuery := filterRiskQuery(q, ranker)
	riskPagination := filterRiskPagination(q)

	return &splitQueries{
		deploymentDatastore: deploymentDatastore,
		riskDatastore:       riskDatastore,

		deploymentQuery:      deploymentQuery,
		deploymentPagination: deploymentPagination,

		riskQuery:      riskQuery,
		riskPagination: riskPagination,
	}
}

func (s *splitQueries) getListDeployments(ctx context.Context) ([]*storage.ListDeployment, error) {
	// If we need to paginate or filter on risk, then update our deployment query and pagination.
	if s.riskQuery != nil || s.riskPagination != nil {
		if err := s.updateDeploymentWithRiskQuery(ctx); err != nil {
			return nil, err
		}
	}

	// Get the deployments that we might need to return.
	deployments, err := s.getPossibleDeployments(ctx)
	if err != nil {
		return nil, err
	}

	// If risk processing resulted in an ordering we need to follow, then sort deployments.
	if s.riskSortOrder != nil {
		sort.SliceStable(deployments, func(i, j int) bool {
			iVal, iHasVal := s.riskSortOrder[deployments[i].GetId()]
			jVal, jHasVal := s.riskSortOrder[deployments[j].GetId()]
			if !iHasVal && !jHasVal {
				return true
			} else if !iHasVal {
				return false
			} else if !jHasVal {
				return true
			}
			return iVal < jVal
		})
	}

	// If risk processing resulted in an item range to use, paginate the deployments to only that range.
	if s.riskItemRange != nil {
		deployments = paginateDeployments(s.riskItemRange, deployments)
	}
	return deployments, nil
}

func (s *splitQueries) getPossibleDeployments(ctx context.Context) ([]*storage.ListDeployment, error) {
	// Construct a query for the possible deployments to return.
	var localQuery *v1.Query
	if s.deploymentQuery != nil {
		localQuery = proto.Clone(s.deploymentQuery).(*v1.Query)
	} else {
		localQuery = search.EmptyQuery()
	}
	if s.deploymentPagination != nil {
		localQuery.Pagination = proto.Clone(s.deploymentPagination).(*v1.QueryPagination)
	}

	// Get the deployments.
	deployments, err := s.deploymentDatastore.SearchListDeployments(ctx, localQuery)
	if err != nil {
		return nil, err
	}
	return deployments, nil
}

// If we have a risk query AND risk page option, then we need to both update the deployment query to only fetch the
// matching risk ids, and use any risk pagination.
// Should only be called if we have either riskQuery, riskPagination, or both.
func (s *splitQueries) updateDeploymentWithRiskQuery(ctx context.Context) error {
	// We need to construct a query that will only get deployments if no query is present, and add pagination if necessary.
	var localQuery *v1.Query
	if s.riskQuery != nil {
		localQuery = proto.Clone(s.riskQuery).(*v1.Query)
	} else {
		localQuery = search.NewQueryBuilder().
			AddStrings(search.RiskSubjectType, storage.RiskSubjectType_DEPLOYMENT.String()).
			ProtoQuery()
	}

	// If we are paginating and filtering on queries, then we can paginate completely.
	if s.riskPagination != nil {
		localQuery.Pagination = proto.Clone(s.riskPagination).(*v1.QueryPagination)
	}

	// If we are paginating but not filtering, then we need to ignore the ranges so that we can filter after sorting the deployments.
	if s.riskQuery == nil {
		localQuery.Pagination.Limit = 0
		localQuery.Pagination.Offset = 0
	}

	// Fetch the risks that match our configured query.
	risks, err := s.riskDatastore.SearchRawRisks(ctx, localQuery)
	if err != nil {
		return err
	}

	// If we have a risk query, we need to update the deployment query to filter based on teh returned risk.
	if s.riskQuery != nil {
		deploymentIDQueryBuilder := search.NewQueryBuilder()
		for _, risk := range risks {
			deploymentIDQueryBuilder.AddDocIDs(risk.GetSubject().GetId())
		}
		deploymentIDQuery := deploymentIDQueryBuilder.ProtoQuery()
		s.deploymentQuery = search.NewConjunctionQuery(s.deploymentQuery, deploymentIDQuery)
	}

	// If we paginated with risk, add an ordering for the deployments.
	if s.riskPagination != nil {
		s.riskSortOrder = make(map[string]int, len(risks))
		for i, risk := range risks {
			s.riskSortOrder[risk.GetSubject().GetId()] = i
		}
	}

	// If we paginated on risk, but didn't filter on risk, we need to create a paging entry.
	if s.riskQuery == nil && (s.riskPagination.GetOffset() != 0 || s.riskPagination.GetLimit() != 0) {
		s.riskItemRange = &itemRange{
			offset: int(s.riskPagination.GetOffset()),
			limit:  int(s.riskPagination.GetLimit()),
		}
	}
	return nil
}

// Static helper functions.
///////////////////////////

func paginateDeployments(paging *itemRange, results []*storage.ListDeployment) []*storage.ListDeployment {
	if len(results) == 0 {
		return nil
	}

	remnants := len(results) - paging.offset
	if remnants <= 0 {
		return nil
	}

	var end int
	if paging.limit == 0 || remnants < paging.limit {
		end = paging.offset + remnants
	} else {
		end = paging.offset + paging.limit
	}

	return results[paging.offset:end]
}
