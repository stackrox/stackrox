package runtime

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestRuntimeDetector(t *testing.T) {
	suite.Run(t, new(RuntimeDetectorTestSuite))
}

type RuntimeDetectorTestSuite struct {
	suite.Suite
}

func (s *RuntimeDetectorTestSuite) TestUpdateSecrets() {
	policySet := detection.NewPolicySet()

	err := policySet.UpsertPolicy(s.getUpdateSecretPolicy())
	s.NoError(err, "upsert policy should succeed")

	d := NewDetector(policySet)

	kubeEvent := s.getKubeEvent(storage.KubernetesEvent_Object_SECRETS, storage.KubernetesEvent_UPDATE, "cluster-id", "namespace", "secret-name", false)
	alerts, err := d.DetectForAuditEvents([]*storage.KubernetesEvent{kubeEvent})

	s.NoError(err)
	s.NotNil(alerts)
	j, _ := json.Marshal(alerts[0])
	fmt.Printf("%+v\n", alerts[0])
	fmt.Println(string(j))
}

func (s *RuntimeDetectorTestSuite) TestCreateConfigMap() {
	policySet := detection.NewPolicySet()

	err := policySet.UpsertPolicy(s.getCreateConfigmapPolicy())
	s.NoError(err, "upsert policy should succeed")

	d := NewDetector(policySet)

	kubeEvent := s.getKubeEvent(storage.KubernetesEvent_Object_CONFIGMAPS, storage.KubernetesEvent_CREATE, "cluster-id", "namespace", "secret-name", true)
	alerts, err := d.DetectForAuditEvents([]*storage.KubernetesEvent{kubeEvent})

	s.NoError(err)
	s.NotNil(alerts)
	s.Equalf(1, len(alerts), "incorrect number of alerts received")
	j, _ := json.Marshal(alerts[0])
	s.NotNil(j)
}

func (s *RuntimeDetectorTestSuite) TestConfigMapPolicyWithRegex() {
	policySet := detection.NewPolicySet()

	cmPolicy := s.getCreateConfigmapPolicy()
	pv := &storage.PolicyValue{}
	pv.SetValue("r/config-.*")
	pg := &storage.PolicyGroup{}
	pg.SetFieldName("Kubernetes Resource Name")
	pg.SetValues([]*storage.PolicyValue{
		pv,
	})
	cmPolicy.GetPolicySections()[0].SetPolicyGroups(append(cmPolicy.GetPolicySections()[0].GetPolicyGroups(), pg))
	err := policySet.UpsertPolicy(cmPolicy)
	s.NoError(err, "upsert policy should succeed")

	d := NewDetector(policySet)

	kubeEvent := s.getKubeEvent(storage.KubernetesEvent_Object_CONFIGMAPS, storage.KubernetesEvent_CREATE, "cluster-id", "namespace", "config-name", true)
	alerts, err := d.DetectForAuditEvents([]*storage.KubernetesEvent{kubeEvent})
	s.NoError(err)
	s.Len(alerts, 1, "incorrect number of alerts received")

	kubeEvent.GetObject().SetName("secret-hello")
	alerts, err = d.DetectForAuditEvents([]*storage.KubernetesEvent{kubeEvent})
	s.NoError(err)
	s.Len(alerts, 0, "incorrect number of alerts received")
}

func (s *RuntimeDetectorTestSuite) getKubeEvent(resource storage.KubernetesEvent_Object_Resource, verb storage.KubernetesEvent_APIVerb, clusterID, namespace, name string, isImpersonated bool) *storage.KubernetesEvent {
	ko := &storage.KubernetesEvent_Object{}
	ko.SetName(name)
	ko.SetResource(resource)
	ko.SetClusterId(clusterID)
	ko.SetNamespace(namespace)
	ku := &storage.KubernetesEvent_User{}
	ku.SetUsername("username")
	ku.SetGroups([]string{"groupA", "groupB"})
	kr := &storage.KubernetesEvent_ResponseStatus{}
	kr.SetStatusCode(200)
	kr.SetReason("cause")
	event := &storage.KubernetesEvent{}
	event.SetId(uuid.NewV4().String())
	event.SetObject(ko)
	event.SetTimestamp(protocompat.TimestampNow())
	event.SetApiVerb(verb)
	event.SetUser(ku)
	event.SetSourceIps([]string{"192.168.1.1", "127.0.0.1"})
	event.SetUserAgent("curl")
	event.SetResponseStatus(kr)
	if isImpersonated {
		ku2 := &storage.KubernetesEvent_User{}
		ku2.SetUsername("impersonatedUser")
		ku2.SetGroups([]string{"groupC"})
		event.SetImpersonatedUser(ku2)
	}
	return event
}

func (s *RuntimeDetectorTestSuite) getUpdateSecretPolicy() *storage.Policy {
	return storage.Policy_builder{
		Id:            "9dc8b85e-7b35-4423-847b-165cd9b92fc7",
		PolicyVersion: "1.1",
		Name:          "Secrets Access",
		Severity:      storage.Severity_LOW_SEVERITY,
		Categories:    []string{"Kubernetes Events"},
		PolicySections: []*storage.PolicySection{
			storage.PolicySection_builder{
				SectionName: "section 1",
				PolicyGroups: []*storage.PolicyGroup{
					storage.PolicyGroup_builder{
						FieldName: "Kubernetes Resource",
						Negate:    false,
						Values:    []*storage.PolicyValue{storage.PolicyValue_builder{Value: "SECRETS"}.Build()},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Kubernetes API Verb",
						Negate:    false,
						Values:    []*storage.PolicyValue{storage.PolicyValue_builder{Value: "UPDATE"}.Build()},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Is Impersonated User",
						Values:    []*storage.PolicyValue{storage.PolicyValue_builder{Value: "false"}.Build()},
					}.Build(),
				},
			}.Build(),
		},
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
	}.Build()
}

func (s *RuntimeDetectorTestSuite) getCreateConfigmapPolicy() *storage.Policy {
	return storage.Policy_builder{
		Id:            "9dc8b85e-7b35-4423-847b-165cd9b92fc7",
		PolicyVersion: "1.1",
		Name:          "Secrets Access",
		Severity:      storage.Severity_LOW_SEVERITY,
		Categories:    []string{"Kubernetes Events"},
		PolicySections: []*storage.PolicySection{
			storage.PolicySection_builder{
				SectionName: "section 1",
				PolicyGroups: []*storage.PolicyGroup{
					storage.PolicyGroup_builder{
						FieldName: "Kubernetes Resource",
						Negate:    false,
						Values:    []*storage.PolicyValue{storage.PolicyValue_builder{Value: "CONFIGMAPS"}.Build()},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Kubernetes API Verb",
						Negate:    false,
						Values:    []*storage.PolicyValue{storage.PolicyValue_builder{Value: "CREATE"}.Build()},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Is Impersonated User",
						Values:    []*storage.PolicyValue{storage.PolicyValue_builder{Value: "true"}.Build()},
					}.Build(),
				},
			}.Build(),
		},
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
	}.Build()
}
