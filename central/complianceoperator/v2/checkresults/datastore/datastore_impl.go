package datastore

import (
	"context"

	store "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
)

var (
	complianceOperatorSAC = sac.ForResource(resources.ComplianceOperator)
)

type datastoreImpl struct {
	store store.Store
	db    postgres.DB
}

// ResourceCountByResultByCluster represents shape of the stats query for compliance operator results
type ResourceCountByResultByCluster struct {
	PassCount          int    `db:"pass_count"`
	FailCount          int    `db:"fail_count"`
	ErrorCount         int    `db:"error_count"`
	InfoCount          int    `db:"info_count"`
	ManualCount        int    `db:"manual_count"`
	NotApplicableCount int    `db:"not_applicable_count"`
	InconsistentCount  int    `db:"inconsistent_count"`
	ClusterID          string `db:"cluster_id"`
	ClusterName        string `db:"cluster"`
	ScanConfigID       string `db:"compliance_scan_config_id"`
	ScanConfigName     string `db:"compliance_scan_name"`
}

// UpsertResult adds the result to the database
func (d *datastoreImpl) UpsertResult(ctx context.Context, result *storage.ComplianceOperatorCheckResultV2) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	// TODO (ROX-18102): populate the standard and control from the rule so that lookup only happens
	// one time on insert and not everytime we pull the results.

	return d.store.Upsert(ctx, result)
}

// DeleteResult removes a result from the database
func (d *datastoreImpl) DeleteResult(ctx context.Context, id string) error {
	if ok, err := complianceOperatorSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return d.store.Delete(ctx, id)
}

// SearchComplianceCheckResults retrieves the scan results specified by query
func (d *datastoreImpl) SearchComplianceCheckResults(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorCheckResultV2, error) {
	if ok, err := complianceOperatorSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	return d.store.GetByQuery(ctx, query)
}

// ComplianceCheckResultStats retrieves the scan results stats specified by query
func (d *datastoreImpl) ComplianceCheckResultStats(ctx context.Context, query *v1.Query) ([]*ResourceCountByResultByCluster, error) {
	if ok, err := complianceOperatorSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	cloned := query.Clone()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.ClusterID).Proto(),
		search.NewQuerySelect(search.Cluster).Proto(),
		search.NewQuerySelect(search.ComplianceOperatorScanConfig).Proto(),
		search.NewQuerySelect(search.ComplianceOperatorScanName).Proto(),
	}
	cloned.GroupBy = &v1.QueryGroupBy{
		Fields: []string{
			search.ClusterID.String(),
			search.Cluster.String(),
			search.ComplianceOperatorScanConfig.String(),
			search.ComplianceOperatorScanName.String(),
		},
	}

	if cloned.Pagination == nil {
		cloned.Pagination = &v1.QueryPagination{}
		cloned.Pagination.SortOptions = []*v1.QuerySortOption{
			{
				Field: search.ClusterID.String(),
			},
			{
				Field: search.Cluster.String(),
			},
			{
				Field: search.ComplianceOperatorScanConfig.String(),
			},
			{
				Field: search.ComplianceOperatorScanName.String(),
			},
		}
	}

	countQuery := d.withCountByResultSelectQuery(cloned, search.ClusterID)
	countResults, err := pgSearch.RunSelectRequestForSchema[ResourceCountByResultByCluster](ctx, d.db, schema.ComplianceOperatorCheckResultV2Schema, countQuery)
	if err != nil {
		return nil, err
	}

	return countResults, nil
}

func (d *datastoreImpl) withCountByResultSelectQuery(q *v1.Query, countOn search.FieldLabel) *v1.Query {
	cloned := q.Clone()
	cloned.Selects = append(cloned.Selects,
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter("pass_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_PASS.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter("fail_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_FAIL.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter("error_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_ERROR.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter("info_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_INFO.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter("manual_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_MANUAL.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter("not_applicable_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_NOT_APPLICABLE.String(),
					).ProtoQuery(),
			).Proto(),
		search.NewQuerySelect(countOn).
			AggrFunc(aggregatefunc.Count).
			Filter("inconsistent_count",
				search.NewQueryBuilder().
					AddExactMatches(
						search.ComplianceOperatorCheckStatus,
						storage.ComplianceOperatorCheckResultV2_INCONSISTENT.String(),
					).ProtoQuery(),
			).Proto(),
	)
	return cloned
}
