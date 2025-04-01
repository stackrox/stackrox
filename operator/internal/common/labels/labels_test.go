package labels

import (
	"reflect"
	"testing"
)

func TestMergeLabels(t *testing.T) {
	tests := map[string]struct {
		current   map[string]string
		newLabels map[string]string
		expected  map[string]string
		updated   bool
	}{
		"current is nil": {
			current:   nil,
			newLabels: map[string]string{"foo": "bar"},
			expected:  map[string]string{"foo": "bar"},
			updated:   true,
		},
		"newLabels is nil": {
			current:   map[string]string{"foo": "bar"},
			newLabels: nil,
			expected:  map[string]string{"foo": "bar"},
			updated:   false,
		},
		"current is empty": {
			current:   map[string]string{},
			newLabels: map[string]string{"foo": "bar"},
			expected:  map[string]string{"foo": "bar"},
			updated:   true,
		},
		"newLabels is empty": {
			current:   map[string]string{"foo": "bar"},
			newLabels: map[string]string{},
			expected:  map[string]string{"foo": "bar"},
			updated:   false,
		},
		"both current and newLabels are nil": {
			current:   nil,
			newLabels: nil,
			expected:  map[string]string{},
			updated:   false,
		},
		"current contains keys that are not in newLabels": {
			current:   map[string]string{"foo": "bar"},
			newLabels: map[string]string{"bar": "qux"},
			expected:  map[string]string{"foo": "bar", "bar": "qux"},
			updated:   true,
		},
		"current contains both keys in newLabels and keys not in newLabels": {
			current:   map[string]string{"foo": "bar", "bar": "qux"},
			newLabels: map[string]string{"bar": "snipper"},
			expected:  map[string]string{"foo": "bar", "bar": "snipper"},
			updated:   true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual, updated := MergeLabels(test.current, test.newLabels)

			if !reflect.DeepEqual(actual, test.expected) {
				t.Errorf("expected labels: %v, got: %v", test.expected, actual)
			}
			if updated != test.updated {
				t.Errorf("expected updated: %v, got: %v", test.updated, updated)
			}
		})
	}
}
