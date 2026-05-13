//go:build compliance

package tests

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis"
	complianceoperatorv1 "github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/service"
	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingV1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	cached "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/kubernetes"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	coNamespaceV2       = "openshift-compliance"
	stackroxNamespace   = "stackrox"
	defaultTimeout      = 120 * time.Second
	defaultInterval     = 5 * time.Second
	waitForDoneTimeout  = 5 * time.Minute
	waitForDoneInterval = 30 * time.Second
)

var (
	initialSchedule = &v2.Schedule{
		Hour:         12,
		Minute:       0,
		IntervalType: v2.Schedule_DAILY,
		Interval: &v2.Schedule_DaysOfWeek_{
			DaysOfWeek: &v2.Schedule_DaysOfWeek{
				Days: []int32{0, 1, 2, 3, 4, 5, 6},
			},
		},
	}
	updatedSchedule = &v2.Schedule{
		Hour:         1,
		Minute:       30,
		IntervalType: v2.Schedule_DAILY,
		Interval: &v2.Schedule_DaysOfWeek_{
			DaysOfWeek: &v2.Schedule_DaysOfWeek{
				Days: []int32{2, 3, 4},
			},
		},
	}
)

// profileRef pairs a profile name with its compliance operator kind (Profile/TailoredProfile),
// used to assert both the name and the Kind field on ScanSettingBinding profile entries.
type profileRef struct {
	name         string
	operatorKind v2.ComplianceProfile_OperatorKind
}

func (p profileRef) k8sKind() string {
	if p.operatorKind == v2.ComplianceProfile_TAILORED_PROFILE {
		return "TailoredProfile"
	}
	return "Profile"
}

// profileNames extracts names for the Central API (which takes repeated string).
func profileNames(refs []profileRef) []string {
	names := make([]string, len(refs))
	for i, r := range refs {
		names[i] = r.name
	}
	return names
}

func scaleToN(ctx context.Context, t *testing.T, client kubernetes.Interface, deploymentName string, namespace string, replicas int32) {
	scaleRequest := &autoscalingV1.Scale{
		Spec: autoscalingV1.ScaleSpec{
			Replicas: replicas,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: namespace,
		},
	}

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		_, err := client.AppsV1().Deployments(namespace).UpdateScale(ctx, deploymentName, scaleRequest, metav1.UpdateOptions{})
		require.NoErrorf(c, err, "failed to scale %q to %q replicas", deploymentName, replicas)
	}, defaultTimeout, defaultInterval)
}

func createDynamicClient(t testutils.T) ctrlClient.Client {
	restCfg := getConfig(t)
	restCfg.WarningHandler = rest.NoWarnings{}
	k8sClient := createK8sClient(t)

	k8sScheme := runtime.NewScheme()

	err := cgoscheme.AddToScheme(k8sScheme)
	require.NoError(t, err, "error adding Kubernetes Scheme to client")

	err = extscheme.AddToScheme(k8sScheme)
	require.NoError(t, err, "error adding Kubernetes Scheme to client")

	cachedClientDiscovery := cached.NewMemCacheClient(k8sClient.Discovery())
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedClientDiscovery)
	restMapper.Reset()

	client, err := ctrlClient.New(
		restCfg,
		ctrlClient.Options{
			Scheme: k8sScheme,
			Mapper: restMapper,
		},
	)
	require.NoError(t, err, "failed to create dynamic client")

	// Add all the Compliance Operator schemes to the client so we can use
	// it for dealing with the Compliance Operator directly.
	err = apis.AddToScheme(client.Scheme())
	require.NoError(t, err, "failed to add Compliance Operator schemes to client")
	return client
}

func waitForComplianceSuiteToComplete(t *testing.T, client ctrlClient.Client, suiteName string, interval, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	t.Logf("Waiting for ComplianceSuite %s to reach DONE phase", suiteName)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		callCtx, callCancel := context.WithTimeout(ctx, interval)
		defer callCancel()

		// Assert that ScanSetting and ScanSettingBinding have been created.
		var scanSetting complianceoperatorv1.ScanSetting
		err := client.Get(callCtx,
			types.NamespacedName{Name: suiteName, Namespace: coNamespaceV2},
			&scanSetting,
		)
		assert.NoErrorf(c, err, "failed to get ScanSetting %s", suiteName)

		var scanSettingBinding complianceoperatorv1.ScanSettingBinding
		err = client.Get(callCtx,
			types.NamespacedName{Name: suiteName, Namespace: coNamespaceV2},
			&scanSettingBinding,
		)
		assert.NoErrorf(c, err, "failed to get ScanSettingBinding %s", suiteName)

		var suite complianceoperatorv1.ComplianceSuite
		err = client.Get(callCtx,
			types.NamespacedName{Name: suiteName, Namespace: coNamespaceV2},
			&suite,
		)
		assert.NoErrorf(c, err, "failed to get ComplianceSuite %s", suiteName)
		require.Equalf(c, complianceoperatorv1.PhaseDone, suite.Status.Phase,
			"ComplianceSuite %s not DONE (current phase is %q)", suiteName, suite.Status.Phase)
	}, timeout, interval)
	t.Logf("ComplianceSuite %s has reached DONE phase", suiteName)
}

func deleteResource[T any, PT interface {
	ctrlClient.Object
	*T
}](ctx context.Context, t *testing.T, client ctrlClient.Client, name, namespace string) {
	key := types.NamespacedName{Name: name, Namespace: namespace}

	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		var obj T
		ptr := PT(&obj)
		ptr.SetName(name)
		ptr.SetNamespace(namespace)
		err := client.Delete(ctx, ptr)
		if err != nil && !errors2.IsNotFound(err) {
			t.Logf("failed to delete %T %s/%s: %v", ptr, namespace, name, err)
		}
		err = client.Get(ctx, key, ptr)
		require.True(c, errors2.IsNotFound(err), "%T %s/%s still exists", ptr, namespace, name)
	}, defaultTimeout, defaultInterval)
}

func cleanUpResources(ctx context.Context, t *testing.T, client ctrlClient.Client, resourceName string, namespace string) {
	deleteResource[complianceoperatorv1.ScanSettingBinding](ctx, t, client, resourceName, namespace)
	deleteResource[complianceoperatorv1.ScanSetting](ctx, t, client, resourceName, namespace)
}

