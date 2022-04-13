package k8srbac

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
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
				{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				},
				{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				},
			},
			expected: []*storage.Subject{
				{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				},
			},
		},
		{
			name: "Different names",
			input: []*storage.Subject{
				{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				},
				{
					Name:      "tim",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				},
			},
			expected: []*storage.Subject{
				{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				},
				{
					Name:      "tim",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				},
			},
		},
		{
			name: "Difference namespaces",
			input: []*storage.Subject{
				{
					Name:      "rob",
					Namespace: "default",
					Kind:      storage.SubjectKind_USER,
				},
				{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				},
			},
			expected: []*storage.Subject{
				{
					Name:      "rob",
					Namespace: "default",
					Kind:      storage.SubjectKind_USER,
				},
				{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				},
			},
		},
		{
			name: "Different kinds",
			input: []*storage.Subject{
				{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_GROUP,
				},
				{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				},
			},
			expected: []*storage.Subject{
				{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				},
				{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_GROUP,
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prs := NewSubjectSet()
			prs.Add(c.input...)
			assert.Equal(t, c.expected, prs.ToSlice())
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
				{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				},
				{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				},
			},
			contains: &storage.Subject{
				Name:      "rob",
				Namespace: "stackrox",
				Kind:      storage.SubjectKind_USER,
			},
			expected: true,
		},
		{
			name: "Different name",
			initial: []*storage.Subject{
				{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				},
				{
					Name:      "tim",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				},
			},
			contains: &storage.Subject{
				Name:      "joe",
				Namespace: "stackrox",
				Kind:      storage.SubjectKind_USER,
			},
			expected: false,
		},
		{
			name: "Different namespace",
			initial: []*storage.Subject{
				{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				},
				{
					Name:      "tim",
					Namespace: "default",
					Kind:      storage.SubjectKind_USER,
				},
			},
			contains: &storage.Subject{
				Name:      "tim",
				Namespace: "stackrox",
				Kind:      storage.SubjectKind_USER,
			},
			expected: false,
		},
		{
			name: "Different kind",
			initial: []*storage.Subject{
				{
					Name:      "rob",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_USER,
				},
				{
					Name:      "tim",
					Namespace: "stackrox",
					Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
				},
			},
			contains: &storage.Subject{
				Name:      "tim",
				Namespace: "stackrox",
				Kind:      storage.SubjectKind_USER,
			},
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
