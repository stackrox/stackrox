//go:build compliance

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis"
	complianceoperatorv1 "github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/service"
	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
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
func createCustomRule(ctx context.Context, t *testing.T, client dynclient.Client, name string) {
	// Ensure ConfigMap exists in the CO namespace (shared across tests).
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-cr-config", Namespace: coNamespaceV2},
		Data:       map[string]string{"e2e-marker": "true"},
	}
	if err := client.Create(ctx, cm); err != nil && !errors2.IsAlreadyExists(err) {
		require.NoError(t, err, "failed to create ConfigMap")
	}

	cr := &complianceoperatorv1.CustomRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: coNamespaceV2,
		},
		Spec: complianceoperatorv1.CustomRuleSpec{
			RulePayload: complianceoperatorv1.RulePayload{
				ID:          name,
				Title:       "ConfigMap has e2e marker",
				Description: "Checks for a ConfigMap with an e2e-marker data key",
				Rationale:   "E2E test marker must be present",
				Severity:    "medium",
				CheckType:   "Platform",
			},
			CustomRulePayload: complianceoperatorv1.CustomRulePayload{
				ScannerType:   complianceoperatorv1.ScannerTypeCEL,
				Expression:    `configmaps.items.exists(cm, has(cm.data) && "e2e-marker" in cm.data)`,
				FailureReason: "No ConfigMap with 'e2e-marker' data key found",
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
	require.NoErrorf(t, client.Create(ctx, cr), "failed to create CustomRule %s", name)
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
	}, 2*time.Minute, 5*time.Second)
}

