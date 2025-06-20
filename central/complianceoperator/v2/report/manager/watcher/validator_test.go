package watcher

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	mocksComplianceIntegrationDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/mocks"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

const (
	oldVersion = "v1.5.0"
	clusterID  = "cluster-id"
	scanName   = "test"
)

func withExpectCall(fn func(*mocksComplianceIntegrationDS.MockDataStore)) func(*mocksComplianceIntegrationDS.MockDataStore) {
	if fn == nil {
		return func(_ *mocksComplianceIntegrationDS.MockDataStore) {}
	}
	return fn
}

func TestValidateScanConfigResults(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cases := map[string]struct {
		results                *ScanConfigWatcherResults
		expectFn               func(*mocksComplianceIntegrationDS.MockDataStore)
		expectedFailedClusters map[string]*report.FailedCluster
		expectedError          bool
		expectedExactError     error
	}{
		"no error": {
			results:                getScanConfigResults(2, 0, 0, 1, nil),
			expectFn:               withExpectCall(nil),
			expectedFailedClusters: make(map[string]*report.FailedCluster),
		},
		"two failed clusters": {
			results: getScanConfigResults(2, 2, 0, 1, nil),
			expectFn: withExpectCall(func(ds *mocksComplianceIntegrationDS.MockDataStore) {
				ds.EXPECT().GetComplianceIntegrationByCluster(ctx, newClusterIdMatcher(2, 2)).
					Times(2).Return([]*storage.ComplianceIntegration{
					{
						OperatorInstalled: true,
						Version:           minimumComplianceOperatorVersion,
						OperatorStatus:    storage.COStatus_HEALTHY,
					},
				}, nil)
			}),
			expectedFailedClusters: getFailedClusters(2, 2, 0, 1),
			expectedError:          true,
		},
		"two failed clusters with two scans": {
			results: getScanConfigResults(2, 2, 0, 2, nil),
			expectFn: withExpectCall(func(ds *mocksComplianceIntegrationDS.MockDataStore) {
				ds.EXPECT().GetComplianceIntegrationByCluster(ctx, newClusterIdMatcher(2, 2)).
					Times(4).Return([]*storage.ComplianceIntegration{
					{
						OperatorInstalled: true,
						Version:           minimumComplianceOperatorVersion,
						OperatorStatus:    storage.COStatus_HEALTHY,
					},
				}, nil)
			}),
			expectedFailedClusters: getFailedClusters(2, 2, 0, 2),
			expectedError:          true,
		},
		"two failed clusters scan config watcher timeout": {
			results: getScanConfigResults(2, 2, 0, 1, ErrScanConfigTimeout),
			expectFn: withExpectCall(func(ds *mocksComplianceIntegrationDS.MockDataStore) {
				ds.EXPECT().GetComplianceIntegrationByCluster(ctx, newClusterIdMatcher(2, 2)).
					Times(2).Return([]*storage.ComplianceIntegration{
					{
						OperatorInstalled: true,
						Version:           minimumComplianceOperatorVersion,
						OperatorStatus:    storage.COStatus_HEALTHY,
					},
				}, nil)
			}),
			expectedFailedClusters: getFailedClusters(2, 2, 0, 1),
			expectedError:          true,
			expectedExactError:     report.ErrScanConfigWatcherTimeout,
		},
		"two failed clusters scan config watcher failed": {
			results: getScanConfigResults(2, 2, 0, 1, errors.New("some error")),
			expectFn: withExpectCall(func(ds *mocksComplianceIntegrationDS.MockDataStore) {
				ds.EXPECT().GetComplianceIntegrationByCluster(ctx, newClusterIdMatcher(2, 2)).
					Times(2).Return([]*storage.ComplianceIntegration{
					{
						OperatorInstalled: true,
						Version:           minimumComplianceOperatorVersion,
						OperatorStatus:    storage.COStatus_HEALTHY,
					},
				}, nil)
			}),
			expectedFailedClusters: getFailedClusters(2, 2, 0, 1),
			expectedError:          true,
			expectedExactError:     report.ErrScanWatchersFailed,
		},
		"two missing clusters": {
			results: getScanConfigResults(2, 0, 2, 1, nil),
			expectFn: withExpectCall(func(ds *mocksComplianceIntegrationDS.MockDataStore) {
				ds.EXPECT().GetComplianceIntegrationByCluster(ctx, newClusterIdMatcher(2, 2)).
					Times(2).Return([]*storage.ComplianceIntegration{
					{
						OperatorInstalled: true,
						Version:           minimumComplianceOperatorVersion,
						OperatorStatus:    storage.COStatus_HEALTHY,
					},
				}, nil)
			}),
			expectedFailedClusters: getFailedClusters(2, 0, 2, 1),
			expectedError:          true,
		},
		"two missing clusters and two failed clusters": {
			results: getScanConfigResults(2, 2, 2, 1, nil),
			expectFn: withExpectCall(func(ds *mocksComplianceIntegrationDS.MockDataStore) {
				ds.EXPECT().GetComplianceIntegrationByCluster(ctx, newClusterIdMatcher(2, 4)).
					Times(4).Return([]*storage.ComplianceIntegration{
					{
						OperatorInstalled: true,
						Version:           minimumComplianceOperatorVersion,
						OperatorStatus:    storage.COStatus_HEALTHY,
					},
				}, nil)
			}),
			expectedFailedClusters: getFailedClusters(2, 2, 2, 1),
			expectedError:          true,
		},
	}
	for tName, tCase := range cases {
		t.Run(tName, func(tt *testing.T) {
			coIntegrationDS := getMockedIntegration(tt)
			tCase.expectFn(coIntegrationDS)
			res, err := ValidateScanConfigResults(ctx, tCase.results, coIntegrationDS)
			assert.Equal(tt, len(tCase.expectedFailedClusters), len(res))
			for id, failedCluster := range tCase.expectedFailedClusters {
				actual, ok := res[id]
				require.True(tt, ok)
				assertFailedCluster(tt, failedCluster, actual)
			}
			if tCase.expectedError {
				assert.Error(tt, err)
			} else {
				assert.NoError(tt, err)
				if tCase.expectedExactError != nil {
					assert.ErrorIs(tt, err, tCase.expectedExactError)
				}
			}
		})
	}
}