// createCustomRule creates a CEL CustomRule with the given name, waits for it
// to reach Ready phase, and registers cleanup.
func createCustomRule(ctx context.Context, t *testing.T, client ctrlClient.Client, name string) {
	// Create a ConfigMap with a test-specific marker key so the CEL
	// expression matches only this test's ConfigMap, not other tests'.
	markerKey := "e2e-marker-" + name
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: coNamespaceV2},
		Data:       map[string]string{markerKey: "true"},
	}
	if err := client.Create(ctx, cm); err != nil && !errors2.IsAlreadyExists(err) {
		require.NoError(t, err, "failed to create ConfigMap")
	}
	t.Cleanup(func() {
		deleteResource[corev1.ConfigMap](ctx, t, client, name, coNamespaceV2)
	})

	cr := &complianceoperatorv1.CustomRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: coNamespaceV2,
		},
		Spec: complianceoperatorv1.CustomRuleSpec{
			RulePayload: complianceoperatorv1.RulePayload{
				// CO recommendation: set metadata.name and spec.rulePayload.id
				// to the same DNS-friendly value (lowercase, hyphens, no underscores).
				ID:          name,
				Title:       "ConfigMap has e2e marker",
				Description: "Checks for a ConfigMap with an e2e-marker data key",
				Rationale:   "E2E test marker must be present",
				Severity:    "medium",
				CheckType:   "Platform",
			},
			CustomRulePayload: complianceoperatorv1.CustomRulePayload{
				ScannerType:   complianceoperatorv1.ScannerTypeCEL,
				Expression:    fmt.Sprintf(`configmaps.items.exists(cm, has(cm.data) && "%s" in cm.data)`, markerKey),
				FailureReason: fmt.Sprintf("No ConfigMap with '%s' data key found", markerKey),
				Inputs: []complianceoperatorv1.InputPayload{
					{
						Name: "configmaps",
						KubernetesInputSpec: complianceoperatorv1.KubernetesInputSpec{
							APIVersion:        "v1",
							Resource:          "configmaps",
							ResourceNamespace: coNamespaceV2,
						},
					},
				},
			},
		},
	}
	if err := client.Create(ctx, cr); err != nil && !errors2.IsAlreadyExists(err) {
		require.NoError(t, err, "failed to create CustomRule")
	}
	t.Cleanup(func() {
		deleteResource[complianceoperatorv1.CustomRule](ctx, t, client, name, coNamespaceV2)
	})

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var current complianceoperatorv1.CustomRule
		err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: coNamespaceV2}, &current)
		require.NoError(c, err)
		require.Equalf(c, complianceoperatorv1.CustomRulePhaseReady, current.Status.Phase,
			"CustomRule %s not Ready (phase: %s, error: %s)",
			name, current.Status.Phase, current.Status.ErrorMessage)
	}, 10*time.Second, 1*time.Second)
}

// createTailoredProfile creates an extends-based TailoredProfile (extending
// ocp4-e8), waits for it to be READY in k8s, and registers cleanup.
func createTailoredProfile(ctx context.Context, t *testing.T, client ctrlClient.Client, name string) {
	tp := &complianceoperatorv1.TailoredProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: coNamespaceV2,
		},
		Spec: complianceoperatorv1.TailoredProfileSpec{
			Extends:     "ocp4-e8",
			Title:       fmt.Sprintf("E2E TailoredProfile %s", name),
			Description: "Extends ocp4-e8 for e2e testing",
			DisableRules: []complianceoperatorv1.RuleReferenceSpec{
				{Name: "ocp4-api-server-encryption-provider-cipher", Rationale: "e2e test"},
			},
		},
	}

	if err := client.Create(ctx, tp); err != nil && !errors2.IsAlreadyExists(err) {
		require.NoError(t, err, "failed to create TailoredProfile")
	}

	t.Cleanup(func() {
		deleteResource[complianceoperatorv1.TailoredProfile](ctx, t, client, name, coNamespaceV2)
	})

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var current complianceoperatorv1.TailoredProfile
		err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: coNamespaceV2}, &current)
		require.NoErrorf(c, err, "failed to get TailoredProfile %s", name)
		require.Equalf(c, complianceoperatorv1.TailoredProfileStateReady, current.Status.State,
			"TailoredProfile %s not READY (state: %q, error: %q)",
			name, current.Status.State, current.Status.ErrorMessage)
	}, 10*time.Second, 1*time.Second)
}

// waitUntilTPInCentralDB waits for a tailored profile to appear in Central's
// database (via the compliance profile API) and returns it.
func waitUntilTPInCentralDB(ctx context.Context, t *testing.T,
	client v2.ComplianceProfileServiceClient, clusterID, name string,
) *v2.ComplianceProfile {
	var (
		mu      sync.Mutex
		profile *v2.ComplianceProfile
	)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		profileList, err := client.ListComplianceProfiles(ctx,
			&v2.ProfilesForClusterRequest{
				ClusterId: clusterID,
				Query:     &v2.RawQuery{Query: "Compliance Profile Name:" + name},
			})
		require.NoErrorf(c, err, "failed to list profiles")
		for _, p := range profileList.GetProfiles() {
			if p.GetName() == name {
				concurrency.WithLock(&mu, func() {
					profile = p
				})
				return
			}
		}
		require.Failf(c, "TailoredProfile not yet in Central DB", "profile %q not found", name)
	}, 10*time.Second, 1*time.Second)
	mu.Lock()
	defer mu.Unlock()
	return profile
}

func assertResourceDoesNotExist[T any, PT interface {
	ctrlClient.Object
	*T
}](ctx context.Context, t testutils.T, client ctrlClient.Client, name, namespace string) {
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var obj T
		err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, PT(&obj))
		require.True(c, errors2.IsNotFound(err), "%T %s/%s still exists", obj, namespace, name)
	}, defaultTimeout, defaultInterval)
}

func assertScanSetting(ctx context.Context, t testutils.T, client ctrlClient.Client, name, namespace string, scanConfig *v2.ComplianceScanConfiguration) {
	scanSetting := &complianceoperatorv1.ScanSetting{}
	err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, scanSetting)
	require.NoErrorf(t, err, "ScanSetting %s/%s does not exist", namespace, name)

	cron, err := schedule.ConvertToCronTab(service.ConvertV2ScheduleToProto(scanConfig.GetScanConfig().GetScanSchedule()))
	require.NoError(t, err)
	assert.Equal(t, scanConfig.GetScanName(), scanSetting.GetName())
	assert.Equal(t, cron, scanSetting.ComplianceSuiteSettings.Schedule)
	require.Contains(t, scanSetting.GetLabels(), "app.kubernetes.io/name")
	assert.Equal(t, scanSetting.GetLabels()["app.kubernetes.io/name"], "stackrox")
	require.Contains(t, scanSetting.GetAnnotations(), "owner")
	assert.Equal(t, scanSetting.GetAnnotations()["owner"], "stackrox")
}

