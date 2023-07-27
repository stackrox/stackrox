package usagecsv

import (
	"bytes"
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
			Ts: protoconv.ConvertTimeToTimestamp(ts1),
			Sr: &storage.Usage_SecuredResources{
				Nodes: 1,
				Cores: 2,
			},
		},
		{
			Ts: protoconv.ConvertTimeToTimestamp(ts2),
			Sr: &storage.Usage_SecuredResources{
				Nodes: 3,
				Cores: 4,
			},
		},
	}
	var data []byte
	buf := bytes.NewBuffer(data)
	err := writeCSV(metrics, buf)

	assert.NoError(t, err)
	assert.Equal(t, "Timestamp,Nodes,Cores\n2023-07-24T10:13:21Z,1,2\n2023-07-24T15:13:21Z,3,4\n", buf.String())
}
