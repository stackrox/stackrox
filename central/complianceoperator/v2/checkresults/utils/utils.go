package utils

import (
	"context"

	"github.com/pkg/errors"
	benchmarkDS "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/datastore"
	complianceCheckResultDS "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	complianceProfileDS "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	complianceRuleDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	complianceScanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	types "github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
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

func DeleteOldResults(ctx context.Context, profileRefID string, resultDS complianceCheckResultDS.DataStore, scanDS complianceScanDS.DataStore, profileDS complianceProfileDS.DataStore) error {
	scanRefQuery := search.NewQueryBuilder().
		AddExactMatches(
			search.ComplianceOperatorProfileRef,
			profileRefID,
		).ProtoQuery()
	// Find all the Scans that are associated with this profile
	log.Debugf("searching scans with profile ref %q", profileRefID)
	scans, err := scanDS.SearchScans(ctx, scanRefQuery)
	if err != nil {
		return errors.Wrapf(err, "unable to retrieve scans with profile ref %q", profileRefID)
	}
	if len(scans) == 0 {
		return errors.Errorf("unable to find scans asociated with profile ref %q", profileRefID)
	}
	errList := errorhelpers.NewErrorList("delete old CheckResults")
	for _, s := range scans {
		// If the scan failed we delete the last CheckResults too
		log.Debugf("deleting CheckResults from scan %q", s.GetScanName())
		if err := resultDS.DeleteOldResults(ctx, s.GetLastStartedTime(), s.GetScanRefId(), true); err != nil {
			errList.AddError(errors.Wrapf(err, "unable to delete results for scan ref %q", s.GetScanRefId()))
		}
	}
	return errList.ToError()
}