func assertScanSettingBinding(ctx context.Context, t testutils.T, client ctrlClient.Client,
	name, namespace string, expectedProfiles []profileRef) {
	ssb := &complianceoperatorv1.ScanSettingBinding{}
	err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, ssb)
	require.NoErrorf(t, err, "ScanSettingBinding %s/%s does not exist", namespace, name)

	assert.Equal(t, name, ssb.GetName())
	require.Len(t, ssb.Profiles, len(expectedProfiles))
	for _, expected := range expectedProfiles {
		found := false
		for _, actual := range ssb.Profiles {
			if actual.Name == expected.name {
				found = true
				assert.Equalf(t, expected.k8sKind(), actual.Kind,
					"SSB profile %q: expected Kind %q, got %q",
					expected.name, expected.k8sKind(), actual.Kind)
				break
			}
		}
		assert.Truef(t, found, "profile %q not found in SSB", expected.name)
	}
	require.Contains(t, ssb.Labels, "app.kubernetes.io/name")
	assert.Equal(t, "stackrox", ssb.Labels["app.kubernetes.io/name"])
	require.Contains(t, ssb.Annotations, "owner")
	assert.Equal(t, "stackrox", ssb.Annotations["owner"])
}

func waitForDeploymentReady(ctx context.Context, t *testing.T, client ctrlClient.Client, name string, namespace string, numReplicas int32) {
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		deployment := &appsv1.Deployment{}
		err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, deployment)
		require.NoErrorf(c, err, "failed to get deployment %q", name)
		require.Equal(c, numReplicas, deployment.Status.ReadyReplicas)
	}, defaultTimeout, defaultInterval)
}

// Run this test outside of other parallel tests because of the Sensor side effects.
func TestComplianceV2CentralSendsScanConfiguration(t *testing.T) {
	ctx := context.Background()
	k8sClient := createK8sClient(t)
	dynClient := createDynamicClient(t)

	conn := centralgrpc.GRPCConnectionToCentral(t)
	// Create the ScanConfiguration service
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)

	// Get cluster ID
	serviceCluster := v1.NewClustersServiceClient(conn)
	clusters, err := serviceCluster.GetClusters(ctx, &v1.GetClustersRequest{})
	assert.NoError(t, err)
	require.Greater(t, len(clusters.GetClusters()), 0)
	clusterID := clusters.GetClusters()[0].GetId()

	// Create tailored profile and wait until it appears in Central.
	testID := fmt.Sprintf("sync-%s", uuid.NewV4().String())
	tpName := testID
	createTailoredProfile(ctx, t, dynClient, tpName)
	profileClient := v2.NewComplianceProfileServiceClient(conn)
	waitUntilTPInCentralDB(ctx, t, profileClient, clusterID, tpName)

	// Use mixed profiles (Profile + TailoredProfile) to validate that the startup
	// sync path preserves profile_refs with correct kinds.
	initialProfiles := []profileRef{
		{name: "ocp4-cis", operatorKind: v2.ComplianceProfile_PROFILE},
		{name: tpName, operatorKind: v2.ComplianceProfile_TAILORED_PROFILE},
	}

	// Create local scan config with UUID-based name for test isolation.
	scanConfig := v2.ComplianceScanConfiguration{
		ScanName: testID,
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			Description:  testID,
			OneTimeScan:  false,
			Profiles:     profileNames(initialProfiles),
			ScanSchedule: initialSchedule,
		},
	}

	// Scale down Sensor
	scaleToN(ctx, t, k8sClient, "sensor", stackroxNamespace, 0)
	waitForDeploymentReady(ctx, t, dynClient, "sensor", stackroxNamespace, 0)

	// Create ScanConfig in Central
	res, err := scanConfigService.CreateComplianceScanConfiguration(ctx, &scanConfig)
	assert.NoError(t, err)

	// Cleanup just in case the test fails
	t.Cleanup(func() {
		reqDelete := &v2.ResourceByID{
			Id: res.GetId(),
		}
		_, _ = scanConfigService.DeleteComplianceScanConfiguration(ctx, reqDelete)
		cleanUpResources(ctx, t, dynClient, testID, coNamespaceV2)
	})

	// Scale up Sensor
	scaleToN(ctx, t, k8sClient, "sensor", stackroxNamespace, 1)
	waitForDeploymentReady(ctx, t, dynClient, "sensor", stackroxNamespace, 1)

	// Assert the ScanSetting and the ScanSettingBinding are created with correct profile kinds.
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assertScanSetting(ctx, wrapCollectT(t, c), dynClient, testID, coNamespaceV2, &scanConfig)
		assertScanSettingBinding(ctx, wrapCollectT(t, c), dynClient, testID, coNamespaceV2, initialProfiles)
	}, defaultTimeout, defaultInterval)

	// Scale down Sensor
	scaleToN(ctx, t, k8sClient, "sensor", stackroxNamespace, 0)
	waitForDeploymentReady(ctx, t, dynClient, "sensor", stackroxNamespace, 0)

	// Update the ScanConfig in Central with a different set of mixed profiles.
	updatedProfiles := []profileRef{
		{name: "ocp4-pci-dss", operatorKind: v2.ComplianceProfile_PROFILE},
		{name: tpName, operatorKind: v2.ComplianceProfile_TAILORED_PROFILE},
	}
	scanConfig.Id = res.GetId()
	scanConfig.ScanConfig.Profiles = profileNames(updatedProfiles)
	scanConfig.ScanConfig.ScanSchedule = updatedSchedule
	_, err = scanConfigService.UpdateComplianceScanConfiguration(ctx, &scanConfig)
	assert.NoError(t, err)

	// Scale up Sensor
	scaleToN(ctx, t, k8sClient, "sensor", stackroxNamespace, 1)
	waitForDeploymentReady(ctx, t, dynClient, "sensor", stackroxNamespace, 1)

	// Assert the ScanSetting and the ScanSettingBinding are updated with correct profile kinds.
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assertScanSetting(ctx, wrapCollectT(t, c), dynClient, testID, coNamespaceV2, &scanConfig)
		assertScanSettingBinding(ctx, wrapCollectT(t, c), dynClient, testID, coNamespaceV2, updatedProfiles)
	}, defaultTimeout, defaultInterval)

	// Scale down Sensor
	scaleToN(ctx, t, k8sClient, "sensor", stackroxNamespace, 0)
	waitForDeploymentReady(ctx, t, dynClient, "sensor", stackroxNamespace, 0)

	// Delete the ScanConfig in Central
	reqDelete := &v2.ResourceByID{
		Id: res.GetId(),
	}
	_, _ = scanConfigService.DeleteComplianceScanConfiguration(ctx, reqDelete)

	// Scale up Sensor
	scaleToN(ctx, t, k8sClient, "sensor", stackroxNamespace, 1)
	waitForDeploymentReady(ctx, t, dynClient, "sensor", stackroxNamespace, 1)

	// Assert the ScanSetting and the ScanSettingBinding are deleted
	assertResourceDoesNotExist[complianceoperatorv1.ScanSetting](ctx, t, dynClient, testID, coNamespaceV2)
	assertResourceDoesNotExist[complianceoperatorv1.ScanSettingBinding](ctx, t, dynClient, testID, coNamespaceV2)
}

