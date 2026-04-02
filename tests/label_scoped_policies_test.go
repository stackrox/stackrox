//go:build test_e2e

package tests

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/require"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultNamespace = "default"
)

func TestClusterLabelPolicyScoping(t *testing.T) {
	clusterID := getClusterID(t)

	setClusterLabels(t, clusterID, map[string]string{"env": "prod"})
	defer setClusterLabels(t, clusterID, nil)

	// Wait for labels to sync to Central and Sensor
	time.Sleep(5 * time.Second)

	conn := centralgrpc.GRPCConnectionToCentral(t)
	policyService := v1.NewPolicyServiceClient(conn)
	alertService := v1.NewAlertServiceClient(conn)

	policy := &storage.Policy{
		Name:            "Test - Cluster Label Prod",
		Description:     "Test cluster label scoping",
		Severity:        storage.Severity_HIGH_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		Categories:      []string{"Test"},
		Scope: []*storage.Scope{
			{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod",
				},
			},
		},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Privileged Container",
						Values: []*storage.PolicyValue{
							{Value: "true"},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	createdPolicy, err := policyService.PostPolicy(ctx, &v1.PostPolicyRequest{Policy: policy})
	require.NoError(t, err)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_, _ = policyService.DeletePolicy(ctx, &v1.ResourceByID{Id: createdPolicy.GetId()})
	}()

	// Use random suffix to avoid conflicts between parallel test runs
	deploymentName := fmt.Sprintf("test-cluster-label-%d", rand.IntN(10000))

	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", deploymentName, 1, defaultNamespace, true)
	defer teardownDeploymentWithoutCheck(t, deploymentName, defaultNamespace)

	qb := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, deploymentName).
		AddStrings(search.PolicyName, policy.GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(t, alertService, &v1.ListAlertsRequest{Query: qb.Query()}, 1)
	t.Logf("Alert appeared for deployment in cluster with env=prod")

	setClusterLabels(t, clusterID, map[string]string{"env": "dev"})
	time.Sleep(5 * time.Second) // Wait for label change to propagate

	deploymentName2 := fmt.Sprintf("test-cluster-label-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", deploymentName2, 1, defaultNamespace, true)
	defer teardownDeploymentWithoutCheck(t, deploymentName2, defaultNamespace)

	qb2 := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, deploymentName2).
		AddStrings(search.PolicyName, policy.GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(t, alertService, &v1.ListAlertsRequest{Query: qb2.Query()}, 0)
	t.Logf("No alert for deployment after cluster label changed to env=dev")
}

