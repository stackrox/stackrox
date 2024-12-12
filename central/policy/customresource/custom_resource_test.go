package customresource

import (
	"bytes"
	_ "embed"
	"strings"
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
	converted, err := generateCustomResource(policy)
	require.NoError(t, err)

	assert.YAMLEq(t, templateFile, converted)
}

func getTestPolicy() *storage.Policy {
	p := fixtures.GetPolicy()
	p.Notifiers = []string{
		"email-notifier-uuid",
	}
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

func TestToDNSSubdomainName(t *testing.T) {
	tests := []struct {
		description string
		input       string
		expected    string
		prefix      string
	}{
		{
			description: "Valid name, unchanged",
			input:       "valid-name",
			expected:    "valid-name",
		},
		{
			description: "Uppercase converted to lowercase",
			input:       "Valid-Name",
			expected:    "valid-name",
		},
		{
			description: "Spaces replaced by dots",
			input:       "some name with spaces",
			expected:    "some-name-with-spaces",
		},
		{
			description: "Special characters replaced by hyphens",
			input:       "invalid@name#with$.special&characters",
			expected:    "invalid-name-with-special-characters",
		},
		{
			description: "Consecutive dots or hyphens reduced to single hyphen",
			input:       "multiple--dots..and-hyphens",
			expected:    "multiple-dots-and-hyphens",
		},
		{
			description: "Name longer than 253 characters should be truncated",
			input:       strings.Repeat("a", 300),
			expected:    strings.Repeat("a", 253),
		},
		{
			description: "Empty input should return default value",
			input:       "",
			prefix:      "rhacs-",
		},
		{
			description: "All invalid input should return default value",
			input:       "@!@#$%^&*()",
			prefix:      "rhacs-",
		},
		{
			description: "Leading and trailing invalid characters should be trimmed",
			input:       "-leading.trailing.",
			expected:    "leading.trailing",
		},
		{
			description: "A comprehensive test case",
			input:       " 这是一个严肃的 @-@ セキュリティポリシ ",
			prefix:      "rhacs-",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			result := toDNSSubdomainName(test.input)
			if len(test.expected) > 0 {
				assert.Equal(t, test.expected, result, "For input %q, expected %q, but got %q", test.input, test.expected, result)
			}
			if len(test.prefix) > 0 {
				assert.True(t, strings.HasPrefix(result, test.prefix) && len(result) > len(test.prefix))
			}
		})
	}
}

// generateCustomResource generate custom resource in YAML text from a policy
func generateCustomResource(policy *storage.Policy) (string, error) {
	w := &bytes.Buffer{}
	if err := WriteCustomResource(w, ConvertPolicyToCustomResource(policy)); err != nil {
		return "", err
	}
	return w.String(), nil
}
