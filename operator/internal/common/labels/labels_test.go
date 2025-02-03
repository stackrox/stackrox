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
