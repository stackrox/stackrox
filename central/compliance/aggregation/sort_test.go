package aggregation

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
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
	ca := &storage.ComplianceAggregation_AggregationKey{}
	ca.SetScope(storage.ComplianceAggregation_CONTROL)
	ca.SetId("1_20_a")
	ca2 := &storage.ComplianceAggregation_AggregationKey{}
	ca2.SetScope(storage.ComplianceAggregation_DEPLOYMENT)
	ca2.SetId("d1")
	a := []*storage.ComplianceAggregation_AggregationKey{
		ca,
		ca2,
	}
	ca3 := &storage.ComplianceAggregation_AggregationKey{}
	ca3.SetScope(storage.ComplianceAggregation_CONTROL)
	ca3.SetId("1_2_a")
	ca4 := &storage.ComplianceAggregation_AggregationKey{}
	ca4.SetScope(storage.ComplianceAggregation_DEPLOYMENT)
	ca4.SetId("d1")
	b := []*storage.ComplianceAggregation_AggregationKey{
		ca3,
		ca4,
	}
	assert.Equal(t, false, aBeforeB(a, b))
}

func TestComparatorWithoutControlScope(t *testing.T) {
	// A should be before b because it has the same scope, but a lesser ID.
	ca := &storage.ComplianceAggregation_AggregationKey{}
	ca.SetScope(storage.ComplianceAggregation_CLUSTER)
	ca.SetId("c1")
	ca2 := &storage.ComplianceAggregation_AggregationKey{}
	ca2.SetScope(storage.ComplianceAggregation_DEPLOYMENT)
	ca2.SetId("d1")
	a := []*storage.ComplianceAggregation_AggregationKey{
		ca,
		ca2,
	}
	ca3 := &storage.ComplianceAggregation_AggregationKey{}
	ca3.SetScope(storage.ComplianceAggregation_CLUSTER)
	ca3.SetId("c2")
	ca4 := &storage.ComplianceAggregation_AggregationKey{}
	ca4.SetScope(storage.ComplianceAggregation_DEPLOYMENT)
	ca4.SetId("d1")
	b := []*storage.ComplianceAggregation_AggregationKey{
		ca3,
		ca4,
	}
	assert.Equal(t, true, aBeforeB(a, b))
}

func TestDifferentKeyLengthsMatter(t *testing.T) {
	// A should be after b because it is more scoped.
	ca := &storage.ComplianceAggregation_AggregationKey{}
	ca.SetScope(storage.ComplianceAggregation_CLUSTER)
	ca.SetId("c1")
	ca2 := &storage.ComplianceAggregation_AggregationKey{}
	ca2.SetScope(storage.ComplianceAggregation_DEPLOYMENT)
	ca2.SetId("d1")
	ca3 := &storage.ComplianceAggregation_AggregationKey{}
	ca3.SetScope(storage.ComplianceAggregation_NAMESPACE)
	ca3.SetId("n1")
	a := []*storage.ComplianceAggregation_AggregationKey{
		ca,
		ca2,
		ca3,
	}
	ca4 := &storage.ComplianceAggregation_AggregationKey{}
	ca4.SetScope(storage.ComplianceAggregation_CLUSTER)
	ca4.SetId("c2")
	ca5 := &storage.ComplianceAggregation_AggregationKey{}
	ca5.SetScope(storage.ComplianceAggregation_DEPLOYMENT)
	ca5.SetId("d1")
	b := []*storage.ComplianceAggregation_AggregationKey{
		ca4,
		ca5,
	}
	assert.Equal(t, false, aBeforeB(a, b))
}
