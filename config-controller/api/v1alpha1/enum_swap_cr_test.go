package v1alpha1

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	v1 "github.com/stackrox/rox/generated/test/enumswapv1"
	v2 "github.com/stackrox/rox/generated/test/enumswapv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

// These tests cover the SecurityPolicy CR side of swapping the enum type of
// EvaluationFilter.skip_container_types (see pkg/protocompat/enum_swap_test.go for the
// serialized-blob side). The CR path never carries enum numbers: the CR field is a Go
// string and ToProtobuf resolves it through the generated <Enum>_value name map. So the
// swap is invisible to CR authors as long as the enum *value* names are preserved, which
// nesting the enum inside a message does.

const skipInitPolicyYAML = `
apiVersion: config.stackrox.io/v1alpha1
kind: SecurityPolicy
metadata:
  name: skip-init-containers
spec:
  policyName: skip-init-containers
  severity: HIGH_SEVERITY
  categories:
    - Test
  lifecycleStages:
    - DEPLOY
  evaluationFilter:
    skipContainerTypes:
      - INIT
  policySections:
    - policyGroups:
        - fieldName: Image Tag
          values:
            - value: latest
`

func emptyCaches() map[CacheType]map[string]string {
	return map[CacheType]map[string]string{
		Notifier: {},
		Cluster:  {},
	}
}

// TestCREnumSwapAcceptsPlainValueName is the contract an operator-facing CR author sees:
// the YAML says INIT, not the Go identifier and not a number.
func TestCREnumSwapAcceptsPlainValueName(t *testing.T) {
	var policy SecurityPolicy
	require.NoError(t, yaml.Unmarshal([]byte(skipInitPolicyYAML), &policy))
	require.Equal(t, []ContainerType{"INIT"}, policy.Spec.EvaluationFilter.SkipContainerTypes)

	proto, err := policy.Spec.ToProtobuf(emptyCaches())
	require.NoError(t, err)
	assert.Equal(t,
		[]storage.ContainerType{storage.ContainerType_INIT},
		proto.GetEvaluationFilter().GetSkipContainerTypes())
}

// TestCREnumSwapRejectsGoIdentifierSpelling pins that the Go-level spelling is not a CR
// value. ToProtobuf drops names it cannot resolve, so this must not silently become INIT.
func TestCREnumSwapRejectsGoIdentifierSpelling(t *testing.T) {
	spec := minimalSpec(&EvaluationFilter{SkipContainerTypes: []ContainerType{"SkipContainerType_INIT"}})

	proto, err := spec.ToProtobuf(emptyCaches())
	require.NoError(t, err)
	assert.Empty(t, proto.GetEvaluationFilter().GetSkipContainerTypes())
}

// TestCREnumSwapNameLookupIsTypeAgnostic is the core of the question: ToProtobuf resolves
// by name against the generated _value map, and that map is keyed on the enum value name,
// which nesting does not change. "INIT" resolves to the same number under either enum.
func TestCREnumSwapNameLookupIsTypeAgnostic(t *testing.T) {
	beforeVal, beforeFound := v1.ContainerType_value["INIT"]
	afterVal, afterFound := v2.EvaluationFilter_SkipContainerType_value["INIT"]

	assert.True(t, beforeFound)
	assert.True(t, afterFound)
	assert.Equal(t, beforeVal, afterVal)
	assert.EqualValues(t, 1, afterVal)
}

// TestCREnumSwapDropsRegularByName shows the CR path degrades REGULAR more safely than the
// binary wire path does. On the wire, REGULAR(0) silently becomes UNSET(0) because only the
// number is stored. In the CR the lookup is by name, REGULAR is simply absent from the new
// map, and ToProtobuf's `found` guard drops it instead of aliasing it onto UNSET.
func TestCREnumSwapDropsRegularByName(t *testing.T) {
	_, beforeFound := v1.ContainerType_value["REGULAR"]
	assert.True(t, beforeFound, "REGULAR is a valid name under the current enum")

	_, afterFound := v2.EvaluationFilter_SkipContainerType_value["REGULAR"]
	assert.False(t, afterFound, "REGULAR must not resolve under the new enum")

	// The trap this guards against: resolving by number instead of by name would map the
	// old REGULAR onto the new UNSET, preserving the behaviour we are trying to remove.
	assert.Equal(t, v2.EvaluationFilter_UNSET, v2.EvaluationFilter_SkipContainerType(v1.ContainerType_value["REGULAR"]))
}

// TestCREnumSwapAllUnresolvedYieldsEmptyFilter documents an edge in the current conversion:
// when every requested name is dropped, EvaluationFilter is still set, just empty. Worth
// knowing when REGULAR stops being a valid CR value and old CRs are re-applied.
func TestCREnumSwapAllUnresolvedYieldsEmptyFilter(t *testing.T) {
	spec := minimalSpec(&EvaluationFilter{SkipContainerTypes: []ContainerType{"NOT_A_CONTAINER_TYPE"}})

	proto, err := spec.ToProtobuf(emptyCaches())
	require.NoError(t, err)
	assert.NotNil(t, proto.GetEvaluationFilter(), "filter is set even though nothing resolved")
	assert.Empty(t, proto.GetEvaluationFilter().GetSkipContainerTypes())
}

func minimalSpec(filter *EvaluationFilter) SecurityPolicySpec {
	return SecurityPolicySpec{
		PolicyName:      "test-policy",
		Severity:        "HIGH_SEVERITY",
		Categories:      []string{"Test"},
		LifecycleStages: []LifecycleStage{"DEPLOY"},
		PolicySections: []PolicySection{{
			PolicyGroups: []PolicyGroup{{
				FieldName: "Image Tag",
				Values:    []PolicyValue{{Value: "latest"}},
			}},
		}},
		EvaluationFilter: filter,
	}
}
