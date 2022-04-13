package aggregation

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestVersionComparator(t *testing.T) {
	// 1.20 is after 1.2
	assert.Equal(t, 1, versionCompare("1_20", "1_2"))

	// 1.20.a is before 1.20.b
	assert.Equal(t, -1, versionCompare("1_20_a", "1_20_b"))

	// 1.2.a is before 1.20.b
	assert.Equal(t, -1, versionCompare("1_2_a", "1_20_b"))

	// 1.20.b is after 1.2.b
	assert.Equal(t, 1, versionCompare("1_20_b", "1_2_b"))

	// a.20.b is before 1.20.b.iii
	assert.Equal(t, -1, versionCompare("1_20_b", "1_20_b_iii"))

	// Numbers before letters
	assert.Equal(t, -1, versionCompare("1_20_30", "1_20_b"))
}

func TestComparatorWithControlScope(t *testing.T) {
	// A should be after b because it has the same scopes but a greater id.
	a := []*storage.ComplianceAggregation_AggregationKey{
		{
			Scope: storage.ComplianceAggregation_CONTROL,
			Id:    "1_20_a",
		},
		{
			Scope: storage.ComplianceAggregation_DEPLOYMENT,
			Id:    "d1",
		},
	}
	b := []*storage.ComplianceAggregation_AggregationKey{
		{
			Scope: storage.ComplianceAggregation_CONTROL,
			Id:    "1_2_a",
		},
		{
			Scope: storage.ComplianceAggregation_DEPLOYMENT,
			Id:    "d1",
		},
	}
	assert.Equal(t, false, aBeforeB(a, b))
}

func TestComparatorWithoutControlScope(t *testing.T) {
	// A should be before b because it has the same scope, but a lesser ID.
	a := []*storage.ComplianceAggregation_AggregationKey{
		{
			Scope: storage.ComplianceAggregation_CLUSTER,
			Id:    "c1",
		},
		{
			Scope: storage.ComplianceAggregation_DEPLOYMENT,
			Id:    "d1",
		},
	}
	b := []*storage.ComplianceAggregation_AggregationKey{
		{
			Scope: storage.ComplianceAggregation_CLUSTER,
			Id:    "c2",
		},
		{
			Scope: storage.ComplianceAggregation_DEPLOYMENT,
			Id:    "d1",
		},
	}
	assert.Equal(t, true, aBeforeB(a, b))
}

func TestDifferentKeyLengthsMatter(t *testing.T) {
	// A should be after b because it is more scoped.
	a := []*storage.ComplianceAggregation_AggregationKey{
		{
			Scope: storage.ComplianceAggregation_CLUSTER,
			Id:    "c1",
		},
		{
			Scope: storage.ComplianceAggregation_DEPLOYMENT,
			Id:    "d1",
		},
		{
			Scope: storage.ComplianceAggregation_NAMESPACE,
			Id:    "n1",
		},
	}
	b := []*storage.ComplianceAggregation_AggregationKey{
		{
			Scope: storage.ComplianceAggregation_CLUSTER,
			Id:    "c2",
		},
		{
			Scope: storage.ComplianceAggregation_DEPLOYMENT,
			Id:    "d1",
		},
	}
	assert.Equal(t, false, aBeforeB(a, b))
}
