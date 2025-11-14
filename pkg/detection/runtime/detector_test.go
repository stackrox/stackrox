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
	cmPolicy.PolicySections[0].PolicyGroups = append(cmPolicy.PolicySections[0].PolicyGroups, &storage.PolicyGroup{
		FieldName: "Kubernetes Resource Name",
		Values: []*storage.PolicyValue{
			{
				Value: "r/config-.*",
			},
		},
	})
	err := policySet.UpsertPolicy(cmPolicy)
	s.NoError(err, "upsert policy should succeed")

	d := NewDetector(policySet)

	kubeEvent := s.getKubeEvent(storage.KubernetesEvent_Object_CONFIGMAPS, storage.KubernetesEvent_CREATE, "cluster-id", "namespace", "config-name", true)
	alerts, err := d.DetectForAuditEvents([]*storage.KubernetesEvent{kubeEvent})
	s.NoError(err)
	s.Len(alerts, 1, "incorrect number of alerts received")

	kubeEvent.Object.Name = "secret-hello"
	alerts, err = d.DetectForAuditEvents([]*storage.KubernetesEvent{kubeEvent})
	s.NoError(err)
	s.Len(alerts, 0, "incorrect number of alerts received")
}

func (s *RuntimeDetectorTestSuite) getKubeEvent(resource storage.KubernetesEvent_Object_Resource, verb storage.KubernetesEvent_APIVerb, clusterID, namespace, name string, isImpersonated bool) *storage.KubernetesEvent {
	event := &storage.KubernetesEvent{
		Id: uuid.NewV4().String(),
		Object: &storage.KubernetesEvent_Object{
			Name:      name,
			Resource:  resource,
			ClusterId: clusterID,
			Namespace: namespace,
		},
		Timestamp: protocompat.TimestampNow(),
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
	if isImpersonated {
		event.ImpersonatedUser = &storage.KubernetesEvent_User{
			Username: "impersonatedUser",
			Groups:   []string{"groupC"},
		}
	}
	return event
}

func (s *RuntimeDetectorTestSuite) getUpdateSecretPolicy() *storage.Policy {
	return &storage.Policy{
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
						Values:    []*storage.PolicyValue{{Value: "UPDATE"}},
					},
					{
						FieldName: "Is Impersonated User",
						Values:    []*storage.PolicyValue{{Value: "false"}},
					},
				},
			},
		},
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
	}
}

func (s *RuntimeDetectorTestSuite) getCreateConfigmapPolicy() *storage.Policy {
	return &storage.Policy{
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
					{
						FieldName: "Is Impersonated User",
						Values:    []*storage.PolicyValue{{Value: "true"}},
					},
				},
			},
		},
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
	}
}

func (s *RuntimeDetectorTestSuite) TestNodeFileAccessDetection() {
	policySet := detection.NewPolicySet()

	err := policySet.UpsertPolicy(s.getNodeFileAccessPolicy("/etc/shadow"))
	s.NoError(err, "upsert policy should succeed")

	d := NewDetector(policySet)

	node := &storage.Node{
		Id:   "node-123",
		Name: "test-node",
	}

	fileAccess := &storage.FileAccess{
		File: &storage.FileAccess_File{
			NodePath: "/etc/shadow",
		},
		Operation: storage.FileAccess_OPEN,
		Process: &storage.ProcessIndicator{
			Signal: &storage.ProcessSignal{Name: "cat"},
		},
		Hostname: "test-node",
	}

	alerts, err := d.DetectForNodeAndFileAccess(node, fileAccess)

	s.NoError(err)
	s.Len(alerts, 1, "expected one alert for sensitive file access")
	s.Equal(storage.LifecycleStage_RUNTIME, alerts[0].GetLifecycleStage())
	s.NotNil(alerts[0].GetNode(), "alert should have node information")
	s.Equal("node-123", alerts[0].GetNode().GetId())
	s.Equal("test-node", alerts[0].GetNode().GetName())
	s.NotNil(alerts[0].GetFileAccessViolation(), "alert should have file access violation")
	s.Equal("Sensitive File Access on Node", alerts[0].GetPolicy().GetName())
}

