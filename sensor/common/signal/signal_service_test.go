package signal

import (
	"fmt"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestApiIntoSensorSignal(t *testing.T) {
	tests := []struct {
		input    *v1.Signal
		expected *sensor.ProcessSignal
	}{
		{input: nil, expected: nil},
		{input: &v1.Signal{}, expected: nil},
		{
			input: &v1.Signal{
				Signal: &v1.Signal_ProcessSignal{
					ProcessSignal: &storage.ProcessSignal{
						Id:           "1234",
						ContainerId:  "0123456789ab",
						Name:         "mock",
						Args:         "--help",
						ExecFilePath: "/usr/local/bin/mock",
						Pid:          1234,
						Uid:          4321,
						Gid:          2345,
						Scraped:      false,
					},
				},
			},
			expected: &sensor.ProcessSignal{
				Id:           "1234",
				ContainerId:  "0123456789ab",
				Name:         "mock",
				Args:         "--help",
				ExecFilePath: "/usr/local/bin/mock",
				Pid:          1234,
				Uid:          4321,
				Gid:          2345,
				Scraped:      false,
			},
		},
		{
			input: &v1.Signal{
				Signal: &v1.Signal_ProcessSignal{
					ProcessSignal: &storage.ProcessSignal{
						Id:           "1234",
						ContainerId:  "0123456789ab",
						Name:         "mock",
						Args:         "--help",
						ExecFilePath: "/usr/local/bin/mock",
						Pid:          1234,
						Uid:          4321,
						Gid:          2345,
						Scraped:      false,
						LineageInfo: []*storage.ProcessSignal_LineageInfo{
							{ParentUid: 5432, ParentExecFilePath: "parent"},
						},
					},
				},
			},
			expected: &sensor.ProcessSignal{
				Id:           "1234",
				ContainerId:  "0123456789ab",
				Name:         "mock",
				Args:         "--help",
				ExecFilePath: "/usr/local/bin/mock",
				Pid:          1234,
				Uid:          4321,
				Gid:          2345,
				Scraped:      false,
				LineageInfo: []*sensor.ProcessSignal_LineageInfo{
					{ParentUid: 5432, ParentExecFilePath: "parent"},
				},
			},
		},
	}

	for _, test := range tests {
		fmt.Println("Starting test")
		assert.EqualExportedValues(t, apiToSensorSignal(test.input), test.expected)
	}
}
