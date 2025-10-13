package search

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitQuery(t *testing.T) {
	cases := []struct {
		query string
		pairs []string
	}{
		{
			query: NewQueryBuilder().AddStrings(DeploymentLabel, "label").Query(),
			pairs: []string{"Deployment Label:label"},
		},
		{
			query: NewQueryBuilder().AddStrings(DeploymentLabel, "label,label").Query(),
			pairs: []string{"Deployment Label:label,label"},
		},
		{
			query: NewQueryBuilder().AddStrings(DeploymentLabel, "label+label").Query(),
			pairs: []string{"Deployment Label:label+label"},
		},
		{
			query: NewQueryBuilder().AddStrings(DeploymentLabel, "label+label").AddStrings(DeploymentAnnotation, "annotation").Query(),
			pairs: []string{"Deployment Label:label+label", "Deployment Annotation:annotation"},
		},
		{
			query: NewQueryBuilder().AddStrings(DeploymentLabel, "label+").Query(),
			pairs: []string{"Deployment Label:label+"},
		},
		{
			query: DeploymentLabel.String(),
			pairs: []string{DeploymentLabel.String()},
		},
		{
			query: "Deployment:attempted-alerts-dep-6+Policy:Kubernetes Actions: Exec into Pod",
			pairs: []string{"Deployment:attempted-alerts-dep-6", "Policy:Kubernetes Actions: Exec into Pod"},
		},
		{
			query: "Deployment:attempted-alerts-dep-6+Policy:Kubernetes Actions: Exec into Pod,hello+hi+New Deployment Label:value",
			pairs: []string{"Deployment:attempted-alerts-dep-6", "Policy:Kubernetes Actions: Exec into Pod,hello+hi", "New Deployment Label:value"},
		},
		{
			query: "Deployment:field1,\"field12+some:thing\",field13+Category:\"field2+something\"",
			pairs: []string{"Deployment:field1,\"field12+some:thing\",field13", "Category:\"field2+something\""},
		},
		{
			query: "Deployment:field1,\"field12+some,thi:ng\",field13+Category:\"field2+some,thing\"",
			pairs: []string{"Deployment:field1,\"field12+some,thi:ng\",field13", "Category:\"field2+some,thing\""},
		},
	}

	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			assert.ElementsMatch(t, c.pairs, splitQuery(c.query))
		})
	}
}

func TestSplitCommaSeparateValues(t *testing.T) {
	cases := []struct {
		commaSeparatedVals string
		splitValues        []string
	}{
		{
			commaSeparatedVals: "val1,val2,val3+val4",
			splitValues:        []string{"val1", "val2", "val3+val4"},
		},
		{
			commaSeparatedVals: "val1,val2,val3+val:4+val5",
			splitValues:        []string{"val1", "val2", "val3+val:4+val5"},
		},
		{
			commaSeparatedVals: "val1,val2,\"val3,val:4+val5\",val6",
			splitValues:        []string{"val1", "val2", "\"val3,val:4+val5\"", "val6"},
		},
		{
			commaSeparatedVals: ",val1,val2,val3,",
			splitValues:        []string{"", "val1", "val2", "val3", ""},
		},
		{
			commaSeparatedVals: ",,",
			splitValues:        []string{"", "", ""},
		},
		{
			commaSeparatedVals: ",",
			splitValues:        []string{"", ""},
		},
	}

	for _, c := range cases {
		t.Run(c.commaSeparatedVals, func(t *testing.T) {
			assert.ElementsMatch(t, c.splitValues, splitCommaSeparatedValues(c.commaSeparatedVals))
		})
	}
}

func TestParsePairs(t *testing.T) {
	cases := []struct {
		query      string
		allowEmpty bool
		pairs      []string
		valid      bool
	}{
		{
			query:      "Deployment Label:label",
			allowEmpty: false,
			pairs:      []string{"Deployment Label", "label"},
			valid:      true,
		},
		{
			query:      "Deployment Label:label,label",
			allowEmpty: false,
			pairs:      []string{"Deployment Label", "label,label"},
			valid:      true,
		},
		{
			query:      "  Deployment Label :  label+label  ",
			allowEmpty: false,
			pairs:      []string{"Deployment Label", "label+label"},
			valid:      true,
		},
		{
			query:      "Deployment Label:label+label",
			allowEmpty: false,
			pairs:      []string{"Deployment Label", "label+label"},
			valid:      true,
		},
		{
			query:      "Deployment Label",
			allowEmpty: false,
			pairs:      []string{"", ""},
			valid:      false,
		},
		{
			query:      "Deployment Label",
			allowEmpty: true,
			pairs:      []string{"", ""},
			valid:      false,
		},
		{
			query:      "Deployment Label:",
			allowEmpty: false,
			pairs:      []string{"", ""},
			valid:      false,
		},
		{
			query:      "Deployment Label:",
			allowEmpty: true,
			pairs:      []string{"Deployment Label", WildcardString},
			valid:      true,
		},
		{
			query:      "Deployment Label:attempted-alerts-dep-6+Policy:Kubernetes Actions: Exec into Pod",
			allowEmpty: false,
			pairs:      []string{"Deployment Label", "attempted-alerts-dep-6+Policy:Kubernetes Actions: Exec into Pod"},
			valid:      true,
		},
		{
			query:      "Deployment Label:label1,label2,",
			allowEmpty: true,
			pairs:      []string{"Deployment Label", fmt.Sprintf("label1,label2,%s", WildcardString)},
			valid:      true,
		},
	}

	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			key, value, valid := parsePair(c.query, c.allowEmpty)
			assert.Equal(t, valid, c.valid)
			assert.Equal(t, key, c.pairs[0])
			assert.Equal(t, value, c.pairs[1])
		})
	}
}

func TestValueAndModifierFromString(t *testing.T) {
	cases := []struct {
		value            string
		expectedValue    string
		expectedModifier []QueryModifier
	}{
		{
			value:            "test",
			expectedValue:    "test",
			expectedModifier: nil,
		},
		{
			value:            "\"test\"",
			expectedValue:    "test",
			expectedModifier: []QueryModifier{Equality},
		},
		{
			value:            "\"\"",
			expectedValue:    "",
			expectedModifier: []QueryModifier{Equality},
		},
		{
			value:            "\"",
			expectedValue:    "\"",
			expectedModifier: nil,
		},
		{
			value:            "\"test",
			expectedValue:    "\"test",
			expectedModifier: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			value, modifier := GetValueAndModifiersFromString(c.value)
			assert.Equal(t, c.expectedValue, value)
			assert.Equal(t, c.expectedModifier, modifier)
		})
	}
}
