package search

import (
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
	}

	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			assert.ElementsMatch(t, c.pairs, splitQuery(c.query))
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
