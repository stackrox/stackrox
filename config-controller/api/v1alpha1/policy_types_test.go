package v1alpha1

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protoconv"
)

const (
	expirationTS = "2006-01-02T15:04:05Z"
)

func TestToProtobuf(t *testing.T) {
	policyCRSpec := SecurityPolicySpec{
		PolicyName:      "This is a test policy",
		Description:     "This is a test description",
		Rationale:       "This is a test rationale",
		Remediation:     "This is a test remediation",
		Categories:      []string{"Security Best Practices"},
		LifecycleStages: []LifecycleStage{"BUILD", "DEPLOY"},
		Exclusions: []Exclusion{
			{
				Name: "Don't alert on deployment collector in namespace stackrox",
				Deployment: Deployment{
					Name: "collector",
					Scope: Scope{
						Namespace: "stackrox",
						Cluster:   "test",
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
		CriteriaLocked:     true,
		MitreVectorsLocked: true,
		IsDefault:          false,
	}

	expectedProto := &storage.Policy{
		Name:            "This is a test policy",
		Description:     "This is a test description",
		Rationale:       "This is a test rationale",
		Remediation:     "This is a test remediation",
		Categories:      []string{"Security Best Practices"},
		PolicyVersion:   "1.1",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD, storage.LifecycleStage_DEPLOY},
		Exclusions: []*storage.Exclusion{
			{
				Name: "Don't alert on deployment collector in namespace stackrox",
				Deployment: &storage.Exclusion_Deployment{
					Name: "collector",
					Scope: &storage.Scope{
						Namespace: "stackrox",
						Cluster:   "test",
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
		CriteriaLocked:     true,
		MitreVectorsLocked: true,
		IsDefault:          false,
	}
	protoPolicy := policyCRSpec.ToProtobuf()
	// Hack: Reset the source field for us to be able to compare
	protoPolicy.Source = storage.PolicySource_IMPERATIVE
	protoassert.Equal(t, expectedProto, protoPolicy, "proto message derived from custom resource not as expected")
}