func TestValidateScanResults(t *testing.T) {
	ctx, validCtxCancelFn := context.WithCancel(context.Background())
	defer validCtxCancelFn()
	canceledCtx, canceledCtxCancelFn := context.WithCancel(context.Background())
	canceledCtxCancelFn()
	cases := map[string]struct {
		operatorStatus            []*storage.ComplianceIntegration
		expectDSError             error
		results                   *ScanWatcherResults
		expectedFailedCluster     *report.FailedCluster
		expectedInstallationError bool
	}{
		"no error": {
			results: &ScanWatcherResults{
				Error: nil,
			},
			expectedFailedCluster:     nil,
			expectedInstallationError: false,
		},
		"internal error due to data store error": {
			results: &ScanWatcherResults{
				Scan: &storage.ComplianceOperatorScanV2{
					ClusterId: clusterID,
				},
				Error: errors.New("some error"),
			},
			expectDSError:             errors.New("some error"),
			expectedFailedCluster:     newFailedCluster(clusterID, "", []string{report.INTERNAL_ERROR}, false),
			expectedInstallationError: true,
		},
		"internal error due to no integration retrieved from data store": {
			results: &ScanWatcherResults{
				Scan: &storage.ComplianceOperatorScanV2{
					ClusterId: clusterID,
				},
				Error: errors.New("some error"),
			},
			operatorStatus:            []*storage.ComplianceIntegration{},
			expectDSError:             nil,
			expectedFailedCluster:     newFailedCluster(clusterID, "", []string{report.INTERNAL_ERROR}, false),
			expectedInstallationError: true,
		},
		"operator not installed": {
			results: &ScanWatcherResults{
				Scan: &storage.ComplianceOperatorScanV2{
					ClusterId: clusterID,
				},
				Error: errors.New("some error"),
			},
			operatorStatus: []*storage.ComplianceIntegration{
				{
					OperatorInstalled: false,
				},
			},
			expectDSError:             nil,
			expectedFailedCluster:     newFailedCluster(clusterID, "", []string{report.COMPLIANCE_NOT_INSTALLED}, false),
			expectedInstallationError: true,
		},
		"operator old version": {
			results: &ScanWatcherResults{
				Scan: &storage.ComplianceOperatorScanV2{
					ClusterId: clusterID,
				},
				Error: errors.New("some error"),
			},
			operatorStatus: []*storage.ComplianceIntegration{
				{
					OperatorInstalled: true,
					Version:           oldVersion,
					OperatorStatus:    storage.COStatus_HEALTHY,
				},
			},
			expectDSError:             nil,
			expectedFailedCluster:     newFailedCluster(clusterID, oldVersion, []string{report.COMPLIANCE_VERSION_ERROR}, false),
			expectedInstallationError: true,
		},
		"scan removed error": {
			results: &ScanWatcherResults{
				Scan: &storage.ComplianceOperatorScanV2{
					ClusterId: clusterID,
					ScanName:  scanName,
				},
				Error: ErrScanRemoved,
			},
			operatorStatus: []*storage.ComplianceIntegration{
				{
					OperatorInstalled: true,
					Version:           minimumComplianceOperatorVersion,
					OperatorStatus:    storage.COStatus_HEALTHY,
				},
			},
			expectDSError:         nil,
			expectedFailedCluster: newFailedCluster(clusterID, minimumComplianceOperatorVersion, []string{fmt.Sprintf(report.SCAN_REMOVED_FMT, scanName)}, true),
		},
		"scan timeout error": {
			results: &ScanWatcherResults{
				SensorCtx: ctx,
				Scan: &storage.ComplianceOperatorScanV2{
					ClusterId: clusterID,
					ScanName:  scanName,
				},
				Error: ErrScanTimeout,
			},
			operatorStatus: []*storage.ComplianceIntegration{
				{
					OperatorInstalled: true,
					Version:           minimumComplianceOperatorVersion,
					OperatorStatus:    storage.COStatus_HEALTHY,
				},
			},
			expectDSError:         nil,
			expectedFailedCluster: newFailedCluster(clusterID, minimumComplianceOperatorVersion, []string{fmt.Sprintf(report.SCAN_TIMEOUT_FMT, scanName)}, true),
		},
		"sensor context canceled error": {
			results: &ScanWatcherResults{
				SensorCtx: canceledCtx,
				Scan: &storage.ComplianceOperatorScanV2{
					ClusterId: clusterID,
					ScanName:  scanName,
				},
				Error: ErrScanTimeout,
			},
			operatorStatus: []*storage.ComplianceIntegration{
				{
					OperatorInstalled: true,
					Version:           minimumComplianceOperatorVersion,
					OperatorStatus:    storage.COStatus_HEALTHY,
				},
			},
			expectDSError:         nil,
			expectedFailedCluster: newFailedCluster(clusterID, minimumComplianceOperatorVersion, []string{fmt.Sprintf(report.SCAN_TIMEOUT_SENSOR_DISCONNECTED_FMT, scanName)}, true),
		},
		"internal error due context canceled error": {
			results: &ScanWatcherResults{
				SensorCtx: ctx,
				Scan: &storage.ComplianceOperatorScanV2{
					ClusterId: clusterID,
					ScanName:  scanName,
				},
				Error: ErrScanContextCancelled,
			},
			operatorStatus: []*storage.ComplianceIntegration{
				{
					OperatorInstalled: true,
					Version:           minimumComplianceOperatorVersion,
					OperatorStatus:    storage.COStatus_HEALTHY,
				},
			},
			expectDSError:         nil,
			expectedFailedCluster: newFailedCluster(clusterID, minimumComplianceOperatorVersion, []string{report.INTERNAL_ERROR}, true),
		},
	}
	for tName, tCase := range cases {
		t.Run(tName, func(tt *testing.T) {
			coIntegrationDS := getMockedIntegration(tt)
			if tCase.operatorStatus != nil || tCase.expectDSError != nil {
				coIntegrationDS.EXPECT().GetComplianceIntegrationByCluster(ctx, clusterID).Times(1).
					Return(tCase.operatorStatus, tCase.expectDSError)
			}
			res, isInstallationError := ValidateScanResults(ctx, tCase.results, coIntegrationDS)
			assertFailedCluster(tt, tCase.expectedFailedCluster, res)
			assert.Equal(tt, tCase.expectedInstallationError, isInstallationError)
		})
	}
}