// ACS API test suite for integration testing for the Compliance Operator.
func TestComplianceV2Integration(t *testing.T) {
	t.Parallel()
	resp := getIntegrations(t)
	assert.Equal(t, resp.GetIntegrations()[0].GetClusterName(), "remote", "failed to find integration for cluster called \"remote\"")
	assert.Equal(t, resp.GetIntegrations()[0].GetNamespace(), "openshift-compliance", "failed to find integration for \"openshift-compliance\" namespace")
}

func TestComplianceV2ProfileGet(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceProfileServiceClient(conn)
	clusterID := getIntegrations(t).GetIntegrations()[0].GetClusterId()

	// Create tailored profile and wait until it appears in Central.
	tpName := fmt.Sprintf("profile-get-%s", uuid.NewV4().String())
	createTailoredProfile(ctx, t, dynClient, tpName)
	tailoredProfile := waitUntilTPInCentralDB(ctx, t, client, clusterID, tpName)

	assert.Equal(t, v2.ComplianceProfile_TAILORED_PROFILE, tailoredProfile.GetOperatorKind(),
		"e2e tailored profile should have operator_kind TAILORED_PROFILE")

	// Find a regular Profile to contrast.
	profileList, err := client.ListComplianceProfiles(ctx, &v2.ProfilesForClusterRequest{ClusterId: clusterID})
	assert.NoError(t, err)
	var regularProfile *v2.ComplianceProfile
	for _, p := range profileList.GetProfiles() {
		if p.GetOperatorKind() == v2.ComplianceProfile_PROFILE {
			regularProfile = p
			break
		}
	}
	require.NotNil(t, regularProfile, "no regular profile found in profile list")
	assert.Equal(t, v2.ComplianceProfile_PROFILE, regularProfile.GetOperatorKind(),
		"regular profile should have operator_kind PROFILE")

	// Get the TailoredProfile by ID and verify rules.
	tp, err := client.GetComplianceProfile(ctx, &v2.ResourceByID{Id: tailoredProfile.GetId()})
	require.NoError(t, err)
	assert.Greater(t, len(tp.GetRules()), 0, "tailored profile should have rules")

	// Verify a regular profile also has rules via GetComplianceProfile.
	regProfile, err := client.GetComplianceProfile(ctx, &v2.ResourceByID{Id: regularProfile.GetId()})
	require.NoError(t, err)
	assert.Greater(t, len(regProfile.GetRules()), 0, "regular profile should have rules")
}

func TestComplianceV2ProfileGetSummaries(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceProfileServiceClient(conn)
	clusterID := getIntegrations(t).GetIntegrations()[0].GetClusterId()

	// Create tailored profile and wait until it appears in Central.
	tpName := fmt.Sprintf("summaries-%s", uuid.NewV4().String())
	createTailoredProfile(ctx, t, dynClient, tpName)
	waitUntilTPInCentralDB(ctx, t, client, clusterID, tpName)

	profileSummaries, err := client.ListProfileSummaries(ctx, &v2.ClustersProfileSummaryRequest{ClusterIds: []string{clusterID}})
	assert.NoError(t, err)
	assert.Greater(t, len(profileSummaries.GetProfiles()), 0, "failed to assert the cluster has profiles")

	var foundTP bool
	var foundRegular bool
	for _, p := range profileSummaries.GetProfiles() {
		if p.GetName() == tpName {
			assert.Equal(t, v2.ComplianceProfileSummary_TAILORED_PROFILE, p.GetOperatorKind(),
				"e2e tailored profile summary should have operator_kind TAILORED_PROFILE")
			foundTP = true
		} else if p.GetOperatorKind() == v2.ComplianceProfileSummary_PROFILE {
			foundRegular = true
		}
	}
	assert.True(t, foundTP, "e2e tailored profile %q not found in profile summaries", tpName)
	assert.True(t, foundRegular, "no regular profile found in profile summaries")
}

// Helper to get the integrations as the cluster id is needed in many API calls
func getIntegrations(t *testing.T) *v2.ListComplianceIntegrationsResponse {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceIntegrationServiceClient(conn)

	q := &v2.RawQuery{Query: ""}
	resp, err := client.ListComplianceIntegrations(context.TODO(), q)
	if err != nil {
		t.Fatal(err)
	}
	require.Len(t, resp.GetIntegrations(), 1, "failed to assert there is only a single compliance integration")

	return resp
}

