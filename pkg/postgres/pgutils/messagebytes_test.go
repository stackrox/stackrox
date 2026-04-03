package pgutils

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshalRepeatedMessages_RoundTrip(t *testing.T) {
	original := []*storage.ProcessSignalNoSerialized_LineageInfoNoSerialized{
		{ParentUid: 22, ParentExecFilePath: "/bin/bash"},
		{ParentUid: 1, ParentExecFilePath: "/sbin/init"},
	}

	data, err := MarshalRepeatedMessages(original)
	require.NoError(t, err)
	require.NotNil(t, data)

	result, err := UnmarshalRepeatedMessages(data, func() *storage.ProcessSignalNoSerialized_LineageInfoNoSerialized {
		return &storage.ProcessSignalNoSerialized_LineageInfoNoSerialized{}
	})
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, uint32(22), result[0].ParentUid)
	assert.Equal(t, "/bin/bash", result[0].ParentExecFilePath)
	assert.Equal(t, uint32(1), result[1].ParentUid)
	assert.Equal(t, "/sbin/init", result[1].ParentExecFilePath)
}

func TestMarshalUnmarshalRepeatedMessages_Empty(t *testing.T) {
	data, err := MarshalRepeatedMessages[*storage.ProcessSignalNoSerialized_LineageInfoNoSerialized](nil)
	require.NoError(t, err)
	require.Nil(t, data)

	result, err := UnmarshalRepeatedMessages(nil, func() *storage.ProcessSignalNoSerialized_LineageInfoNoSerialized {
		return &storage.ProcessSignalNoSerialized_LineageInfoNoSerialized{}
	})
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestMarshalUnmarshalRepeatedMessages_SingleElement(t *testing.T) {
	original := []*storage.ProcessSignalNoSerialized_LineageInfoNoSerialized{
		{ParentUid: 42, ParentExecFilePath: "/usr/bin/env"},
	}

	data := MustMarshalRepeatedMessages(original)
	result := MustUnmarshalRepeatedMessages(data, func() *storage.ProcessSignalNoSerialized_LineageInfoNoSerialized {
		return &storage.ProcessSignalNoSerialized_LineageInfoNoSerialized{}
	})
	require.Len(t, result, 1)
	assert.Equal(t, uint32(42), result[0].ParentUid)
	assert.Equal(t, "/usr/bin/env", result[0].ParentExecFilePath)
}

func TestUnmarshalRepeatedMessages_TruncatedData(t *testing.T) {
	original := []*storage.ProcessSignalNoSerialized_LineageInfoNoSerialized{
		{ParentUid: 1, ParentExecFilePath: "/bin/sh"},
	}
	data := MustMarshalRepeatedMessages(original)

	// Truncate the data
	_, err := UnmarshalRepeatedMessages(data[:len(data)-2], func() *storage.ProcessSignalNoSerialized_LineageInfoNoSerialized {
		return &storage.ProcessSignalNoSerialized_LineageInfoNoSerialized{}
	})
	require.Error(t, err)
}
