package component

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

const (
	containerID1 = "1e43ac4f61f9"
)

func TestSensorIntoStorageSignal(t *testing.T) {
	tests := []struct {
		input    *sensor.ProcessSignal
		expected *storage.ProcessSignal
	}{
		{input: nil, expected: nil},
		{
			input: &sensor.ProcessSignal{
				Id:           "1234",
				ContainerId:  containerID1,
				Name:         "mock",
				Args:         "--help",
				ExecFilePath: "/usr/local/bin/mock",
				Pid:          4321,
				Uid:          5432,
				Gid:          1234,
				Scraped:      false,
			},
			expected: &storage.ProcessSignal{
				Id:           "1234",
				ContainerId:  containerID1,
				Name:         "mock",
				Args:         "--help",
				ExecFilePath: "/usr/local/bin/mock",
				Pid:          4321,
				Uid:          5432,
				Gid:          1234,
				Scraped:      false,
			},
		},
		{
			input: &sensor.ProcessSignal{
				Id:           "1234",
				ContainerId:  containerID1,
				Name:         "mock",
				Args:         "--help",
				ExecFilePath: "/usr/local/bin/mock",
				Pid:          4321,
				Uid:          5432,
				Gid:          1234,
				Scraped:      false,
				LineageInfo: []*sensor.ProcessSignal_LineageInfo{
					{ParentUid: 2345, ParentExecFilePath: "parent"},
				},
			},
			expected: &storage.ProcessSignal{
				Id:           "1234",
				ContainerId:  containerID1,
				Name:         "mock",
				Args:         "--help",
				ExecFilePath: "/usr/local/bin/mock",
				Pid:          4321,
				Uid:          5432,
				Gid:          1234,
				Scraped:      false,
				LineageInfo: []*storage.ProcessSignal_LineageInfo{
					{ParentUid: 2345, ParentExecFilePath: "parent"},
				},
			},
		},
	}

	for _, test := range tests {
		assert.EqualExportedValues(t, sensorIntoStorageSignal(test.input), test.expected)
	}
}
