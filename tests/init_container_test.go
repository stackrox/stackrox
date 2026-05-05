//go:build test_e2e

package tests

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/suite"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	busyboxLatest = "quay.io/rhacs-eng/qa-multi-arch-busybox:latest"
	nginxLatest   = "quay.io/rhacs-eng/qa-multi-arch-nginx:latest"
	nginxTagged   = "quay.io/rhacs-eng/qa-multi-arch:nginx-1.21.1"
)

type InitContainerSuite struct {
	suite.Suite
	deploymentService v1.DeploymentServiceClient
	policyService     v1.PolicyServiceClient
	alertService      v1.AlertServiceClient
}

func TestInitContainers(t *testing.T) {
	suite.Run(t, new(InitContainerSuite))
}

func (s *InitContainerSuite) SetupSuite() {
	conn := centralgrpc.GRPCConnectionToCentral(s.T())
	s.deploymentService = v1.NewDeploymentServiceClient(conn)
	s.policyService = v1.NewPolicyServiceClient(conn)
	s.alertService = v1.NewAlertServiceClient(conn)

	// Skip if init container support is not enabled on Central
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ffService := v1.NewFeatureFlagServiceClient(conn)
	flags, err := ffService.GetFeatureFlags(ctx, &v1.Empty{})
	s.Require().NoError(err)

	for _, f := range flags.GetFeatureFlags() {
		if f.GetEnvVar() == features.InitContainerSupport.EnvVar() {
			if !f.GetEnabled() {
				s.T().Skip("ROX_INIT_CONTAINER_SUPPORT is not enabled, skipping init container tests")
			}
			return
		}
	}
	s.T().Skip("ROX_INIT_CONTAINER_SUPPORT feature flag not found")
}

func (s *InitContainerSuite) createDeploymentWithInitContainers(name, namespace string, initImages []string, mainImage string) {
	t := s.T()
	client := createK8sClient(t)

	pullPolicy := coreV1.PullIfNotPresent
	if policy := os.Getenv("IMAGE_PULL_POLICY_FOR_QUAY_IO"); policy != "" {
		pullPolicy = coreV1.PullPolicy(policy)
	}

	initContainers := make([]coreV1.Container, len(initImages))
	for i, img := range initImages {
		p := pullPolicy
		if !strings.HasPrefix(img, "quay.io/") {
			p = coreV1.PullIfNotPresent
		}
		initContainers[i] = coreV1.Container{
			Name:            fmt.Sprintf("init-%d", i),
			Image:           img,
			Command:         []string{"sh", "-c", "echo init done"},
			ImagePullPolicy: p,
		}
	}

	mainPullPolicy := pullPolicy
	if !strings.HasPrefix(mainImage, "quay.io/") {
		mainPullPolicy = coreV1.PullIfNotPresent
	}

	deployment := &appsV1.Deployment{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"app": name},
		},
		Spec: appsV1.DeploymentSpec{
			Replicas: pointers.Int32(1),
			Selector: &metaV1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: coreV1.PodSpec{
					InitContainers: initContainers,
					Containers:     []coreV1.Container{buildContainer(name, mainImage, mainPullPolicy)},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := client.AppsV1().Deployments(namespace).Create(ctx, deployment, metaV1.CreateOptions{})
	s.Require().NoError(err)
	t.Logf("Created deployment %q with %d init container(s) in namespace %q", name, len(initImages), namespace)
}

// waitForDeploymentWithContainers waits for a deployment to appear in Central with at least
// expectedContainers containers and at least one enriched image.
func (s *InitContainerSuite) waitForDeploymentWithContainers(deploymentName string, expectedContainers int) *storage.Deployment {
	var result *storage.Deployment
	qb := search.NewQueryBuilder().AddExactMatches(search.DeploymentName, deploymentName)

	waitForCondition(s.T(), func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		listResp, err := s.deploymentService.ListDeployments(ctx, &v1.RawQuery{Query: qb.Query()})
		if err != nil || len(listResp.GetDeployments()) == 0 {
			return false
		}

		dep, err := s.deploymentService.GetDeployment(ctx, &v1.ResourceByID{Id: listResp.GetDeployments()[0].GetId()})
		if err != nil || len(dep.GetContainers()) < expectedContainers {
			return false
		}

		for _, c := range dep.GetContainers() {
			if c.GetImage().GetId() != "" {
				result = dep
				return true
			}
		}
		return false
	}, fmt.Sprintf("deployment %s with %d containers", deploymentName, expectedContainers), waitTimeout, time.Second)

	return result
}

