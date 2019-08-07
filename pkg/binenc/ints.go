package binenc

import "encoding/binary"

// UintEncoder allows encoding uint values directly to byte slices.
type UintEncoder interface {
	binary.ByteOrder

	EncodeUint16(x uint16) []byte
	EncodeUint32(x uint32) []byte
	EncodeUint64(x uint64) []byte
}

var (
	// BigEndian provides encoding functions for the big endian byte order.
	BigEndian = uintEncoder{ByteOrder: binary.BigEndian}
	// LittleEndian provides encoding functions for the little endian byte order.
	LittleEndian = uintEncoder{ByteOrder: binary.LittleEndian}
)

type uintEncoder struct {
	binary.ByteOrder
}

func (e uintEncoder) EncodeUint16(x uint16) []byte {
	buf := make([]byte, 2)
	e.PutUint16(buf, x)
	return buf
}

func (e uintEncoder) EncodeUint32(x uint32) []byte {
	buf := make([]byte, 4)
	e.PutUint32(buf, x)
	return buf
}

func (e uintEncoder) EncodeUint64(x uint64) []byte {
	buf := make([]byte, 8)
	e.PutUint64(buf, x)
	return buf
}
