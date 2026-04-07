//go:build compliance

package tests

import (
	"context"
	"fmt"
	"strings"
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
	coNamespaceV2          = "openshift-compliance"
	stackroxNamespace      = "stackrox"
	defaultTimeout         = 120 * time.Second
	defaultInterval        = 5 * time.Second
	waitForDoneTimeout     = 5 * time.Minute
	waitForDoneInterval    = 30 * time.Second
	e2eTailoredProfileName = "e2e-tailored-profile"
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

// profileRef pairs a profile name with its compliance operator kind, used to
// assert both the name and the Kind field on ScanSettingBinding profile entries.
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

	// Use mixed profiles (platform Profile + TailoredProfile) to validate that the startup
	// sync path preserves profile_refs with correct kinds.
	initialProfiles := []profileRef{
		{name: "ocp4-cis", operatorKind: v2.ComplianceProfile_PROFILE},
		{name: e2eTailoredProfileName, operatorKind: v2.ComplianceProfile_TAILORED_PROFILE},
	}

	scanName := fmt.Sprintf("sync-test-%s", uuid.NewV4().String())
	scanConfig := v2.ComplianceScanConfiguration{
		ScanName: scanName,
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			Description:  scanName,
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
		cleanUpResources(ctx, t, dynClient, scanName, coNamespaceV2)
	})

	// Scale up Sensor
	scaleToN(ctx, t, k8sClient, "sensor", stackroxNamespace, 1)
	waitForDeploymentReady(ctx, t, dynClient, "sensor", stackroxNamespace, 1)

	// Assert the ScanSetting and the ScanSettingBinding are created with correct profile kinds.
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assertScanSetting(ctx, wrapCollectT(t, c), dynClient, scanName, coNamespaceV2, &scanConfig)
		assertScanSettingBinding(ctx, wrapCollectT(t, c), dynClient, scanName, coNamespaceV2, initialProfiles)
	}, defaultTimeout, defaultInterval)

	// Scale down Sensor
	scaleToN(ctx, t, k8sClient, "sensor", stackroxNamespace, 0)
	waitForDeploymentReady(ctx, t, dynClient, "sensor", stackroxNamespace, 0)

	// Update the ScanConfig in Central with a different set of mixed profiles.
	updatedProfiles := []profileRef{
		{name: "ocp4-pci-dss", operatorKind: v2.ComplianceProfile_PROFILE},
		{name: e2eTailoredProfileName, operatorKind: v2.ComplianceProfile_TAILORED_PROFILE},
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
		assertScanSetting(ctx, wrapCollectT(t, c), dynClient, scanName, coNamespaceV2, &scanConfig)
		assertScanSettingBinding(ctx, wrapCollectT(t, c), dynClient, scanName, coNamespaceV2, updatedProfiles)
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
	assertResourceDoesNotExist[complianceoperatorv1.ScanSetting](ctx, t, dynClient, scanName, coNamespaceV2)
	assertResourceDoesNotExist[complianceoperatorv1.ScanSettingBinding](ctx, t, dynClient, scanName, coNamespaceV2)
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
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceProfileServiceClient(conn)

	// Get the profiles for the cluster
	clusterID := getIntegrations(t).GetIntegrations()[0].GetClusterId()
	profileList, err := client.ListComplianceProfiles(context.TODO(), &v2.ProfilesForClusterRequest{ClusterId: clusterID})
	assert.NoError(t, err)
	assert.Greater(t, len(profileList.GetProfiles()), 0, "failed to assert the cluster has profiles")

	// Find the e2e TailoredProfile and a regular Profile, asserting operator_kind on each.
	var tpProfile *v2.ComplianceProfile
	var regularProfile *v2.ComplianceProfile
	for _, p := range profileList.GetProfiles() {
		if p.GetName() == e2eTailoredProfileName {
			tpProfile = p
		} else if regularProfile == nil && p.GetOperatorKind() == v2.ComplianceProfile_PROFILE {
			regularProfile = p
		}
	}
	require.NotNilf(t, tpProfile, "e2e tailored profile %q not found in profile list", e2eTailoredProfileName)
	assert.Equal(t, v2.ComplianceProfile_TAILORED_PROFILE, tpProfile.GetOperatorKind(),
		"e2e tailored profile should have operator_kind TAILORED_PROFILE")

	require.NotNil(t, regularProfile, "no regular profile found in profile list")
	assert.Equal(t, v2.ComplianceProfile_PROFILE, regularProfile.GetOperatorKind(),
		"regular profile should have operator_kind PROFILE")

	// Get the TailoredProfile by ID and verify operator_kind + custom rule in rules list.
	tp, err := client.GetComplianceProfile(context.TODO(), &v2.ResourceByID{Id: tpProfile.GetId()})
	require.NoError(t, err)
	assert.Equal(t, v2.ComplianceProfile_TAILORED_PROFILE, tp.GetOperatorKind(),
		"GetComplianceProfile response should have operator_kind TAILORED_PROFILE")
	foundCustomRule := false
	for _, r := range tp.GetRules() {
		if strings.Contains(r.GetName(), "check-cm-marker") {
			foundCustomRule = true
			break
		}
	}
	assert.True(t, foundCustomRule, "e2e tailored profile should contain the custom rule check-cm-marker")

	// Verify a regular profile also has rules and correct operator_kind via GetComplianceProfile.
	regProfile, err := client.GetComplianceProfile(context.TODO(), &v2.ResourceByID{Id: regularProfile.GetId()})
	require.NoError(t, err)
	assert.Equal(t, v2.ComplianceProfile_PROFILE, regProfile.GetOperatorKind(),
		"regular profile GetComplianceProfile should have operator_kind PROFILE")
	assert.Greater(t, len(regProfile.GetRules()), 0, "regular profile should have rules")
}

