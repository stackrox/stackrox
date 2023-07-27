package usagecsv

import (
	"bytes"
	"net/url"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/assert"
)

func TestCSV(t *testing.T) {
	ts1, _ := time.Parse(time.RFC3339Nano, "2023-07-24T10:13:21.702316Z")
	ts2, _ := time.Parse(time.RFC3339Nano, "2023-07-24T15:13:21.702316Z")
	metrics := []*storage.Usage{
		{
			Timestamp:   protoconv.ConvertTimeToTimestamp(ts1),
			NumNodes:    1,
			NumCpuUnits: 2,
		},
		{
			Timestamp:   protoconv.ConvertTimeToTimestamp(ts2),
			NumNodes:    3,
			NumCpuUnits: 4,
		},
	}
	var data []byte
	buf := bytes.NewBuffer(data)
	err := writeCSV(metrics, buf)

	assert.NoError(t, err)
	assert.Equal(t, "Timestamp,Nodes,CPU Units\n2023-07-24T10:13:21Z,1,2\n2023-07-24T15:13:21Z,3,4\n", buf.String())
}

func TestGetTimeParam(t *testing.T) {
	{
		from, _ := time.Parse(time.RFC3339Nano, "2023-07-24T10:13:21.702316Z")
		formValues := url.Values{"from": {from.Format(time.RFC3339Nano)}, "to": {"not a time"}}
		v, err := getTimeParameter(formValues, "from", zeroTime)
		assert.NoError(t, err)
		assert.Equal(t, from.Unix(), v.GetSeconds())
		assert.Equal(t, int32(from.Nanosecond()), v.GetNanos())
	}
	{
		formValues := url.Values{"from": {"not a time"}}
		_, err := getTimeParameter(formValues, "from", zeroTime)
		assert.Error(t, err)
		now := time.Now()
		to, err := getTimeParameter(formValues, "to", now)
		assert.NoError(t, err)
		assert.Equal(t, now.Unix(), to.GetSeconds())
	}
}
