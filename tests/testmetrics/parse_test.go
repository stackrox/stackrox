//go:build test

package testmetrics

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	testCases := map[string]struct {
		text    string
		queries []Query
		want    map[string]Value
	}{
		"should return found and missing metrics": {
			text: strings.Join([]string{
				"some_counter_total 2",
				"other_counter_total 1",
				`labeled_total{status="ok"} 7`,
			}, "\n"),
			queries: []Query{
				{Name: "some_counter_total"},
				{Name: "other_counter_total"},
				{Name: "labeled_total", LabelFilter: `status="ok"`},
				{Name: "labeled_total", LabelFilter: `status="err"`},
				{Name: "absent_total"},
			},
			want: map[string]Value{
				Key(Query{Name: "some_counter_total"}):                         {Val: 2, Found: true},
				Key(Query{Name: "other_counter_total"}):                        {Val: 1, Found: true},
				Key(Query{Name: "labeled_total", LabelFilter: `status="ok"`}):  {Val: 7, Found: true},
				Key(Query{Name: "labeled_total", LabelFilter: `status="err"`}): {Val: 0, Found: false},
				Key(Query{Name: "absent_total"}):                               {Val: 0, Found: false},
			},
		},
		"should parse value before optional timestamp": {
			text: strings.Join([]string{
				"some_counter_total 2 1718462400000",
				`labeled_total{status="ok"} 7 1718462400001`,
			}, "\n"),
			queries: []Query{
				{Name: "some_counter_total"},
				{Name: "labeled_total", LabelFilter: `status="ok"`},
			},
			want: map[string]Value{
				Key(Query{Name: "some_counter_total"}):                        {Val: 2, Found: true},
				Key(Query{Name: "labeled_total", LabelFilter: `status="ok"`}): {Val: 7, Found: true},
			},
		},
		"should return missing values for empty input": {
			text: "",
			queries: []Query{
				{Name: "foo_total"},
				{Name: "bar_total"},
			},
			want: map[string]Value{
				Key(Query{Name: "foo_total"}): {Val: 0, Found: false},
				Key(Query{Name: "bar_total"}): {Val: 0, Found: false},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, parse(tc.text, tc.queries))
		})
	}
}

func TestParse_PrefixDoesNotFalseMatch(t *testing.T) {
	text := strings.Join([]string{
		"rox_scan_connections_total 5",
		`rox_scan_connections_errors_total{reason="timeout"} 3`,
	}, "\n")

	m := parse(text, []Query{
		{Name: "rox_scan_connections_total"},
		{Name: "rox_scan_connections_errors_total"},
		{Name: "rox_scan_connections"},
	})

	v := m[Key(Query{Name: "rox_scan_connections_total"})]
	require.True(t, v.Found)
	require.Equal(t, float64(5), v.Val)

	v = m[Key(Query{Name: "rox_scan_connections_errors_total"})]
	require.True(t, v.Found)
	require.Equal(t, float64(3), v.Val)

	v = m[Key(Query{Name: "rox_scan_connections"})]
	require.False(t, v.Found, "rox_scan_connections should NOT match rox_scan_connections_total")
}

func TestParse_LabelFilterMatchesExactLabelName(t *testing.T) {
	text := `labeled_total{my_status="ok"} 7`

	m := parse(text, []Query{
		{Name: "labeled_total", LabelFilter: `status="ok"`},
	})

	v := m[Key(Query{Name: "labeled_total", LabelFilter: `status="ok"`})]
	require.False(t, v.Found, `status="ok" should NOT match my_status="ok"`)
}
