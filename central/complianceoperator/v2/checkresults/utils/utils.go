package utils

import (
	"context"

	"github.com/pkg/errors"
	complianceScanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	types "github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
)

func GetLastScanTime(ctx context.Context, clusterID string, profileName string, scanDS complianceScanDS.DataStore) (*types.Timestamp, error) {
	// Check the Compliance Scan object to get the scan time.
	scanQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, profileName).
		AddExactMatches(search.ClusterID, clusterID).
		ProtoQuery()
	scans, err := scanDS.SearchScans(ctx, scanQuery)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to retrieve scan data for cluster %q and profile %q", clusterID, profileName)
	}
	// There should only be a single object for a profile/cluster pair
	if len(scans) == 0 {
		return nil, errors.Errorf("Unable to retrieve scan data for cluster %q and profile %q", clusterID, profileName)
	}

	var lastScanTime *types.Timestamp
	for _, scan := range scans {
		if types.CompareTimestamps(scan.LastExecutedTime, lastScanTime) > 0 {
			lastScanTime = scan.LastExecutedTime
		}
	}

	return lastScanTime, nil
}
