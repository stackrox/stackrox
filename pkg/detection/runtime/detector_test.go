package runtime

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/protoassert"
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

func (s *RuntimeDetectorTestSuite) TestNodeFileAccess() {
	node := &storage.Node{
		Name: "test-node-1",
		Id:   "test-node-1",
	}

	type eventWrapper struct {
		access      *storage.FileAccess
		expectAlert bool
	}

	for _, tc := range []struct {
		description string
		policy      *storage.Policy
		events      []eventWrapper
	}{
		{
			description: "Node file open policy with matching event",
			policy: s.getNodeFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Node file open policy with mismatching event (UNLINK)",
			policy: s.getNodeFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: false,
				},
			},
		},
		{
			description: "Node file open policy with mismatching event (/tmp/foo)",
			policy: s.getNodeFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getNodeFileAccessEvent("/tmp/foo", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Node file policy with negated file operation",
			policy: s.getNodeFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, true,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: false, // open is the only event we should ignore
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Node file policy with multiple operations",
			policy: s.getNodeFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: false,
				},
			},
		},
		{
			description: "Node file policy with multiple negated operations",
			policy: s.getNodeFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, true,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: false,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Node file policy with multiple files and single operation",
			policy: s.getNodeFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd", "/etc/shadow",
			),
			events: []eventWrapper{
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Node file policy with multiple files and multiple operations",
			policy: s.getNodeFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, false,
				"/etc/passwd", "/etc/shadow",
			),
			events: []eventWrapper{
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/shadow", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/tmp/foo", storage.FileAccess_CREATE),
					expectAlert: false,
				},
				{
					access:      s.getNodeFileAccessEvent("/tmp/foo", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Node file policy with no operations",
			policy:      s.getNodeFileAccessPolicy("/etc/passwd"),
			events: []eventWrapper{
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      s.getNodeFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
	} {
		s.Run(tc.description, func() {
			policySet := detection.NewPolicySet()
			err := policySet.UpsertPolicy(tc.policy)
			s.NoError(err, "upsert policy should succeed")

			d := NewDetector(policySet)

			for _, event := range tc.events {
				alerts, err := d.DetectForNodeAndFileAccess(node, event.access)
				s.NoError(err)

				if event.expectAlert {
					s.Len(alerts, 1, "expected one alert")

					fileAccessViolation := alerts[0].GetFileAccessViolation()
					s.NotNil(fileAccessViolation, "expected file access violation in alert")
					s.Len(fileAccessViolation.GetAccesses(), 1, "expected one file access in alert")

					protoassert.Equal(s.T(), event.access, fileAccessViolation.GetAccesses()[0])
				} else {
					s.Empty(alerts, "expected no alerts")
				}
			}
		})
	}
}

func (s *RuntimeDetectorTestSuite) getNodeFileAccessEvent(path string, operation storage.FileAccess_Operation) *storage.FileAccess {
	return &storage.FileAccess{
		File:      &storage.FileAccess_File{NodePath: path},
		Operation: operation,
	}
}

func (s *RuntimeDetectorTestSuite) getNodeFileAccessPolicyWithOperations(operations []storage.FileAccess_Operation, negate bool, paths ...string) *storage.Policy {
	var pathValues []*storage.PolicyValue
	for _, path := range paths {
		pathValues = append(pathValues, &storage.PolicyValue{
			Value: path,
		})
	}

	var operationValues []*storage.PolicyValue
	for _, op := range operations {
		operationValues = append(operationValues, &storage.PolicyValue{
			Value: op.String(),
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
						FieldName: fieldnames.NodeFilePath,
						Values:    pathValues,
					},
					{
						FieldName: fieldnames.FileOperation,
						Values:    operationValues,
						Negate:    negate,
					},
				},
			},
		},
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_NODE_EVENT,
	}
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
