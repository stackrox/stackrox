package labels

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileSelector_NilMatchesNothing(t *testing.T) {
	t.Parallel()

	sel, err := CompileSelector(nil)
	require.NoError(t, err)

	assert.False(t, sel.Matches(nil))
	assert.False(t, sel.Matches(map[string]string{}))
	assert.False(t, sel.Matches(map[string]string{"foo": "bar"}))
}

func TestCompileSelector_EmptyMatchesEverything(t *testing.T) {
	t.Parallel()

	sel, err := CompileSelector(&storage.LabelSelector{})
	require.NoError(t, err)

	assert.True(t, sel.Matches(nil))
	assert.True(t, sel.Matches(map[string]string{}))
	assert.True(t, sel.Matches(map[string]string{"foo": "bar"}))
}

func TestCompileSelector_Simple(t *testing.T) {
	t.Parallel()
	sel := &storage.LabelSelector{
		MatchLabels: map[string]string{
			"foo": "bar",
			"baz": "qux",
		},
	}

	csel, err := CompileSelector(sel)
	require.NoError(t, err)

	assert.True(t, csel.Matches(map[string]string{
		"foo":  "bar",
		"baz":  "qux",
		"quux": "quuz",
	}))

	assert.False(t, csel.Matches(map[string]string{
		"foo":  "bar",
		"baz":  "quux",
		"quux": "quuz",
	}))
	assert.False(t, csel.Matches(map[string]string{
		"foo": "bar",
	}))
}

func TestCompileSelector_WithRequirements(t *testing.T) {
	t.Parallel()
	sel := &storage.LabelSelector{
		MatchLabels: map[string]string{
			"foo": "bar",
		},
		Requirements: []*storage.LabelSelector_Requirement{
			{
				Key:    "baz",
				Op:     storage.LabelSelector_IN,
				Values: []string{"a", "b", "c", "d"},
			},
			{
				Key:    "qux",
				Op:     storage.LabelSelector_NOT_IN,
				Values: []string{"a", "b", "c"},
			},
			{
				Key:    "baz",
				Op:     storage.LabelSelector_NOT_IN,
				Values: []string{"c", "d", "e"},
			},
			{
				Key: "quux",
				Op:  storage.LabelSelector_EXISTS,
			},
			{
				Key: "quuz",
				Op:  storage.LabelSelector_NOT_EXISTS,
			},
			{
				Key:    "corge",
				Op:     storage.LabelSelector_NOT_IN,
				Values: []string{"a", "b", "c"},
			},
			{
				Key: "corge",
				Op:  storage.LabelSelector_EXISTS,
			},
		},
	}

	csel, err := CompileSelector(sel)
	require.NoError(t, err)

	matchedLabelSets := []map[string]string{
		{
			"foo":     "bar",
			"no_qux":  "bla",
			"baz":     "a",
			"quux":    "something",
			"no_quuz": "bla",
			"corge":   "something_not_a_b_c",
		},
		{
			"foo":     "bar",
			"qux":     "something_not_a_b_c",
			"baz":     "a",
			"quux":    "something",
			"no_quuz": "bla",
			"corge":   "something_not_a_b_c",
		},
	}

	nonMatchedLabelSets := []map[string]string{
		{
			"foo":     "baz", // should be bar
			"no_qux":  "bla",
			"baz":     "a",
			"quux":    "something",
			"no_quuz": "bla",
			"corge":   "something_not_a_b_c",
		},
		{
			"foo":     "bar",
			"no_qux":  "bla",
			"baz":     "e", // should be a, b
			"quux":    "something",
			"no_quuz": "bla",
			"corge":   "something_not_a_b_c",
		},
		{
			"foo":     "bar",
			"qux":     "a", // should not be a, b, c
			"baz":     "a",
			"quux":    "something",
			"no_quuz": "bla",
			"corge":   "something_not_a_b_c",
		},
		{
			"foo":     "bar",
			"no_qux":  "bla",
			"baz":     "c", // should be a, b
			"quux":    "something",
			"no_quuz": "bla",
			"corge":   "something_not_a_b_c",
		},
		{
			"foo":     "bar",
			"no_qux":  "bla",
			"baz":     "a",
			"no_quux": "should_exist",
			"no_quuz": "bla",
			"corge":   "something_not_a_b_c",
		},
		{
			"foo":    "bar",
			"no_qux": "bla",
			"baz":    "a",
			"quux":   "something",
			"quuz":   "should_not_exist",
			"corge":  "something_not_a_b_c",
		},
		{
			"foo":     "bar",
			"no_qux":  "bla",
			"baz":     "a",
			"quux":    "something",
			"no_quuz": "bla",
			"corge":   "a", // should not be a, b, c
		},
		{
			"foo":      "bar",
			"no_qux":   "bla",
			"baz":      "a",
			"quux":     "something",
			"no_quuz":  "bla",
			"no_corge": "should_exist",
		},
	}

	for _, ls := range matchedLabelSets {
		assert.Truef(t, csel.Matches(ls), "label set not matched, but should be matched: %+v", ls)
	}

	for _, ls := range nonMatchedLabelSets {
		assert.Falsef(t, csel.Matches(ls), "label set matched, but should not be matched: %+v", ls)
	}
}
