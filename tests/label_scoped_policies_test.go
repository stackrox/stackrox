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
	"github.com/stretchr/testify/suite"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultNamespace = "default"
)

type LabelScopedPoliciesSuite struct {
	suite.Suite
	policyService v1.PolicyServiceClient
	alertService  v1.AlertServiceClient
}

func TestLabelScopedPolicies(t *testing.T) {
	suite.Run(t, new(LabelScopedPoliciesSuite))
}

func (s *LabelScopedPoliciesSuite) SetupSuite() {
	conn := centralgrpc.GRPCConnectionToCentral(s.T())
	s.policyService = v1.NewPolicyServiceClient(conn)
	s.alertService = v1.NewAlertServiceClient(conn)
}

// newPrivilegedContainerPolicy creates a privileged container detection policy with namespace label scoping
func (s *LabelScopedPoliciesSuite) newPrivilegedContainerPolicy(name, description, labelKey, labelValue string) *storage.Policy {
	return &storage.Policy{
		Name:            name,
		Description:     description,
		Severity:        storage.Severity_HIGH_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		Categories:      []string{"Test"},
		Scope: []*storage.Scope{
			{
				NamespaceLabel: &storage.Scope_Label{
					Key:   labelKey,
					Value: labelValue,
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
}

// createPolicyWithCleanup creates a policy and registers cleanup
func (s *LabelScopedPoliciesSuite) createPolicyWithCleanup(policy *storage.Policy) *storage.Policy {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	createdPolicy, err := s.policyService.PostPolicy(ctx, &v1.PostPolicyRequest{Policy: policy})
	s.Require().NoError(err)

	s.T().Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		_, _ = s.policyService.DeletePolicy(ctx, &v1.ResourceByID{Id: createdPolicy.GetId()})
	})

	return createdPolicy
}

// waitForViolationAlert waits for alert matching deployment and policy
func (s *LabelScopedPoliciesSuite) waitForViolationAlert(deploymentName, policyName string, expectedCount int) {
	query := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, deploymentName).
		AddStrings(search.PolicyName, policyName).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(s.T(), s.alertService, &v1.ListAlertsRequest{Query: query.Query()}, expectedCount)
}

func (s *LabelScopedPoliciesSuite) TestNamespaceLabelPolicyScoping() {
	t := s.T()
	backendNS := fmt.Sprintf("test-backend-%d", rand.IntN(10000))
	frontendNS := fmt.Sprintf("test-frontend-%d", rand.IntN(10000))

	createNamespaceWithLabels(t, backendNS, map[string]string{"team": "backend"})
	defer deleteNamespace(t, backendNS)

	createNamespaceWithLabels(t, frontendNS, map[string]string{"team": "frontend"})
	defer deleteNamespace(t, frontendNS)

	policy := s.newPrivilegedContainerPolicy(
		"Test - Namespace Label Backend",
		"Test namespace label scoping",
		"team",
		"backend",
	)
	createdPolicy := s.createPolicyWithCleanup(policy)

	backendDeployment := fmt.Sprintf("test-ns-backend-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", backendDeployment, 1, backendNS, true)
	defer teardownDeploymentWithoutCheck(t, backendDeployment, backendNS)

	s.waitForViolationAlert(backendDeployment, createdPolicy.GetName(), 1)
	t.Logf("Alert appeared for deployment in namespace with team=backend")

	frontendDeployment := fmt.Sprintf("test-ns-frontend-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", frontendDeployment, 1, frontendNS, true)
	defer teardownDeploymentWithoutCheck(t, frontendDeployment, frontendNS)

	s.waitForViolationAlert(frontendDeployment, createdPolicy.GetName(), 0)
	t.Logf("No alert for deployment in namespace with team=frontend")
}

func (s *LabelScopedPoliciesSuite) TestPolicyDryRunWithNamespaceLabel() {
	t := s.T()
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

	policy := s.newPrivilegedContainerPolicy(
		"Test - Dry Run Namespace Label",
		"Dry run test for namespace label scoping",
		"team",
		"backend",
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	resp, err := s.policyService.DryRunPolicy(ctx, policy)
	cancel()
	s.Require().NoError(err)

	alerts := resp.GetAlerts()
	s.Require().Len(alerts, 1, "Expected 1 alert for namespace label policy (only backend deployment)")
	t.Logf("Dry run with namespace label policy: %d alert (expected 1)", len(alerts))
}

func (s *LabelScopedPoliciesSuite) TestRuntimeDetectionWithNamespaceLabels() {
	t := s.T()
	backendNS := fmt.Sprintf("test-backend-%d", rand.IntN(10000))
	frontendNS := fmt.Sprintf("test-frontend-%d", rand.IntN(10000))

	createNamespaceWithLabels(t, backendNS, map[string]string{"team": "backend"})
	defer deleteNamespace(t, backendNS)

	createNamespaceWithLabels(t, frontendNS, map[string]string{"team": "frontend"})
	defer deleteNamespace(t, frontendNS)

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
	createdPolicy := s.createPolicyWithCleanup(policy)

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
	s.waitForViolationAlert(backendDeployment, createdPolicy.GetName(), 1)
	t.Logf("Runtime alert appeared for deployment in namespace with team=backend")

	execInDeployment(t, client, frontendDeployment, frontendNS, "apt", "--help")
	s.waitForViolationAlert(frontendDeployment, createdPolicy.GetName(), 0)
	t.Logf("No runtime alert for deployment in namespace with team=frontend")
}

func (s *LabelScopedPoliciesSuite) TestNamespaceLabelRemoval() {
	t := s.T()
	testNS := "test-label-removal-ns"
	createNamespaceWithLabels(t, testNS, map[string]string{"team": "backend"})
	defer deleteNamespace(t, testNS)

	policy := s.newPrivilegedContainerPolicy(
		"Test - Label Removal Namespace",
		"Test namespace label removal",
		"team",
		"backend",
	)
	createdPolicy := s.createPolicyWithCleanup(policy)

	deployment1 := fmt.Sprintf("test-ns-removal-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", deployment1, 1, testNS, true)
	defer teardownDeploymentWithoutCheck(t, deployment1, testNS)

	s.waitForViolationAlert(deployment1, createdPolicy.GetName(), 1)
	t.Logf("Alert appeared for deployment in namespace with team=backend")

	// Remove namespace labels by replacing with empty label set
	client := createK8sClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ns, err := client.CoreV1().Namespaces().Get(ctx, testNS, metaV1.GetOptions{})
	s.Require().NoError(err)
	ns.Labels = map[string]string{}

	_, err = client.CoreV1().Namespaces().Update(ctx, ns, metaV1.UpdateOptions{})
	s.Require().NoError(err)

	deployment2 := fmt.Sprintf("test-ns-removal-%d", rand.IntN(10000))
	setupDeploymentWithReplicasInNamespace(t, "quay.io/rhacs-eng/qa-multi-arch-nginx:latest", deployment2, 1, testNS, true)
	defer teardownDeploymentWithoutCheck(t, deployment2, testNS)

	s.waitForViolationAlert(deployment2, createdPolicy.GetName(), 0)
	t.Logf("No alert for deployment after namespace labels removed entirely")
}