func TestValidateClusterHealth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cases := map[string]struct {
		operatorStatus []*storage.ComplianceIntegration
		expectDSError  error
		expectedReason []string
	}{
		"no error": {
			operatorStatus: []*storage.ComplianceIntegration{
				{
					OperatorStatus:    storage.COStatus_HEALTHY,
					OperatorInstalled: true,
					Version:           minimumComplianceOperatorVersion,
				},
			},
			expectDSError: nil,
		},
		"unsupported version": {
			operatorStatus: []*storage.ComplianceIntegration{
				{
					OperatorStatus:    storage.COStatus_HEALTHY,
					OperatorInstalled: true,
					Version:           oldVersion,
				},
			},
			expectDSError:  nil,
			expectedReason: []string{report.COMPLIANCE_VERSION_ERROR},
		},
		"operator not installed": {
			operatorStatus: []*storage.ComplianceIntegration{
				{
					OperatorInstalled: false,
				},
			},
			expectDSError:  nil,
			expectedReason: []string{report.COMPLIANCE_NOT_INSTALLED},
		},
		"internal error due to data store error": {
			expectDSError:  errors.New("some error"),
			expectedReason: []string{report.INTERNAL_ERROR},
		},
		"internal error due to no integration retrieved from data store": {
			expectDSError:  nil,
			expectedReason: []string{report.INTERNAL_ERROR},
		},
	}
	for tName, tCase := range cases {
		t.Run(tName, func(tt *testing.T) {
			coIntegrationDS := getMockedIntegration(tt)
			coIntegrationDS.EXPECT().GetComplianceIntegrationByCluster(ctx, clusterID).Times(1).
				Return(tCase.operatorStatus, tCase.expectDSError)
			res := ValidateClusterHealth(ctx, clusterID, coIntegrationDS)
			require.NotNil(tt, res)
			assert.Equal(tt, clusterID, res.ClusterId)
			assert.Equal(tt, tCase.expectedReason, res.Reasons)
			if len(tCase.operatorStatus) > 0 {
				assert.Equal(tt, tCase.operatorStatus[0].GetVersion(), res.OperatorVersion)
			}
		})
	}
}

