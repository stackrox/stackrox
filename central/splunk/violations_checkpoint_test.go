package splunk

// This file contains tests for splunkCheckpoint behavior.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCheckpointParam(t *testing.T) {
	const (
		now         = "now"
		longTimeAgo = "long time ago"
	)

	samples := []struct {
		value                 string
		fromTs, toTs, alertID string
		error                 string
	}{
		{
			value:   "2020-12-31T00:00:00.000Z",
			fromTs:  "2020-12-31T00:00:00Z",
			toTs:    now,
			alertID: "",
		}, {
			value:   "2021-03-22T14:31:58.987654321Z__2021-03-22T21:41:00.123456789Z__fancy-uuid-here",
			fromTs:  "2021-03-22T14:31:58.987654321Z",
			toTs:    "2021-03-22T21:41:00.123456789Z",
			alertID: "fancy-uuid-here",
		}, {
			value:   "2021-03-22T14:31:58.987654321Z__2021-04-01T00:00:08Z",
			fromTs:  "2021-03-22T14:31:58.987654321Z",
			toTs:    "2021-04-01T00:00:08Z",
			alertID: "",
		}, {
			value:   "2021-03-22T14:31:58Z__2021-04-01T00:00:08.987654321Z__",
			fromTs:  "2021-03-22T14:31:58Z",
			toTs:    "2021-04-01T00:00:08.987654321Z",
			alertID: "",
		}, {
			value:   "",
			fromTs:  longTimeAgo,
			toTs:    now,
			alertID: "",
		}, {
			value: "__2021-04-01T00:00:08Z",
			error: "could not parse FromTimestamp",
		}, {
			value: "blah__2021-04-01T00:00:08Z",
			error: "could not parse FromTimestamp",
		}, {
			value: "2021-03-22T14:31:58.987654321Z__42__blah",
			error: "could not parse ToTimestamp",
		}, {
			value: "2021-03-22T14:31:58.987654321Z____blah",
			error: "could not parse ToTimestamp",
		}, {
			value: "1__2__3__4",
			error: "too many parts in checkpoint value",
		}, {
			value: nowWithOffsetStr(-1),
			error: "FromTimestamp.*within eventual consistency margin",
		}, {
			value: nowWithOffsetStr(-11*time.Second) + "__" + nowWithOffsetStr(-2),
			error: "ToTimestamp.*within eventual consistency margin",
		}, {
			value: nowWithOffsetStr(-7*time.Second) + "__" + nowWithOffsetStr(-3*time.Second),
			error: "(FromTimestamp|ToTimestamp).*within eventual consistency margin",
		}, {
			value: "2221-03-25T22:56:00Z",
			error: "FromTimestamp.*in the future",
		}, {
			value: "2021-03-25T20:56:00Z__2021-03-25T07:40:00Z",
			error: "FromTimestamp.*after.*ToTimestamp",
		}, {
			value: "2021-03-25T20:56:00Z__2221-03-25T22:56:00Z",
			error: "ToTimestamp.*in the future",
		}, {
			value: "2221-03-25T07:40:00Z__2221-03-25T20:56:00Z",
			error: "(FromTimestamp|ToTimestamp).*in the future",
		}, {
			// Having both timestamps equal is technically allowed, would not be an error, and the implementation should
			// handle it normally. However this won't be very useful.
			value:  "2021-03-25T07:40:00Z__2021-03-25T07:40:00Z",
			fromTs: "2021-03-25T07:40:00Z",
			toTs:   "2021-03-25T07:40:00Z",
		}, {
			value: nowWithOffsetStr(-15*time.Second) + "__" + nowWithOffsetStr(-30*time.Second),
			error: "FromTimestamp.*after ToTimestamp",
		}, {
			value: nowWithOffsetStr(30*time.Second) + "__" + nowWithOffsetStr(-15*time.Second),
			error: "FromTimestamp.*in the future",
		}, {
			value: nowWithOffsetStr(-27*time.Second) + "__" + nowWithOffsetStr(19*time.Second),
			error: "ToTimestamp.*in the future",
		}, {
			value: nowWithOffsetStr(10*time.Second) + "__" + nowWithOffsetStr(15*time.Second),
			error: "(FromTimestamp|ToTimestamp).*in the future",
		},
	}

	for _, s := range samples {
		t.Run(s.value, func(t *testing.T) {
			cp, err := parseCheckpointParam(s.value)

			if s.error != "" {
				require.Error(t, err)
				assert.Regexp(t, s.error, err.Error())
				return
			}
			require.NoError(t, err)

			if s.fromTs == longTimeAgo {
				assertFromTimestampIsLongTimeAgo(t, cp)
			} else if s.fromTs == now {
				assertTimestampIsNow(t, cp.fromTimestamp)
			} else {
				assert.Equal(t, mustParseTime(s.fromTs), cp.fromTimestamp)
			}

			if s.toTs == now {
				assertTimestampIsNow(t, cp.toTimestamp)
			} else {
				assert.Equal(t, mustParseTime(s.toTs), cp.toTimestamp)
			}

			assert.Falsef(t, cp.fromTimestamp.After(cp.toTimestamp), "Checkpoint's FromTimestamp %q must not be after ToTimestamp %q", cp.fromTimestamp, cp.toTimestamp)

			assert.Equal(t, s.alertID, cp.fromAlertID)
		})
	}
}

