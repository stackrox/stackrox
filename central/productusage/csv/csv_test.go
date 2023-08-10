package usagecsv

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetTimeParam(t *testing.T) {
	t.Run("good from, bad to", func(t *testing.T) {
		from, _ := time.Parse(time.RFC3339Nano, "2023-07-24T10:13:21.702316Z")
		formValues := url.Values{
			"from": {from.Format(time.RFC3339Nano)},
			"to":   {"not a time"},
		}
		v, err := getTimeParameter(formValues, "from", zeroTime)
		assert.NoError(t, err)
		assert.Equal(t, from.Unix(), v.GetSeconds())
		assert.Equal(t, int32(from.Nanosecond()), v.GetNanos())
	})
	t.Run("bad from", func(t *testing.T) {
		formValues := url.Values{
			"from": {"not a time"},
		}
		_, err := getTimeParameter(formValues, "from", zeroTime)
		assert.Error(t, err)
		now := time.Now()
		to, err := getTimeParameter(formValues, "to", now)
		assert.NoError(t, err)
		assert.Equal(t, now.Unix(), to.GetSeconds())
	})
}
