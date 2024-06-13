package utils

import (
	"context"

	"github.com/pkg/errors"
	benchmarkDS "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/datastore"
	complianceRuleDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	complianceScanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/pkg/errox"
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

func GetControlsForScanResults(ctx context.Context, ruleDS complianceRuleDS.DataStore, ruleNames []string, profileName string, benchmarkDS benchmarkDS.DataStore) ([]*complianceRuleDS.ControlResult, error) {
	benchmarks, err := benchmarkDS.GetBenchmarksByProfileName(ctx, profileName)
	if err != nil {
		return nil, errors.Wrapf(errox.NotFound, "Unable to retrieve benchmarks for profile %v", profileName)
	}
	// If the profile does not map to a benchmark then we cannot map control data.
	if len(benchmarks) == 0 {
		return nil, nil
	}

	var benchmarkShortNames []string
	for _, benchmark := range benchmarks {
		benchmarkShortNames = append(benchmarkShortNames, benchmark.GetShortName())
	}

	controls, err := ruleDS.GetControlsByRulesAndBenchmarks(ctx, ruleNames, benchmarkShortNames)
	if err != nil {
		return nil, errors.Wrap(err, "could not receive controls by rule controls")
	}
	return controls, nil
}
