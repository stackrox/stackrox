//go:build test

package testmetrics

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse_FoundAndMissing(t *testing.T) {
	text := strings.Join([]string{
		"some_counter_total 2",
		"other_counter_total 1",
		`labeled_total{status="ok"} 7`,
	}, "\n")

	queries := []Query{
		{Name: "some_counter_total"},
		{Name: "other_counter_total"},
		{Name: "labeled_total", LabelFilter: `status="ok"`},
		{Name: "labeled_total", LabelFilter: `status="err"`},
		{Name: "absent_total"},
	}
	m := Parse(text, queries)

	assertFound := func(q Query, expected float64) {
		t.Helper()
		v := m[Key(q)]
		require.True(t, v.Found, "expected %s to be found", Key(q))
		require.Equal(t, expected, v.Val, "wrong value for %s", Key(q))
	}
	assertMissing := func(q Query) {
		t.Helper()
		v := m[Key(q)]
		require.False(t, v.Found, "expected %s to be absent", Key(q))
	}

	assertFound(queries[0], 2)
	assertFound(queries[1], 1)
	assertFound(queries[2], 7)
	assertMissing(queries[3])
	assertMissing(queries[4])
}

func TestParse_EmptyInput(t *testing.T) {
	queries := []Query{
		{Name: "foo_total"},
		{Name: "bar_total"},
	}
	m := Parse("", queries)

	for _, q := range queries {
		v := m[Key(q)]
		require.False(t, v.Found, "expected %s absent on empty input", Key(q))
		require.Equal(t, float64(0), v.Val)
	}
}

func TestKey(t *testing.T) {
	require.Equal(t, "foo_total", Key(Query{Name: "foo_total"}))
	require.Equal(t, `foo_total{status="ok"}`, Key(Query{Name: "foo_total", LabelFilter: `status="ok"`}))
}

func TestValuesEqual(t *testing.T) {
	a := map[string]Value{"x": {Val: 1, Found: true}}
	b := map[string]Value{"x": {Val: 1, Found: true}}
	require.True(t, ValuesEqual(a, b))

	c := map[string]Value{"x": {Val: 2, Found: true}}
	require.False(t, ValuesEqual(a, c))

	d := map[string]Value{"y": {Val: 1, Found: true}}
	require.False(t, ValuesEqual(a, d))
}