func getMockedIntegration(t *testing.T) *mocksComplianceIntegrationDS.MockDataStore {
	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})
	return mocksComplianceIntegrationDS.NewMockDataStore(ctrl)
}

func getScanConfigResults(numSuccessfulClusters, numFailedClusters, numMissingClusters, numScansPerCluster int, err error) *ScanConfigWatcherResults {
	scanResults := make(map[string]*ScanWatcherResults)
	var clusters []*storage.ComplianceOperatorScanConfigurationV2_Cluster
	for i := 0; i < numSuccessfulClusters; i++ {
		id := fmt.Sprintf("cluster-%d", i)
		clusters = append(clusters, &storage.ComplianceOperatorScanConfigurationV2_Cluster{
			ClusterId: id,
		})
		for j := 0; j < numScansPerCluster; j++ {
			resultsID := fmt.Sprintf("%s:scan-%d", id, j)
			scanResults[resultsID] = &ScanWatcherResults{
				Scan: &storage.ComplianceOperatorScanV2{
					ClusterId: id,
				},
			}
		}
	}
	for i := numSuccessfulClusters; i < numSuccessfulClusters+numFailedClusters; i++ {
		id := fmt.Sprintf("cluster-%d", i)
		clusters = append(clusters, &storage.ComplianceOperatorScanConfigurationV2_Cluster{
			ClusterId: id,
		})
		for j := 0; j < numScansPerCluster; j++ {
			resultsID := fmt.Sprintf("%s:scan-%d", id, j)
			scanResults[resultsID] = &ScanWatcherResults{
				Scan: &storage.ComplianceOperatorScanV2{
					ClusterId: id,
				},
				SensorCtx: context.Background(),
				Error:     errors.New("some error"),
			}
		}
	}
	for i := numSuccessfulClusters + numFailedClusters; i < numSuccessfulClusters+numFailedClusters+numMissingClusters; i++ {
		id := fmt.Sprintf("cluster-%d", i)
		clusters = append(clusters, &storage.ComplianceOperatorScanConfigurationV2_Cluster{
			ClusterId: id,
		})
	}
	return &ScanConfigWatcherResults{
		ScanResults: scanResults,
		Error:       err,
		ScanConfig: &storage.ComplianceOperatorScanConfigurationV2{
			Clusters: clusters,
		},
	}
}

