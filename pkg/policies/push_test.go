package policies

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stretchr/testify/assert"
)

func TestPush(t *testing.T) {
	t.Skip("Skipping in CI since this is only for local testing / verification")

	policy := &storage.Policy{
		Id:          "c19c0cea-b5df-40c4-80e7-836a1b0785e6",
		Name:        "First pushed policy",
		Description: "This is the first pushed policy",
		Rationale:   "Because",
		Remediation: "Because",
		Categories: []string{
			"Anomalous Activity",
		},
		LifecycleStages: []storage.LifecycleStage{
			storage.LifecycleStage_DEPLOY,
		},
		EventSource:        0,
		Severity:           storage.Severity_MEDIUM_SEVERITY,
		SORTName:           "First pushed policy",
		SORTLifecycleStage: storage.LifecycleStage_DEPLOY.String(),
		PolicyVersion:      "1.1",
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "Policy Section 1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName:       "Disallowed Annotation",
						BooleanOperator: storage.BooleanOperator_OR,
						Negate:          false,
						Values: []*storage.PolicyValue{
							{
								Value: "foo=bar",
							},
						},
					},
				},
			},
		},
	}

	registryConfig := &types.Config{
		// TODO(dhaus): For local testing, add creds here.
		RegistryHostname: "registry-1.docker.io",
	}

	p := NewPusher()

	s, err := p.Push(context.Background(), policy, registryConfig, "daha97/policies")
	assert.NoError(t, err)
	assert.NotEmpty(t, s)
}