func TestCheckpointToStringWithNanos(t *testing.T) {
	cp := splunkCheckpoint{
		fromTimestamp: mustParseTime("2019-01-02T03:04:05.678901234Z"),
		toTimestamp:   mustParseTime("2021-12-11T10:09:08.7Z"),
		fromAlertID:   "1e81b203-91e5-47d2-a472-b975bc932f10",
	}
	assert.Equal(t, "2019-01-02T03:04:05.678901234Z__2021-12-11T10:09:08.7Z__1e81b203-91e5-47d2-a472-b975bc932f10", cp.String())
}
func TestCheckpointToStringWholeSeconds(t *testing.T) {
	cp := splunkCheckpoint{
		fromTimestamp: mustParseTime("2019-01-02T03:04:05Z"),
		toTimestamp:   mustParseTime("2021-12-11T10:09:08Z"),
		fromAlertID:   "1e81b203-91e5-47d2-a472-b975bc932f10",
	}
	assert.Equal(t, "2019-01-02T03:04:05Z__2021-12-11T10:09:08Z__1e81b203-91e5-47d2-a472-b975bc932f10", cp.String())
}
func TestCheckpointToStringEmptyAlertID(t *testing.T) {
	cp := splunkCheckpoint{
		fromTimestamp: mustParseTime("2019-01-02T03:04:05Z"),
		toTimestamp:   mustParseTime("2021-12-11T10:09:08Z"),
	}
	assert.Equal(t, "2019-01-02T03:04:05Z__2021-12-11T10:09:08Z__", cp.String())
}
func TestCheckpointToStringOnlyFromTimestamp(t *testing.T) {
	cp := splunkCheckpoint{
		fromTimestamp: mustParseTime("2019-01-02T03:04:05Z"),
	}
	assert.Equal(t, "2019-01-02T03:04:05Z", cp.String())
}
func TestCheckpointToStringNoToTimestamp(t *testing.T) {
	cp := splunkCheckpoint{
		fromTimestamp: mustParseTime("2019-01-02T03:04:05Z"),
		fromAlertID:   "1e81b203-91e5-47d2-a472-b975bc932f10",
	}
	// There will be time.Now().UTC().Format(time.RFC3339Nanos) in the middle, therefore using regexp.
	assert.Regexp(t, "2019-01-02T03:04:05Z__.+Z__1e81b203-91e5-47d2-a472-b975bc932f10", cp.String())
}

func TestMakeNextCheckpoint(t *testing.T) {
	cp1 := mustParseCheckpoint(t, "2021-01-01T00:00:00Z__2021-02-28T23:59:59Z__abcdefg")
	cp2 := cp1.makeNextCheckpoint()
	assert.Equal(t, "2021-02-28T23:59:59Z", cp2.String())
}

func TestMakeNextCheckpointNoToTimestamp(t *testing.T) {
	cp1 := mustParseCheckpoint(t, "2021-01-01T00:00:00Z")
	// Although hardly practical, it is possible to make the next checkpoint after the one without ToTimestamp.
	cp2 := cp1.makeNextCheckpoint()
	// cp2 will have only FromTimestamp equal to time.Now()
	assertCheckpointIsNow(t, cp2.String())
}

func assertFromTimestampIsLongTimeAgo(t *testing.T, cp splunkCheckpoint) {
	// The date here is the first commit in k8s repo. There certainly could be no Alerts before that date.
	assert.True(t, cp.fromTimestamp.Before(mustParseTime("2014-06-06T23:40:48Z")), "Checkpoint FromTimestamp is not old enough to allow querying all alerts", cp.fromTimestamp)
}

func assertTimestampIsNow(t *testing.T, ts time.Time) {
	assert.True(t, nowWithOffset(-eventualConsistencyMargin).After(ts), "Timestamp should not be in the future", ts)
	assert.True(t, nowWithOffset(-5*time.Second-eventualConsistencyMargin).Before(ts), "Timestamp should not be in the past", ts)
}

func mustParseCheckpoint(t *testing.T, value string) splunkCheckpoint {
	cp, err := parseCheckpointParam(value)
	require.NoError(t, err)
	return cp
}

func assertCheckpointIsNow(t *testing.T, checkpoint string) {
	// Since checkpoint is just FromTimestamp, we can directly parse it as timestamp.
	ts, err := time.Parse(time.RFC3339Nano, checkpoint)
	require.NoError(t, err)
	assertTimestampIsNow(t, ts)
}

func nowWithOffset(offset time.Duration) time.Time {
	return time.Now().UTC().Add(offset)
}

func nowWithOffsetStr(offset time.Duration) string {
	return nowWithOffset(offset).Format(time.RFC3339Nano)
}
