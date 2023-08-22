package logging

import (
	"strconv"
	"time"

	"go.uber.org/zap/zapcore"
)

var (
	_ zapcore.ObjectEncoder = (*stringObjectEncoder)(nil)
)

// stringObjectEncoder encodes zapcore.Fields to a map[string]string for conversion reasons.
// Currently, following types will not be ignored, since those probably do not make much sense to expose:
//   - Complex numbers
//   - Arrays, Objects
//   - Binary data
type stringObjectEncoder struct {
	m map[string]string
}

func (z *stringObjectEncoder) AddBool(key string, value bool) {
	z.m[key] = strconv.FormatBool(value)
}
func (z *stringObjectEncoder) AddDuration(key string, value time.Duration) {
	z.m[key] = value.String()
}

func (z *stringObjectEncoder) AddFloat64(key string, value float64) {
	z.m[key] = strconv.FormatFloat(value, 'f', -1, 64)
}

func (z *stringObjectEncoder) AddInt64(key string, value int64) {
	z.m[key] = strconv.FormatInt(value, 10)
}

func (z *stringObjectEncoder) AddString(key, value string) {
	z.m[key] = value
}

func (z *stringObjectEncoder) AddTime(key string, value time.Time) {
	z.m[key] = value.Format(time.RFC3339)
}

func (z *stringObjectEncoder) AddUint64(key string, value uint64) {
	z.m[key] = strconv.FormatUint(value, 10)
}

func (z *stringObjectEncoder) AddInt(key string, value int)         { z.AddInt64(key, int64(value)) }
func (z *stringObjectEncoder) AddInt32(key string, value int32)     { z.AddInt64(key, int64(value)) }
func (z *stringObjectEncoder) AddInt16(key string, value int16)     { z.AddInt64(key, int64(value)) }
func (z *stringObjectEncoder) AddInt8(key string, value int8)       { z.AddInt64(key, int64(value)) }
func (z *stringObjectEncoder) AddUint32(key string, value uint32)   { z.AddUint64(key, uint64(value)) }
func (z *stringObjectEncoder) AddUint16(key string, value uint16)   { z.AddUint64(key, uint64(value)) }
func (z *stringObjectEncoder) AddUint8(key string, value uint8)     { z.AddUint64(key, uint64(value)) }
func (z *stringObjectEncoder) AddUint(key string, value uint)       { z.AddUint64(key, uint64(value)) }
func (z *stringObjectEncoder) AddUintptr(key string, value uintptr) { z.AddUint64(key, uint64(value)) }
func (z *stringObjectEncoder) AddFloat32(key string, value float32) {
	z.AddFloat64(key, float64(value))
}

// All fields below are currently not supported and expected to be exposed as labels.
// In case we need those, we simply need to implement the interface accordingly.

func (z *stringObjectEncoder) AddByteString(_ string, _ []byte)                    {}
func (z *stringObjectEncoder) AddBinary(_ string, _ []byte)                        {}
func (z *stringObjectEncoder) AddObject(_ string, _ zapcore.ObjectMarshaler) error { return nil }
func (z *stringObjectEncoder) AddArray(_ string, _ zapcore.ArrayMarshaler) error   { return nil }
func (z *stringObjectEncoder) AddComplex128(_ string, _ complex128)                {}
func (z *stringObjectEncoder) AddReflected(_ string, _ interface{}) error          { return nil }
func (z *stringObjectEncoder) AddComplex64(_ string, _ complex64)                  {}
func (z *stringObjectEncoder) OpenNamespace(_ string)                              {}
