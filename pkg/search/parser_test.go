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
			pairs: []string{"Deployment Label:label+label", "Annotation:annotation"},
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
