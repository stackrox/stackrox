package complianceoperator

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

var (
	machineConfigPoolGVR = schema.GroupVersionResource{
		Group:    "machineconfiguration.openshift.io",
		Version:  "v1",
		Resource: "machineconfigpools",
	}

	crdGVR = schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}
)

// Test_NodeRoleConfiguration verifies that MachineConfigPool resources drive
// the roles written into compliance ScanSetting objects. It covers:
//  1. Scan settings created via SyncScanConfigs use MCP names as roles.
//  2. Adding a new MCP triggers debounced reconciliation that updates
//     existing scan settings.
func Test_NodeRoleConfiguration(t *testing.T) {
	t.Setenv(features.ComplianceCustomNodeRoles.EnvVar(), "true")
	t.Setenv(env.ConnectionRetryInitialInterval.EnvVar(), "1s")
	t.Setenv(env.ConnectionRetryMaxInterval.EnvVar(), "2s")

	ctx := context.Background()

	// The MCP CRD must exist before sensor starts so the listener
	// initializes the MachineConfigPool informer during handleAllEvents.
	restConfig, err := clientcmd.BuildConfigFromFlags("", conf.ResolveKubeConfigFile())
	require.NoError(t, err)

	preStartResources, err := resources.New(restConfig)
	require.NoError(t, err)

	require.NoError(t, decoder.ApplyWithManifestDir(ctx, preStartResources,
		"../../../tests/complianceoperator/mcpcrds", "*", []resources.CreateOption{}))
	t.Cleanup(func() {
		utils.IgnoreError(func() error {
			return decoder.DeleteWithManifestDir(ctx, preStartResources,
				"../../../tests/complianceoperator/mcpcrds", "*", []resources.DeleteOption{})
		})
	})

	waitForMCPCRDEstablished(t, restConfig)

	centralCaps := []string{
		centralsensor.SendDeduperStateOnReconnect,
		centralsensor.ComplianceV2Integrations,
	}
	tc, err := helper.NewContextWithConfig(t, helper.Config{
		CentralCaps:           centralCaps,
		InitialSystemPolicies: nil,
		CertFilePath:          "../../../tools/local-sensor/certs/",
	})
	require.NoError(t, err)
	t.Cleanup(tc.Stop)

	tc.RunTest(t, helper.WithTestCase(func(t *testing.T, tc *helper.TestContext, _ map[string]k8s.Object) {
		ctx := context.Background()

		t.Log("Wait for initial sync")
		tc.WaitForSyncEvent(t, 10*time.Second)
		tc.GetFakeCentral().ClearReceivedBuffer()

		t.Log("Creating MachineConfigPool resources")
		dynClient := tc.GetKubernetesClient().Dynamic()
		createMCP(t, ctx, dynClient, "infra")
		createMCP(t, ctx, dynClient, "storage")

		t.Log("Creating Compliance Operator CRDs")
		deleteCRDsFn, err := tc.ApplyWithManifestDir(ctx, "../../../tests/complianceoperator/crds", "*")
		require.NoError(t, err)
		t.Cleanup(func() {
			utils.IgnoreError(deleteCRDsFn)
			tc.WaitForResourceDelete(ctx, t, coNamespace, "", namespaceAPIResource)
		})

		t.Log("Creating fake Compliance Operator deployment")
		coDep := &appsv1.Deployment{}
		_, err = tc.ApplyResourceAndWait(ctx, t, coNamespace, &coDeployment, coDep, nil)
		require.NoError(t, err)

		t.Log("Waiting for sensor to sync after CRDs detected")
		tc.WaitForSyncEventf(t, 10*time.Second, "expected restart after CRDs detected")

		t.Log("Sending SyncScanConfigs from Central")
		tc.GetFakeCentral().StubMessage(createSyncScanConfigsMessage(testScanConfig))

		t.Log("Asserting ScanSetting was created with MCP node roles")
		assertScanSettingHasRoles(t, tc,
			testScanConfig.GetUpdateScan().GetScanSettings().GetScanName(),
			[]string{"infra", "storage"})

		t.Log("Adding a new MachineConfigPool to trigger role reconciliation")
		createMCP(t, ctx, dynClient, "custom-pool")

		t.Log("Asserting ScanSetting roles were reconciled after MCP change")
		assertScanSettingHasRoles(t, tc,
			testScanConfig.GetUpdateScan().GetScanSettings().GetScanName(),
			[]string{"custom-pool", "infra", "storage"})
	}))
}

func createMCP(t *testing.T, ctx context.Context, dynClient dynamic.Interface, name string) {
	t.Helper()
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "machineconfiguration.openshift.io/v1",
			"kind":       "MachineConfigPool",
			"metadata": map[string]interface{}{
				"name": name,
			},
		},
	}
	_, err := dynClient.Resource(machineConfigPoolGVR).Create(ctx, obj, metav1.CreateOptions{})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = dynClient.Resource(machineConfigPoolGVR).Delete(ctx, name, metav1.DeleteOptions{})
	})
}

func waitForMCPCRDEstablished(t *testing.T, restConfig *rest.Config) {
	t.Helper()
	dynClient, err := dynamic.NewForConfig(restConfig)
	require.NoError(t, err)

	require.Eventuallyf(t, func() bool {
		obj, err := dynClient.Resource(crdGVR).Get(context.Background(),
			"machineconfigpools.machineconfiguration.openshift.io", metav1.GetOptions{})
		if err != nil {
			return false
		}
		conditions, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
		if err != nil || !found {
			return false
		}
		for _, c := range conditions {
			cond, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			if cond["type"] == "Established" && cond["status"] == "True" {
				return true
			}
		}
		return false
	}, 30*time.Second, 500*time.Millisecond, "MachineConfigPool CRD should be established")
}

func assertScanSettingHasRoles(t *testing.T, tc *helper.TestContext, scanName string, expectedRoles []string) {
	t.Helper()
	dynClient, err := dynamic.NewForConfig(tc.Resources().GetConfig())
	require.NoError(t, err)

	ssGVR := complianceoperator.ScanSetting.GroupVersionResource()

	expected := make([]string, len(expectedRoles))
	copy(expected, expectedRoles)
	slices.Sort(expected)

	require.Eventuallyf(t, func() bool {
		obj, err := dynClient.Resource(ssGVR).Namespace(coNamespace).Get(
			context.Background(), scanName, metav1.GetOptions{})
		if err != nil {
			return false
		}
		var ss v1alpha1.ScanSetting
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &ss); err != nil {
			return false
		}
		actual := make([]string, len(ss.Roles))
		copy(actual, ss.Roles)
		slices.Sort(actual)
		return slices.Equal(actual, expected)
	}, 30*time.Second, 500*time.Millisecond, "expected scan setting %q to have roles %v", scanName, expectedRoles)
}