func TestComplianceV2CreateGetScanConfigurations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	serviceCluster := v1.NewClustersServiceClient(conn)
	clusters, err := serviceCluster.GetClusters(ctx, &v1.GetClustersRequest{})
	assert.NoError(t, err)
	clusterID := clusters.GetClusters()[0].GetId()

	// Create tailored profile and wait until it appears in Central.
	testID := fmt.Sprintf("create-get-%s", uuid.NewV4().String())
	tpName := testID
	createTailoredProfile(ctx, t, dynClient, tpName)
	profileClient := v2.NewComplianceProfileServiceClient(conn)
	waitUntilTPInCentralDB(ctx, t, profileClient, clusterID, tpName)

	// Use mixed profiles: a regular Profile and a TailoredProfile.
	initialProfiles := []profileRef{
		{name: "rhcos4-e8", operatorKind: v2.ComplianceProfile_PROFILE},
		{name: tpName, operatorKind: v2.ComplianceProfile_TAILORED_PROFILE},
	}

	req := &v2.ComplianceScanConfiguration{
		ScanName: testID,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan:  false,
			Profiles:     profileNames(initialProfiles),
			Description:  "test config",
			ScanSchedule: initialSchedule,
		},
	}

	resp, err := scanConfigService.CreateComplianceScanConfiguration(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, req.GetScanName(), resp.GetScanName())
	t.Cleanup(func() {
		_ = deleteScanConfig(ctx, resp.GetId(), scanConfigService)
		cleanUpResources(ctx, t, dynClient, testID, coNamespaceV2)
	})

	query := &v2.RawQuery{Query: ""}
	scanConfigs, err := scanConfigService.ListComplianceScanConfigurations(ctx, query)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(scanConfigs.GetConfigurations()), 1)
	assert.GreaterOrEqual(t, scanConfigs.GetTotalCount(), int32(1))

	serviceResult := v2.NewComplianceResultsServiceClient(conn)
	query = &v2.RawQuery{Query: ""}
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		results, err := serviceResult.GetComplianceScanResults(ctx, query)
		require.NoError(c, err)

		resultsList := results.GetScanResults()
		var found bool
		for _, result := range resultsList {
			if result.GetScanName() == testID {
				found = true
				break
			}
		}
		require.True(c, found, "scan result not found for %s", testID)
	}, 10*time.Minute, 30*time.Second)

	// Create a different scan configuration with the same profile (duplicate rejection).
	duplicateTestName := fmt.Sprintf("create-get-dup-%s", uuid.NewV4().String())
	duplicateProfileReq := &v2.ComplianceScanConfiguration{
		ScanName: duplicateTestName,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan:  false,
			Profiles:     []string{"rhcos4-e8"},
			Description:  "test config with duplicate profile",
			ScanSchedule: initialSchedule,
		},
	}

	// Verify that the duplicate profile was not created and the error message is correct.
	_, err = scanConfigService.CreateComplianceScanConfiguration(ctx, duplicateProfileReq)
	require.Error(t, err, "expected duplicate profile scan config to be rejected")
	assert.Contains(t, err.Error(), "already uses profile")

	// Also verify that creating with the TP name is rejected (duplicate TP).
	duplicateTPReq := &v2.ComplianceScanConfiguration{
		ScanName: fmt.Sprintf("create-get-dup-tp-%s", uuid.NewV4().String()),
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan:  false,
			Profiles:     []string{tpName},
			Description:  "test config with duplicate tailored profile",
			ScanSchedule: initialSchedule,
		},
	}
	_, err = scanConfigService.CreateComplianceScanConfiguration(ctx, duplicateTPReq)
	require.Error(t, err, "expected duplicate TP scan config to be rejected")
	assert.Contains(t, err.Error(), "already uses profile")

	query = &v2.RawQuery{Query: ""}
	scanConfigs, err = scanConfigService.ListComplianceScanConfigurations(ctx, query)
	assert.NoError(t, err)
	// Verify the original config exists but duplicates were not created.
	assert.NotEmpty(t, getscanConfigID(testID, scanConfigs.GetConfigurations()), "expected original scan config %s to exist", testID)
	assert.Empty(t, getscanConfigID(duplicateTestName, scanConfigs.GetConfigurations()), "expected duplicate scan config %s to not exist", duplicateTestName)

	// Create a scan configuration with profiles with different products (rhcos, cis-node).
	// This should be valid in version >= 4.9
	differentProductProfileTestName := fmt.Sprintf("create-get-diffprod-%s", uuid.NewV4().String())
	differentProductProfileReq := &v2.ComplianceScanConfiguration{
		ScanName: differentProductProfileTestName,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan:  false,
			Profiles:     []string{"rhcos4-stig", "ocp4-cis-node"},
			Description:  "test config with invalid profiles",
			ScanSchedule: initialSchedule,
		},
	}

	res, err := scanConfigService.CreateComplianceScanConfiguration(ctx, differentProductProfileReq)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = deleteScanConfig(ctx, res.GetId(), scanConfigService)
		cleanUpResources(ctx, t, dynClient, differentProductProfileTestName, coNamespaceV2)
	})

	query = &v2.RawQuery{Query: ""}
	scanConfigs, err = scanConfigService.ListComplianceScanConfigurations(ctx, query)
	assert.NoError(t, err)
	// Verify both scan configs exist
	assert.NotEmpty(t, getscanConfigID(testID, scanConfigs.GetConfigurations()), "expected original scan config %s to exist", testID)
	assert.NotEmpty(t, getscanConfigID(differentProductProfileTestName, scanConfigs.GetConfigurations()), "expected different product scan config %s to exist", differentProductProfileTestName)
}

func TestComplianceV2UpdateScanConfigurations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	serviceCluster := v1.NewClustersServiceClient(conn)
	clusters, err := serviceCluster.GetClusters(ctx, &v1.GetClustersRequest{})
	assert.NoError(t, err)
	require.Greater(t, len(clusters.GetClusters()), 0)
	clusterID := clusters.GetClusters()[0].GetId()

	testID := fmt.Sprintf("update-%s", uuid.NewV4().String())

	// Create a scan configuration with a single regular profile.
	initialProfiles := []profileRef{
		{name: "ocp4-moderate", operatorKind: v2.ComplianceProfile_PROFILE},
	}
	req := &v2.ComplianceScanConfiguration{
		ScanName: testID,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan:  false,
			Profiles:     profileNames(initialProfiles),
			Description:  "test config",
			ScanSchedule: initialSchedule,
		},
	}

	resp, err := scanConfigService.CreateComplianceScanConfiguration(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, req.GetScanName(), resp.GetScanName())
	t.Cleanup(func() {
		_ = deleteScanConfig(ctx, resp.GetId(), scanConfigService)
		cleanUpResources(ctx, t, dynClient, req.GetScanName(), coNamespaceV2)
	})

	query := &v2.RawQuery{Query: ""}
	scanConfigs, err := scanConfigService.ListComplianceScanConfigurations(ctx, query)
	assert.NoError(t, err)
	require.GreaterOrEqual(t, len(scanConfigs.GetConfigurations()), 1)
	assert.GreaterOrEqual(t, scanConfigs.GetTotalCount(), int32(1))

	// Assert the ScanSetting and the ScanSettingBinding are created with Kind: Profile.
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assertScanSetting(ctx, wrapCollectT(t, c), dynClient, testID, coNamespaceV2, req)
		assertScanSettingBinding(ctx, wrapCollectT(t, c), dynClient, testID, coNamespaceV2, initialProfiles)
	}, defaultTimeout, defaultInterval)

	// Create tailored profile and wait until it appears in Central.
	tpName := testID
	createTailoredProfile(ctx, t, dynClient, tpName)
	profileClient := v2.NewComplianceProfileServiceClient(conn)
	waitUntilTPInCentralDB(ctx, t, profileClient, clusterID, tpName)

	// Update to mixed profiles: a regular profile + the TailoredProfile.
	updatedProfiles := []profileRef{
		{name: "ocp4-moderate-node", operatorKind: v2.ComplianceProfile_PROFILE},
		{name: tpName, operatorKind: v2.ComplianceProfile_TAILORED_PROFILE},
	}
	updateReq := req.CloneVT()
	updateReq.Id = resp.GetId()
	updateReq.ScanConfig.ScanSchedule = &v2.Schedule{
		IntervalType: 1,
		Hour:         12,
		Minute:       30,
		Interval: &v2.Schedule_DaysOfWeek_{
			DaysOfWeek: &v2.Schedule_DaysOfWeek{
				Days: []int32{2, 4, 6},
			},
		},
	}
	updateReq.ScanConfig.Profiles = profileNames(updatedProfiles)
	_, err = scanConfigService.UpdateComplianceScanConfiguration(ctx, updateReq)
	assert.NoError(t, err)

	query = &v2.RawQuery{Query: ""}
	scanConfigs, err = scanConfigService.ListComplianceScanConfigurations(ctx, query)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(scanConfigs.GetConfigurations()), 1)
	assert.GreaterOrEqual(t, scanConfigs.GetTotalCount(), int32(1))

	// Assert the ScanSetting and the ScanSettingBinding are updated with correct profile kinds.
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assertScanSetting(ctx, wrapCollectT(t, c), dynClient, testID, coNamespaceV2, updateReq)
		assertScanSettingBinding(ctx, wrapCollectT(t, c), dynClient, testID, coNamespaceV2, updatedProfiles)
	}, defaultTimeout, defaultInterval)
}

