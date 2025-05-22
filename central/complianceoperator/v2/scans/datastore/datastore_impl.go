package datastore

import (
	"context"
	"fmt"
	"strings"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/scans/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
)

type datastoreImpl struct {
	db    postgres.DB
	store pgStore.Store
}

// GetScan retrieves the scan object from the database
func (d *datastoreImpl) GetScan(ctx context.Context, id string) (*storage.ComplianceOperatorScanV2, bool, error) {
	return d.store.Get(ctx, id)
}

// UpsertScan adds the scan object to the database.  If enabling the use of this
// method from a service, the creation of the `ProfileRefID` and `ScanRefID` must be accounted for.  In reality this
// method should only be used by the pipeline as this is a compliance operator object we are storing.
func (d *datastoreImpl) UpsertScan(ctx context.Context, scan *storage.ComplianceOperatorScanV2) error {
	return d.store.Upsert(ctx, scan)
}

// DeleteScan removes a scan object from the database
func (d *datastoreImpl) DeleteScan(ctx context.Context, id string) error {
	return d.store.Delete(ctx, id)
}

// GetScansByCluster retrieves scan objects by cluster
func (d *datastoreImpl) GetScansByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorScanV2, error) {
	return d.store.GetByQuery(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).ProtoQuery())
}

// DeleteScanByCluster deletes scans by cluster
func (d *datastoreImpl) DeleteScanByCluster(ctx context.Context, clusterID string) error {
	query := search.NewQueryBuilder().AddStrings(search.ClusterID, clusterID).ProtoQuery()
	_, err := d.store.DeleteByQuery(ctx, query)
	if err != nil {
		return err
	}
	return nil
}

// SearchScans returns the scans for the given query
func (d *datastoreImpl) SearchScans(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorScanV2, error) {
	return d.store.GetByQuery(ctx, query)
}

func (d *datastoreImpl) GetProfilesScanNamesByScanConfigAndCluster(ctx context.Context, scanConfigID, clusterID string) (map[string]string, error) {
	return d.GetProfileScanNamesByScanConfigClusterAndProfileRef(ctx, scanConfigID, clusterID, []string{})
}

func (d *datastoreImpl) GetProfileScanNamesByScanConfigClusterAndProfileRef(ctx context.Context, scanConfigID, clusterID string, profileRefs []string) (map[string]string, error) {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).
		AddExactMatches(search.ComplianceOperatorScanConfig, scanConfigID).
		ProtoQuery()
	if len(profileRefs) > 0 {
		query = search.ConjunctionQuery(
			search.NewQueryBuilder().
				AddExactMatches(search.ComplianceOperatorProfileRef, profileRefs...).
				ProtoQuery(),
			query)
	}
	type ScanProfiles struct {
		ProfileName string `db:"compliance_profile_name"`
		ScanName    string `db:"compliance_scan_name"`
	}
	query.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(search.ComplianceOperatorProfileName).Proto(),
		search.NewQuerySelect(search.ComplianceOperatorScanName).Proto(),
	}
	results, err := pgSearch.RunSelectRequestForSchema[ScanProfiles](ctx, d.db, schema.ComplianceOperatorScanV2Schema, query)
	if err != nil {
		return nil, err
	}
	expandedProfileNames := make(map[string]string)
	for _, result := range results {
		name := result.ProfileName
		if result.ProfileName != result.ScanName {
			name = fmt.Sprintf("%s%s", result.ProfileName, strings.TrimPrefix(result.ScanName, result.ProfileName))
		}
		expandedProfileNames[result.ScanName] = name
	}
	return expandedProfileNames, nil
}
