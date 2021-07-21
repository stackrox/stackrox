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
			query: NewQueryBuilder().AddStrings(Label, "label").Query(),
			pairs: []string{"Label:label"},
		},
		{
			query: NewQueryBuilder().AddStrings(Label, "label,label").Query(),
			pairs: []string{"Label:label,label"},
		},
		{
			query: NewQueryBuilder().AddStrings(Label, "label+label").Query(),
			pairs: []string{"Label:label+label"},
		},
		{
			query: NewQueryBuilder().AddStrings(Label, "label+label").AddStrings(Annotation, "annotation").Query(),
			pairs: []string{"Label:label+label", "Annotation:annotation"},
		},
		{
			query: NewQueryBuilder().AddStrings(Label, "label+").Query(),
			pairs: []string{"Label:label+"},
		},
		{
			query: Label.String(),
			pairs: []string{Label.String()},
		},
		{
			query: "Deployment:attempted-alerts-dep-6+Policy:Kubernetes Actions: Exec into Pod",
			pairs: []string{"Deployment:attempted-alerts-dep-6", "Policy:Kubernetes Actions: Exec into Pod"},
		},
		{
			query: "Deployment:attempted-alerts-dep-6+Policy:Kubernetes Actions: Exec into Pod,hello+hi+New Label:value",
			pairs: []string{"Deployment:attempted-alerts-dep-6", "Policy:Kubernetes Actions: Exec into Pod,hello+hi", "New Label:value"},
		},
	}

	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			assert.ElementsMatch(t, c.pairs, splitQuery(c.query))
		})
	}
}