func TestNamespaceLabelPolicyScoping(t *testing.T) {
	backendNS := fmt.Sprintf("test-backend-%d", rand.IntN(10000))
	frontendNS := fmt.Sprintf("test-frontend-%d", rand.IntN(10000))

	createNamespaceWithLabels(t, backendNS, map[string]string{"team": "backend"})
	defer deleteNamespace(t, backendNS)

	createNamespaceWithLabels(t, frontendNS, map[string]string{"team": "frontend"})
	defer deleteNamespace(t, frontendNS)

	// Wait for namespaces to sync to Central
	time.Sleep(5 * time.Second)

	conn := centralgrpc.GRPCConnectionToCentral(t)
	policyService := v1.NewPolicyServiceClient(conn)
	alertService := v1.NewAlertServiceClient(conn)

	policy := &storage.Policy{
		Name:            "Test - Namespace Label Backend",
		Description:     "Test namespace label scoping",
		Severity:        storage.Severity_HIGH_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		Categories:      []string{"Test"},
		Scope: []*storage.Scope{
			{
				NamespaceLabel: &storage.Scope_Label{
					Key:   "team",
					Value: "backend",
				},
			},
		},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Privileged Container",
						Values: []*storage.PolicyValue{
							{Value: "true"},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	createdPolicy, err := policyService.PostPolicy(ctx, &v1.PostPolicyRequest{Policy: policy})
	require.NoError(t, err)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_, _ = policyService.DeletePolicy(ctx, &v1.ResourceByID{Id: createdPolicy.GetId()})
	}()

	backendDeployment := fmt.Sprintf("test-ns-backend-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", backendDeployment, 1, backendNS, true)
	defer teardownDeploymentWithoutCheck(t, backendDeployment, backendNS)

	qbBackend := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, backendDeployment).
		AddStrings(search.PolicyName, policy.GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(t, alertService, &v1.ListAlertsRequest{Query: qbBackend.Query()}, 1)
	t.Logf("Alert appeared for deployment in namespace with team=backend")

	frontendDeployment := fmt.Sprintf("test-ns-frontend-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", frontendDeployment, 1, frontendNS, true)
	defer teardownDeploymentWithoutCheck(t, frontendDeployment, frontendNS)

	qbFrontend := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, frontendDeployment).
		AddStrings(search.PolicyName, policy.GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(t, alertService, &v1.ListAlertsRequest{Query: qbFrontend.Query()}, 0)
	t.Logf("No alert for deployment in namespace with team=frontend")
}

func TestCombinedLabelPolicyScoping(t *testing.T) {
	backendNS := fmt.Sprintf("test-backend-%d", rand.IntN(10000))
	frontendNS := fmt.Sprintf("test-frontend-%d", rand.IntN(10000))

	clusterID := getClusterID(t)

	setClusterLabels(t, clusterID, map[string]string{"env": "prod"})
	defer setClusterLabels(t, clusterID, nil)

	createNamespaceWithLabels(t, backendNS, map[string]string{"team": "backend"})
	defer deleteNamespace(t, backendNS)

	createNamespaceWithLabels(t, frontendNS, map[string]string{"team": "frontend"})
	defer deleteNamespace(t, frontendNS)

	time.Sleep(5 * time.Second)

	conn := centralgrpc.GRPCConnectionToCentral(t)
	policyService := v1.NewPolicyServiceClient(conn)
	alertService := v1.NewAlertServiceClient(conn)

	policy := &storage.Policy{
		Name:            "Test - Combined Labels",
		Description:     "Test combined cluster and namespace label scoping",
		Severity:        storage.Severity_HIGH_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		Categories:      []string{"Test"},
		Scope: []*storage.Scope{
			{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod",
				},
				NamespaceLabel: &storage.Scope_Label{
					Key:   "team",
					Value: "backend",
				},
			},
		},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Privileged Container",
						Values: []*storage.PolicyValue{
							{Value: "true"},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	createdPolicy, err := policyService.PostPolicy(ctx, &v1.PostPolicyRequest{Policy: policy})
	require.NoError(t, err)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_, _ = policyService.DeletePolicy(ctx, &v1.ResourceByID{Id: createdPolicy.GetId()})
	}()

	// Should violate: cluster=prod AND namespace=backend both match
	backendDeployment := fmt.Sprintf("test-combined-backend-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", backendDeployment, 1, backendNS, true)
	defer teardownDeploymentWithoutCheck(t, backendDeployment, backendNS)

	qbBackend := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, backendDeployment).
		AddStrings(search.PolicyName, policy.GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(t, alertService, &v1.ListAlertsRequest{Query: qbBackend.Query()}, 1)
	t.Logf("Alert appeared for deployment in cluster=prod AND namespace=backend")

	// Should NOT violate: namespace=frontend doesn't match
	frontendDeployment := fmt.Sprintf("test-combined-frontend-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", frontendDeployment, 1, frontendNS, true)
	defer teardownDeploymentWithoutCheck(t, frontendDeployment, frontendNS)

	qbFrontend := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, frontendDeployment).
		AddStrings(search.PolicyName, policy.GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(t, alertService, &v1.ListAlertsRequest{Query: qbFrontend.Query()}, 0)
	t.Logf("No alert for deployment in cluster=prod but namespace=frontend (namespace label doesn't match)")

	// Should NOT violate: namespace has no team label
	defaultDeployment := fmt.Sprintf("test-combined-default-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", defaultDeployment, 1, defaultNamespace, true)
	defer teardownDeploymentWithoutCheck(t, defaultDeployment, defaultNamespace)

	qbDefault := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, defaultDeployment).
		AddStrings(search.PolicyName, policy.GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(t, alertService, &v1.ListAlertsRequest{Query: qbDefault.Query()}, 0)
	t.Logf("No alert for deployment in cluster=prod but namespace has no team label")

	// Should NOT violate after cluster label changes: cluster=dev doesn't match
	setClusterLabels(t, clusterID, map[string]string{"env": "dev"})
	time.Sleep(5 * time.Second)

	backendDeployment2 := fmt.Sprintf("test-combined-backend-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", backendDeployment2, 1, backendNS, true)
	defer teardownDeploymentWithoutCheck(t, backendDeployment2, backendNS)

	qbBackend2 := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, backendDeployment2).
		AddStrings(search.PolicyName, policy.GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(t, alertService, &v1.ListAlertsRequest{Query: qbBackend2.Query()}, 0)
	t.Logf("No alert for deployment after cluster label changed to env=dev (cluster label doesn't match)")
}

func TestPolicyDryRunWithClusterLabel(t *testing.T) {
	backendNS := fmt.Sprintf("test-backend-%d", rand.IntN(10000))
	frontendNS := fmt.Sprintf("test-frontend-%d", rand.IntN(10000))

	clusterID := getClusterID(t)

	setClusterLabels(t, clusterID, map[string]string{"env": "prod"})
	defer setClusterLabels(t, clusterID, nil)

	createNamespaceWithLabels(t, backendNS, map[string]string{"team": "backend"})
	defer deleteNamespace(t, backendNS)

	createNamespaceWithLabels(t, frontendNS, map[string]string{"team": "frontend"})
	defer deleteNamespace(t, frontendNS)

	time.Sleep(5 * time.Second)

	backendDeployment := fmt.Sprintf("test-dryrun-backend-%d", rand.IntN(10000))
	setupDeploymentNoWaitInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", backendDeployment, 1, backendNS, true)
	defer teardownDeploymentWithoutCheck(t, backendDeployment, backendNS)

	frontendDeployment := fmt.Sprintf("test-dryrun-frontend-%d", rand.IntN(10000))
	setupDeploymentNoWaitInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", frontendDeployment, 1, frontendNS, true)
	defer teardownDeploymentWithoutCheck(t, frontendDeployment, frontendNS)

	defaultDeployment := fmt.Sprintf("test-dryrun-default-%d", rand.IntN(10000))
	setupDeploymentNoWaitInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", defaultDeployment, 1, defaultNamespace, true)
	defer teardownDeploymentWithoutCheck(t, defaultDeployment, defaultNamespace)

	waitForDeploymentReadyInK8s(t, backendDeployment, backendNS)
	waitForDeploymentReadyInK8s(t, frontendDeployment, frontendNS)
	waitForDeploymentReadyInK8s(t, defaultDeployment, defaultNamespace)

	waitForDeploymentInCentral(t, backendDeployment)
	waitForDeploymentInCentral(t, frontendDeployment)
	waitForDeploymentInCentral(t, defaultDeployment)

	conn := centralgrpc.GRPCConnectionToCentral(t)
	policyService := v1.NewPolicyServiceClient(conn)

	// Dry run evaluates existing deployments without persisting the policy
	policy := &storage.Policy{
		Name:            "Test - Dry Run Cluster Label",
		Description:     "Dry run test for cluster label scoping",
		Severity:        storage.Severity_HIGH_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		Categories:      []string{"Test"},
		Scope: []*storage.Scope{
			{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod",
				},
			},
		},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Privileged Container",
						Values: []*storage.PolicyValue{
							{Value: "true"},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	resp, err := policyService.DryRunPolicy(ctx, policy)
	cancel()
	require.NoError(t, err)

	alerts := resp.GetAlerts()
	require.Len(t, alerts, 3, "Expected 3 alerts for cluster label policy (all deployments in env=prod)")
	t.Logf("Dry run with cluster label policy: %d alerts (expected 3)", len(alerts))
}

func TestPolicyDryRunWithNamespaceLabel(t *testing.T) {
	backendNS := fmt.Sprintf("test-backend-%d", rand.IntN(10000))
	frontendNS := fmt.Sprintf("test-frontend-%d", rand.IntN(10000))

	clusterID := getClusterID(t)

	setClusterLabels(t, clusterID, map[string]string{"env": "prod"})
	defer setClusterLabels(t, clusterID, nil)

	createNamespaceWithLabels(t, backendNS, map[string]string{"team": "backend"})
	defer deleteNamespace(t, backendNS)

	createNamespaceWithLabels(t, frontendNS, map[string]string{"team": "frontend"})
	defer deleteNamespace(t, frontendNS)

	time.Sleep(5 * time.Second)

	backendDeployment := fmt.Sprintf("test-dryrun-backend-%d", rand.IntN(10000))
	setupDeploymentNoWaitInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", backendDeployment, 1, backendNS, true)
	defer teardownDeploymentWithoutCheck(t, backendDeployment, backendNS)

	frontendDeployment := fmt.Sprintf("test-dryrun-frontend-%d", rand.IntN(10000))
	setupDeploymentNoWaitInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", frontendDeployment, 1, frontendNS, true)
	defer teardownDeploymentWithoutCheck(t, frontendDeployment, frontendNS)

	defaultDeployment := fmt.Sprintf("test-dryrun-default-%d", rand.IntN(10000))
	setupDeploymentNoWaitInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", defaultDeployment, 1, defaultNamespace, true)
	defer teardownDeploymentWithoutCheck(t, defaultDeployment, defaultNamespace)

	waitForDeploymentReadyInK8s(t, backendDeployment, backendNS)
	waitForDeploymentReadyInK8s(t, frontendDeployment, frontendNS)
	waitForDeploymentReadyInK8s(t, defaultDeployment, defaultNamespace)

	waitForDeploymentInCentral(t, backendDeployment)
	waitForDeploymentInCentral(t, frontendDeployment)
	waitForDeploymentInCentral(t, defaultDeployment)

	conn := centralgrpc.GRPCConnectionToCentral(t)
	policyService := v1.NewPolicyServiceClient(conn)

	policy := &storage.Policy{
		Name:            "Test - Dry Run Namespace Label",
		Description:     "Dry run test for namespace label scoping",
		Severity:        storage.Severity_HIGH_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		Categories:      []string{"Test"},
		Scope: []*storage.Scope{
			{
				NamespaceLabel: &storage.Scope_Label{
					Key:   "team",
					Value: "backend",
				},
			},
		},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Privileged Container",
						Values: []*storage.PolicyValue{
							{Value: "true"},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	resp, err := policyService.DryRunPolicy(ctx, policy)
	cancel()
	require.NoError(t, err)

	alerts := resp.GetAlerts()
	require.Len(t, alerts, 1, "Expected 1 alert for namespace label policy (only backend deployment)")
	t.Logf("Dry run with namespace label policy: %d alert (expected 1)", len(alerts))
}

func TestPolicyDryRunWithCombinedLabels(t *testing.T) {
	backendNS := fmt.Sprintf("test-backend-%d", rand.IntN(10000))
	frontendNS := fmt.Sprintf("test-frontend-%d", rand.IntN(10000))

	clusterID := getClusterID(t)

	setClusterLabels(t, clusterID, map[string]string{"env": "prod"})
	defer setClusterLabels(t, clusterID, nil)

	createNamespaceWithLabels(t, backendNS, map[string]string{"team": "backend"})
	defer deleteNamespace(t, backendNS)

	createNamespaceWithLabels(t, frontendNS, map[string]string{"team": "frontend"})
	defer deleteNamespace(t, frontendNS)

	time.Sleep(5 * time.Second)

	backendDeployment := fmt.Sprintf("test-dryrun-backend-%d", rand.IntN(10000))
	setupDeploymentNoWaitInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", backendDeployment, 1, backendNS, true)
	defer teardownDeploymentWithoutCheck(t, backendDeployment, backendNS)

	frontendDeployment := fmt.Sprintf("test-dryrun-frontend-%d", rand.IntN(10000))
	setupDeploymentNoWaitInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", frontendDeployment, 1, frontendNS, true)
	defer teardownDeploymentWithoutCheck(t, frontendDeployment, frontendNS)

	defaultDeployment := fmt.Sprintf("test-dryrun-default-%d", rand.IntN(10000))
	setupDeploymentNoWaitInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", defaultDeployment, 1, defaultNamespace, true)
	defer teardownDeploymentWithoutCheck(t, defaultDeployment, defaultNamespace)

	waitForDeploymentReadyInK8s(t, backendDeployment, backendNS)
	waitForDeploymentReadyInK8s(t, frontendDeployment, frontendNS)
	waitForDeploymentReadyInK8s(t, defaultDeployment, defaultNamespace)

	waitForDeploymentInCentral(t, backendDeployment)
	waitForDeploymentInCentral(t, frontendDeployment)
	waitForDeploymentInCentral(t, defaultDeployment)

	conn := centralgrpc.GRPCConnectionToCentral(t)
	policyService := v1.NewPolicyServiceClient(conn)

	policy := &storage.Policy{
		Name:            "Test - Dry Run Combined Labels",
		Description:     "Dry run test for combined label scoping",
		Severity:        storage.Severity_HIGH_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		Categories:      []string{"Test"},
		Scope: []*storage.Scope{
			{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod",
				},
				NamespaceLabel: &storage.Scope_Label{
					Key:   "team",
					Value: "backend",
				},
			},
		},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Privileged Container",
						Values: []*storage.PolicyValue{
							{Value: "true"},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	resp, err := policyService.DryRunPolicy(ctx, policy)
	cancel()
	require.NoError(t, err)

	alerts := resp.GetAlerts()
	require.Len(t, alerts, 1, "Expected 1 alert for combined label policy (only backend deployment)")
	t.Logf("Dry run with combined label policy: %d alert (expected 1)", len(alerts))
}

func TestPolicyDryRunWithLabelMismatch(t *testing.T) {
	backendNS := fmt.Sprintf("test-backend-%d", rand.IntN(10000))
	frontendNS := fmt.Sprintf("test-frontend-%d", rand.IntN(10000))

	clusterID := getClusterID(t)

	setClusterLabels(t, clusterID, map[string]string{"env": "prod"})
	defer setClusterLabels(t, clusterID, nil)

	createNamespaceWithLabels(t, backendNS, map[string]string{"team": "backend"})
	defer deleteNamespace(t, backendNS)

	createNamespaceWithLabels(t, frontendNS, map[string]string{"team": "frontend"})
	defer deleteNamespace(t, frontendNS)

	time.Sleep(5 * time.Second)

	backendDeployment := fmt.Sprintf("test-dryrun-backend-%d", rand.IntN(10000))
	setupDeploymentNoWaitInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", backendDeployment, 1, backendNS, true)
	defer teardownDeploymentWithoutCheck(t, backendDeployment, backendNS)

	frontendDeployment := fmt.Sprintf("test-dryrun-frontend-%d", rand.IntN(10000))
	setupDeploymentNoWaitInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", frontendDeployment, 1, frontendNS, true)
	defer teardownDeploymentWithoutCheck(t, frontendDeployment, frontendNS)

	defaultDeployment := fmt.Sprintf("test-dryrun-default-%d", rand.IntN(10000))
	setupDeploymentNoWaitInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", defaultDeployment, 1, defaultNamespace, true)
	defer teardownDeploymentWithoutCheck(t, defaultDeployment, defaultNamespace)

	waitForDeploymentReadyInK8s(t, backendDeployment, backendNS)
	waitForDeploymentReadyInK8s(t, frontendDeployment, frontendNS)
	waitForDeploymentReadyInK8s(t, defaultDeployment, defaultNamespace)

	waitForDeploymentInCentral(t, backendDeployment)
	waitForDeploymentInCentral(t, frontendDeployment)
	waitForDeploymentInCentral(t, defaultDeployment)

	conn := centralgrpc.GRPCConnectionToCentral(t)
	policyService := v1.NewPolicyServiceClient(conn)

	policy := &storage.Policy{
		Name:            "Test - Dry Run Cluster Label Mismatch",
		Description:     "Dry run test for cluster label mismatch",
		Severity:        storage.Severity_HIGH_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		Categories:      []string{"Test"},
		Scope: []*storage.Scope{
			{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "dev",
				},
			},
		},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Privileged Container",
						Values: []*storage.PolicyValue{
							{Value: "true"},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	resp, err := policyService.DryRunPolicy(ctx, policy)
	cancel()
	require.NoError(t, err)

	alerts := resp.GetAlerts()
	require.Len(t, alerts, 0, "Expected 0 alerts for cluster label mismatch (cluster is prod, not dev)")
	t.Logf("Dry run with cluster label mismatch: %d alerts (expected 0)", len(alerts))
}

func TestRuntimeDetectionWithNamespaceLabels(t *testing.T) {
	backendNS := fmt.Sprintf("test-backend-%d", rand.IntN(10000))
	frontendNS := fmt.Sprintf("test-frontend-%d", rand.IntN(10000))

	createNamespaceWithLabels(t, backendNS, map[string]string{"team": "backend"})
	defer deleteNamespace(t, backendNS)

	createNamespaceWithLabels(t, frontendNS, map[string]string{"team": "frontend"})
	defer deleteNamespace(t, frontendNS)

	time.Sleep(5 * time.Second)

	conn := centralgrpc.GRPCConnectionToCentral(t)
	policyService := v1.NewPolicyServiceClient(conn)
	alertService := v1.NewAlertServiceClient(conn)

	// Triggers when "apt" process executes (detects runtime package manager usage)
	policy := &storage.Policy{
		Name:            "Test - Runtime Namespace Label",
		Description:     "Runtime detection test for namespace label scoping",
		Severity:        storage.Severity_HIGH_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		Categories:      []string{"Test"},
		Scope: []*storage.Scope{
			{
				NamespaceLabel: &storage.Scope_Label{
					Key:   "team",
					Value: "backend",
				},
			},
		},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Process Name",
						Values: []*storage.PolicyValue{
							{Value: "apt"},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	createdPolicy, err := policyService.PostPolicy(ctx, &v1.PostPolicyRequest{Policy: policy})
	require.NoError(t, err)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_, _ = policyService.DeletePolicy(ctx, &v1.ResourceByID{Id: createdPolicy.GetId()})
	}()

	backendDeployment := fmt.Sprintf("test-runtime-backend-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", backendDeployment, 1, backendNS, true)
	defer teardownDeploymentWithoutCheck(t, backendDeployment, backendNS)
	waitForDeploymentReadyInK8s(t, backendDeployment, backendNS)

	frontendDeployment := fmt.Sprintf("test-runtime-frontend-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", frontendDeployment, 1, frontendNS, true)
	defer teardownDeploymentWithoutCheck(t, frontendDeployment, frontendNS)
	waitForDeploymentReadyInK8s(t, frontendDeployment, frontendNS)

	client := createK8sClient(t)

	// Execute apt --help to trigger runtime detection (safe no-op that matches "Process Name: apt")
	execInDeployment(t, client, backendDeployment, backendNS, "apt", "--help")

	qbBackend := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, backendDeployment).
		AddStrings(search.PolicyName, policy.GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(t, alertService, &v1.ListAlertsRequest{Query: qbBackend.Query()}, 1)
	t.Logf("Runtime alert appeared for deployment in namespace with team=backend")

	execInDeployment(t, client, frontendDeployment, frontendNS, "apt", "--help")

	qbFrontend := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, frontendDeployment).
		AddStrings(search.PolicyName, policy.GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(t, alertService, &v1.ListAlertsRequest{Query: qbFrontend.Query()}, 0)
	t.Logf("No runtime alert for deployment in namespace with team=frontend")
}

func TestRuntimeDetectionWithClusterLabels(t *testing.T) {
	clusterID := getClusterID(t)

	setClusterLabels(t, clusterID, map[string]string{"env": "prod"})
	defer setClusterLabels(t, clusterID, nil)

	time.Sleep(5 * time.Second)

	conn := centralgrpc.GRPCConnectionToCentral(t)
	policyService := v1.NewPolicyServiceClient(conn)
	alertService := v1.NewAlertServiceClient(conn)

	// Triggers when "apt" process executes (detects runtime package manager usage)
	policy := &storage.Policy{
		Name:            "Test - Runtime Cluster Label",
		Description:     "Runtime detection test for cluster label scoping",
		Severity:        storage.Severity_HIGH_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		Categories:      []string{"Test"},
		Scope: []*storage.Scope{
			{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod",
				},
			},
		},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Process Name",
						Values: []*storage.PolicyValue{
							{Value: "apt"},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	createdPolicy, err := policyService.PostPolicy(ctx, &v1.PostPolicyRequest{Policy: policy})
	require.NoError(t, err)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_, _ = policyService.DeletePolicy(ctx, &v1.ResourceByID{Id: createdPolicy.GetId()})
	}()

	deployment1 := fmt.Sprintf("test-runtime-cluster-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", deployment1, 1, defaultNamespace, true)
	defer teardownDeploymentWithoutCheck(t, deployment1, defaultNamespace)
	waitForDeploymentReadyInK8s(t, deployment1, defaultNamespace)

	client := createK8sClient(t)

	// Execute apt --help to trigger runtime detection (safe no-op that matches "Process Name: apt")
	execInDeployment(t, client, deployment1, "default", "apt", "--help")

	qb1 := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, deployment1).
		AddStrings(search.PolicyName, policy.GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(t, alertService, &v1.ListAlertsRequest{Query: qb1.Query()}, 1)
	t.Logf("Runtime alert appeared for deployment in cluster with env=prod")

	// Change cluster label to dev (tests hot-reload)
	setClusterLabels(t, clusterID, map[string]string{"env": "dev"})
	time.Sleep(5 * time.Second)

	deployment2 := fmt.Sprintf("test-runtime-cluster-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", deployment2, 1, defaultNamespace, true)
	defer teardownDeploymentWithoutCheck(t, deployment2, defaultNamespace)
	waitForDeploymentReadyInK8s(t, deployment2, defaultNamespace)

	execInDeployment(t, client, deployment2, "default", "apt", "--help")

	qb2 := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, deployment2).
		AddStrings(search.PolicyName, policy.GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(t, alertService, &v1.ListAlertsRequest{Query: qb2.Query()}, 0)
	t.Logf("No runtime alert for deployment after cluster label changed to env=dev")
}

func TestLabelRemoval(t *testing.T) {
	clusterID := getClusterID(t)

	// Test cluster label removal
	setClusterLabels(t, clusterID, map[string]string{"env": "prod"})
	defer setClusterLabels(t, clusterID, nil)

	time.Sleep(5 * time.Second)

	conn := centralgrpc.GRPCConnectionToCentral(t)
	policyService := v1.NewPolicyServiceClient(conn)
	alertService := v1.NewAlertServiceClient(conn)

	clusterLabelPolicy := &storage.Policy{
		Name:            "Test - Label Removal Cluster",
		Description:     "Test cluster label removal",
		Severity:        storage.Severity_HIGH_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		Categories:      []string{"Test"},
		Scope: []*storage.Scope{
			{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod",
				},
			},
		},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Privileged Container",
						Values: []*storage.PolicyValue{
							{Value: "true"},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	createdPolicy, err := policyService.PostPolicy(ctx, &v1.PostPolicyRequest{Policy: clusterLabelPolicy})
	require.NoError(t, err)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_, _ = policyService.DeletePolicy(ctx, &v1.ResourceByID{Id: createdPolicy.GetId()})
	}()

	deployment1 := fmt.Sprintf("test-label-removal-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", deployment1, 1, defaultNamespace, true)
	defer teardownDeploymentWithoutCheck(t, deployment1, defaultNamespace)

	qb1 := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, deployment1).
		AddStrings(search.PolicyName, clusterLabelPolicy.GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(t, alertService, &v1.ListAlertsRequest{Query: qb1.Query()}, 1)
	t.Logf("Alert appeared for deployment with cluster label env=prod")

	// Remove cluster labels entirely
	setClusterLabels(t, clusterID, nil)
	time.Sleep(5 * time.Second)

	deployment2 := fmt.Sprintf("test-label-removal-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", deployment2, 1, defaultNamespace, true)
	defer teardownDeploymentWithoutCheck(t, deployment2, defaultNamespace)

	qb2 := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, deployment2).
		AddStrings(search.PolicyName, clusterLabelPolicy.GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(t, alertService, &v1.ListAlertsRequest{Query: qb2.Query()}, 0)
	t.Logf("No alert for deployment after cluster labels removed entirely")

	// Test namespace label removal
	testNS := "test-label-removal-ns"
	createNamespaceWithLabels(t, testNS, map[string]string{"team": "backend"})
	defer deleteNamespace(t, testNS)

	time.Sleep(5 * time.Second)

	nsLabelPolicy := &storage.Policy{
		Name:            "Test - Label Removal Namespace",
		Description:     "Test namespace label removal",
		Severity:        storage.Severity_HIGH_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		Categories:      []string{"Test"},
		Scope: []*storage.Scope{
			{
				NamespaceLabel: &storage.Scope_Label{
					Key:   "team",
					Value: "backend",
				},
			},
		},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Privileged Container",
						Values: []*storage.PolicyValue{
							{Value: "true"},
						},
					},
				},
			},
		},
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	createdNSPolicy, err := policyService.PostPolicy(ctx, &v1.PostPolicyRequest{Policy: nsLabelPolicy})
	require.NoError(t, err)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_, _ = policyService.DeletePolicy(ctx, &v1.ResourceByID{Id: createdNSPolicy.GetId()})
	}()

	deployment3 := fmt.Sprintf("test-ns-removal-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", deployment3, 1, testNS, true)
	defer teardownDeploymentWithoutCheck(t, deployment3, testNS)

	qb3 := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, deployment3).
		AddStrings(search.PolicyName, nsLabelPolicy.GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(t, alertService, &v1.ListAlertsRequest{Query: qb3.Query()}, 1)
	t.Logf("Alert appeared for deployment in namespace with team=backend")

	// Remove namespace labels by replacing with empty label set
	client := createK8sClient(t)
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ns, err := client.CoreV1().Namespaces().Get(ctx, testNS, metaV1.GetOptions{})
	require.NoError(t, err)
	ns.Labels = map[string]string{}

	_, err = client.CoreV1().Namespaces().Update(ctx, ns, metaV1.UpdateOptions{})
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	deployment4 := fmt.Sprintf("test-ns-removal-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", deployment4, 1, testNS, true)
	defer teardownDeploymentWithoutCheck(t, deployment4, testNS)

	qb4 := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, deployment4).
		AddStrings(search.PolicyName, nsLabelPolicy.GetName()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(t, alertService, &v1.ListAlertsRequest{Query: qb4.Query()}, 0)
	t.Logf("No alert for deployment after namespace labels removed entirely")
}
