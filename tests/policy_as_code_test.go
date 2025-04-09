//go:build test_e2e

package tests

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/config-controller/api/v1alpha1"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/notifiers"
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

var (
	policyGVR = schema.GroupVersionResource{
		Group:    "config.stackrox.io",
		Version:  "v1alpha1",
		Resource: "securitypolicies",
	}
)

const (
	server      = "smtp.mailgun.org"
	user        = "postmaster@sandboxd6576ea8be3c477989eba2c14735d2e6.mailgun.org"
	clusterName = fixtureconsts.ClusterName1
)

func TestPolicyAsCode(t *testing.T) {
	suite.Run(t, new(PolicyAsCodeSuite))
}

type PolicyAsCodeSuite struct {
	suite.Suite
	policyClient      v1.PolicyServiceClient
	notifierClient    v1.NotifierServiceClient
	clusterClient     v1.ClustersServiceClient
	centralHTTPClient *http.Client
	k8sClient         dynamic.ResourceInterface
	informerfactory   dynamicinformer.DynamicSharedInformerFactory
	informer          informers.GenericInformer
	policies          []*storage.Policy
	cluster           *storage.Cluster
	notifier          *storage.Notifier
	ctx               context.Context
	cleanupCtx        context.Context
	cancel            func()
	stopCh            chan struct{}
}

