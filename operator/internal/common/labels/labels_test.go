package labels

import (
	"reflect"
	"testing"
)

func TestMergeLabels(t *testing.T) {
	cases := []struct {
		name      string
		current   map[string]string
		newLabels map[string]string
		expected  map[string]string
		updated   bool
	}{
		{
			name:      "No change",
			current:   map[string]string{"a": "1"},
			newLabels: map[string]string{"a": "1"},
			expected:  map[string]string{"a": "1"},
			updated:   false,
		},

		{
			name:      "Existing label updated",
			current:   map[string]string{"a": "1"},
			newLabels: map[string]string{"a": "2"},
			expected:  map[string]string{"a": "2"},
			updated:   true,
		},
		{
			name:      "Current is nil",
			current:   nil,
			newLabels: map[string]string{"a": "1"},
			expected:  map[string]string{"a": "1"},
			updated:   true,
		},
		{
			name:      "newLabel is nil",
			current:   map[string]string{"a": "1"},
			newLabels: nil,
			expected:  map[string]string{"a": "1"},
			updated:   false,
		},
		{
			name:      "Current is empty",
			current:   map[string]string{},
			newLabels: map[string]string{"a": "1"},
			expected:  map[string]string{"a": "1"},
			updated:   true,
		},
		{
			name:      "newLabel is empty",
			current:   map[string]string{"a": "1"},
			newLabels: map[string]string{},
			expected:  map[string]string{"a": "1"},
			updated:   false,
		},
		{
			name:      "both current and newLabel is nil",
			current:   nil,
			newLabels: nil,
			expected:  map[string]string{},
			updated:   false,
		},
		{
			name:      "current has keys that are not in newLabels",
			current:   map[string]string{"foo": "bar"},
			newLabels: map[string]string{"bar": "qux"},
			expected:  map[string]string{"foo": "bar", "bar": "qux"},
			updated:   true,
		},
		{
			name:      "current has keys that are and are not in newLabels",
			current:   map[string]string{"foo": "bar", "bar": "qux"},
			newLabels: map[string]string{"bar": "snipper"},
			expected:  map[string]string{"foo": "bar", "bar": "snipper"},
			updated:   true,
		},
	}

	for _, x := range cases {
		t.Run(x.name, func(t *testing.T) {
			merged, updated := MergeLabels(x.current, x.newLabels)
			if updated != x.updated {
				t.Errorf("expected updated %v, got %v", x.updated, updated)
			}
			if !reflect.DeepEqual(merged, x.expected) {
				t.Errorf("expected merged %v, got %v", x.updated, merged)
			}
		})
	}

}
