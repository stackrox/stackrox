package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVariablesAPI(t *testing.T) {
	tc := []struct {
		name  string
		v     *Variable
		value string
		err   bool
	}{
		{
			name: "string variable values: sets a non-empty string",
			v: &Variable{
				VariablePayload: VariablePayload{
					ID:    "foo_id",
					Type:  "string",
					Value: "foo",
				},
			},
			value: "foo",
			err:   false,
		},
		{
			name: "string variable values: denies empty string",
			v: &Variable{
				VariablePayload: VariablePayload{
					ID:    "foo_id",
					Type:  "string",
					Value: "foo",
				},
			},
			value: "",
			err:   true,
		},
		{
			name: "string variable value selections: allowed values are used",
			v: &Variable{
				VariablePayload: VariablePayload{
					ID:    "beatles",
					Type:  "string",
					Value: "john",
					Selections: []ValueSelection{
						{
							"vocals",
							"john",
						},
						{
							"bass",
							"paul",
						},
						{
							"drums",
							"ringo",
						},
						{
							"guitar",
							"george",
						},
					},
				},
			},
			value: "john",
			err:   false,
		},
		{
			name: "string variable value selections: denied values are not used",
			v: &Variable{
				VariablePayload: VariablePayload{
					ID:    "beatles",
					Type:  "string",
					Value: "john",
					Selections: []ValueSelection{
						{
							"vocals",
							"john",
						},
						{
							"bass",
							"paul",
						},
						{
							"drums",
							"ringo",
						},
						{
							"guitar",
							"george",
						},
					},
				},
			},
			value: "ringo_deathstarr",
			err:   true,
		},
		{
			name: "bool variable values: true and false values are used",
			v: &Variable{
				VariablePayload: VariablePayload{
					ID:    "bool_test",
					Type:  "bool",
					Value: "true",
				},
			},
			value: "false",
			err:   false,
		},
		{
			name: "bool variable values: nonbool values are not used",
			v: &Variable{
				VariablePayload: VariablePayload{
					ID:    "bool_test",
					Type:  "bool",
					Value: "true",
				},
			},
			value: "xxx",
			err:   true,
		},
		{
			name: "number variable value selections: allowed values are used",
			v: &Variable{
				VariablePayload: VariablePayload{
					ID:    "number_test",
					Type:  "number",
					Value: "42",
					Selections: []ValueSelection{
						{
							"fourty two",
							"42",
						},
						{
							"fourty two times two",
							"84",
						},
					},
				},
			},
			value: "84",
			err:   false,
		},
		{
			name: "number variable value selections: disallowed values are not used",
			v: &Variable{
				VariablePayload: VariablePayload{
					ID:    "number_test",
					Type:  "number",
					Value: "42",
					Selections: []ValueSelection{
						{
							"fourty two",
							"42",
						},
						{
							"fourty two times two",
							"84",
						},
					},
				},
			},
			value: "123",
			err:   true,
		},
	}
	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			oldValue := tt.v.Value
			err := tt.v.SetValue(tt.value)
			if tt.err {
				assert.Error(t, err)
				assert.Equal(t, tt.v.Value, oldValue)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.v.Value, tt.value)
			}
		})
	}
}