func (s *RuntimeDetectorTestSuite) TestNodeFileAccessNoMatch() {
	policySet := detection.NewPolicySet()

	err := policySet.UpsertPolicy(s.getNodeFileAccessPolicy("/etc/passwd"))
	s.NoError(err, "upsert policy should succeed")

	d := NewDetector(policySet)

	node := &storage.Node{
		Id:   "node-123",
		Name: "test-node",
	}

	// File path that doesn't match the policy
	fileAccess := &storage.FileAccess{
		File: &storage.FileAccess_File{
			NodePath: "/tmp/some-file",
		},
		Operation: storage.FileAccess_OPEN,
		Process: &storage.ProcessIndicator{
			Signal: &storage.ProcessSignal{Name: "cat"},
		},
		Hostname: "test-node",
	}

	alerts, err := d.DetectForNodeAndFileAccess(node, fileAccess)

	s.NoError(err)
	s.Len(alerts, 0, "expected no alerts for non-matching file access")
}

func (s *RuntimeDetectorTestSuite) TestNodeFileAccessDisabledPolicy() {
	policySet := detection.NewPolicySet()

	policy := s.getNodeFileAccessPolicy("/etc/shadow")
	policy.Disabled = true
	err := policySet.UpsertPolicy(policy)
	s.NoError(err, "upsert policy should succeed")

	d := NewDetector(policySet)

	node := &storage.Node{
		Id:   "node-123",
		Name: "test-node",
	}

	fileAccess := &storage.FileAccess{
		File: &storage.FileAccess_File{
			NodePath: "/etc/shadow",
		},
		Operation: storage.FileAccess_OPEN,
		Process: &storage.ProcessIndicator{
			Signal: &storage.ProcessSignal{Name: "cat"},
		},
		Hostname: "test-node",
	}

	alerts, err := d.DetectForNodeAndFileAccess(node, fileAccess)

	s.NoError(err)
	s.Len(alerts, 0, "expected no alerts for disabled policy")
}

func (s *RuntimeDetectorTestSuite) TestNodeFileAccessMultiplePaths() {
	policySet := detection.NewPolicySet()

	// Test policy with multiple file paths
	policy := s.getNodeFileAccessPolicy("/etc/passwd", "/etc/shadow")
	err := policySet.UpsertPolicy(policy)
	s.NoError(err, "upsert policy should succeed")

	d := NewDetector(policySet)

	node := &storage.Node{
		Id:   "node-123",
		Name: "test-node",
	}

	// Test matching /etc/passwd
	fileAccess := &storage.FileAccess{
		File: &storage.FileAccess_File{
			NodePath: "/etc/passwd",
		},
		Operation: storage.FileAccess_OPEN,
		Process: &storage.ProcessIndicator{
			Signal: &storage.ProcessSignal{Name: "cat"},
		},
		Hostname: "test-node",
	}

	alerts, err := d.DetectForNodeAndFileAccess(node, fileAccess)
	s.NoError(err)
	s.Len(alerts, 1, "expected one alert for /etc/passwd")

	// Test matching /etc/shadow
	fileAccess.File.NodePath = "/etc/shadow"
	alerts, err = d.DetectForNodeAndFileAccess(node, fileAccess)
	s.NoError(err)
	s.Len(alerts, 1, "expected one alert for /etc/shadow")

	// Test non-matching file
	fileAccess.File.NodePath = "/etc/hosts"
	alerts, err = d.DetectForNodeAndFileAccess(node, fileAccess)
	s.NoError(err)
	s.Len(alerts, 0, "expected no alerts for non-matching file")
}

func (s *RuntimeDetectorTestSuite) getNodeFileAccessPolicy(paths ...string) *storage.Policy {
	var policyValues []*storage.PolicyValue
	for _, path := range paths {
		policyValues = append(policyValues, &storage.PolicyValue{
			Value: path,
		})
	}

	return &storage.Policy{
		Id:            uuid.NewV4().String(),
		PolicyVersion: "1.1",
		Name:          "Sensitive File Access on Node",
		Severity:      storage.Severity_HIGH_SEVERITY,
		Categories:    []string{"File System"},
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section 1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Node File Path",
						Values:    policyValues,
					},
				},
			},
		},
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_NODE_EVENT,
	}
}
