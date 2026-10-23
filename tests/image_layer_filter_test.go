//go:build test_e2e

package tests

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stackrox/rox/config-controller/api/v1alpha1"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// TestImageLayerFilterE2E verifies the image-layer evaluation filter's configuration persistence
// and feature-gating plumbing. This e2e test covers config/persistence/plumbing only; it does NOT
// verify runtime SKIP_BASE filtering behavior (base-layer CVEs actually being suppressed).
//
// Behavioral coverage is provided by unit tests (pkg/booleanpolicy/filter/image_layer_test.go and
// central/policy/service/validator_test.go) plus manual end-to-end validation performed during
// development. Behavioral e2e was deferred because it requires ROX_BASE_IMAGE_DETECTION plus a test
// image with populated baseImageInfo (exact recompressed layer-blob digest matching via the podman
// pipeline), which is not reliably reproducible in CI.
func TestImageLayerFilterE2E(t *testing.T) {
	suite.Run(t, new(ImageLayerFilterSuite))
}

type ImageLayerFilterSuite struct {
	suite.Suite
	policyClient v1.PolicyServiceClient
	k8sClient    dynamic.ResourceInterface
	policies     []*storage.Policy
	ctx          context.Context
	cleanupCtx   context.Context
	cancel       func()
}

func (s *ImageLayerFilterSuite) SetupSuite() {
	s.ctx, s.cleanupCtx, s.cancel = testContexts(s.T(), "TestImageLayerFilterE2E", 5*time.Minute)

	conn := centralgrpc.GRPCConnectionToCentral(s.T())
	s.policyClient = v1.NewPolicyServiceClient(conn)

	dynamicClient := dynamic.NewForConfigOrDie(getConfig(s.T()))
	s.k8sClient = dynamicClient.Resource(policyGVR).Namespace("stackrox")
}

func (s *ImageLayerFilterSuite) TearDownSuite() {
	for _, policy := range s.policies {
		s.T().Logf("Deleting policy with name %q", policy.GetName())
		if policy.GetSource() == storage.PolicySource_DECLARATIVE {
			err := s.k8sClient.Delete(s.ctx, policy.GetName(), metav1.DeleteOptions{})
			if err != nil {
				s.T().Logf("Warning: failed to delete CR policy %q: %v", policy.GetName(), err)
			}
		} else {
			_, err := s.policyClient.DeletePolicy(s.ctx, &v1.ResourceByID{Id: policy.GetId()})
			if err != nil {
				s.T().Logf("Warning: failed to delete policy %q: %v", policy.GetName(), err)
			}
		}
	}
	s.cancel()
}

// TestCRWithSkipImageLayers verifies that a SecurityPolicy CR with evaluationFilter.skipImageLayers
// round-trips correctly: CR → Central protobuf policy → retrieved via API, with skip_image_layers preserved.
func (s *ImageLayerFilterSuite) TestCRWithSkipImageLayers() {
	tests := map[string]struct {
		crValue    v1alpha1.SkipImageLayers
		protoValue storage.SkipImageLayers
	}{
		"SKIP_NONE": {crValue: "SKIP_NONE", protoValue: storage.SkipImageLayers_SKIP_NONE},
		"SKIP_BASE": {crValue: "SKIP_BASE", protoValue: storage.SkipImageLayers_SKIP_BASE},
		"SKIP_APP":  {crValue: "SKIP_APP", protoValue: storage.SkipImageLayers_SKIP_APP},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			crName := fmt.Sprintf("test-image-layer-filter-cr-%s-%d", name, rand.IntN(10000))
			k8sPolicy := s.createBasePolicyCR(crName)
			k8sPolicy.Spec.EvaluationFilter = &v1alpha1.EvaluationFilter{
				SkipImageLayers: tc.crValue,
			}

			id := s.createCRAndObserveInCentral(k8sPolicy)
			s.Require().NotEmpty(id)

			policy, err := s.policyClient.GetPolicy(s.ctx, &v1.ResourceByID{Id: id})
			s.Require().NoError(err)
			s.policies = append(s.policies, policy)

			s.Require().NotNil(policy.GetEvaluationFilter())
			s.Require().Equal(
				tc.protoValue,
				policy.GetEvaluationFilter().GetSkipImageLayers(),
				"SkipImageLayers should round-trip through CR conversion and Central storage",
			)
		})
	}
}

