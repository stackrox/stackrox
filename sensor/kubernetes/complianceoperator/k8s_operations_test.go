package complianceoperator

import (
	"context"
	"testing"
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/utils/pointer"
)

const testNS = "openshift-compliance"

func TestScanSettingWriter(t *testing.T) {
	suite.Run(t, new(ScanSettingWriterTestSuite))
}

type ScanSettingWriterTestSuite struct {
	suite.Suite
	client *fake.FakeDynamicClient
	writer *scanSettingWriter
}

func (s *ScanSettingWriterTestSuite) SetupTest() {
	gvrToListKind := map[schema.GroupVersionResource]string{
		complianceoperator.ScanSetting.GroupVersionResource():        "ScanSettingList",
		complianceoperator.ScanSettingBinding.GroupVersionResource(): "ScanSettingBindingList",
		machineConfigPoolGVR: "MachineConfigPoolList",
	}
	s.client = fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToListKind)
	r := &retrier{
		maxRetries:     2,
		apiCallTimeout: 100 * time.Millisecond,
		retryTimeout:   500 * time.Millisecond,
	}
	s.writer = &scanSettingWriter{
		client:       s.client,
		retrier:      r,
		roleProvider: func() ([]string, bool) { return []string{masterRole, workerRole}, true },
	}
}

func (s *ScanSettingWriterTestSuite) TestCreateScanResourcesSuccess() {
	request := &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
		ScanName:       "test-scan",
		StrictNodeScan: true,
		Profiles:       []string{"ocp4-cis"},
	}
	err := s.writer.createScanResources(context.Background(), testNS, request, "0 0 * * *")
	s.Require().NoError(err)

	ss, err := s.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).
		Namespace(testNS).Get(context.Background(), "test-scan", v1.GetOptions{})
	s.Require().NoError(err)
	s.Equal("test-scan", ss.GetName())

	ssb, err := s.client.Resource(complianceoperator.ScanSettingBinding.GroupVersionResource()).
		Namespace(testNS).Get(context.Background(), "test-scan", v1.GetOptions{})
	s.Require().NoError(err)
	s.Equal("test-scan", ssb.GetName())
}

func (s *ScanSettingWriterTestSuite) TestCreateScanResourcesDuplicate() {
	request := &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
		ScanName:       "dup-scan",
		StrictNodeScan: true,
		Profiles:       []string{"ocp4-cis"},
	}
	err := s.writer.createScanResources(context.Background(), testNS, request, "0 0 * * *")
	s.Require().NoError(err)

	err = s.writer.createScanResources(context.Background(), testNS, request, "0 0 * * *")
	s.Require().Error(err)
	s.Contains(err.Error(), "already exists")
}

func (s *ScanSettingWriterTestSuite) TestDeleteScanConfigSuccess() {
	request := &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
		ScanName:       "del-scan",
		StrictNodeScan: true,
		Profiles:       []string{"ocp4-cis"},
	}
	err := s.writer.createScanResources(context.Background(), testNS, request, "")
	s.Require().NoError(err)

	err = s.writer.deleteScanConfig(context.Background(), testNS, "del-scan")
	s.Require().NoError(err)

	_, err = s.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).
		Namespace(testNS).Get(context.Background(), "del-scan", v1.GetOptions{})
	s.Error(err)

	_, err = s.client.Resource(complianceoperator.ScanSettingBinding.GroupVersionResource()).
		Namespace(testNS).Get(context.Background(), "del-scan", v1.GetOptions{})
	s.Error(err)
}

func (s *ScanSettingWriterTestSuite) TestDeleteScanConfigNotFound() {
	err := s.writer.deleteScanConfig(context.Background(), testNS, "nonexistent")
	s.Require().NoError(err)
}