func (pc *PolicyAsCodeSuite) SetupSuite() {
	pc.ctx, pc.cleanupCtx, pc.cancel = testContexts(pc.T(), "TestPolicyAsCode", 15*time.Minute)

	conn := centralgrpc.GRPCConnectionToCentral(pc.T())
	pc.policyClient = v1.NewPolicyServiceClient(conn)
	pc.notifierClient = v1.NewNotifierServiceClient(conn)
	pc.clusterClient = v1.NewClustersServiceClient(conn)
	pc.cluster = pc.createClusterInCentral()
	pc.notifier = pc.createNotifierInCentral()

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

func (pc *PolicyAsCodeSuite) TestSaveAsCRUpdateDelete() {
	policy := pc.createPolicyInCentral()
	pc.policies = append(pc.policies, policy)
	k8sPolicy := pc.saveAsCustomResource(policy)
	k8sPolicy = pc.createPolicyInK8s(k8sPolicy)

	// Make sure the ID from Central is used to ensure controller didn't create a duplicate
	pc.checkPolicyIsDeclarative(policy.Id)
	pc.updateCRandObserveInCentral(k8sPolicy, policy.Id)
	pc.deleteCRandObserveInCentral(k8sPolicy, policy.Id)
}

func createBasePolicyStruct(name string) *v1alpha1.SecurityPolicy {
	k8sPolicy := &v1alpha1.SecurityPolicy{
		Spec: v1alpha1.SecurityPolicySpec{
			PolicyName:      name,
			Description:     "This is a description",
			Categories:      []string{"Vulnerability Management"},
			Severity:        storage.Severity_MEDIUM_SEVERITY.String(),
			LifecycleStages: []v1alpha1.LifecycleStage{"DEPLOY"},
			PolicySections: []v1alpha1.PolicySection{
				{
					SectionName: "Section 1",
					PolicyGroups: []v1alpha1.PolicyGroup{
						{
							FieldName: "Days Since CVE Was First Discovered In Image",
							Values: []v1alpha1.PolicyValue{
								{
									Value: "5",
								},
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
		"test": "policy-as-code",
	})

	return k8sPolicy
}

func (pc *PolicyAsCodeSuite) TestCreateCR() {
	k8sPolicy := createBasePolicyStruct("test-policy-create")
	k8sPolicy.Spec.Notifiers = []string{pc.notifier.GetName()}
	id := pc.createCRAndObserveInCentral(k8sPolicy)
	pc.Require().NotEmpty(id)
	pc.checkPolicyIsDeclarative(id)

	// Add the policy to the policy set so it can be deleted at teardown
	policy, err := pc.policyClient.GetPolicy(pc.ctx, &v1.ResourceByID{
		Id: id,
	})
	pc.Require().NoError(err)
	pc.policies = append(pc.policies, policy)
}

func (pc *PolicyAsCodeSuite) TestClusterIDResolution() {
	k8sPolicy := createBasePolicyStruct("test-cluster-id-resolution")
	k8sPolicy.Spec.Notifiers = []string{pc.notifier.GetName()}
	k8sPolicy.Spec.Scope = []v1alpha1.Scope{{
		Cluster: clusterName,
	}}

	id := pc.createCRAndObserveInCentral(k8sPolicy)
	pc.Require().NotEmpty(id)
	pc.checkPolicyIsDeclarative(id)

	policy, err := pc.policyClient.GetPolicy(pc.ctx, &v1.ResourceByID{
		Id: id,
	})
	pc.Require().NoError(err)
	pc.policies = append(pc.policies, policy)

	pc.Equal(pc.cluster.Id, policy.Scope[0].Cluster)
}

func (pc *PolicyAsCodeSuite) TestCreateDefaultCR() {
	k8sPolicy := createBasePolicyStruct("90-Day Image Age")
	k8sPolicy.SetName("90-day-image-age")
	ch, watchStop := pc.watch(k8sPolicy.GetName())
	defer watchStop()
	_, err := pc.k8sClient.Create(pc.ctx, pc.toUnstructured(k8sPolicy), metav1.CreateOptions{})
	pc.Require().NoError(err)

	message := "status never updated"
	timer := time.NewTimer(time.Second * 5)
	for {
		select {
		case <-timer.C:
			pc.FailNowf("Policy never marked as rejected as duplicate of default policy", message+": %s", k8sPolicy.Spec.PolicyName)
		case p := <-ch:
			if condition := p.Status.Condition.GetCondition(v1alpha1.PolicyValidated); !p.Status.Condition.IsPolicyValidated() && strings.Contains(condition.Message, "existing default policy with the same name") {
				return
			}
			message = p.Status.Condition.GetCondition(v1alpha1.PolicyValidated).Message
		}
	}
}

func (pc *PolicyAsCodeSuite) TestRenameToDefaultCR() {
	k8sPolicy := createBasePolicyStruct("rename-to-default-test")
	ch, watchStop := pc.watch(k8sPolicy.GetName())
	defer watchStop()

	u, err := pc.k8sClient.Create(pc.ctx, pc.toUnstructured(k8sPolicy), metav1.CreateOptions{})
	pc.Require().NoError(err)
	pc.fromUnstructured(u, k8sPolicy)

	message := "status never updated"
	timer := time.NewTimer(time.Second * 5)
	for {
		accepted := false
		select {
		case <-timer.C:
			pc.FailNowf("Policy never marked as accepted", message+": %s", k8sPolicy.Spec.PolicyName)
		case p := <-ch:
			acceptedCondition := p.Status.Condition.GetCondition(v1alpha1.AcceptedByCentral)
			accepted = acceptedCondition.Status == "True"
			message = acceptedCondition.Message
		}

		if accepted {
			break
		}
	}

	k8sPolicy.Spec.PolicyName = "90-Day Image Age"
	pc.Require().EventuallyWithT(func(collect *assert.CollectT) {
		_, err = pc.k8sClient.Update(pc.ctx, pc.toUnstructured(k8sPolicy), metav1.UpdateOptions{})
		assert.NoError(collect, err)
		u, err = pc.k8sClient.Get(pc.ctx, k8sPolicy.GetName(), metav1.GetOptions{})
		pc.fromUnstructured(u, k8sPolicy)
		k8sPolicy.Spec.PolicyName = "90-Day Image Age"
	}, time.Second*5, time.Millisecond*30)

	timer = time.NewTimer(time.Second * 5)
	for {
		var policy *v1alpha1.SecurityPolicy
		select {
		case <-timer.C:
			pc.FailNowf("Policy never marked as duplicate of default policy", message+": %s", k8sPolicy.Spec.PolicyName)
		case p := <-ch:
			policy = p
		}

		if condition := policy.Status.Condition.GetCondition(v1alpha1.PolicyValidated); !policy.Status.Condition.IsPolicyValidated() && strings.Contains(condition.Message, "existing default policy with the same name") {
			break
		}
	}
}

func (pc *PolicyAsCodeSuite) createPolicyInCentral() *storage.Policy {
	policyName := "This is a test policy"
	log.Infof("Adding policy with name \"%s\"", policyName)
	policy, err := pc.policyClient.PostPolicy(pc.ctx, &v1.PostPolicyRequest{
		Policy: &storage.Policy{
			Name:            policyName,
			Description:     "This is a description",
			Categories:      []string{"Vulnerability Management"},
			Notifiers:       []string{pc.notifier.GetId()},
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

func (pc *PolicyAsCodeSuite) createClusterInCentral() *storage.Cluster {
	log.Infof("Adding cluster with name \"%s\"", clusterName)
	cluster, err := pc.clusterClient.PostCluster(pc.ctx, &storage.Cluster{Name: fixtureconsts.ClusterName1, MainImage: "docker.io/stackrox/rox:latest", CentralApiEndpoint: "central.stackrox:443"})
	pc.NoError(err)
	return cluster.Cluster
}

func (pc *PolicyAsCodeSuite) createNotifierInCentral() *storage.Notifier {
	notifierName := "email-notifier"
	log.Infof("Adding notifier with name \"%s\"", notifierName)
	notifier, err := pc.notifierClient.PostNotifier(pc.ctx, &storage.Notifier{
		Name:       notifierName,
		Type:       notifiers.EmailType,
		UiEndpoint: "http://google.com",
		Config: &storage.Notifier_Email{
			Email: &storage.Email{
				Server:                   server,
				Sender:                   user,
				AllowUnauthenticatedSmtp: true,
				DisableTLS:               true,
			},
		},
	})
	pc.Require().NoError(err)
	return notifier
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

	ch, watchStop := pc.watch(toCreate.GetName())
	defer watchStop()

	_, err := pc.k8sClient.Create(pc.ctx, pc.toUnstructured(toCreate), metav1.CreateOptions{})
	pc.Require().NoError(err)

	message := "status never updated"
	timer := time.NewTimer(time.Second * 5)
	for {
		select {
		case <-timer.C:
			pc.FailNowf("Policy never marked as accepted", message+": %s", toCreate.Spec.PolicyName)
		case p := <-ch:
			if p.Status.Condition.IsAcceptedByCentral() && p.Status.PolicyId != "" {
				return p
			}
			message = p.Status.Condition.GetCondition(v1alpha1.AcceptedByCentral).Message
		}
	}
}

func (pc *PolicyAsCodeSuite) watch(name string) (chan *v1alpha1.SecurityPolicy, func()) {
	ch := make(chan *v1alpha1.SecurityPolicy)
	reg, err := pc.informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			p := pc.toPolicy(newObj)
			if p.GetName() == name {
				ch <- p
			}
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

func (pc *PolicyAsCodeSuite) fromUnstructured(u *unstructured.Unstructured, i interface{}) {
	pc.Require().NoError(runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, i))
}

func (pc *PolicyAsCodeSuite) checkPolicyIsDeclarative(id string) {
	pc.Require().EventuallyWithT(func(collect *assert.CollectT) {
		policy, err := pc.policyClient.GetPolicy(pc.ctx, &v1.ResourceByID{Id: id})
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
		policy, err := pc.policyClient.GetPolicy(pc.ctx, &v1.ResourceByID{Id: id})
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
		_, err := pc.policyClient.GetPolicy(pc.ctx, &v1.ResourceByID{Id: id})
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

func (pc *PolicyAsCodeSuite) createCRAndObserveInCentral(policyCR *v1alpha1.SecurityPolicy) string {
	_, err := pc.k8sClient.Create(pc.ctx, pc.toUnstructured(policyCR), metav1.CreateOptions{})
	pc.Require().NoError(err)

	var policyId string
	pc.Require().EventuallyWithT(func(collect *assert.CollectT) {
		resp, err := pc.policyClient.ListPolicies(pc.ctx, &v1.RawQuery{})
		if err != nil {
			collect.Errorf("Failed to list policies: %s", err.Error())
		}
		for _, p := range resp.GetPolicies() {
			if p.GetName() == policyCR.Spec.PolicyName {
				policyId = p.GetId()
				break
			}
		}
		assert.NotEmpty(collect, policyId)
	}, time.Second*5, time.Millisecond*30)
	return policyId
}

func (pc *PolicyAsCodeSuite) TearDownSuite() {
	// TODO: Don't double delete
	for _, policy := range pc.policies {
		log.Infof("Deleting policy with name \"%s\"", policy.Name)
		_, err := pc.policyClient.DeletePolicy(pc.ctx, &v1.ResourceByID{
			Id: policy.Id,
		})
		pc.Require().NoError(err)
	}

	if pc.notifier != nil {
		log.Infof("Deleting notifier with name \"%s\"", pc.notifier.Name)
		_, err := pc.notifierClient.DeleteNotifier(pc.ctx, &v1.DeleteNotifierRequest{
			Id:    pc.notifier.Id,
			Force: true,
		})
		pc.Require().NoError(err)
	}

	if pc.cluster != nil {
		log.Infof("Deleting cluster with name \"%s\"", pc.cluster.Name)
		_, err := pc.clusterClient.DeleteCluster(pc.ctx, &v1.ResourceByID{
			Id: pc.cluster.Id,
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