func TestComplianceV2DeleteComplianceScanConfigurations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	// Retrieve the results from the scan configuration once the scan is complete
	serviceCluster := v1.NewClustersServiceClient(conn)
	clusters, err := serviceCluster.GetClusters(ctx, &v1.GetClustersRequest{})
	assert.NoError(t, err)

	clusterID := clusters.GetClusters()[0].GetId()

	// Create tailored profile and wait until it appears in Central.
	testID := fmt.Sprintf("delete-%s", uuid.NewV4().String())
	tpName := testID
	createTailoredProfile(ctx, t, dynClient, tpName)
	profileClient := v2.NewComplianceProfileServiceClient(conn)
	waitUntilTPInCentralDB(ctx, t, profileClient, clusterID, tpName)

	req := &v2.ComplianceScanConfiguration{
		ScanName: testID,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan:  false,
			Profiles:     []string{"rhcos4-high", tpName},
			Description:  "test config",
			ScanSchedule: initialSchedule,
		},
	}

	resp, err := scanConfigService.CreateComplianceScanConfiguration(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, req.GetScanName(), resp.GetScanName())
	t.Cleanup(func() {
		_ = deleteScanConfig(ctx, resp.GetId(), scanConfigService)
		cleanUpResources(ctx, t, dynClient, testID, coNamespaceV2)
	})

	query := &v2.RawQuery{Query: ""}
	scanConfigs, err := scanConfigService.ListComplianceScanConfigurations(ctx, query)
	require.NoError(t, err)
	configs := scanConfigs.GetConfigurations()
	scanconfigID := getscanConfigID(testID, configs)
	reqDelete := &v2.ResourceByID{
		Id: scanconfigID,
	}
	_, err = scanConfigService.DeleteComplianceScanConfiguration(ctx, reqDelete)
	assert.NoError(t, err)

	// Verify scan configuration no longer exists
	scanConfigs, err = scanConfigService.ListComplianceScanConfigurations(ctx, query)
	require.NoError(t, err)
	configs = scanConfigs.GetConfigurations()
	scanconfigID = getscanConfigID(testID, configs)
	assert.Empty(t, scanconfigID)
}

func TestComplianceV2ComplianceObjectMetadata(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	serviceCluster := v1.NewClustersServiceClient(conn)
	clusters, err := serviceCluster.GetClusters(ctx, &v1.GetClustersRequest{})
	assert.NoError(t, err)
	clusterID := clusters.GetClusters()[0].GetId()
	testName := fmt.Sprintf("metadata-%s", uuid.NewV4().String())
	req := &v2.ComplianceScanConfiguration{
		ScanName: testName,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan:  false,
			Profiles:     []string{"rhcos4-nerc-cip"},
			Description:  "test config",
			ScanSchedule: initialSchedule,
		},
	}

	resp, err := scanConfigService.CreateComplianceScanConfiguration(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, req.GetScanName(), resp.GetScanName())
	t.Cleanup(func() {
		_ = deleteScanConfig(ctx, resp.GetId(), scanConfigService)
		cleanUpResources(ctx, t, dynClient, testName, coNamespaceV2)
	})

	query := &v2.RawQuery{Query: ""}
	scanConfigs, err := scanConfigService.ListComplianceScanConfigurations(ctx, query)
	require.NoError(t, err)
	configs := scanConfigs.GetConfigurations()
	_ = getscanConfigID(testName, configs) // verify config exists

	// Ensure the ScanSetting and ScanSettingBinding have ACS metadata
	var scanSetting complianceoperatorv1.ScanSetting
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		callCtx, callCancel := context.WithTimeout(ctx, 10*time.Second)
		defer callCancel()

		err := dynClient.Get(callCtx,
			types.NamespacedName{Name: testName, Namespace: "openshift-compliance"},
			&scanSetting,
		)
		require.NoErrorf(c, err, "failed to get ScanSetting %s", testName)
	}, defaultTimeout, defaultInterval)

	assert.Contains(t, scanSetting.Labels, "app.kubernetes.io/name")
	assert.Equal(t, scanSetting.Labels["app.kubernetes.io/name"], "stackrox")
	assert.Contains(t, scanSetting.Annotations, "owner")
	assert.Equal(t, scanSetting.Annotations["owner"], "stackrox")

	var scanSettingBinding complianceoperatorv1.ScanSetting
	err = dynClient.Get(context.TODO(), types.NamespacedName{Name: testName, Namespace: "openshift-compliance"}, &scanSettingBinding)
	require.NoErrorf(t, err, "failed to get ScanSettingBinding %s", testName)
	assert.Contains(t, scanSettingBinding.Labels, "app.kubernetes.io/name")
	assert.Equal(t, scanSettingBinding.Labels["app.kubernetes.io/name"], "stackrox")
	assert.Contains(t, scanSettingBinding.Annotations, "owner")
	assert.Equal(t, scanSettingBinding.Annotations["owner"], "stackrox")
}

func deleteScanConfig(ctx context.Context, scanID string, service v2.ComplianceScanConfigurationServiceClient) error {
	req := &v2.ResourceByID{
		Id: scanID,
	}
	_, err := service.DeleteComplianceScanConfiguration(ctx, req)
	return err
}

func getscanConfigID(configName string, scanConfigs []*v2.ComplianceScanConfigurationStatus) string {
	configID := ""
	for i := 0; i < len(scanConfigs); i++ {
		if scanConfigs[i].GetScanName() == configName {
			configID = scanConfigs[i].GetId()
		}
	}
	return configID
}

func TestComplianceV2ScheduleRescan(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceScanConfigurationServiceClient(conn)
	clusterId := getIntegrations(t).GetIntegrations()[0].GetClusterId()

	// Create tailored profile and wait until it appears in Central.
	testID := fmt.Sprintf("rescan-%s", uuid.NewV4().String())
	tpName := testID
	createTailoredProfile(ctx, t, dynClient, tpName)
	profileClient := v2.NewComplianceProfileServiceClient(conn)
	waitUntilTPInCentralDB(ctx, t, profileClient, clusterId, tpName)

	sc := v2.ComplianceScanConfiguration{
		ScanName: testID,
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"ocp4-e8", tpName},
			ScanSchedule: &v2.Schedule{
				IntervalType: 3,
				Hour:         0,
				Minute:       0,
			},
			Description: "Scan schedule for the Australian Essential Eight profile to run daily.",
		},
		Clusters: []string{clusterId},
	}
	scanConfig, err := client.CreateComplianceScanConfiguration(context.TODO(), &sc)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = client.DeleteComplianceScanConfiguration(context.TODO(), &v2.ResourceByID{Id: scanConfig.GetId()})
		cleanUpResources(context.Background(), t, dynClient, testID, coNamespaceV2)
	})

	waitForComplianceSuiteToComplete(t, dynClient, scanConfig.GetScanName(), waitForDoneInterval, waitForDoneTimeout)

	// Invoke a rescan
	_, err = client.RunComplianceScanConfiguration(context.TODO(), &v2.ResourceByID{Id: scanConfig.GetId()})
	require.NoErrorf(t, err, "failed to rerun scan schedule %s", testID)

	// Assert the scan is rerunning on the cluster using the Compliance Operator CRDs
	waitForComplianceSuiteToComplete(t, dynClient, scanConfig.GetScanName(), waitForDoneInterval, waitForDoneTimeout)
}

