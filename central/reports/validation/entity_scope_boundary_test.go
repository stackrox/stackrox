package validation

import (
	"testing"

	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/require"
)

func TestValidateEntityScopeRejectsUnsupportedRulesAndValues(t *testing.T) {
	tests := map[string]struct {
		entity apiV2.ScopeEntity
		field  apiV2.ScopeField
		value  *apiV2.RuleValue
	}{
		"unsupported cluster ID": {
			entity: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
			field:  apiV2.ScopeField_FIELD_ID,
			value:  &apiV2.RuleValue{Value: "cluster-id"},
		},
		"unknown entity": {
			entity: apiV2.ScopeEntity(99),
			field:  apiV2.ScopeField_FIELD_NAME,
			value:  &apiV2.RuleValue{Value: "prod"},
		},
		"unknown field": {
			entity: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
			field:  apiV2.ScopeField(99),
			value:  &apiV2.RuleValue{Value: "prod"},
		},
		"unknown match type": {
			entity: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
			field:  apiV2.ScopeField_FIELD_NAME,
			value:  &apiV2.RuleValue{Value: "prod", MatchType: apiV2.MatchType(99)},
		},
		"nil value": {
			entity: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
			field:  apiV2.ScopeField_FIELD_NAME,
		},
		"empty scalar value": {
			entity: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
			field:  apiV2.ScopeField_FIELD_NAME,
			value:  &apiV2.RuleValue{},
		},
		"invalid regex key": {
			entity: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
			field:  apiV2.ScopeField_FIELD_LABEL,
			value:  &apiV2.RuleValue{Value: "[=prod", MatchType: apiV2.MatchType_REGEX},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateEntityScope(&apiV2.EntityScope{Rules: []*apiV2.EntityScopeRule{{
				Entity: tc.entity,
				Field:  tc.field,
				Values: []*apiV2.RuleValue{tc.value},
			}}})
			require.ErrorIs(t, err, errox.InvalidArgs)
		})
	}
}