func TestComplianceV2ProfileGetSummaries(t *testing.T) {
	t.Parallel()
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceProfileServiceClient(conn)

	// Get the profiles for the cluster
	clusterID := getIntegrations(t).GetIntegrations()[0].GetClusterId()
	profileSummaries, err := client.ListProfileSummaries(context.TODO(), &v2.ClustersProfileSummaryRequest{ClusterIds: []string{clusterID}})
	assert.NoError(t, err)
	assert.Greater(t, len(profileSummaries.GetProfiles()), 0, "failed to assert the cluster has profiles")

	// Find the e2e TailoredProfile in summaries and assert operator_kind.
	var foundTP bool
	var foundRegular bool
	for _, p := range profileSummaries.GetProfiles() {
		if p.GetName() == e2eTailoredProfileName {
			assert.Equal(t, v2.ComplianceProfileSummary_TAILORED_PROFILE, p.GetOperatorKind(),
				"e2e tailored profile summary should have operator_kind TAILORED_PROFILE")
			foundTP = true
		} else if p.GetOperatorKind() == v2.ComplianceProfileSummary_PROFILE {
			foundRegular = true
		}
	}
	assert.True(t, foundTP, "e2e tailored profile %q not found in profile summaries", e2eTailoredProfileName)
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
	testName := fmt.Sprintf("create-get-%s", uuid.NewV4().String())

	// Use mixed profiles: a regular Profile and a TailoredProfile.
	initialProfiles := []profileRef{
		{name: "rhcos4-e8", operatorKind: v2.ComplianceProfile_PROFILE},
		{name: e2eTailoredProfileName, operatorKind: v2.ComplianceProfile_TAILORED_PROFILE},
	}

	req := &v2.ComplianceScanConfiguration{
		ScanName: testName,
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
		cleanUpResources(ctx, t, dynClient, testName, coNamespaceV2)
	})

	query := &v2.RawQuery{Query: ""}
	scanConfigs, err := scanConfigService.ListComplianceScanConfigurations(ctx, query)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(scanConfigs.GetConfigurations()), 1)
	assert.GreaterOrEqual(t, scanConfigs.GetTotalCount(), int32(1))

	// Wait for SSB and assert profile kinds.
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assertScanSettingBinding(ctx, wrapCollectT(t, c), dynClient, testName, coNamespaceV2, initialProfiles)
	}, defaultTimeout, defaultInterval)

	serviceResult := v2.NewComplianceResultsServiceClient(conn)
	query = &v2.RawQuery{Query: ""}
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		results, err := serviceResult.GetComplianceScanResults(ctx, query)
		require.NoError(c, err)

		resultsList := results.GetScanResults()
		var found bool
		for _, result := range resultsList {
			if result.GetScanName() == testName {
				found = true
				break
			}
		}
		require.True(c, found, "scan result not found for %s", testName)
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
	assert.Contains(t, err.Error(), "already uses profile")

	// Also verify that creating with the TP name is rejected (duplicate TP).
	duplicateTPReq := &v2.ComplianceScanConfiguration{
		ScanName: fmt.Sprintf("create-get-dup-tp-%s", uuid.NewV4().String()),
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{e2eTailoredProfileName},
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
	assert.Contains(t, err.Error(), "already uses profile")

	query = &v2.RawQuery{Query: ""}
	scanConfigs, err = scanConfigService.ListComplianceScanConfigurations(ctx, query)
	assert.NoError(t, err)
	// Verify the original config exists but duplicates were not created.
	assert.NotEmpty(t, getscanConfigID(testName, scanConfigs.GetConfigurations()), "expected original scan config %s to exist", testName)
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
	assert.NotEmpty(t, getscanConfigID(testName, scanConfigs.GetConfigurations()), "expected original scan config %s to exist", testName)
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

	// Create a scan configuration with a single regular profile.
	initialProfiles := []profileRef{
		{name: "ocp4-moderate", operatorKind: v2.ComplianceProfile_PROFILE},
	}
	scanName := fmt.Sprintf("update-%s", uuid.NewV4().String())
	req := &v2.ComplianceScanConfiguration{
		ScanName: scanName,
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
		assertScanSetting(ctx, wrapCollectT(t, c), dynClient, scanName, coNamespaceV2, req)
		assertScanSettingBinding(ctx, wrapCollectT(t, c), dynClient, scanName, coNamespaceV2, initialProfiles)
	}, defaultTimeout, defaultInterval)

	// Update to mixed profiles: a regular profile + the TailoredProfile.
	updatedProfiles := []profileRef{
		{name: "ocp4-moderate-node", operatorKind: v2.ComplianceProfile_PROFILE},
		{name: e2eTailoredProfileName, operatorKind: v2.ComplianceProfile_TAILORED_PROFILE},
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
		assertScanSetting(ctx, wrapCollectT(t, c), dynClient, scanName, coNamespaceV2, updateReq)
		assertScanSettingBinding(ctx, wrapCollectT(t, c), dynClient, scanName, coNamespaceV2, updatedProfiles)
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
	testName := fmt.Sprintf("delete-%s", uuid.NewV4().String())
	req := &v2.ComplianceScanConfiguration{
		ScanName: testName,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"rhcos4-high", e2eTailoredProfileName},
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
	scanConfigs, _ := scanConfigService.ListComplianceScanConfigurations(ctx, query)
	configs := scanConfigs.GetConfigurations()
	scanconfigID := getscanConfigID(testName, configs)
	reqDelete := &v2.ResourceByID{
		Id: scanconfigID,
	}
	_, err = scanConfigService.DeleteComplianceScanConfiguration(ctx, reqDelete)
	assert.NoError(t, err)

	// Verify scan configuration no longer exists
	scanConfigs, _ = scanConfigService.ListComplianceScanConfigurations(ctx, query)
	configs = scanConfigs.GetConfigurations()
	scanconfigID = getscanConfigID(testName, configs)
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
	scanConfigs, _ := scanConfigService.ListComplianceScanConfigurations(ctx, query)
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
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceScanConfigurationServiceClient(conn)
	clusterId := getIntegrations(t).GetIntegrations()[0].GetClusterId()

	scanConfigName := fmt.Sprintf("schedule-rescan-%s", uuid.NewV4().String())
	sc := v2.ComplianceScanConfiguration{
		ScanName: scanConfigName,
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"ocp4-e8", e2eTailoredProfileName},
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
		cleanUpResources(context.Background(), t, dynClient, scanConfigName, coNamespaceV2)
	})

	waitForComplianceSuiteToComplete(t, dynClient, scanConfig.GetScanName(), waitForDoneInterval, waitForDoneTimeout)

	// Invoke a rescan
	_, err = client.RunComplianceScanConfiguration(context.TODO(), &v2.ResourceByID{Id: scanConfig.GetId()})
	require.NoErrorf(t, err, "failed to rerun scan schedule %s", scanConfigName)

	// Assert the scan is rerunning on the cluster using the Compliance Operator CRDs
	waitForComplianceSuiteToComplete(t, dynClient, scanConfig.GetScanName(), waitForDoneInterval, waitForDoneTimeout)
}