// TestComplianceV2TailoredProfileVariants verifies that ACS correctly tracks
// different TailoredProfile variants: extends-base (with enabled and disabled
// rules), from-scratch with custom rules, and from-scratch with regular rules.
func TestComplianceV2TailoredProfileVariants(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	profileClient := v2.NewComplianceProfileServiceClient(conn)
	clusterID := getIntegrations(t).GetIntegrations()[0].GetClusterId()
	testID := fmt.Sprintf("variants-%s", uuid.NewV4().String())

	// Create custom rule needed by the "custom-rules" variant.
	crName := testID
	createCustomRule(ctx, t, dynClient, crName)

	variants := map[string]complianceoperatorv1.TailoredProfileSpec{
		"extends": {
			Extends:     "ocp4-cis",
			Title:       "E2E Extends Base",
			Description: "TP extending ocp4-cis for e2e testing",
			EnableRules: []complianceoperatorv1.RuleReferenceSpec{
				{Name: "ocp4-api-server-admission-control-plugin-alwaysadmit", Rationale: "e2e test"},
			},
			DisableRules: []complianceoperatorv1.RuleReferenceSpec{
				{Name: "ocp4-api-server-encryption-provider-cipher", Rationale: "e2e test"},
			},
		},
		"custom-rules": {
			Title:       "E2E Custom Rules",
			Description: "From-scratch TP with a custom rule",
			EnableRules: []complianceoperatorv1.RuleReferenceSpec{
				{Name: crName, Kind: complianceoperatorv1.CustomRuleKind, Rationale: "e2e test"},
			},
		},
		"from-scratch": {
			Title:       "E2E From-Scratch",
			Description: "From-scratch TP with regular rules",
			EnableRules: []complianceoperatorv1.RuleReferenceSpec{
				{Name: "ocp4-api-server-audit-log-maxbackup", Kind: complianceoperatorv1.RuleKind, Rationale: "e2e test"},
			},
		},
	}

	for name, spec := range variants {
		t.Run(name, func(t *testing.T) {
			tpName := testID + "-" + name

			tp := &complianceoperatorv1.TailoredProfile{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tpName,
					Namespace: coNamespaceV2,
				},
				Spec: spec,
			}
			require.NoErrorf(t, dynClient.Create(ctx, tp), "failed to create TP %s", tpName)
			t.Cleanup(func() {
				deleteResource[complianceoperatorv1.TailoredProfile](ctx, t, dynClient, tpName, coNamespaceV2)
			})

			require.EventuallyWithT(t, func(c *assert.CollectT) {
				var current complianceoperatorv1.TailoredProfile
				err := dynClient.Get(ctx, types.NamespacedName{Name: tpName, Namespace: coNamespaceV2}, &current)
				require.NoErrorf(c, err, "failed to get TailoredProfile %s", tpName)
				require.Equalf(c, complianceoperatorv1.TailoredProfileStateReady, current.Status.State,
					"TailoredProfile %s not READY (state: %q, error: %q)",
					tpName, current.Status.State, current.Status.ErrorMessage)
			}, 10*time.Second, 1*time.Second)

			storedTP := waitUntilTPInCentralDB(ctx, t, profileClient, clusterID, tpName)
			assert.Equal(t, v2.ComplianceProfile_TAILORED_PROFILE, storedTP.GetOperatorKind(),
				"TP should have operator_kind TAILORED_PROFILE")

			detail, err := profileClient.GetComplianceProfile(ctx, &v2.ResourceByID{Id: storedTP.GetId()})
			require.NoError(t, err)
			assert.Greater(t, len(detail.GetRules()), 0, "TP should have at least one rule")

			ruleNames := make([]string, 0, len(detail.GetRules()))
			for _, r := range detail.GetRules() {
				ruleNames = append(ruleNames, r.GetName())
			}

			for _, r := range spec.EnableRules {
				assert.Contains(t, ruleNames, r.Name, "expected rule not found")
			}
			for _, r := range spec.DisableRules {
				assert.NotContains(t, ruleNames, r.Name, "found unexpected rule")
			}
		})
	}
}

// assessmentTimeFormat is the expected format for timestamps in compliance report CSVs.
// The format is MM/DD/YYYY HH:MM:SS in 24-hour time.
var assessmentTimeFormat = regexp.MustCompile(`^\d{2}/\d{2}/\d{4} \d{2}:\d{2}:\d{2}$`)

// TestComplianceV2ReportDownloadTimestampFormat verifies that the Assessment Time
// column in downloaded compliance report CSVs uses MM/DD/YYYY HH:MM:SS format.
func TestComplianceV2ReportDownloadTimestampFormat(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	serviceCluster := v1.NewClustersServiceClient(conn)

	clusters, err := serviceCluster.GetClusters(ctx, &v1.GetClustersRequest{})
	require.NoError(t, err)
	require.Greater(t, len(clusters.GetClusters()), 0, "no clusters found")
	clusterID := clusters.GetClusters()[0].GetId()

	testID := fmt.Sprintf("ts-fmt-%s", uuid.NewV4().String())

	req := &v2.ComplianceScanConfiguration{
		ScanName: testID,
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: true,
			Profiles:    []string{"ocp4-e8"},
			Description: "timestamp format e2e test",
			ScanSchedule: &v2.Schedule{
				IntervalType: v2.Schedule_DAILY,
				Hour:         12,
				Minute:       0,
				Interval: &v2.Schedule_DaysOfWeek_{
					DaysOfWeek: &v2.Schedule_DaysOfWeek{Days: []int32{0, 1, 2, 3, 4, 5, 6}},
				},
			},
		},
	}

	scanConfig, err := scanConfigService.CreateComplianceScanConfiguration(ctx, req)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = deleteScanConfig(ctx, scanConfig.GetId(), scanConfigService)
		cleanUpResources(ctx, t, dynClient, testID, coNamespaceV2)
	})

	// Wait for the scan to complete on the cluster.
	waitForComplianceSuiteToComplete(t, dynClient, testID, waitForDoneInterval, waitForDoneTimeout)

	runAndDownloadReport(t, ctx, scanConfigService, scanConfig.GetId())
}

