package signal

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func TestApiIntoSensorSignal(t *testing.T) {
	signal := &storage.ProcessSignal{
		Id:           "1234",
		ContainerId:  "0123456789ab",
		Time:         &timestamppb.Timestamp{},
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
	}
	input := v1.Signal{
		Signal: &v1.Signal_ProcessSignal{
			ProcessSignal: signal,
		},
	}
	expected := sensor.ProcessSignal{
		Id:           signal.Id,
		ContainerId:  signal.ContainerId,
		CreationTime: &timestamppb.Timestamp{},
		Name:         signal.Name,
		Args:         signal.Args,
		ExecFilePath: signal.ExecFilePath,
		Pid:          signal.Pid,
		Uid:          signal.Uid,
		Gid:          signal.Gid,
		Scraped:      false,
		LineageInfo: []*sensor.ProcessSignal_LineageInfo{{
			ParentUid:          signal.LineageInfo[0].ParentUid,
			ParentExecFilePath: signal.LineageInfo[0].ParentExecFilePath,
		}},
	}
	assert.Equal(t, apiToSensorSignal(&input), &expected)
}