func (s *ScanSettingWriterTestSuite) TestSyncScanConfigsCreatesResources() {
	request := &central.SyncComplianceScanConfigRequest{
		ScanConfigs: []*central.ApplyComplianceScanConfigRequest{
			{
				ScanRequest: &central.ApplyComplianceScanConfigRequest_UpdateScan{
					UpdateScan: &central.ApplyComplianceScanConfigRequest_UpdateScheduledScan{
						ScanSettings: &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
							ScanName:       "synced-scan",
							StrictNodeScan: true,
							Profiles:       []string{"ocp4-cis"},
						},
						Cron: "0 0 * * *",
					},
				},
			},
		},
	}
	err := s.writer.syncScanConfigs(context.Background(), testNS, request)
	s.Require().NoError(err)

	ss, err := s.client.Resource(complianceoperator.ScanSetting.GroupVersionResource()).
		Namespace(testNS).Get(context.Background(), "synced-scan", v1.GetOptions{})
	s.Require().NoError(err)
	s.Equal("synced-scan", ss.GetName())

	ssb, err := s.client.Resource(complianceoperator.ScanSettingBinding.GroupVersionResource()).
		Namespace(testNS).Get(context.Background(), "synced-scan", v1.GetOptions{})
	s.Require().NoError(err)
	s.Equal("synced-scan", ssb.GetName())
}

func TestGetScanForReRun(t *testing.T) {
	scan := &v1alpha1.ComplianceScan{
		TypeMeta: v1.TypeMeta{
			Kind:       complianceoperator.ComplianceScan.Kind,
			APIVersion: complianceoperator.GetGroupVersion().String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "my-scan",
			Namespace: "ns",
		},
	}
	obj, err := runtimeObjToUnstructured(scan)
	require.NoError(t, err)

	name, result, err := getScanForReRun(func() (*unstructured.Unstructured, error) {
		return obj, nil
	}, "my-scan")
	require.NoError(t, err)
	assert.Equal(t, "my-scan", name)
	require.NotNil(t, result)

	annotations := result.GetAnnotations()
	_, hasRescan := annotations[rescanAnnotation]
	assert.True(t, hasRescan, "expected rescan annotation to be set")
}

func TestGetScanForReRunNotFound(t *testing.T) {
	name, result, err := getScanForReRun(func() (*unstructured.Unstructured, error) {
		return nil, nil
	}, "missing-scan")
	require.NoError(t, err)
	assert.Equal(t, "missing-scan", name)
	assert.Nil(t, result)
}

func TestGetScanSettingForUpdate(t *testing.T) {
	scanSetting := &v1alpha1.ScanSetting{
		TypeMeta: v1.TypeMeta{
			Kind:       complianceoperator.ScanSetting.Kind,
			APIVersion: complianceoperator.GetGroupVersion().String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "my-setting",
			Namespace: "ns",
		},
	}
	obj, err := runtimeObjToUnstructured(scanSetting)
	require.NoError(t, err)

	config := scanScheduleConfiguration{
		Suspend:  pointer.Bool(true),
		ScanName: "my-setting",
	}

	result, err := getScanSettingForUpdate(func() (*unstructured.Unstructured, error) {
		return obj, nil
	}, config)
	require.NoError(t, err)
	require.NotNil(t, result)

	var updated v1alpha1.ScanSetting
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(result.Object, &updated)
	require.NoError(t, err)
	assert.True(t, updated.Suspend, "expected Suspend to be true")
}

func TestGetScanSettingForUpdateNotFound(t *testing.T) {
	config := scanScheduleConfiguration{
		Suspend:  pointer.Bool(true),
		ScanName: "missing",
	}
	result, err := getScanSettingForUpdate(func() (*unstructured.Unstructured, error) {
		return nil, nil
	}, config)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestGenerateScanIndex(t *testing.T) {
	tests := map[string]struct {
		namespace string
		scanName  string
		expected  string
	}{
		"standard": {
			namespace: "openshift-compliance",
			scanName:  "my-scan",
			expected:  "openshift-compliance-my-scan",
		},
		"empty namespace": {
			namespace: "",
			scanName:  "scan",
			expected:  "-scan",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, generateScanIndex(tc.namespace, tc.scanName))
		})
	}
}
