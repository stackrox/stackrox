package v1alpha1

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	expirationTS = "2006-01-02T15:04:05Z"
)

var (
	emailNotifierID = uuid.NewV4().String()
	jiraNotifierID  = uuid.NewV4().String()
	clusterID       = uuid.NewV4().String()
)

func TestToProtobuf(t *testing.T) {
	policyCRSpec := SecurityPolicySpec{
		PolicyName:      "This is a test policy",
		Description:     "This is a test description",
		Rationale:       "This is a test rationale",
		Remediation:     "This is a test remediation",
		Categories:      []string{"Security Best Practices"},
		LifecycleStages: []LifecycleStage{"BUILD", "DEPLOY"},
		Notifiers: []string{
			"email-notifier",
		},
		Exclusions: []Exclusion{
			{
				Name: "Don't alert on deployment collector in namespace stackrox",
				Deployment: Deployment{
					Name: "collector",
					Scope: Scope{
						Namespace: "stackrox",
						Cluster:   "test-cluster",
					},
				},
				Expiration: expirationTS,
			},
		},
		Severity:           "LOW_SEVERITY",
		EventSource:        "DEPLOYMENT_EVENT",
		EnforcementActions: []EnforcementAction{"SCALE_TO_ZERO_ENFORCEMENT"},
		PolicySections: []PolicySection{
			{
				SectionName: "Section name",
				PolicyGroups: []PolicyGroup{
					{
						FieldName: "Image Component",
						Values: []PolicyValue{{
							Value: "rpm|microdnf|dnf|yum=",
						}},
					},
				},
			},
		},
		Scope: []Scope{
			{
				Cluster: "test-cluster",
			},
		},
	}

	expectedProto := &storage.Policy{
		Name:            "This is a test policy",
		Description:     "This is a test description",
		Rationale:       "This is a test rationale",
		Remediation:     "This is a test remediation",
		Categories:      []string{"Security Best Practices"},
		PolicyVersion:   "1.1",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD, storage.LifecycleStage_DEPLOY},
		Notifiers: []string{
			emailNotifierID,
		},
		Exclusions: []*storage.Exclusion{
			{
				Name: "Don't alert on deployment collector in namespace stackrox",
				Deployment: &storage.Exclusion_Deployment{
					Name: "collector",
					Scope: &storage.Scope{
						Namespace: "stackrox",
						Cluster:   clusterID,
					},
				},
				Expiration: protoconv.ConvertTimeString(expirationTS),
			},
		},
		Severity:           storage.Severity_LOW_SEVERITY,
		EventSource:        storage.EventSource_DEPLOYMENT_EVENT,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "Section name",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Image Component",
						Values: []*storage.PolicyValue{
							{
								Value: "rpm|microdnf|dnf|yum=",
							},
						},
					},
				},
			},
		},
		Scope: []*storage.Scope{
			{
				Cluster: clusterID,
			},
		},
	}

	notifiers := map[string]string{
		"email-notifier": emailNotifierID,
		"jira-notifier":  jiraNotifierID,
	}
	clusters := map[string]string{
		"test-cluster": clusterID,
	}
	protoPolicy, err := policyCRSpec.ToProtobuf(map[CacheType]map[string]string{
		Notifier: notifiers,
		Cluster:  clusters,
	})
	assert.NoError(t, err, "unexpected error in converting to policy proto")
	// Hack: Reset the source field for us to be able to compare
	protoPolicy.Source = storage.PolicySource_IMPERATIVE
	protoassert.Equal(t, expectedProto, protoPolicy, "proto message derived from custom resource not as expected")
}

func TestConditionUpdates(t *testing.T) {
	startTime := metav1.Now()
	policy := &SecurityPolicy{}
	policy.Status = SecurityPolicyStatus{
		Conditions: SecurityPolicyConditions{
			SecurityPolicyCondition{
				Type:               CentralDataFresh,
				Status:             "False",
				Message:            "",
				LastTransitionTime: startTime,
			},
			SecurityPolicyCondition{
				Type:               PolicyValidated,
				Status:             "False",
				Message:            "",
				LastTransitionTime: startTime,
			},
			SecurityPolicyCondition{
				Type:               AcceptedByCentral,
				Status:             "False",
				Message:            "",
				LastTransitionTime: startTime,
			},
		},
	}
	assert.Equal(t, "False", policy.Status.Conditions.GetCondition(CentralDataFresh).Status)
	assert.Equal(t, false, policy.Status.Conditions.IsCentralDataFresh())
	policy.Status.Conditions.UpdateCondition(SecurityPolicyCondition{
		Type:    CentralDataFresh,
		Status:  "True",
		Message: "Central data updated",
	})
	// Check that the condition was properly updated
	assert.Equal(t, true, policy.Status.Conditions.IsCentralDataFresh())
	newCentralDataFreshCondition := policy.Status.Conditions.GetCondition(CentralDataFresh)
	assert.Equal(t, "Central data updated", newCentralDataFreshCondition.Message)
	assert.Equal(t, "True", newCentralDataFreshCondition.Status)
	assert.NotEqual(t, startTime, newCentralDataFreshCondition.LastTransitionTime)
	// Ensure no other fields were changed
	assert.Equal(t, "False", policy.Status.Conditions.GetCondition(PolicyValidated).Status)
	assert.Equal(t, false, policy.Status.Conditions.IsPolicyValidated())
	assert.Equal(t, "False", policy.Status.Conditions.GetCondition(AcceptedByCentral).Status)
	assert.Equal(t, false, policy.Status.Conditions.IsAcceptedByCentral())
	// Ensure the length of the conditions array is still 3
	assert.Equal(t, 3, len(policy.Status.Conditions))
}
