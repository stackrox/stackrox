package runtime

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestRuntimeDetector(t *testing.T) {
	suite.Run(t, new(RuntimeDetectorTestSuite))
}

type RuntimeDetectorTestSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func (s *RuntimeDetectorTestSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
}

func (s *RuntimeDetectorTestSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *RuntimeDetectorTestSuite) TestCreateSecrets() {
	s.envIsolator.Setenv(features.K8sAuditLogDetection.EnvVar(), "true")
	if !features.K8sAuditLogDetection.Enabled() {
		s.T().Skipf("%s feature flag not enabled, skipping...", features.K8sAuditLogDetection.Name())
	}

	policySet := detection.NewPolicySet()

	err := policySet.UpsertPolicy(s.getListSecretPolicy())
	s.NoError(err, "upsert policy should succeed")

	d := NewDetector(policySet)

	kubeEvent := s.getKubeEvent(storage.KubernetesEvent_Object_SECRETS, storage.KubernetesEvent_LIST, "cluster-id", "namespace", "secret-name")
	alerts, err := d.DetectForAuditEvents([]*storage.KubernetesEvent{kubeEvent})

	s.NoError(err)
	s.NotNil(alerts)
	j, _ := json.Marshal(alerts[0])
	fmt.Printf("%+v\n", alerts[0])
	fmt.Println(string(j))
}

func (s *RuntimeDetectorTestSuite) TestCreateConfigMap() {
	s.envIsolator.Setenv(features.K8sAuditLogDetection.EnvVar(), "true")
	if !features.K8sAuditLogDetection.Enabled() {
		s.T().Skipf("%s feature flag not enabled, skipping...", features.K8sAuditLogDetection.Name())
	}

	policySet := detection.NewPolicySet()

	err := policySet.UpsertPolicy(s.getCreateConfigmapPolicy())
	s.NoError(err, "upsert policy should succeed")

	d := NewDetector(policySet)

	kubeEvent := s.getKubeEvent(storage.KubernetesEvent_Object_CONFIGMAPS, storage.KubernetesEvent_CREATE, "cluster-id", "namespace", "secret-name")
	alerts, err := d.DetectForAuditEvents([]*storage.KubernetesEvent{kubeEvent})

	s.NoError(err)
	s.NotNil(alerts)
	s.Equalf(1, len(alerts), "incorrect number of alerts received")
	j, _ := json.Marshal(alerts[0])
	s.NotNil(j)
}

func (s *RuntimeDetectorTestSuite) getKubeEvent(resource storage.KubernetesEvent_Object_Resource, verb storage.KubernetesEvent_APIVerb, clusterID, namespace, name string) *storage.KubernetesEvent {
	return &storage.KubernetesEvent{
		Id: uuid.NewV4().String(),
		Object: &storage.KubernetesEvent_Object{
			Name:      name,
			Resource:  resource,
			ClusterId: clusterID,
			Namespace: namespace,
		},
		Timestamp: types.TimestampNow(),
		ApiVerb:   verb,
		User: &storage.KubernetesEvent_User{
			Username: "username",
			Groups:   []string{"groupA", "groupB"},
		},
		SourceIps: []string{"192.168.1.1", "127.0.0.1"},
		UserAgent: "curl",
		ResponseStatus: &storage.KubernetesEvent_ResponseStatus{
			StatusCode: 200,
			Reason:     "cause",
		},
	}
}

func (s *RuntimeDetectorTestSuite) getListSecretPolicy() *storage.Policy {
	return policyversion.MustEnsureConverted(&storage.Policy{
		Id:            "9dc8b85e-7b35-4423-847b-165cd9b92fc7",
		PolicyVersion: "1.1",
		Name:          "Secrets Access",
		Severity:      storage.Severity_LOW_SEVERITY,
		Categories:    []string{"Kubernetes Events"},
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section 1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Kubernetes Resource",
						Negate:    false,
						Values:    []*storage.PolicyValue{{Value: "SECRETS"}},
					},
					{
						FieldName: "Kubernetes API Verb",
						Negate:    false,
						Values:    []*storage.PolicyValue{{Value: "LIST"}},
					},
					/*{
						FieldName:            "Is Impersonated User",
						Negate:               false,
						Values:               []*storage.PolicyValue{{Value: "false"}},
					},*/
				},
			},
		},
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
	})
}

func (s *RuntimeDetectorTestSuite) getCreateConfigmapPolicy() *storage.Policy {
	return policyversion.MustEnsureConverted(&storage.Policy{
		Id:            "9dc8b85e-7b35-4423-847b-165cd9b92fc7",
		PolicyVersion: "1.1",
		Name:          "Secrets Access",
		Severity:      storage.Severity_LOW_SEVERITY,
		Categories:    []string{"Kubernetes Events"},
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section 1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Kubernetes Resource",
						Negate:    false,
						Values:    []*storage.PolicyValue{{Value: "CONFIGMAPS"}},
					},
					{
						FieldName: "Kubernetes API Verb",
						Negate:    false,
						Values:    []*storage.PolicyValue{{Value: "CREATE"}},
					},
					/*{
						FieldName:            "Is Impersonated User",
						Negate:               false,
						Values:               []*storage.PolicyValue{{Value: "false"}},
					},*/
				},
			},
		},
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
	})
}
