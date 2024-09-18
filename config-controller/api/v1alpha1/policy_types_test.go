package v1alpha1

import (
	"embed"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:embed test.json

var files embed.FS

func TestMarshalJSON(t *testing.T) {

	bytes, err := files.ReadFile("test.json")

	assert.NoError(t, err, "Failed to read policy file")

	policyCRSpec := SecurityPolicySpec{}

	assert.NoError(t, json.Unmarshal(bytes, &policyCRSpec), "Failed to unmarshal policy spec CR JSON")

	expected := SecurityPolicySpec{
		Description:     "This is a test description",
		Rationale:       "This is a test rationale",
		Remediation:     "This is a test remediation",
		Categories:      []string{"Security Best Practices"},
		LifecycleStages: []LifecycleStage{"BUILD", "DEPLOY"},
		Exclusions: []Exclusion{{
			Name: "Don't alert on deployment collector in namespace stackrox",
			Deployment: Deployment{
				Name: "collector",
				Scope: Scope{
					Namespace: "stackrox",
					Cluster:   "test",
				},
			}},
		},
		Severity:           "LOW_SEVERITY",
		EventSource:        "DEPLOYMENT_EVENT",
		EnforcementActions: []EnforcementAction{"SCALE_TO_ZERO_ENFORCEMENT"},
		PolicyVersion:      "1.1",
		PolicySections: []PolicySection{{
			SectionName: "Section name",
			PolicyGroups: []PolicyGroup{{
				FieldName: "Image Component",
				Values: []PolicyValue{{
					Value: "rpm|microdnf|dnf|yum=",
				}},
			}},
		}},
		CriteriaLocked:     true,
		MitreVectorsLocked: true,
		IsDefault:          false,
	}

	assert.Equal(t, expected, policyCRSpec)

	protoPolicy := policyCRSpec.ToProtobuf()

	protoBytes, err := protojson.MarshalOptions{
		Multiline: true,
	}.Marshal(protoPolicy)

	assert.NoError(t, err, "Failed to marshal protobuf")

	assert.Equal(t, string(bytes), strings.ReplaceAll(string(protoBytes), ":  ", ": ")+"\n")
}
