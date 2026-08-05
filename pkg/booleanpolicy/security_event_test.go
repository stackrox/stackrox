package booleanpolicy

import (
	"testing"

	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityEventEvaluator(t *testing.T) {
	t.Setenv("ROX_POLICY_REPORTS", "true")
	ResetFieldMetadataSingleton(t)

	obj := augmentedobjs.ConstructSecurityEvent("Kyverno")
	factory := evaluator.MustCreateNewFactory(augmentedobjs.SecurityEventMeta)

	cases := map[string]struct {
		fieldQuery *query.FieldQuery
		expect     bool
	}{
		"exact match": {
			fieldQuery: &query.FieldQuery{Field: "Security Event Source", Values: []string{"Kyverno"}},
			expect:     true,
		},
		"regex match-all": {
			fieldQuery: &query.FieldQuery{Field: "Security Event Source", Values: []string{"r/.*"}},
			expect:     true,
		},
		"regex partial": {
			fieldQuery: &query.FieldQuery{Field: "Security Event Source", Values: []string{"r/Kyv.*"}},
			expect:     true,
		},
		"matchAll": {
			fieldQuery: &query.FieldQuery{Field: "Security Event Source", MatchAll: true},
			expect:     true,
		},
		"no match": {
			fieldQuery: &query.FieldQuery{Field: "Security Event Source", Values: []string{"Falco"}},
			expect:     false,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			q := &query.Query{FieldQueries: []*query.FieldQuery{tc.fieldQuery}}
			eval, err := factory.GenerateEvaluator(q)
			require.NoError(t, err)
			_, matched := eval.Evaluate(obj.Value())
			assert.Equal(t, tc.expect, matched)
		})
	}
}
