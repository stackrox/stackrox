//go:build test_e2e

package tests

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stackrox/rox/config-controller/api/v1alpha1"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

var policyGVR = schema.GroupVersionResource{
	Group:    "config.stackrox.io",
	Version:  "v1alpha1",
	Resource: "securitypolicies",
}

func TestPolicyAsCode(t *testing.T) {
	suite.Run(t, new(PolicyAsCodeSuite))
}

type PolicyAsCodeSuite struct {
	suite.Suite
	centralClient     v1.PolicyServiceClient
	centralHTTPClient *http.Client
	k8sClient         dynamic.ResourceInterface
	informerfactory   dynamicinformer.DynamicSharedInformerFactory
	informer          informers.GenericInformer
	policies          []*storage.Policy
	ctx               context.Context
	cleanupCtx        context.Context
	cancel            func()
	stopCh            chan struct{}
}

func (pc *PolicyAsCodeSuite) SetupSuite() {
	pc.ctx, pc.cleanupCtx, pc.cancel = testContexts(pc.T(), "TestPolicyAsCode", 15*time.Minute)

	conn := centralgrpc.GRPCConnectionToCentral(pc.T())
	pc.centralClient = v1.NewPolicyServiceClient(conn)
	pc.centralHTTPClient = centralgrpc.HTTPClientForCentral(pc.T())

	dynamicClient := dynamic.NewForConfigOrDie(getConfig(pc.T()))
	pc.k8sClient = dynamicClient.Resource(policyGVR).Namespace("stackrox")

	pc.informerfactory = dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamicClient, time.Hour, "stackrox", func(opts *metav1.ListOptions) {
		opts.LabelSelector = "test=policy-as-code"
	})
	pc.informer = pc.informerfactory.ForResource(policyGVR)

	pc.stopCh = make(chan struct{})
	pc.informerfactory.Start(pc.stopCh)
	pc.informerfactory.WaitForCacheSync(pc.stopCh)
}

func (pc *PolicyAsCodeSuite) TestPolicyAsCode() {
	policy := pc.createPolicyInCentral()
	pc.policies = append(pc.policies, policy)
	k8sPolicy := pc.saveAsCustomResource(policy)
	k8sPolicy = pc.createPolicyInK8s(k8sPolicy)
	// Make sure the ID from Central is used to ensure controller didn't create a duplicate
	pc.checkPolicyIsDeclarative(policy.Id)
	pc.updateCRandObserveInCentral(k8sPolicy, policy.Id)
	pc.deleteCRandObserveInCentral(k8sPolicy, policy.Id)
}

func (pc *PolicyAsCodeSuite) createPolicyInCentral() *storage.Policy {
	policyName := "This is a test policy"
	log.Infof("Adding policy with name \"%s\"", policyName)
	policy, err := pc.centralClient.PostPolicy(pc.ctx, &v1.PostPolicyRequest{
		Policy: &storage.Policy{
			Name:            policyName,
			Description:     "This is a description",
			Categories:      []string{"Vulnerability Management"},
			Severity:        storage.Severity_MEDIUM_SEVERITY,
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
			PolicySections: []*storage.PolicySection{{
				SectionName: "Section 1",
				PolicyGroups: []*storage.PolicyGroup{{
					FieldName: "Days Since CVE Was First Discovered In Image",
					Values: []*storage.PolicyValue{{
						Value: "5",
					}},
				}},
			}},
		},
	})

	pc.Require().NoError(err)
	return policy
}

func (pc *PolicyAsCodeSuite) saveAsCustomResource(policy *storage.Policy) *v1alpha1.SecurityPolicy {
	req := map[string][]string{"ids": {policy.Id}}
	jsonReq, err := json.Marshal(req)
	pc.Require().NoError(err)

	resp, err := pc.centralHTTPClient.Post("/api/policy/custom-resource/save-as-zip", "application/json", bytes.NewBuffer(jsonReq))
	pc.Require().NoError(err)
	defer utils.IgnoreError(resp.Body.Close)
	pc.Require().Equal("application/zip", resp.Header.Get("content-type"), "Unexpected content type")

	// Load the zip file
	buff := bytes.NewBuffer([]byte{})
	size, err := io.Copy(buff, resp.Body)
	pc.Require().NoError(err)
	zipReader, err := zip.NewReader(bytes.NewReader(buff.Bytes()), size)
	pc.Require().NoError(err)
	pc.Require().Len(zipReader.File, 1, "Unexpected number of CRs in zip file")

	// Load the yaml file from the zip
	pc.Require().Equal("this-is-a-test-policy.yaml", zipReader.File[0].Name, "Unexpected name in zip file")
	yamlFileBuff := bytes.NewBuffer([]byte{})
	yamlReader, err := zipReader.File[0].Open()
	pc.Require().NoError(err)
	_, err = io.Copy(yamlFileBuff, yamlReader)
	pc.Require().NoError(err)

	// Parse the yaml and do basic validation
	decoder := yaml.NewDecoder(bytes.NewReader(yamlFileBuff.Bytes()))
	u := &unstructured.Unstructured{}
	pc.Require().NoError(decoder.Decode(&u.Object))
	pc.Require().Equal("SecurityPolicy", u.Object["kind"], "Failed to correctly marshal CR")
	return pc.toPolicy(u)
}

