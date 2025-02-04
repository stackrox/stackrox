package grpc

import (
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/mem"
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
	vt, err := c.marshalVT(m)
	if err != nil {
		return c.CodecV2.Marshal(v)
	}
	return vt, nil
}

func (c *codec) marshalVT(m vtprotoMessage) (mem.BufferSlice, error) {
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
	err := m.UnmarshalVT(buf.ReadOnlyData())
	if err != nil {
		return c.CodecV2.Unmarshal(data, v)
	}
	return nil
}

func init() {
	// Replace the original codec with vt wrapper.
	encoding.RegisterCodecV2(&codec{
		CodecV2: encoding.GetCodecV2(proto.Name),
	})
}