// TestComplianceV2ReportDownloadTimestampFormatTailoredProfile verifies timestamps
// in report CSVs when using a tailored profile with custom rules.
func TestComplianceV2ReportDownloadTimestampFormatTailoredProfile(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	serviceCluster := v1.NewClustersServiceClient(conn)
	profileClient := v2.NewComplianceProfileServiceClient(conn)

	clusters, err := serviceCluster.GetClusters(ctx, &v1.GetClustersRequest{})
	require.NoError(t, err)
	require.Greater(t, len(clusters.GetClusters()), 0, "no clusters found")
	clusterID := clusters.GetClusters()[0].GetId()

	testID := fmt.Sprintf("ts-tp-%s", uuid.NewV4().String())

	// Create a custom rule for the tailored profile.
	crName := testID
	createCustomRule(ctx, t, dynClient, crName)

	// Create a tailored profile that adds the custom rule.
	tp := &complianceoperatorv1.TailoredProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testID,
			Namespace: coNamespaceV2,
		},
		Spec: complianceoperatorv1.TailoredProfileSpec{
			Title:       fmt.Sprintf("E2E TS TP %s", testID),
			Description: "Tailored profile for timestamp format e2e test",
			EnableRules: []complianceoperatorv1.RuleReferenceSpec{
				{Name: crName, Kind: complianceoperatorv1.CustomRuleKind, Rationale: "e2e timestamp test"},
			},
		},
	}
	require.NoError(t, dynClient.Create(ctx, tp), "failed to create TailoredProfile")
	t.Cleanup(func() {
		deleteResource[complianceoperatorv1.TailoredProfile](ctx, t, dynClient, testID, coNamespaceV2)
	})

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var current complianceoperatorv1.TailoredProfile
		tpErr := dynClient.Get(ctx, types.NamespacedName{Name: testID, Namespace: coNamespaceV2}, &current)
		require.NoErrorf(c, tpErr, "failed to get TailoredProfile %s", testID)
		require.Equalf(c, complianceoperatorv1.TailoredProfileStateReady, current.Status.State,
			"TailoredProfile %s not READY (state: %q, error: %q)",
			testID, current.Status.State, current.Status.ErrorMessage)
	}, 30*time.Second, 2*time.Second)

	// Wait for the tailored profile to appear in Central.
	waitUntilTPInCentralDB(ctx, t, profileClient, clusterID, testID)

	req := &v2.ComplianceScanConfiguration{
		ScanName: testID,
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: true,
			Profiles:    []string{testID},
			Description: "tailored profile timestamp format e2e test",
			ScanSchedule: &v2.Schedule{
				IntervalType: v2.Schedule_DAILY,
				Hour:         12,
				Minute:       0,
				Interval: &v2.Schedule_DaysOfWeek_{
					DaysOfWeek: &v2.Schedule_DaysOfWeek{Days: []int32{0, 1, 2, 3, 4, 5, 6}},
				},
			},
		},
	}

	scanConfig, err := scanConfigService.CreateComplianceScanConfiguration(ctx, req)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = deleteScanConfig(ctx, scanConfig.GetId(), scanConfigService)
		cleanUpResources(ctx, t, dynClient, testID, coNamespaceV2)
	})

	waitForComplianceSuiteToComplete(t, dynClient, testID, waitForDoneInterval, waitForDoneTimeout)

	runAndDownloadReport(t, ctx, scanConfigService, scanConfig.GetId())
}

// runAndDownloadReport triggers an on-demand download report for the given scan config,
// polls until the report is generated, downloads the zip, and verifies all Assessment
// Time values in the CSV files match MM/DD/YYYY HH:MM:SS format.
func runAndDownloadReport(t *testing.T, ctx context.Context, scanConfigService v2.ComplianceScanConfigurationServiceClient, scanConfigID string) {
	t.Helper()

	// Trigger an on-demand download report.
	runResp, err := scanConfigService.RunReport(ctx, &v2.ComplianceRunReportRequest{
		ScanConfigId:             scanConfigID,
		ReportNotificationMethod: v2.NotificationMethod_DOWNLOAD,
	})
	require.NoError(t, err)
	submittedAt := runResp.GetSubmittedAt()
	require.NotNil(t, submittedAt, "expected a submission timestamp in RunReport response")

	// Poll until the report is ready (GENERATED state), finding it by submission time.
	var reportID string
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		history, histErr := scanConfigService.GetMyReportHistory(ctx, &v2.ComplianceReportHistoryRequest{
			Id: scanConfigID,
		})
		require.NoError(c, histErr)
		for _, snap := range history.GetComplianceReportSnapshots() {
			snapTime := snap.GetReportStatus().GetStartedAt()
			if snapTime == nil || snapTime.AsTime().Before(submittedAt.AsTime().Add(-5*time.Second)) {
				continue
			}
			state := snap.GetReportStatus().GetRunState()
			require.NotEqualf(c, v2.ComplianceReportStatus_FAILURE, state,
				"report failed: %s", snap.GetReportStatus().GetErrorMsg())
			require.Equalf(c, v2.ComplianceReportStatus_GENERATED, state,
				"report not yet ready (state: %s)", state)
			reportID = snap.GetReportJobId()
			return
		}
		require.Fail(c, "report snapshot not yet found in history")
	}, waitForDoneTimeout, waitForDoneInterval)

	// Download the report zip via the HTTP endpoint.
	httpClient := centralgrpc.HTTPClientForCentral(t)
	downloadURL := fmt.Sprintf("/v2/compliance/scan/configurations/reports/download?id=%s", reportID)
	resp, err := httpClient.Get(downloadURL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 for report download")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	verifyReportTimestamps(t, body)
}

// verifyReportTimestamps unzips a compliance report zip, reads every CSV file,
// and asserts that non-empty Assessment Time values match MM/DD/YYYY HH:MM:SS.
func verifyReportTimestamps(t *testing.T, zipData []byte) {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err, "failed to open report zip")

	var checkedRows int
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		rc, openErr := f.Open()
		require.NoErrorf(t, openErr, "failed to open %s in zip", f.Name)

		rows, readErr := csv.NewReader(rc).ReadAll()
		_ = rc.Close()
		require.NoErrorf(t, readErr, "failed to read CSV %s", f.Name)

		if len(rows) < 2 {
			continue
		}

		// Find the "Assessment Time" column index.
		header := rows[0]
		assessmentColIdx := -1
		for i, col := range header {
			if col == "Assessment Time" {
				assessmentColIdx = i
				break
			}
		}
		if assessmentColIdx < 0 {
			continue
		}

		for rowIdx, row := range rows[1:] {
			if assessmentColIdx >= len(row) {
				continue
			}
			val := row[assessmentColIdx]
			switch val {
			case "", "N/A", "Data not found for the cluster":
				continue
			case "ERR":
				assert.Failf(t, "invalid timestamp in report",
					"Assessment Time in %s row %d: got %q (invalid protobuf timestamp)",
					f.Name, rowIdx+2, val)
				continue
			}
			assert.Truef(t, assessmentTimeFormat.MatchString(val),
				"Assessment Time in %s row %d: got %q, want MM/DD/YYYY HH:MM:SS format",
				f.Name, rowIdx+2, val)
			checkedRows++
		}
	}
	assert.Greater(t, checkedRows, 0, "no Assessment Time values were checked — report may be empty")
}
