package complianceoperator

import (
	"context"
	"testing"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

const (
	coNamespace = "openshift-compliance"
)

var (
	coDeployment = helper.K8sResourceInfo{Kind: "Deployment", YamlFile: "co-deployment.yaml", Name: "compliance-operator"}

	testScanConfig = &central.ApplyComplianceScanConfigRequest{
		ScanRequest: &central.ApplyComplianceScanConfigRequest_UpdateScan{
			UpdateScan: &central.ApplyComplianceScanConfigRequest_UpdateScheduledScan{
				ScanSettings: &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
					ScanName: "test",
					Profiles: []string{"ocp4-cis"},
				},
				Cron: "0 1 * * *",
			},
		},
	}
	updatedTestScanConfig = &central.ApplyComplianceScanConfigRequest{
		ScanRequest: &central.ApplyComplianceScanConfigRequest_UpdateScan{
			UpdateScan: &central.ApplyComplianceScanConfigRequest_UpdateScheduledScan{
				ScanSettings: &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
					ScanName: "test",
					Profiles: []string{"ocp4-cis", "ocp4-cis-node"},
				},
				Cron: "0 2 * * *",
			},
		},
	}
	testScanConfig2 = &central.ApplyComplianceScanConfigRequest{
		ScanRequest: &central.ApplyComplianceScanConfigRequest_UpdateScan{
			UpdateScan: &central.ApplyComplianceScanConfigRequest_UpdateScheduledScan{
				ScanSettings: &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
					ScanName: "test-2",
					Profiles: []string{"ocp4-moderate"},
				},
				Cron: "0 1 * * *",
			},
		},
	}

	namespaceAPIResource = v1.APIResource{
		Name:    "namespaces",
		Kind:    "Namespace",
		Group:   "",
		Version: "v1",
	}
)