func getFailedClusters(idx, numFailedClusters, numMissingClusters, numScans int) map[string]*report.FailedCluster {
	ret := make(map[string]*report.FailedCluster)
	for i := idx; i < idx+numFailedClusters; i++ {
		id := fmt.Sprintf("cluster-%d", i)
		failedCluster := &report.FailedCluster{
			ClusterId:       id,
			OperatorVersion: minimumComplianceOperatorVersion,
			Reasons:         []string{report.INTERNAL_ERROR},
		}
		ret[id] = failedCluster
		var reasons []string
		for j := 0; j < numScans; j++ {
			reasons = append(reasons, report.INTERNAL_ERROR)
			failedCluster.FailedScans = append(failedCluster.FailedScans, &storage.ComplianceOperatorScanV2{
				ClusterId: id,
			})
		}
		ret[id].Reasons = reasons
	}
	for i := idx + numFailedClusters; i < idx+numFailedClusters+numMissingClusters; i++ {
		id := fmt.Sprintf("cluster-%d", i)
		failedCluster := &report.FailedCluster{
			ClusterId:       id,
			OperatorVersion: minimumComplianceOperatorVersion,
			Reasons:         []string{report.INTERNAL_ERROR},
		}
		ret[id] = failedCluster
	}
	return ret
}

func newFailedCluster(clusterID, coVersion string, reasons []string, expectScan bool) *report.FailedCluster {
	ret := &report.FailedCluster{
		ClusterId:       clusterID,
		OperatorVersion: coVersion,
		Reasons:         reasons,
	}
	if expectScan {
		ret.FailedScans = []*storage.ComplianceOperatorScanV2{
			{
				ClusterId: clusterID,
				ScanName:  scanName,
			},
		}
	}
	return ret
}

func assertFailedCluster(t *testing.T, expected, actual *report.FailedCluster) {
	if expected == nil && actual == nil {
		return
	}
	assert.Equal(t, expected.ClusterId, actual.ClusterId)
	assert.Equal(t, expected.ClusterName, actual.ClusterName)
	assert.Equal(t, expected.OperatorVersion, actual.OperatorVersion)
	assert.Equal(t, expected.Reasons, actual.Reasons)
	assert.Equal(t, len(expected.FailedScans), len(actual.FailedScans))
	for _, expectedScan := range expected.FailedScans {
		found := false
		for _, actualScan := range actual.FailedScans {
			if proto.Equal(expectedScan, actualScan) {
				found = true
				break
			}
		}
		assert.Truef(t, found, "expected scan %v not found", expectedScan)
	}
}

type clusterIdMatcher struct {
	ids   set.StringSet
	error string
}

func newClusterIdMatcher(idx, numClusters int) *clusterIdMatcher {
	ids := make([]string, 0, numClusters)
	for i := idx; i < idx+numClusters; i++ {
		ids = append(ids, fmt.Sprintf("cluster-%d", i))
	}
	return &clusterIdMatcher{
		ids: set.NewStringSet(ids...),
	}
}

func (m *clusterIdMatcher) Matches(target interface{}) bool {
	id, ok := target.(string)
	if !ok {
		m.error = "target is not of type string"
		return false
	}
	if !m.ids.Contains(id) {
		m.error = fmt.Sprintf("got unexpected id %q", id)
		return false
	}
	return true
}

func (m *clusterIdMatcher) String() string {
	return m.error
}
