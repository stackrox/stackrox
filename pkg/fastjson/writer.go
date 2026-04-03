package fastjson

import (
	"encoding/base64"
	"math"
	"strconv"
	"unicode/utf8"
)

// Writer is an append-based JSON writer that builds JSON output
// into a byte buffer with minimal allocations.
type Writer struct {
	buf   []byte
	depth int
	first bool // tracks whether next element needs a comma
}

// NewWriter creates a new Writer with the given estimated capacity.
func NewWriter(estimatedSize int) Writer {
	return Writer{
		buf:   make([]byte, 0, estimatedSize),
		first: true,
	}
}

// Bytes returns the accumulated JSON bytes.
func (w *Writer) Bytes() []byte {
	return w.buf
}

// BeginObject writes '{' and enters a new nesting level.
func (w *Writer) BeginObject() {
	w.buf = append(w.buf, '{')
	w.depth++
	w.first = true
}

// EndObject writes '}' and exits the nesting level.
func (w *Writer) EndObject() {
	w.buf = append(w.buf, '}')
	w.depth--
	w.first = false
}

// BeginArray writes '[' and enters a new nesting level.
func (w *Writer) BeginArray() {
	w.buf = append(w.buf, '[')
	w.depth++
	w.first = true
}

// EndArray writes ']' and exits the nesting level.
func (w *Writer) EndArray() {
	w.buf = append(w.buf, ']')
	w.depth--
	w.first = false
}

// comma writes a comma separator if this is not the first element.
func (w *Writer) comma() {
	if !w.first {
		w.buf = append(w.buf, ',')
	}
	w.first = false
}

// FieldName writes a JSON object field name with comma management.
// Writes: ,"name": (or "name": if first field)
func (w *Writer) FieldName(name string) {
	w.comma()
	w.buf = append(w.buf, '"')
	w.buf = append(w.buf, name...) // field names are known safe (no escaping needed)
	w.buf = append(w.buf, '"', ':')
}

// String writes a JSON string value with proper escaping.
func (w *Writer) String(s string) {
	w.buf = appendJSONString(w.buf, s)
}

// Uint32 writes a JSON number for a uint32 value.
func (w *Writer) Uint32(v uint32) {
	w.buf = strconv.AppendUint(w.buf, uint64(v), 10)
}

// Int32 writes a JSON number for an int32 value.
func (w *Writer) Int32(v int32) {
	w.buf = strconv.AppendInt(w.buf, int64(v), 10)
}

// Int64 writes a quoted JSON string for an int64 value (proto3 spec).
func (w *Writer) Int64(v int64) {
	w.buf = append(w.buf, '"')
	w.buf = strconv.AppendInt(w.buf, v, 10)
	w.buf = append(w.buf, '"')
}

// Uint64 writes a quoted JSON string for a uint64 value (proto3 spec).
func (w *Writer) Uint64(v uint64) {
	w.buf = append(w.buf, '"')
	w.buf = strconv.AppendUint(w.buf, v, 10)
	w.buf = append(w.buf, '"')
}

// Float32 writes a JSON number for a float32 value.
// Handles NaN, +Inf, -Inf as quoted strings per proto3 spec.
func (w *Writer) Float32(v float32) {
	switch {
	case math.IsNaN(float64(v)):
		w.buf = append(w.buf, `"NaN"`...)
	case math.IsInf(float64(v), 1):
		w.buf = append(w.buf, `"Infinity"`...)
	case math.IsInf(float64(v), -1):
		w.buf = append(w.buf, `"-Infinity"`...)
	default:
		w.buf = strconv.AppendFloat(w.buf, float64(v), 'g', -1, 32)
	}
}

// Float64 writes a JSON number for a float64 value.
// Handles NaN, +Inf, -Inf as quoted strings per proto3 spec.
func (w *Writer) Float64(v float64) {
	switch {
	case math.IsNaN(v):
		w.buf = append(w.buf, `"NaN"`...)
	case math.IsInf(v, 1):
		w.buf = append(w.buf, `"Infinity"`...)
	case math.IsInf(v, -1):
		w.buf = append(w.buf, `"-Infinity"`...)
	default:
		w.buf = strconv.AppendFloat(w.buf, v, 'g', -1, 64)
	}
}

// Bool writes a JSON boolean value.
func (w *Writer) Bool(v bool) {
	if v {
		w.buf = append(w.buf, "true"...)
	} else {
		w.buf = append(w.buf, "false"...)
	}
}

// Base64 writes a base64-encoded JSON string for bytes.
func (w *Writer) Base64(v []byte) {
	w.buf = append(w.buf, '"')
	encodedLen := base64.StdEncoding.EncodedLen(len(v))
	start := len(w.buf)
	w.buf = append(w.buf, make([]byte, encodedLen)...)
	base64.StdEncoding.Encode(w.buf[start:], v)
	w.buf = append(w.buf, '"')
}

// Null writes the JSON null value.
func (w *Writer) Null() {
	w.buf = append(w.buf, "null"...)
}

// Raw writes pre-formatted JSON bytes directly.
func (w *Writer) Raw(data []byte) {
	w.buf = append(w.buf, data...)
}

// ArrayElem prepends a comma if needed before an array element.
func (w *Writer) ArrayElem() {
	w.comma()
}

// appendJSONString appends a JSON-escaped string (with quotes) to buf.
func appendJSONString(buf []byte, s string) []byte {
	buf = append(buf, '"')
	for i := 0; i < len(s); {
		b := s[i]
		if b < utf8.RuneSelf {
			switch {
			case b == '"':
				buf = append(buf, '\\', '"')
			case b == '\\':
				buf = append(buf, '\\', '\\')
			case b == '\n':
				buf = append(buf, '\\', 'n')
			case b == '\r':
				buf = append(buf, '\\', 'r')
			case b == '\t':
				buf = append(buf, '\\', 't')
			case b < 0x20:
				buf = append(buf, '\\', 'u', '0', '0')
				buf = append(buf, hexDigits[b>>4], hexDigits[b&0x0f])
			default:
				buf = append(buf, b)
			}
			i++
		} else {
			r, size := utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError && size == 1 {
				// Invalid UTF-8 byte, escape it.
				buf = append(buf, '\\', 'u', 'f', 'f', 'f', 'd')
			} else {
				buf = append(buf, s[i:i+size]...)
			}
			i += size
		}
	}
	buf = append(buf, '"')
	return buf
}

const hexDigits = "0123456789abcdef"