func Test_ComplianceOperatorScanConfigSync(t *testing.T) {
	t.Setenv(env.ConnectionRetryInitialInterval.EnvVar(), "1s")
	t.Setenv(env.ConnectionRetryMaxInterval.EnvVar(), "2s")

	centralCaps := []string{
		centralsensor.SendDeduperStateOnReconnect,
		centralsensor.ComplianceV2Integrations,
	}
	c, err := helper.NewContextWithConfig(t, helper.Config{
		CentralCaps:           centralCaps,
		InitialSystemPolicies: nil,
		CertFilePath:          "../../../tools/local-sensor/certs/",
	})
	t.Cleanup(c.Stop)

	require.NoError(t, err)

	c.RunTest(t, helper.WithTestCase(func(t *testing.T, tc *helper.TestContext, _ map[string]k8s.Object) {
		ctx := context.Background()
		// Create the Compliance Operator CRDs
		deleteCRDsFn, err := tc.ApplyWithManifestDir(context.Background(), "../../../tests/complianceoperator/crds", "*")
		t.Cleanup(func() {
			// Deleting the namespace should take care of all the resources created in this test
			utils.IgnoreError(deleteCRDsFn)
			// Wait for the namespace to be deleted
			tc.WaitForResourceDelete(ctx, t, coNamespace, "", namespaceAPIResource)
		})

		require.NoError(t, err)

		// Create a fake Compliance Operator Deployment
		coDep := &appsv1.Deployment{}
		_, err = tc.ApplyResourceAndWait(ctx, t, coNamespace, &coDeployment, coDep, nil)

		require.NoError(t, err)

		// Send a SyncScanConfigs Message
		tc.GetFakeCentral().StubMessage(createSyncScanConfigsMessage(testScanConfig))

		// Assert the resources are created
		scanSetting := tc.AssertResourceDoesExist(ctx, t, testScanConfig.GetUpdateScan().GetScanSettings().GetScanName(), coNamespace, complianceoperator.ScanSetting.APIResource)
		scanSettingBinding := tc.AssertResourceDoesExist(ctx, t, testScanConfig.GetUpdateScan().GetScanSettings().GetScanName(), coNamespace, complianceoperator.ScanSettingBinding.APIResource)

		assertScanSetting(t, testScanConfig, scanSetting)
		assertScanSettingBinding(t, testScanConfig, scanSettingBinding)

		// Restart connection
		tc.RestartFakeCentralConnection(centralCaps...)

		// Send a SyncScanConfigs Message
		tc.GetFakeCentral().StubMessage(createSyncScanConfigsMessage(updatedTestScanConfig, testScanConfig2))

		scanSetting = tc.AssertResourceWasUpdated(ctx, t, updatedTestScanConfig.GetUpdateScan().GetScanSettings().GetScanName(), coNamespace, complianceoperator.ScanSetting.APIResource, scanSetting.GetResourceVersion())
		scanSettingBinding = tc.AssertResourceWasUpdated(ctx, t, updatedTestScanConfig.GetUpdateScan().GetScanSettings().GetScanName(), coNamespace, complianceoperator.ScanSettingBinding.APIResource, scanSettingBinding.GetResourceVersion())

		assertScanSetting(t, updatedTestScanConfig, scanSetting)
		assertScanSettingBinding(t, updatedTestScanConfig, scanSettingBinding)

		scanSetting = tc.AssertResourceDoesExist(ctx, t, testScanConfig2.GetUpdateScan().GetScanSettings().GetScanName(), coNamespace, complianceoperator.ScanSetting.APIResource)
		scanSettingBinding = tc.AssertResourceDoesExist(ctx, t, testScanConfig2.GetUpdateScan().GetScanSettings().GetScanName(), coNamespace, complianceoperator.ScanSettingBinding.APIResource)

		assertScanSetting(t, testScanConfig2, scanSetting)
		assertScanSettingBinding(t, testScanConfig2, scanSettingBinding)

		// Restart connection
		tc.RestartFakeCentralConnection(centralCaps...)

		// Send a SyncScanConfigs Message
		tc.GetFakeCentral().StubMessage(createSyncScanConfigsMessage())

		tc.AssertResourceDoesNotExist(ctx, t, testScanConfig.GetUpdateScan().GetScanSettings().GetScanName(), coNamespace, complianceoperator.ScanSetting.APIResource)
		tc.AssertResourceDoesNotExist(ctx, t, testScanConfig.GetUpdateScan().GetScanSettings().GetScanName(), coNamespace, complianceoperator.ScanSettingBinding.APIResource)
		tc.AssertResourceDoesNotExist(ctx, t, testScanConfig2.GetUpdateScan().GetScanSettings().GetScanName(), coNamespace, complianceoperator.ScanSetting.APIResource)
		tc.AssertResourceDoesNotExist(ctx, t, testScanConfig2.GetUpdateScan().GetScanSettings().GetScanName(), coNamespace, complianceoperator.ScanSettingBinding.APIResource)
	}))
}

func createSyncScanConfigsMessage(scanConfigs ...*central.ApplyComplianceScanConfigRequest) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_ComplianceRequest{
			ComplianceRequest: &central.ComplianceRequest{
				Request: &central.ComplianceRequest_SyncScanConfigs{
					SyncScanConfigs: &central.SyncComplianceScanConfigRequest{
						ScanConfigs: scanConfigs,
					},
				},
			},
		},
	}
}

func assertScanSetting(t *testing.T, expected *central.ApplyComplianceScanConfigRequest, actual *unstructured.Unstructured) {
	require.NotNil(t, actual)
	var scanSetting v1alpha1.ScanSetting
	require.NoError(t, runtime.DefaultUnstructuredConverter.FromUnstructured(actual.Object, &scanSetting))
	assert.Equal(t, expected.GetUpdateScan().GetScanSettings().GetScanName(), scanSetting.GetName())
	assert.Equal(t, expected.GetUpdateScan().GetCron(), scanSetting.ComplianceSuiteSettings.Schedule)
}

func assertScanSettingBinding(t *testing.T, expected *central.ApplyComplianceScanConfigRequest, actual *unstructured.Unstructured) {
	require.NotNil(t, actual)
	var scanSettingBinding v1alpha1.ScanSettingBinding
	require.NoError(t, runtime.DefaultUnstructuredConverter.FromUnstructured(actual.Object, &scanSettingBinding))
	assert.Equal(t, expected.GetUpdateScan().GetScanSettings().GetScanName(), scanSettingBinding.GetName())
	for _, profile := range scanSettingBinding.Profiles {
		assert.Contains(t, expected.GetUpdateScan().GetScanSettings().GetProfiles(), profile.Name)
	}
}
