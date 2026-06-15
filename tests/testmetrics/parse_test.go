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
	m := parse(text, queries)

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

func TestParse_UsesValueBeforeOptionalTimestamp(t *testing.T) {
	text := strings.Join([]string{
		"some_counter_total 2 1718462400000",
		`labeled_total{status="ok"} 7 1718462400001`,
	}, "\n")

	queries := []Query{
		{Name: "some_counter_total"},
		{Name: "labeled_total", LabelFilter: `status="ok"`},
	}
	m := parse(text, queries)

	require.Equal(t, Value{Val: 2, Found: true}, m[Key(queries[0])])
	require.Equal(t, Value{Val: 7, Found: true}, m[Key(queries[1])])
}

func TestParse_PrefixDoesNotFalseMatch(t *testing.T) {
	text := strings.Join([]string{
		"rox_scan_connections_total 5",
		`rox_scan_connections_errors_total{reason="timeout"} 3`,
	}, "\n")

	queries := []Query{
		{Name: "rox_scan_connections_total"},
		{Name: "rox_scan_connections_errors_total"},
		{Name: "rox_scan_connections"},
	}
	m := parse(text, queries)

	v := m[Key(queries[0])]
	require.True(t, v.Found)
	require.Equal(t, float64(5), v.Val)

	v = m[Key(queries[1])]
	require.True(t, v.Found)
	require.Equal(t, float64(3), v.Val)

	v = m[Key(queries[2])]
	require.False(t, v.Found, "rox_scan_connections should NOT match rox_scan_connections_total")
}

func TestParse_EmptyInput(t *testing.T) {
	queries := []Query{
		{Name: "foo_total"},
		{Name: "bar_total"},
	}
	m := parse("", queries)

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
	require.True(t, valuesEqual(a, b))

	c := map[string]Value{"x": {Val: 2, Found: true}}
	require.False(t, valuesEqual(a, c))

	d := map[string]Value{"y": {Val: 1, Found: true}}
	require.False(t, valuesEqual(a, d))
}