// TestComplianceV2TailoredProfileVariants verifies that ACS correctly tracks both
// from-scratch and extends-base TailoredProfiles, including their operator_kind and
// effective rules list (excluded disabled rules).
//
// Note: CO does not allow mixing CustomRules and regular Rules in an extends-based
// TailoredProfile. The extends-base TP here only disables a regular rule (no custom
// rules). The from-scratch TP (applied during cluster setup) covers the custom rule path.
//
// NOT parallel — creates K8s resources directly on the cluster.
func TestComplianceV2TailoredProfileVariants(t *testing.T) {
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	profileClient := v2.NewComplianceProfileServiceClient(conn)
	clusterID := getIntegrations(t).GetIntegrations()[0].GetClusterId()

	// Create an extends-base TailoredProfile that only disables a regular rule.
	// CO rejects mixing CustomRules and regular Rules in the same TP.
	extendsTPName := "e2e-tp-extends-base"
	const disabledRule = "ocp4-api-server-encryption-provider-cipher"
	tp := &complianceoperatorv1.TailoredProfile{
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
	require.NoError(t, dynClient.Create(ctx, tp), "failed to create extends-base TailoredProfile")
	t.Cleanup(func() {
		deleteResource[complianceoperatorv1.TailoredProfile](ctx, t, dynClient, extendsTPName, coNamespaceV2)
	})

	// Wait for the extends-base TP to reach READY state in k8s.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var current complianceoperatorv1.TailoredProfile
		err := dynClient.Get(ctx, types.NamespacedName{Name: extendsTPName, Namespace: coNamespaceV2}, &current)
		require.NoErrorf(c, err, "failed to get TailoredProfile %s", extendsTPName)
		require.Equalf(c, complianceoperatorv1.TailoredProfileStateReady, current.Status.State,
			"TailoredProfile %s not READY (current state: %q, error: %q)",
			extendsTPName, current.Status.State, current.Status.ErrorMessage)
	}, 3*time.Minute, 10*time.Second)

	// Wait for ACS to ingest the extends-base TP (dispatched by Sensor after k8s event).
	var extendsTPInACS *v2.ComplianceProfile
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		profileList, err := profileClient.ListComplianceProfiles(ctx, &v2.ProfilesForClusterRequest{ClusterId: clusterID})
		require.NoErrorf(c, err, "failed to list profiles")
		for _, p := range profileList.GetProfiles() {
			if p.GetName() == extendsTPName {
				extendsTPInACS = p
				return
			}
		}
		require.Failf(c, "extends-base TP not yet in ACS profile list", "profile %q not found", extendsTPName)
	}, 2*time.Minute, 10*time.Second)

	// Verify both TPs in the profile list have operator_kind TAILORED_PROFILE.
	var fromScratchInACS *v2.ComplianceProfile
	profileList, err := profileClient.ListComplianceProfiles(ctx, &v2.ProfilesForClusterRequest{ClusterId: clusterID})
	require.NoError(t, err)
	for _, p := range profileList.GetProfiles() {
		if p.GetName() == e2eTailoredProfileName {
			fromScratchInACS = p
		}
	}
	require.NotNilf(t, fromScratchInACS, "from-scratch TP %q not found in profile list", e2eTailoredProfileName)
	assert.Equal(t, v2.ComplianceProfile_TAILORED_PROFILE, fromScratchInACS.GetOperatorKind(),
		"from-scratch TP should have operator_kind TAILORED_PROFILE")
	assert.Equal(t, v2.ComplianceProfile_TAILORED_PROFILE, extendsTPInACS.GetOperatorKind(),
		"extends-base TP should have operator_kind TAILORED_PROFILE")

	// Get the extends-base TP profile detail and verify the disabled rule is excluded.
	tpDetail, err := profileClient.GetComplianceProfile(ctx, &v2.ResourceByID{Id: extendsTPInACS.GetId()})
	require.NoError(t, err)
	assert.Equal(t, v2.ComplianceProfile_TAILORED_PROFILE, tpDetail.GetOperatorKind(),
		"GetComplianceProfile for extends-base TP should have operator_kind TAILORED_PROFILE")
	assert.Greater(t, len(tpDetail.GetRules()), 0, "extends-base TP should have rules inherited from ocp4-cis")

	// Disabled rule should NOT be present in effective rules.
	foundDisabledRule := false
	for _, r := range tpDetail.GetRules() {
		if r.GetName() == disabledRule {
			foundDisabledRule = true
			break
		}
	}
	assert.False(t, foundDisabledRule,
		"disabled rule %q should not be in extends-base TP rules list", disabledRule)
}