func (s *InitContainerSuite) newLatestTagPolicy(name, namespace string) *storage.Policy {
	return &storage.Policy{
		Name:            name,
		Description:     "Test policy for init container filtering",
		Severity:        storage.Severity_HIGH_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		Categories:      []string{"Test"},
		Scope: []*storage.Scope{
			{
				Namespace: namespace,
			},
		},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Image Tag",
						Values: []*storage.PolicyValue{
							{Value: "latest"},
						},
					},
				},
			},
		},
	}
}

func (s *InitContainerSuite) createPolicyWithCleanup(policy *storage.Policy) *storage.Policy {
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

func (s *InitContainerSuite) waitForViolationAlert(deploymentName, policyName string, expectedCount int) {
	query := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, deploymentName).
		AddStrings(search.PolicyName, policyName).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	waitForAlert(s.T(), s.alertService, &v1.ListAlertsRequest{Query: query.Query()}, expectedCount)
}

func (s *InitContainerSuite) TestInitContainerExtraction() {
	t := s.T()
	ns := fmt.Sprintf("init-test-extract-%d", rand.IntN(10000))
	createNamespaceWithLabels(t, ns, nil)
	defer deleteNamespace(t, ns)

	// Deploy with a fully-specified init container to verify field population
	deployName := fmt.Sprintf("init-extract-%d", rand.IntN(10000))
	client := createK8sClient(t)

	pullPolicy := coreV1.PullIfNotPresent
	if policy := os.Getenv("IMAGE_PULL_POLICY_FOR_QUAY_IO"); policy != "" {
		pullPolicy = coreV1.PullPolicy(policy)
	}

	deployment := &appsV1.Deployment{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      deployName,
			Namespace: ns,
			Labels:    map[string]string{"app": deployName},
		},
		Spec: appsV1.DeploymentSpec{
			Replicas: pointers.Int32(1),
			Selector: &metaV1.LabelSelector{
				MatchLabels: map[string]string{"app": deployName},
			},
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{"app": deployName},
				},
				Spec: coreV1.PodSpec{
					InitContainers: []coreV1.Container{
						{
							Name:            "init-setup",
							Image:           busyboxLatest,
							Command:         []string{"sh", "-c", "echo hello > /work/init-output.txt"},
							ImagePullPolicy: pullPolicy,
							Env: []coreV1.EnvVar{
								{Name: "INIT_VAR", Value: "init-value"},
							},
							Resources: coreV1.ResourceRequirements{
								Requests: coreV1.ResourceList{
									coreV1.ResourceCPU:    resource.MustParse("50m"),
									coreV1.ResourceMemory: resource.MustParse("32Mi"),
								},
								Limits: coreV1.ResourceList{
									coreV1.ResourceCPU:    resource.MustParse("100m"),
									coreV1.ResourceMemory: resource.MustParse("64Mi"),
								},
							},
							SecurityContext: &coreV1.SecurityContext{
								ReadOnlyRootFilesystem: pointers.Bool(true),
							},
							VolumeMounts: []coreV1.VolumeMount{
								{Name: "shared-data", MountPath: "/work"},
							},
						},
					},
					Containers: []coreV1.Container{buildContainer(deployName, nginxLatest, pullPolicy)},
					Volumes: []coreV1.Volume{
						{
							Name:         "shared-data",
							VolumeSource: coreV1.VolumeSource{EmptyDir: &coreV1.EmptyDirVolumeSource{}},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := client.AppsV1().Deployments(ns).Create(ctx, deployment, metaV1.CreateOptions{})
	s.Require().NoError(err)
	defer teardownDeploymentWithoutCheck(t, deployName, ns)

	dep := s.waitForDeploymentWithContainers(deployName, 2)
	s.Require().Len(dep.GetContainers(), 2)

	var initContainer, regularContainer *storage.Container
	for _, c := range dep.GetContainers() {
		switch c.GetType() {
		case storage.ContainerType_INIT:
			initContainer = c
		case storage.ContainerType_REGULAR:
			regularContainer = c
		}
	}

	// Verify types and images
	s.Require().NotNil(initContainer, "expected an init container")
	s.Require().NotNil(regularContainer, "expected a regular container")
	s.Contains(initContainer.GetImage().GetName().GetFullName(), "busybox")
	s.Contains(regularContainer.GetImage().GetName().GetFullName(), "nginx")

	// Verify init container field population
	s.Equal("init-setup", initContainer.GetName())
	s.Equal([]string{"sh", "-c", "echo hello > /work/init-output.txt"}, initContainer.GetConfig().GetCommand())

	envVars := initContainer.GetConfig().GetEnv()
	s.Require().NotEmpty(envVars, "init container should have env vars")
	var foundEnvVar bool
	for _, e := range envVars {
		if e.GetKey() == "INIT_VAR" && e.GetValue() == "init-value" {
			foundEnvVar = true
			break
		}
	}
	s.True(foundEnvVar, "expected INIT_VAR=init-value in init container env")

	s.True(initContainer.GetSecurityContext().GetReadOnlyRootFilesystem(), "init container should have readOnlyRootFilesystem")

	s.Greater(initContainer.GetResources().GetCpuCoresRequest(), float32(0), "init container should have CPU request")
	s.Greater(initContainer.GetResources().GetMemoryMbRequest(), float32(0), "init container should have memory request")

	s.Require().NotEmpty(initContainer.GetVolumes(), "init container should have volumes")
	s.Equal("/work", initContainer.GetVolumes()[0].GetDestination())

	t.Logf("Init container extraction and field population verified")
}

func (s *InitContainerSuite) TestMultipleInitContainers() {
	t := s.T()
	ns := fmt.Sprintf("init-test-multi-%d", rand.IntN(10000))
	createNamespaceWithLabels(t, ns, nil)
	defer deleteNamespace(t, ns)

	deployName := fmt.Sprintf("init-multi-%d", rand.IntN(10000))
	s.createDeploymentWithInitContainers(deployName, ns, []string{busyboxLatest, busyboxLatest}, nginxLatest)
	defer teardownDeploymentWithoutCheck(t, deployName, ns)

	dep := s.waitForDeploymentWithContainers(deployName, 3)

	s.Require().Len(dep.GetContainers(), 3)

	var initCount, regularCount int
	for _, c := range dep.GetContainers() {
		switch c.GetType() {
		case storage.ContainerType_INIT:
			initCount++
		case storage.ContainerType_REGULAR:
			regularCount++
		}
	}

	s.Equal(2, initCount, "expected 2 init containers")
	s.Equal(1, regularCount, "expected 1 regular container")
	t.Logf("Multiple init containers verified: %d init, %d regular", initCount, regularCount)
}

func (s *InitContainerSuite) TestPolicyFilteringInitNotViolated() {
	t := s.T()
	ns := fmt.Sprintf("init-test-policy-%d", rand.IntN(10000))
	createNamespaceWithLabels(t, ns, nil)
	defer deleteNamespace(t, ns)

	createdPolicy := s.createPolicyWithCleanup(s.newLatestTagPolicy(
		fmt.Sprintf("Test - Init Not Violated %d", rand.IntN(10000)), ns,
	))

	// Init container uses :latest (should be filtered), regular uses tagged image (no violation)
	deployName := fmt.Sprintf("init-policy-no-alert-%d", rand.IntN(10000))
	s.createDeploymentWithInitContainers(deployName, ns, []string{busyboxLatest}, nginxTagged)
	defer teardownDeploymentWithoutCheck(t, deployName, ns)

	s.waitForDeploymentWithContainers(deployName, 2)

	s.waitForViolationAlert(deployName, createdPolicy.GetName(), 0)
	t.Logf("Verified: init container with :latest tag did not trigger policy violation")
}

func (s *InitContainerSuite) TestPolicyFilteringRegularStillViolated() {
	t := s.T()
	ns := fmt.Sprintf("init-test-policy-reg-%d", rand.IntN(10000))
	createNamespaceWithLabels(t, ns, nil)
	defer deleteNamespace(t, ns)

	createdPolicy := s.createPolicyWithCleanup(s.newLatestTagPolicy(
		fmt.Sprintf("Test - Regular Still Violated %d", rand.IntN(10000)), ns,
	))

	// Both use :latest — regular should trigger violation, init should be filtered
	deployName := fmt.Sprintf("init-policy-alert-%d", rand.IntN(10000))
	s.createDeploymentWithInitContainers(deployName, ns, []string{busyboxLatest}, nginxLatest)
	defer teardownDeploymentWithoutCheck(t, deployName, ns)

	s.waitForDeploymentWithContainers(deployName, 2)

	s.waitForViolationAlert(deployName, createdPolicy.GetName(), 1)
	t.Logf("Verified: regular container with :latest tag triggered policy violation")
}
