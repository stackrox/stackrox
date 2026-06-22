package vsockclient

import (
	"bytes"
	"io"
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

type fakeStream struct {
	io.Reader
	closed bool
}

func (f *fakeStream) Close() error {
	f.closed = true
	return nil
}

func TestReadVMReport(t *testing.T) {
	want := &v1.VMReport{
		IndexReport: &v1.IndexReport{
			VsockCid: "99",
			IndexV4: &v4.IndexReport{
				Success: true,
			},
		},
	}
	data, err := proto.Marshal(want)
	require.NoError(t, err)

	stream := &fakeStream{Reader: bytes.NewReader(data)}

	got, err := ReadVMReport(stream)
	require.NoError(t, err)
	assert.Equal(t, "99", got.GetIndexReport().GetVsockCid())
	assert.True(t, got.GetIndexReport().GetIndexV4().GetSuccess())
	assert.True(t, stream.closed)
}