func (pc *PolicyAsCodeSuite) createPolicyInK8s(toCreate *v1alpha1.SecurityPolicy) *v1alpha1.SecurityPolicy {
	toCreate.Labels = map[string]string{
		"test": "policy-as-code",
	}

	ch, watchStop := pc.watch()
	defer watchStop()

	_, err := pc.k8sClient.Create(pc.ctx, pc.toUnstructured(toCreate), metav1.CreateOptions{})
	pc.Require().NoError(err)

	message := "status never udpated"
	timer := time.NewTimer(time.Second * 5)
	for {
		select {
		case <-timer.C:
			pc.FailNowf("Policy never marked as accepted", message+": %s", toCreate.Spec.PolicyName)
		case p := <-ch:
			if p.Status.Accepted && p.Status.PolicyId != "" {
				return p
			}
			message = p.Status.Message
		}
	}
}

func (pc *PolicyAsCodeSuite) watch() (chan *v1alpha1.SecurityPolicy, func()) {
	ch := make(chan *v1alpha1.SecurityPolicy)
	reg, err := pc.informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			p := pc.toPolicy(newObj)
			ch <- p
		},
	})
	pc.Require().NoError(err)

	deferFunc := func() {
		pc.Require().NoError(pc.informer.Informer().RemoveEventHandler(reg))
		close(ch)
	}
	return ch, deferFunc
}

func (pc *PolicyAsCodeSuite) toPolicy(i interface{}) *v1alpha1.SecurityPolicy {
	policyCR := v1alpha1.SecurityPolicy{}
	obj := i.(*unstructured.Unstructured)
	pc.Require().NoError(runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &policyCR))
	return &policyCR
}

func (pc *PolicyAsCodeSuite) toUnstructured(i interface{}) *unstructured.Unstructured {
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(i)
	pc.Require().NoError(err)
	return &unstructured.Unstructured{Object: m}
}

func (pc *PolicyAsCodeSuite) checkPolicyIsDeclarative(id string) {
	pc.Require().EventuallyWithT(func(collect *assert.CollectT) {
		policy, err := pc.centralClient.GetPolicy(pc.ctx, &v1.ResourceByID{Id: id})
		if err != nil {
			collect.Errorf("Failed to get policy: %s", err.Error())
		}
		if policy.Source != storage.PolicySource_DECLARATIVE {
			collect.Errorf("Policy %s was not marked as declarative in Central", id)
		}
	}, time.Second*5, time.Millisecond*30)
}

func (pc *PolicyAsCodeSuite) updateCRandObserveInCentral(k8sPolicy *v1alpha1.SecurityPolicy, id string) {
	k8sPolicy.Spec.PolicySections[0].PolicyGroups[0].Values[0].Value = "3"
	_, err := pc.k8sClient.Update(pc.ctx, pc.toUnstructured(k8sPolicy), metav1.UpdateOptions{})
	pc.Require().NoError(err)

	pc.Require().EventuallyWithT(func(collect *assert.CollectT) {
		policy, err := pc.centralClient.GetPolicy(pc.ctx, &v1.ResourceByID{Id: id})
		if err != nil {
			collect.Errorf("Failed to get policy: %s", err.Error())
		}
		criteriaValue := policy.PolicySections[0].PolicyGroups[0].Values[0].Value
		if criteriaValue != "3" {
			collect.Errorf("Policy criteria not updated in Central. Expected 3 but got %s", criteriaValue)
		}
	}, time.Second*5, time.Millisecond*30)
}

func (pc *PolicyAsCodeSuite) deleteCRandObserveInCentral(k8sPolicy *v1alpha1.SecurityPolicy, id string) {
	pc.Require().NoError(pc.k8sClient.Delete(pc.ctx, k8sPolicy.GetName(), metav1.DeleteOptions{}))

	pc.Require().EventuallyWithT(func(collect *assert.CollectT) {
		_, err := pc.centralClient.GetPolicy(pc.ctx, &v1.ResourceByID{Id: id})
		if err != nil {
			statusErr, _ := status.FromError(err)
			if statusErr.Code() != codes.NotFound {
				collect.Errorf("Unexpected error when geting policy: %s", err.Error())
			}
		} else {
			collect.Errorf("Successfully fetched policy %s when it should be deleted", id)
		}
	}, time.Second*5, time.Millisecond*30, "Policy CR deletion not propogated to Central")
}

func (pc *PolicyAsCodeSuite) TearDownSuite() {
	// TODO: Don't double delete
	for _, policy := range pc.policies {
		log.Infof("Deleting policy with name \"%s\"", policy.Name)
		_, err := pc.centralClient.DeletePolicy(pc.ctx, &v1.ResourceByID{
			Id: policy.Id,
		})
		pc.Require().NoError(err)
	}
	// TODO: Remove finalizers if delete fails
	pc.Require().NoError(pc.k8sClient.DeleteCollection(pc.ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: "test=policy-as-code",
	}))
	pc.cancel()
	close(pc.stopCh)
	pc.informerfactory.Shutdown()
}