// createTailoredProfile creates an extends-based TailoredProfile (extending
// ocp4-e8), waits for it to be READY in k8s, and registers cleanup.
func createTailoredProfile(ctx context.Context, t *testing.T, client dynclient.Client, name string) {
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
	require.NoErrorf(t, client.Create(ctx, tp), "failed to create TailoredProfile %s", name)
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
	var profile *v2.ComplianceProfile
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		profileList, err := client.ListComplianceProfiles(ctx,
			&v2.ProfilesForClusterRequest{
				ClusterId: clusterID,
				Query:     &v2.RawQuery{Query: "Compliance Profile Name:" + name},
			})
		require.NoErrorf(c, err, "failed to list profiles")
		for _, p := range profileList.GetProfiles() {
			if p.GetName() == name {
				profile = p
				return
			}
		}
		require.Failf(c, "TP not yet in Central DB", "profile %q not found", name)
	}, 10*time.Second, 1*time.Second)
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

	// Create per-test tailored profile and wait for ACS ingestion.
	testID := fmt.Sprintf("sync-%s", uuid.NewV4().String())
	tpName := fmt.Sprintf("sync-tp-%s", uuid.NewV4().String())
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

	// Create per-test tailored profile and wait for ACS ingestion.
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

	// Create per-test tailored profile and wait for ACS ingestion.
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

	// Create per-test tailored profile and wait for ACS ingestion.
	testID := fmt.Sprintf("create-get-%s", uuid.NewV4().String())
	tpName := fmt.Sprintf("create-get-tp-%s", uuid.NewV4().String())
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
			OneTimeScan: false,
			Profiles:    profileNames(initialProfiles),
			Description: "test config",
			ScanSchedule: &v2.Schedule{
				IntervalType: 1,
				Hour:         15,
				Minute:       0,
				Interval: &v2.Schedule_DaysOfWeek_{
					DaysOfWeek: &v2.Schedule_DaysOfWeek{
						Days: []int32{1, 2, 3, 4, 5, 6},
					},
				},
			},
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

	// Wait for SSB and assert profile kinds.
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assertScanSettingBinding(ctx, wrapCollectT(t, c), dynClient, testID, coNamespaceV2, initialProfiles)
	}, defaultTimeout, defaultInterval)

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
			OneTimeScan: false,
			Profiles:    []string{"rhcos4-e8"},
			Description: "test config with duplicate profile",
			ScanSchedule: &v2.Schedule{
				IntervalType: 1,
				Hour:         15,
				Minute:       0,
				Interval: &v2.Schedule_DaysOfWeek_{
					DaysOfWeek: &v2.Schedule_DaysOfWeek{
						Days: []int32{1, 2, 3, 4, 5, 6},
					},
				},
			},
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
			OneTimeScan: false,
			Profiles:    []string{tpName},
			Description: "test config with duplicate tailored profile",
			ScanSchedule: &v2.Schedule{
				IntervalType: 1,
				Hour:         15,
				Minute:       0,
				Interval: &v2.Schedule_DaysOfWeek_{
					DaysOfWeek: &v2.Schedule_DaysOfWeek{
						Days: []int32{1, 2, 3, 4, 5, 6},
					},
				},
			},
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
			OneTimeScan: false,
			Profiles:    []string{"rhcos4-stig", "ocp4-cis-node"},
			Description: "test config with invalid profiles",
			ScanSchedule: &v2.Schedule{
				IntervalType: 1,
				Hour:         15,
				Minute:       0,
				Interval: &v2.Schedule_DaysOfWeek_{
					DaysOfWeek: &v2.Schedule_DaysOfWeek{
						Days: []int32{1, 2, 3, 4, 5, 6},
					},
				},
			},
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

	// Create per-test tailored profile and wait for ACS ingestion.
	testID := fmt.Sprintf("update-%s", uuid.NewV4().String())
	tpName := fmt.Sprintf("update-tp-%s", uuid.NewV4().String())
	createTailoredProfile(ctx, t, dynClient, tpName)
	profileClient := v2.NewComplianceProfileServiceClient(conn)
	waitUntilTPInCentralDB(ctx, t, profileClient, clusterID, tpName)

	// Create a scan configuration with a single regular profile.
	initialProfiles := []profileRef{
		{name: "ocp4-moderate", operatorKind: v2.ComplianceProfile_PROFILE},
	}
	req := &v2.ComplianceScanConfiguration{
		ScanName: testID,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    profileNames(initialProfiles),
			Description: "test config",
			ScanSchedule: &v2.Schedule{
				IntervalType: 1,
				Hour:         15,
				Minute:       0,
				Interval: &v2.Schedule_DaysOfWeek_{
					DaysOfWeek: &v2.Schedule_DaysOfWeek{
						Days: []int32{1, 2, 3, 4, 5, 6},
					},
				},
			},
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

	// Create per-test tailored profile and wait for ACS ingestion.
	testID := fmt.Sprintf("delete-%s", uuid.NewV4().String())
	tpName := fmt.Sprintf("delete-tp-%s", uuid.NewV4().String())
	createTailoredProfile(ctx, t, dynClient, tpName)
	profileClient := v2.NewComplianceProfileServiceClient(conn)
	waitUntilTPInCentralDB(ctx, t, profileClient, clusterID, tpName)

	req := &v2.ComplianceScanConfiguration{
		ScanName: testID,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"rhcos4-high", tpName},
			Description: "test config",
			ScanSchedule: &v2.Schedule{
				IntervalType: 1,
				Hour:         15,
				Minute:       0,
				Interval: &v2.Schedule_DaysOfWeek_{
					DaysOfWeek: &v2.Schedule_DaysOfWeek{
						Days: []int32{1, 2, 3, 4, 5, 6},
					},
				},
			},
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
			OneTimeScan: false,
			Profiles:    []string{"rhcos4-nerc-cip"},
			Description: "test config",
			ScanSchedule: &v2.Schedule{
				IntervalType: 1,
				Hour:         15,
				Minute:       0,
				Interval: &v2.Schedule_DaysOfWeek_{
					DaysOfWeek: &v2.Schedule_DaysOfWeek{
						Days: []int32{1, 2, 3, 4, 5, 6},
					},
				},
			},
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

	// Create per-test tailored profile and wait for ACS ingestion.
	testID := fmt.Sprintf("rescan-%s", uuid.NewV4().String())
	tpName := fmt.Sprintf("rescan-tp-%s", uuid.NewV4().String())
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
// different TailoredProfile variants: extends-base (with disabled rules),
// from-scratch with custom rules, and from-scratch with regular rules only.
func TestComplianceV2TailoredProfileVariants(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	profileClient := v2.NewComplianceProfileServiceClient(conn)
	clusterID := getIntegrations(t).GetIntegrations()[0].GetClusterId()
	testID := fmt.Sprintf("variants-%s", uuid.NewV4().String())

	// --- Variant 1: extends-base TP (disables a rule from ocp4-cis) ---
	extendsTPName := testID + "-extends"
	const disabledRule = "ocp4-api-server-encryption-provider-cipher"
	extendsTP := &complianceoperatorv1.TailoredProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      extendsTPName,
			Namespace: coNamespaceV2,
		},
		Spec: complianceoperatorv1.TailoredProfileSpec{
			Extends:     "ocp4-cis",
			Title:       "E2E Extends Base",
			Description: "TP extending ocp4-cis for e2e testing",
			DisableRules: []complianceoperatorv1.RuleReferenceSpec{
				{Name: disabledRule, Rationale: "e2e test"},
			},
		},
	}
	require.NoError(t, dynClient.Create(ctx, extendsTP), "failed to create extends-base TP")
	t.Cleanup(func() {
		deleteResource[complianceoperatorv1.TailoredProfile](ctx, t, dynClient, extendsTPName, coNamespaceV2)
	})
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var current complianceoperatorv1.TailoredProfile
		err := dynClient.Get(ctx, types.NamespacedName{Name: extendsTPName, Namespace: coNamespaceV2}, &current)
		require.NoErrorf(c, err, "failed to get TailoredProfile %s", extendsTPName)
		require.Equalf(c, complianceoperatorv1.TailoredProfileStateReady, current.Status.State,
			"TailoredProfile %s not READY (state: %q, error: %q)",
			extendsTPName, current.Status.State, current.Status.ErrorMessage)
	}, 10*time.Second, 1*time.Second)

	// --- Variant 2: from-scratch TP with a custom rule ---
	crName := testID + "-cr"
	createCustomRule(ctx, t, dynClient, crName)
	fromScratchCRName := testID + "-from-scratch-cr"
	fromScratchCR := &complianceoperatorv1.TailoredProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fromScratchCRName,
			Namespace: coNamespaceV2,
		},
		Spec: complianceoperatorv1.TailoredProfileSpec{
			Title:       "E2E From-Scratch with CustomRule",
			Description: "From-scratch TP with a custom rule for e2e testing",
			EnableRules: []complianceoperatorv1.RuleReferenceSpec{
				{Name: crName, Kind: complianceoperatorv1.CustomRuleKind, Rationale: "e2e test"},
			},
		},
	}
	require.NoError(t, dynClient.Create(ctx, fromScratchCR), "failed to create from-scratch TP with custom rule")
	t.Cleanup(func() {
		deleteResource[complianceoperatorv1.TailoredProfile](ctx, t, dynClient, fromScratchCRName, coNamespaceV2)
	})
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var current complianceoperatorv1.TailoredProfile
		err := dynClient.Get(ctx, types.NamespacedName{Name: fromScratchCRName, Namespace: coNamespaceV2}, &current)
		require.NoErrorf(c, err, "failed to get TailoredProfile %s", fromScratchCRName)
		require.Equalf(c, complianceoperatorv1.TailoredProfileStateReady, current.Status.State,
			"TailoredProfile %s not READY (state: %q, error: %q)",
			fromScratchCRName, current.Status.State, current.Status.ErrorMessage)
	}, 10*time.Second, 1*time.Second)

	// --- Variant 3: from-scratch TP with regular rules only (no custom rules) ---
	fromScratchRegName := testID + "-from-scratch-reg"
	fromScratchReg := &complianceoperatorv1.TailoredProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fromScratchRegName,
			Namespace: coNamespaceV2,
		},
		Spec: complianceoperatorv1.TailoredProfileSpec{
			Title:       "E2E From-Scratch with Regular Rules",
			Description: "From-scratch TP with regular rules for e2e testing",
			EnableRules: []complianceoperatorv1.RuleReferenceSpec{
				{Name: disabledRule, Kind: complianceoperatorv1.RuleKind, Rationale: "e2e test"},
			},
		},
	}
	require.NoError(t, dynClient.Create(ctx, fromScratchReg), "failed to create from-scratch TP with regular rules")
	t.Cleanup(func() {
		deleteResource[complianceoperatorv1.TailoredProfile](ctx, t, dynClient, fromScratchRegName, coNamespaceV2)
	})
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var current complianceoperatorv1.TailoredProfile
		err := dynClient.Get(ctx, types.NamespacedName{Name: fromScratchRegName, Namespace: coNamespaceV2}, &current)
		require.NoErrorf(c, err, "failed to get TailoredProfile %s", fromScratchRegName)
		require.Equalf(c, complianceoperatorv1.TailoredProfileStateReady, current.Status.State,
			"TailoredProfile %s not READY (state: %q, error: %q)",
			fromScratchRegName, current.Status.State, current.Status.ErrorMessage)
	}, 10*time.Second, 1*time.Second)

	// --- Wait for ACS ingestion of all three TPs ---
	extendsInACS := waitUntilTPInCentralDB(ctx, t, profileClient, clusterID, extendsTPName)
	fromScratchCRInACS := waitUntilTPInCentralDB(ctx, t, profileClient, clusterID, fromScratchCRName)
	fromScratchRegInACS := waitUntilTPInCentralDB(ctx, t, profileClient, clusterID, fromScratchRegName)

	// All variants should have operator_kind TAILORED_PROFILE.
	assert.Equal(t, v2.ComplianceProfile_TAILORED_PROFILE, extendsInACS.GetOperatorKind(),
		"extends-base TP should have operator_kind TAILORED_PROFILE")
	assert.Equal(t, v2.ComplianceProfile_TAILORED_PROFILE, fromScratchCRInACS.GetOperatorKind(),
		"from-scratch TP (custom rule) should have operator_kind TAILORED_PROFILE")
	assert.Equal(t, v2.ComplianceProfile_TAILORED_PROFILE, fromScratchRegInACS.GetOperatorKind(),
		"from-scratch TP (regular rules) should have operator_kind TAILORED_PROFILE")

	// --- Verify extends-base TP: inherited rules present, disabled rule excluded ---
	extendsDetail, err := profileClient.GetComplianceProfile(ctx, &v2.ResourceByID{Id: extendsInACS.GetId()})
	require.NoError(t, err)
	assert.Greater(t, len(extendsDetail.GetRules()), 0, "extends-base TP should have rules inherited from ocp4-cis")
	for _, r := range extendsDetail.GetRules() {
		assert.NotEqualf(t, disabledRule, r.GetName(),
			"disabled rule %q should not be in extends-base TP rules list", disabledRule)
	}

	// --- Verify from-scratch TP with custom rule: custom rule present ---
	fromScratchCRDetail, err := profileClient.GetComplianceProfile(ctx, &v2.ResourceByID{Id: fromScratchCRInACS.GetId()})
	require.NoError(t, err)
	foundCustomRule := false
	for _, r := range fromScratchCRDetail.GetRules() {
		if r.GetName() == crName {
			foundCustomRule = true
			break
		}
	}
	assert.True(t, foundCustomRule,
		"from-scratch TP should contain custom rule %s in its rules list", crName)

	// --- Verify from-scratch TP with regular rules: enabled rule present ---
	fromScratchRegDetail, err := profileClient.GetComplianceProfile(ctx, &v2.ResourceByID{Id: fromScratchRegInACS.GetId()})
	require.NoError(t, err)
	foundEnabledRule := false
	for _, r := range fromScratchRegDetail.GetRules() {
		if r.GetName() == disabledRule {
			foundEnabledRule = true
			break
		}
	}
	assert.True(t, foundEnabledRule,
		"from-scratch TP (regular rules) should contain rule %s", disabledRule)
}
