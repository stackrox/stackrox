package policyfields

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stretchr/testify/require"
)

// policyWithField builds a minimal policy with a single criteria field.
// The value is arbitrary — these helpers only check field presence.
func policyWithField(fieldName string) *storage.Policy {
	return &storage.Policy{
		PolicySections: []*storage.PolicySection{{
			PolicyGroups: []*storage.PolicyGroup{{
				FieldName: fieldName,
				Values:    []*storage.PolicyValue{{Value: "something"}},
			}},
		}},
	}
}

func TestAlertsOnMissingEnrichment(t *testing.T) {
	require.True(t, AlertsOnMissingEnrichment(policyWithField(fieldnames.UnscannedImage)))
	require.False(t, AlertsOnMissingEnrichment(policyWithField(fieldnames.ImageTag)))
}

func TestContainsImageEnrichmentRequiredFields(t *testing.T) {
	require.True(t, ContainsImageEnrichmentRequiredFields(policyWithField(fieldnames.CVE)))
	require.False(t, ContainsImageEnrichmentRequiredFields(policyWithField(fieldnames.ImageTag)))
}
