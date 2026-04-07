package validation

import (
	"testing"

	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stretchr/testify/assert"
)

func makeEntityScope(rules ...*apiV2.EntityScopeRule) *apiV2.EntityScope {
	return &apiV2.EntityScope{Rules: rules}
}

func makeRule(entity apiV2.ScopeEntity, field apiV2.ScopeField, values ...*apiV2.RuleValue) *apiV2.EntityScopeRule {
	return &apiV2.EntityScopeRule{Entity: entity, Field: field, Values: values}
}

func exactValue(v string) *apiV2.RuleValue {
	return &apiV2.RuleValue{Value: v, MatchType: apiV2.MatchType_EXACT}
}

func regexValue(v string) *apiV2.RuleValue {
	return &apiV2.RuleValue{Value: v, MatchType: apiV2.MatchType_REGEX}
}

func TestValidateEntityScope(t *testing.T) {
	cases := map[string]struct {
		scope       *apiV2.EntityScope
		expectError bool
		errContains string
	}{
		"nil entity scope": {
			scope:       nil,
			expectError: true,
			errContains: "either a collection scope with a valid collection ID or a non-nil entity scope",
		},
		"empty rules": {
			scope:       makeEntityScope(),
			expectError: false,
		},
		"valid deployment name rule": {
			scope: makeEntityScope(
				makeRule(apiV2.ScopeEntity_SCOPE_ENTITY_DEPLOYMENT, apiV2.ScopeField_FIELD_NAME, exactValue("frontend")),
			),
			expectError: false,
		},
		"valid namespace label rule": {
			scope: makeEntityScope(
				makeRule(apiV2.ScopeEntity_SCOPE_ENTITY_NAMESPACE, apiV2.ScopeField_FIELD_LABEL, exactValue("env=prod")),
			),
			expectError: false,
		},
		"valid multiple rules different entities": {
			scope: makeEntityScope(
				makeRule(apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER, apiV2.ScopeField_FIELD_NAME, exactValue("prod")),
				makeRule(apiV2.ScopeEntity_SCOPE_ENTITY_NAMESPACE, apiV2.ScopeField_FIELD_NAME, exactValue("payments")),
			),
			expectError: false,
		},
		"unset entity": {
			scope: makeEntityScope(
				makeRule(apiV2.ScopeEntity_SCOPE_ENTITY_UNSET, apiV2.ScopeField_FIELD_NAME, exactValue("x")),
			),
			expectError: true,
			errContains: "Unexpected entity scope rule:",
		},
		"unset field": {
			scope: makeEntityScope(
				makeRule(apiV2.ScopeEntity_SCOPE_ENTITY_DEPLOYMENT, apiV2.ScopeField_FIELD_UNSET, exactValue("x")),
			),
			expectError: true,
			errContains: "Unexpected entity scope rule with an unset field",
		},
		"duplicate entity+field pair": {
			scope: makeEntityScope(
				makeRule(apiV2.ScopeEntity_SCOPE_ENTITY_DEPLOYMENT, apiV2.ScopeField_FIELD_NAME, exactValue("a")),
				makeRule(apiV2.ScopeEntity_SCOPE_ENTITY_DEPLOYMENT, apiV2.ScopeField_FIELD_NAME, exactValue("b")),
			),
			expectError: true,
			errContains: "Duplicate",
		},
		"no values in rule": {
			scope: makeEntityScope(
				makeRule(apiV2.ScopeEntity_SCOPE_ENTITY_DEPLOYMENT, apiV2.ScopeField_FIELD_NAME),
			),
			expectError: true,
			errContains: "provide at least one matching value",
		},
		"label with regex match type is valid": {
			scope: makeEntityScope(
				makeRule(apiV2.ScopeEntity_SCOPE_ENTITY_DEPLOYMENT, apiV2.ScopeField_FIELD_LABEL, regexValue("env=prod")),
			),
			expectError: false,
		},
		"label value missing equals sign": {
			scope: makeEntityScope(
				makeRule(apiV2.ScopeEntity_SCOPE_ENTITY_DEPLOYMENT, apiV2.ScopeField_FIELD_LABEL, exactValue("noequalssign")),
			),
			expectError: true,
			errContains: "key=value",
		},
		"cluster annotation unsupported": {
			scope: makeEntityScope(
				makeRule(apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER, apiV2.ScopeField_FIELD_ANNOTATION, exactValue("k=v")),
			),
			expectError: true,
			errContains: "Annotation",
		},
		"deployment regex on name is valid": {
			scope: makeEntityScope(
				makeRule(apiV2.ScopeEntity_SCOPE_ENTITY_DEPLOYMENT, apiV2.ScopeField_FIELD_NAME, regexValue("front.*")),
			),
			expectError: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := validateEntityScope(tc.scope)
			if tc.expectError {
				assert.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSchedule(t *testing.T) {
	v := &Validator{}

	var cases = []struct {
		testname    string
		config      *apiV2.ReportConfiguration
		expectError bool
		errContains string
	}{
		{
			testname: "Nil schedule is valid",
			config: &apiV2.ReportConfiguration{
				Schedule: nil,
			},
			expectError: false,
		},
		{
			testname: "UNSET interval type is invalid",
			config: &apiV2.ReportConfiguration{
				Schedule: &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_UNSET,
				},
			},
			expectError: true,
			errContains: "DAILY, WEEKLY, or MONTHLY",
		},
		{
			testname: "Valid daily schedule",
			config: &apiV2.ReportConfiguration{
				Schedule: &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_DAILY,
					Hour:         14,
					Minute:       30,
				},
			},
			expectError: false,
		},
		{
			testname: "Daily schedule with days_of_week is invalid",
			config: &apiV2.ReportConfiguration{
				Schedule: &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_DAILY,
					Hour:         14,
					Minute:       30,
					Interval: &apiV2.ReportSchedule_DaysOfWeek_{
						DaysOfWeek: &apiV2.ReportSchedule_DaysOfWeek{Days: []int32{1}},
					},
				},
			},
			expectError: true,
			errContains: "Daily schedule must not specify",
		},
		{
			testname: "Daily schedule with days_of_month is invalid",
			config: &apiV2.ReportConfiguration{
				Schedule: &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_DAILY,
					Hour:         14,
					Minute:       30,
					Interval: &apiV2.ReportSchedule_DaysOfMonth_{
						DaysOfMonth: &apiV2.ReportSchedule_DaysOfMonth{Days: []int32{1}},
					},
				},
			},
			expectError: true,
			errContains: "Daily schedule must not specify",
		},
		{
			testname: "Valid weekly schedule",
			config: &apiV2.ReportConfiguration{
				Schedule: &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_WEEKLY,
					Hour:         10,
					Minute:       0,
					Interval: &apiV2.ReportSchedule_DaysOfWeek_{
						DaysOfWeek: &apiV2.ReportSchedule_DaysOfWeek{Days: []int32{1, 3}},
					},
				},
			},
			expectError: false,
		},
		{
			testname: "Weekly schedule without days is invalid",
			config: &apiV2.ReportConfiguration{
				Schedule: &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_WEEKLY,
					Hour:         10,
					Minute:       0,
				},
			},
			expectError: true,
			errContains: "days of week",
		},
		{
			testname: "Valid monthly schedule",
			config: &apiV2.ReportConfiguration{
				Schedule: &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_MONTHLY,
					Hour:         8,
					Minute:       45,
					Interval: &apiV2.ReportSchedule_DaysOfMonth_{
						DaysOfMonth: &apiV2.ReportSchedule_DaysOfMonth{Days: []int32{1}},
					},
				},
			},
			expectError: false,
		},
	}

	for _, c := range cases {
		t.Run(c.testname, func(t *testing.T) {
			err := v.validateSchedule(c.config)
			if c.expectError {
				assert.Error(t, err)
				if c.errContains != "" {
					assert.Contains(t, err.Error(), c.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateReportFiltersQuery(t *testing.T) {
	v := &Validator{}
	baseFilters := func(query string) *apiV2.ReportConfiguration {
		return &apiV2.ReportConfiguration{
			Filter: &apiV2.ReportConfiguration_VulnReportFilters{
				VulnReportFilters: &apiV2.VulnerabilityReportFilters{
					ImageTypes: []apiV2.VulnerabilityReportFilters_ImageType{apiV2.VulnerabilityReportFilters_DEPLOYED},
					CvesSince:  &apiV2.VulnerabilityReportFilters_AllVuln{AllVuln: true},
					Query:      query,
				},
			},
		}
	}

	cases := map[string]struct {
		query       string
		expectError bool
	}{
		"empty query is valid":                      {query: "", expectError: false},
		"valid CVE query":                           {query: "CVE:CVE-2024-1234", expectError: false},
		"valid severity query":                      {query: "Severity:CRITICAL", expectError: false},
		"valid compound query":                      {query: "Severity:CRITICAL+CVE:CVE-2024-1234", expectError: false},
		"bare value without field label is invalid": {query: "nofieldvalue", expectError: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := v.validateReportFilters(baseFilters(tc.query))
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "Invalid query")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
