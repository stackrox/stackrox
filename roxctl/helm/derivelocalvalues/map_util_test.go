package derivelocalvalues

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_filterMap(t *testing.T) {
	initialMap := map[string]any{
		"k1": 0,
		"k2": 1,
		"k3": 2,
	}
	tests := map[string]struct {
		keys     []string
		expected map[string]any
	}{
		"nil":        {nil, initialMap},
		"delete all": {[]string{"k1", "k2", "k3"}, nil},
		"delete one key": {
			[]string{"k2"},
			map[string]any{"k1": 0, "k3": 2}},
		"delete two keys": {
			[]string{"k2", "k1"},
			map[string]any{"k3": 2}},
		"delete missing keys": {
			[]string{"k2", "k4"},
			map[string]any{"k1": 0, "k3": 2}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual := filterMap(initialMap, test.keys)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func Test_envVarSliceToObj(t *testing.T) {
	obj := envVarSliceToObj(
		[]any{"a", "b", 1, true, nil,
			map[any]any{"name": "name1", "value": "value1"},
			map[any]any{"name": "name2", "value": "value2"},
			map[any]any{"name": "name3", "value": "value3"},
		})
	assert.Equal(t, map[string]any{
		"name1": "value1",
		"name2": "value2",
		"name3": "value3",
	}, obj)

	obj = envVarSliceToObj(
		[]any{"a", "b", 1, true, nil,
			map[any]string{"name": "name1", "value": "value1"},
			map[string]any{"name": "name2", "value": "value2"},
			map[any]any{"a": "name3", "b": "value3"},
		})
	assert.Nil(t, obj)
}
