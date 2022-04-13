package blevesearch

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/blevesearch/bleve/search/query"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTimeQueryWithDate(t *testing.T) {
	var cases = []struct {
		value, from, to string
	}{
		{
			value: "10/20/2020",
			from:  "=2020-10-20T00:00:00Z",
			to:    "=2020-10-21T00:00:00Z",
		}, {
			value: "11/20/2020 MST", // MST is UTC-7h
			from:  "=2020-11-20T07:00:00Z",
			to:    "=2020-11-21T07:00:00Z",
		}, {
			value: "11/20/2020 CAT", // CAT is UTC+2h
			from:  "=2020-11-19T22:00:00Z",
			to:    "=2020-11-20T22:00:00Z",
		}, {
			value: "03/31/2021 6:50:01 PM UTC",
			from:  "=2021-03-31T18:50:01Z",
			to:    "=2021-04-01T18:50:01Z",
		}, {
			value: "01/31/2021 6:50:01 PM CAT",
			from:  "=2021-01-31T16:50:01Z",
			to:    "=2021-02-01T16:50:01Z",
		}, {
			value: "==06/15/1969",
			from:  "=1969-06-15T00:00:00Z",
			to:    "=1969-06-15T00:00:00Z",
		}, {
			value: ">10/20/2020",
			from:  "2020-10-20T00:00:00Z",
			to:    "",
		}, {
			value: ">=10/20/2021",
			from:  "=2021-10-20T00:00:00Z",
			to:    "",
		}, {
			value: "<=06/15/2020",
			from:  "",
			to:    "=2020-06-15T00:00:00Z",
		}, {
			value: "<06/15/2020",
			from:  "",
			to:    "2020-06-15T00:00:00Z",
		}, {
			value: ">=03/31/2021 7:01:37 AM UTC",
			from:  "=2021-03-31T07:01:37Z",
			to:    "",
		}, {
			value: "<03/29/2021 12:13:14 PM UTC",
			from:  "",
			to:    "2021-03-29T12:13:14Z",
		},
	}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			t1, incl1, t2, incl2 := makeQueryAndGetRange(t, c.value)

			from := c.from
			inclFrom := strings.HasPrefix(from, "=")
			if inclFrom {
				from = from[1:]
			}

			to := c.to
			inclTo := strings.HasPrefix(to, "=")
			if inclTo {
				to = to[1:]
			}

			assert.Equal(t, from, t1)
			assert.Equal(t, inclFrom, incl1)
			assert.Equal(t, to, t2)
			assert.Equal(t, inclTo, incl2)
		})
	}
}

func makeQueryAndGetRange(t *testing.T, value string) (string, bool, string, bool) {
	q, err := newTimeQuery(v1.SearchCategory_ALERTS, "blah", value)
	assert.NoError(t, err)
	qq, ok := q.(*query.NumericRangeQuery)
	assert.True(t, ok, "Query is not of expected type", q)
	assert.Equal(t, "blah", qq.FieldVal)
	var t1, t2 string
	var incl1, incl2 bool
	if qq.Min != nil {
		t1 = floatToTime(*qq.Min).UTC().Format(time.RFC3339)
		incl1 = *qq.InclusiveMin
	}
	if qq.Max != nil {
		t2 = floatToTime(*qq.Max).UTC().Format(time.RFC3339)
		incl2 = *qq.InclusiveMax
	}
	return t1, incl1, t2, incl2
}

func floatToTime(val float64) time.Time {
	seconds, fraction := math.Modf(val)
	nanos := int64(fraction * 1e9)
	return time.Unix(int64(seconds), nanos)
}

func TestParseDuration(t *testing.T) {
	var cases = []struct {
		value    string
		duration time.Duration
		valid    bool
	}{
		{
			value:    "1",
			duration: 24 * 60 * 60 * time.Second,
			valid:    true,
		},
		{
			value:    "1d",
			duration: 24 * 60 * 60 * time.Second,
			valid:    true,
		},
		{
			value:    "lol",
			duration: time.Second,
			valid:    false,
		},
	}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			duration, valid := parseDuration(c.value)
			require.Equal(t, c.valid, valid)
			assert.Equal(t, c.duration, duration)
		})
	}
}
