package k8srbac

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

func TestDeduplicatesSubjectsCorrectly(t *testing.T) {
	cases := []struct {
		name     string
		input    []*storage.Subject
		expected []*storage.Subject
	}{
		{
			name: "Same subject twice",
			input: []*storage.Subject{
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
			},
			expected: []*storage.Subject{
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
			},
		},
		{
			name: "Different names",
			input: []*storage.Subject{
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name:      "tim",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
			},
			expected: []*storage.Subject{
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name:      "tim",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
			},
		},
		{
			name: "Difference namespaces",
			input: []*storage.Subject{
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "default",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
			},
			expected: []*storage.Subject{
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "default",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
			},
		},
		{
			name: "Different kinds",
			input: []*storage.Subject{
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_GROUP,
				}.Build(),
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
			},
			expected: []*storage.Subject{
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_GROUP,
				}.Build(),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prs := NewSubjectSet()
			prs.Add(c.input...)
			protoassert.SlicesEqual(t, c.expected, prs.ToSlice())
		})
	}
}

func TestMatchesSubjectContentsCorrectly(t *testing.T) {
	cases := []struct {
		name     string
		initial  []*storage.Subject
		contains *storage.Subject
		expected bool
	}{
		{
			name: "Same subject twice",
			initial: []*storage.Subject{
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
			},
			contains: storage.Subject_builder{
				Name:      "rob",
				Namespace: "stackrox",
				Kind:      storage.SubjectKind_USER,
			}.Build(),
			expected: true,
		},
		{
			name: "Different name",
			initial: []*storage.Subject{
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name:      "tim",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
			},
			contains: storage.Subject_builder{
				Name:      "joe",
				Namespace: "stackrox",
				Kind:      storage.SubjectKind_USER,
			}.Build(),
			expected: false,
		},
		{
			name: "Different namespace",
			initial: []*storage.Subject{
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name:      "tim",
					Namespace: "default",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
			},
			contains: storage.Subject_builder{
				Name:      "tim",
				Namespace: "stackrox",
				Kind:      storage.SubjectKind_USER,
			}.Build(),
			expected: false,
		},
		{
			name: "Different kind",
			initial: []*storage.Subject{
				storage.Subject_builder{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				}.Build(),
				storage.Subject_builder{
					Name:      "tim",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
				}.Build(),
			},
			contains: storage.Subject_builder{
				Name:      "tim",
				Namespace: "stackrox",
				Kind:      storage.SubjectKind_USER,
			}.Build(),
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prs := NewSubjectSet()
			prs.Add(c.initial...)
			assert.Equal(t, c.expected, prs.Contains(c.contains))
		})
	}
}
