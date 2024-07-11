package pgsearch

import (
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeQuery(t *testing.T) {
	fakeNow := timeutil.MustParse(time.RFC3339, "2022-06-24T12:00:00Z")
	fake1DayAgo := fakeNow.Add(-24 * time.Hour)
	tsNow := fakeNow.Format(sqlTimeStampFormat)
	ts1dayLater := fakeNow.Add(24 * time.Hour).Format(sqlTimeStampFormat)
	ts1dayAgo := fakeNow.Add(-24 * time.Hour).Format(sqlTimeStampFormat)
	ts10daysAgo := fakeNow.Add(-10 * 24 * time.Hour).Format(sqlTimeStampFormat)

	cases := []struct {
		value          string
		expectErr      bool
		expectedQuery  string
		expectedValues []interface{}
	}{
		{
			value:          "1",
			expectedQuery:  "blah = $$",
			expectedValues: []interface{}{ts1dayAgo},
		},
		{
			value:          "1d",
			expectedQuery:  "blah = $$",
			expectedValues: []interface{}{ts1dayAgo},
		},
		{
			value:     ">lol",
			expectErr: true,
		},
		{
			value:          ">1",
			expectedQuery:  "blah <= $$",
			expectedValues: []interface{}{ts1dayAgo},
		},
		{
			value:          "1-10",
			expectedQuery:  "blah > $$ and blah < $$",
			expectedValues: []interface{}{ts10daysAgo, ts1dayAgo},
		},
		{
			value:          "1d-10d",
			expectedQuery:  "blah > $$ and blah < $$",
			expectedValues: []interface{}{ts10daysAgo, ts1dayAgo},
		},
		{
			value:          "-1d-10d",
			expectedQuery:  "blah > $$ and blah < $$",
			expectedValues: []interface{}{ts10daysAgo, ts1dayLater},
		},
		{
			value:          "-1d-10d",
			expectedQuery:  "blah > $$ and blah < $$",
			expectedValues: []interface{}{ts10daysAgo, ts1dayLater},
		},
		{
			value:          fmt.Sprintf("tr/%d-%d", fake1DayAgo.UnixMilli(), fakeNow.UnixMilli()),
			expectedQuery:  "blah >= $$ and blah < $$",
			expectedValues: []interface{}{ts1dayAgo, tsNow},
		},
	}

	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			actual, err := newTimeQuery(&queryAndFieldContext{
				qualifiedColumnName: "blah",
				value:               c.value,
				now:                 fakeNow,
			})
			if c.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.expectedQuery, actual.Where.Query)
			assert.Equal(t, c.expectedValues, actual.Where.Values)
		})
	}
}
