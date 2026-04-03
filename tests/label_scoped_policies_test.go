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
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultNamespace = "default"
)

func TestNamespaceLabelPolicyScoping(t *testing.T) {
	backendNS := fmt.Sprintf("test-backend-%d", rand.IntN(10000))
	frontendNS := fmt.Sprintf("test-frontend-%d", rand.IntN(10000))

	createNamespaceWithLabels(t, backendNS, map[string]string{"team": "backend"})
	defer deleteNamespace(t, backendNS)

	createNamespaceWithLabels(t, frontendNS, map[string]string{"team": "frontend"})
	defer deleteNamespace(t, frontendNS)

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

func TestPolicyDryRunWithNamespaceLabel(t *testing.T) {
	backendNS := fmt.Sprintf("test-backend-%d", rand.IntN(10000))
	frontendNS := fmt.Sprintf("test-frontend-%d", rand.IntN(10000))

	createNamespaceWithLabels(t, backendNS, map[string]string{"team": "backend"})
	defer deleteNamespace(t, backendNS)

	createNamespaceWithLabels(t, frontendNS, map[string]string{"team": "frontend"})
	defer deleteNamespace(t, frontendNS)

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

func TestRuntimeDetectionWithNamespaceLabels(t *testing.T) {
	backendNS := fmt.Sprintf("test-backend-%d", rand.IntN(10000))
	frontendNS := fmt.Sprintf("test-frontend-%d", rand.IntN(10000))

	createNamespaceWithLabels(t, backendNS, map[string]string{"team": "backend"})
	defer deleteNamespace(t, backendNS)

	createNamespaceWithLabels(t, frontendNS, map[string]string{"team": "frontend"})
	defer deleteNamespace(t, frontendNS)

	conn := centralgrpc.GRPCConnectionToCentral(t)
	policyService := v1.NewPolicyServiceClient(conn)
	alertService := v1.NewAlertServiceClient(conn)

	// Triggers when "apt" process executes (detects runtime package manager usage)
	policy := &storage.Policy{
		Name:            "Test - Runtime Namespace Label",
		Description:     "Runtime detection test for namespace label scoping",
		Severity:        storage.Severity_HIGH_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_DEPLOYMENT_EVENT,
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

func TestNamespaceLabelRemoval(t *testing.T) {
	testNS := "test-label-removal-ns"
	createNamespaceWithLabels(t, testNS, map[string]string{"team": "backend"})
	defer deleteNamespace(t, testNS)

	conn := centralgrpc.GRPCConnectionToCentral(t)
	policyService := v1.NewPolicyServiceClient(conn)
	alertService := v1.NewAlertServiceClient(conn)

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

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
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
