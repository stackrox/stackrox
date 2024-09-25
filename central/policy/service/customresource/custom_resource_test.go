package customresource

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/custom_resource.yaml
var templateFile string

func TestConvertToCR(t *testing.T) {
	policy := getTestPolicy()
	converted, err := GenerateCustomResource(policy)
	require.NoError(t, err)
	fmt.Println(converted)
	assert.YAMLEq(t, templateFile, converted)
}

func getTestPolicy() *storage.Policy {
	p := fixtures.GetPolicy()
	p.MitreAttackVectors = []*storage.Policy_MitreAttackVectors{
		{
			Tactic:     "This is a tactic.",
			Techniques: []string{"technique1", "technique2"},
		},
		{
			Tactic:     "This is another tactic.",
			Techniques: []string{"technique1"},
		},
	}
	p.EnforcementActions = []storage.EnforcementAction{
		storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT,
		storage.EnforcementAction_KILL_POD_ENFORCEMENT,
	}
	p.Exclusions = []*storage.Exclusion{
		{
			Name: "exclusionName1",
			Deployment: &storage.Exclusion_Deployment{
				Name: "deployment1",
				Scope: &storage.Scope{
					Cluster:   "cluster1",
					Namespace: "label1",
					Label: &storage.Scope_Label{
						Key:   "key1",
						Value: "value1",
					},
				},
			},
			Expiration: protocompat.GetProtoTimestampFromSeconds(2334221123),
		},
		{
			Name: "exclusionName2",
			Deployment: &storage.Exclusion_Deployment{
				Name: "deployment2",
				Scope: &storage.Scope{
					Cluster:   "cluster2",
					Namespace: "label2",
					Label: &storage.Scope_Label{
						Key:   "key2",
						Value: "value2",
					},
				},
			},
		},
	}
	return p
}
