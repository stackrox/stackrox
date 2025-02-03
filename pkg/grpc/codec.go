package grpc

import (
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/mem"

	// register original proto codec before so it can be wrapped
	_ "google.golang.org/grpc/encoding/proto" // nolint:revive
)

type vtprotoMessage interface {
	MarshalToSizedBufferVT(data []byte) (int, error)
	UnmarshalVT([]byte) error
	SizeVT() int
}

type codec struct {
	// similar to customMarshaler we fall back to original implementation when message is not supported
	encoding.CodecV2
}

func (c *codec) Name() string { return c.CodecV2.Name() }

var defaultBufferPool = mem.DefaultBufferPool()

func (c *codec) Marshal(v any) (mem.BufferSlice, error) {
	m, ok := v.(vtprotoMessage)
	if !ok {
		return c.CodecV2.Marshal(v)
	}
	size := m.SizeVT()
	if mem.IsBelowBufferPoolingThreshold(size) {
		buf := make([]byte, size)
		_, err := m.MarshalToSizedBufferVT(buf)
		if err != nil {
			return nil, err
		}
		return mem.BufferSlice{mem.SliceBuffer(buf)}, nil
	}
	buf := defaultBufferPool.Get(size)
	_, err := m.MarshalToSizedBufferVT(*buf)
	if err != nil {
		defaultBufferPool.Put(buf)
		return nil, err
	}
	return mem.BufferSlice{mem.NewBuffer(buf, defaultBufferPool)}, nil
}

func (c *codec) Unmarshal(data mem.BufferSlice, v any) error {
	m, ok := v.(vtprotoMessage)
	if !ok {
		return c.CodecV2.Unmarshal(data, v)
	}
	buf := data.MaterializeToBuffer(defaultBufferPool)
	defer buf.Free()
	return m.UnmarshalVT(buf.ReadOnlyData())
}

func init() {
	// Replace the original codec with vt wrapper.
	//https://github.com/grpc/grpc-go/blob/v1.70.0/encoding/proto/proto.go#L33
	encoding.RegisterCodecV2(&codec{
		CodecV2: encoding.GetCodecV2("proto"),
	})
}
