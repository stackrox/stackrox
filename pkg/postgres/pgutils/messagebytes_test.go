package pgutils

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshalRepeatedMessages_RoundTrip(t *testing.T) {
	msgs := []*storage.ProcessSignal_LineageInfo{
		{ParentUid: 1, ParentExecFilePath: "/usr/bin/bash"},
		{ParentUid: 2, ParentExecFilePath: "/usr/bin/python"},
		{ParentUid: 3, ParentExecFilePath: "/bin/sh"},
	}

	data, err := MarshalRepeatedMessages(msgs)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	got, err := UnmarshalRepeatedMessages(data, func() *storage.ProcessSignal_LineageInfo {
		return &storage.ProcessSignal_LineageInfo{}
	})
	require.NoError(t, err)
	require.Len(t, got, 3)

	for i, msg := range msgs {
		assert.Equal(t, msg.GetParentUid(), got[i].GetParentUid())
		assert.Equal(t, msg.GetParentExecFilePath(), got[i].GetParentExecFilePath())
	}
}

func TestMarshalUnmarshalRepeatedMessages_Empty(t *testing.T) {
	data, err := MarshalRepeatedMessages[*storage.ProcessSignal_LineageInfo](nil)
	require.NoError(t, err)
	assert.Nil(t, data)

	got, err := UnmarshalRepeatedMessages(nil, func() *storage.ProcessSignal_LineageInfo {
		return &storage.ProcessSignal_LineageInfo{}
	})
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestMarshalUnmarshalRepeatedMessages_SingleElement(t *testing.T) {
	msgs := []*storage.ProcessSignal_LineageInfo{
		{ParentUid: 42, ParentExecFilePath: "/only/one"},
	}

	data, err := MarshalRepeatedMessages(msgs)
	require.NoError(t, err)

	got, err := UnmarshalRepeatedMessages(data, func() *storage.ProcessSignal_LineageInfo {
		return &storage.ProcessSignal_LineageInfo{}
	})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, uint32(42), got[0].GetParentUid())
}

func TestUnmarshalRepeatedMessages_Truncated(t *testing.T) {
	msgs := []*storage.ProcessSignal_LineageInfo{
		{ParentUid: 1, ParentExecFilePath: "/usr/bin/bash"},
	}

	data, err := MarshalRepeatedMessages(msgs)
	require.NoError(t, err)

	// Truncate the data
	_, err = UnmarshalRepeatedMessages(data[:len(data)-2], func() *storage.ProcessSignal_LineageInfo {
		return &storage.ProcessSignal_LineageInfo{}
	})
	assert.Error(t, err)
}

func TestMustMarshalRepeatedMessages(t *testing.T) {
	msgs := []*storage.ProcessSignal_LineageInfo{
		{ParentUid: 1},
	}
	data := MustMarshalRepeatedMessages(msgs)
	assert.NotEmpty(t, data)

	got := MustUnmarshalRepeatedMessages(data, func() *storage.ProcessSignal_LineageInfo {
		return &storage.ProcessSignal_LineageInfo{}
	})
	require.Len(t, got, 1)
	assert.Equal(t, uint32(1), got[0].GetParentUid())
}