// TestGRPCPolicyWithSkipImageLayers verifies that a policy created via the gRPC PolicyService
// with EvaluationFilter.SkipImageLayers = SKIP_BASE is read back unchanged.
func (s *ImageLayerFilterSuite) TestGRPCPolicyWithSkipImageLayers() {
	tests := map[string]struct {
		skipValue storage.SkipImageLayers
	}{
		"SKIP_NONE": {skipValue: storage.SkipImageLayers_SKIP_NONE},
		"SKIP_BASE": {skipValue: storage.SkipImageLayers_SKIP_BASE},
		"SKIP_APP":  {skipValue: storage.SkipImageLayers_SKIP_APP},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			policyName := fmt.Sprintf("Test - Image Layer Filter gRPC %s %d", name, rand.IntN(10000))
			policy := &storage.Policy{
				Name:            policyName,
				Description:     "Test policy for image layer filter via gRPC",
				Severity:        storage.Severity_MEDIUM_SEVERITY,
				LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
				Categories:      []string{"Test"},
				EvaluationFilter: &storage.EvaluationFilter{
					SkipImageLayers: tc.skipValue,
				},
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: "Days Since CVE Was First Discovered In Image",
								Values: []*storage.PolicyValue{
									{Value: "5"},
								},
							},
						},
					},
				},
			}

			createdPolicy, err := s.policyClient.PostPolicy(s.ctx, &v1.PostPolicyRequest{Policy: policy})
			s.Require().NoError(err)
			s.policies = append(s.policies, createdPolicy)

			storedPolicy, err := s.policyClient.GetPolicy(s.ctx, &v1.ResourceByID{Id: createdPolicy.GetId()})
			s.Require().NoError(err)
			s.Require().NotNil(storedPolicy.GetEvaluationFilter())
			s.Require().Equal(
				tc.skipValue,
				storedPolicy.GetEvaluationFilter().GetSkipImageLayers(),
				"SkipImageLayers should persist through gRPC create and retrieve",
			)
		})
	}
}

// createBasePolicyCR creates a minimal SecurityPolicy CR for testing, mirroring createBasePolicyStruct
// from policy_as_code_test.go.
func (s *ImageLayerFilterSuite) createBasePolicyCR(name string) *v1alpha1.SecurityPolicy {
	k8sPolicy := &v1alpha1.SecurityPolicy{
		Spec: v1alpha1.SecurityPolicySpec{
			PolicyName:      name,
			Description:     "Test policy for image layer filter",
			Categories:      []string{"Test"},
			Severity:        storage.Severity_MEDIUM_SEVERITY.String(),
			LifecycleStages: []v1alpha1.LifecycleStage{"DEPLOY"},
			PolicySections: []v1alpha1.PolicySection{
				{
					SectionName: "Section 1",
					PolicyGroups: []v1alpha1.PolicyGroup{
						{
							FieldName: "Days Since CVE Was First Discovered In Image",
							Values: []v1alpha1.PolicyValue{
								{Value: "5"},
							},
						},
					},
				},
			},
		},
	}
	k8sPolicy.SetName(name)
	k8sPolicy.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "config.stackrox.io",
		Version: "v1alpha1",
		Kind:    "SecurityPolicy",
	})
	k8sPolicy.SetLabels(map[string]string{
		"test": "image-layer-filter",
	})

	return k8sPolicy
}

// createCRAndObserveInCentral creates a SecurityPolicy CR and waits for it to appear in Central,
// mirroring the implementation from policy_as_code_test.go.
func (s *ImageLayerFilterSuite) createCRAndObserveInCentral(policyCR *v1alpha1.SecurityPolicy) string {
	_, err := s.k8sClient.Create(s.ctx, s.toUnstructured(policyCR), metav1.CreateOptions{})
	s.Require().NoError(err)

	var policyID string
	s.Require().EventuallyWithT(func(collect *assert.CollectT) {
		resp, err := s.policyClient.ListPolicies(s.ctx, &v1.RawQuery{})
		if err != nil {
			collect.Errorf("Failed to list policies: %s", err.Error())
			return
		}
		for _, p := range resp.GetPolicies() {
			if p.GetName() == policyCR.Spec.PolicyName {
				policyID = p.GetId()
				break
			}
		}
		assert.NotEmpty(collect, policyID, "policy %q should appear in Central", policyCR.Spec.PolicyName)
	}, time.Second*30, time.Millisecond*30)
	return policyID
}

// toUnstructured converts a typed object to unstructured for the dynamic client.
func (s *ImageLayerFilterSuite) toUnstructured(i interface{}) *unstructured.Unstructured {
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(i)
	s.Require().NoError(err)
	return &unstructured.Unstructured{Object: m}
}
