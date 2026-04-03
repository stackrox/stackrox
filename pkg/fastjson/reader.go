package fastjson

import (
	"fmt"
	"strconv"
	"unicode/utf8"
)

// Reader is a zero-copy JSON tokenizer that scans through JSON bytes directly.
type Reader struct {
	data []byte
	pos  int
}

// NewReader creates a Reader from raw JSON bytes.
func NewReader(data []byte) *Reader {
	return &Reader{data: data}
}

// skipWhitespace advances past whitespace.
func (r *Reader) skipWhitespace() {
	for r.pos < len(r.data) {
		switch r.data[r.pos] {
		case ' ', '\t', '\n', '\r':
			r.pos++
		default:
			return
		}
	}
}

// ReadObject iterates over a JSON object's fields, calling handler for each key.
func (r *Reader) ReadObject(handler func(key string) error) error {
	r.skipWhitespace()
	if r.pos >= len(r.data) || r.data[r.pos] != '{' {
		return fmt.Errorf("fastjson: expected '{' at pos %d", r.pos)
	}
	r.pos++ // skip '{'

	r.skipWhitespace()
	if r.pos < len(r.data) && r.data[r.pos] == '}' {
		r.pos++ // empty object
		return nil
	}

	for {
		r.skipWhitespace()
		key, err := r.readStringValue()
		if err != nil {
			return fmt.Errorf("fastjson: reading object key: %w", err)
		}

		r.skipWhitespace()
		if r.pos >= len(r.data) || r.data[r.pos] != ':' {
			return fmt.Errorf("fastjson: expected ':' at pos %d", r.pos)
		}
		r.pos++ // skip ':'

		r.skipWhitespace()
		if err := handler(key); err != nil {
			return err
		}

		r.skipWhitespace()
		if r.pos >= len(r.data) {
			return fmt.Errorf("fastjson: unexpected end of object")
		}
		if r.data[r.pos] == '}' {
			r.pos++
			return nil
		}
		if r.data[r.pos] == ',' {
			r.pos++
			continue
		}
		return fmt.Errorf("fastjson: expected ',' or '}' at pos %d, got %q", r.pos, r.data[r.pos])
	}
}

// ReadArray iterates over a JSON array, calling handler for each element.
func (r *Reader) ReadArray(handler func() error) error {
	r.skipWhitespace()
	if r.pos >= len(r.data) || r.data[r.pos] != '[' {
		return fmt.Errorf("fastjson: expected '[' at pos %d", r.pos)
	}
	r.pos++ // skip '['

	r.skipWhitespace()
	if r.pos < len(r.data) && r.data[r.pos] == ']' {
		r.pos++ // empty array
		return nil
	}

	for {
		r.skipWhitespace()
		if err := handler(); err != nil {
			return err
		}

		r.skipWhitespace()
		if r.pos >= len(r.data) {
			return fmt.Errorf("fastjson: unexpected end of array")
		}
		if r.data[r.pos] == ']' {
			r.pos++
			return nil
		}
		if r.data[r.pos] == ',' {
			r.pos++
			continue
		}
		return fmt.Errorf("fastjson: expected ',' or ']' at pos %d", r.pos)
	}
}

// ReadString reads a JSON string value.
func (r *Reader) ReadString() (string, error) {
	r.skipWhitespace()
	return r.readStringValue()
}

// readStringValue reads a JSON string (must be at opening quote).
func (r *Reader) readStringValue() (string, error) {
	if r.pos >= len(r.data) || r.data[r.pos] != '"' {
		return "", fmt.Errorf("fastjson: expected '\"' at pos %d", r.pos)
	}
	r.pos++ // skip opening quote

	// Fast path: no escapes needed
	start := r.pos
	hasEscape := false
	for r.pos < len(r.data) {
		b := r.data[r.pos]
		if b == '"' {
			if !hasEscape {
				s := string(r.data[start:r.pos])
				r.pos++ // skip closing quote
				return s, nil
			}
			break
		}
		if b == '\\' {
			hasEscape = true
			r.pos += 2 // skip escape sequence (at minimum)
			continue
		}
		r.pos++
	}

	if !hasEscape {
		return "", fmt.Errorf("fastjson: unterminated string")
	}

	// Slow path: unescape
	r.pos = start
	var buf []byte
	for r.pos < len(r.data) {
		b := r.data[r.pos]
		if b == '"' {
			r.pos++
			return string(buf), nil
		}
		if b == '\\' {
			r.pos++
			if r.pos >= len(r.data) {
				return "", fmt.Errorf("fastjson: unterminated escape")
			}
			switch r.data[r.pos] {
			case '"', '\\', '/':
				buf = append(buf, r.data[r.pos])
			case 'n':
				buf = append(buf, '\n')
			case 'r':
				buf = append(buf, '\r')
			case 't':
				buf = append(buf, '\t')
			case 'b':
				buf = append(buf, '\b')
			case 'f':
				buf = append(buf, '\f')
			case 'u':
				if r.pos+4 >= len(r.data) {
					return "", fmt.Errorf("fastjson: short unicode escape")
				}
				r.pos++
				hexStr := string(r.data[r.pos : r.pos+4])
				codePoint, err := strconv.ParseUint(hexStr, 16, 32)
				if err != nil {
					return "", fmt.Errorf("fastjson: invalid unicode escape: %w", err)
				}
				r.pos += 3 // will be incremented to +4 below
				var utfBuf [utf8.UTFMax]byte
				n := utf8.EncodeRune(utfBuf[:], rune(codePoint))
				buf = append(buf, utfBuf[:n]...)
			default:
				return "", fmt.Errorf("fastjson: unknown escape '\\%c'", r.data[r.pos])
			}
			r.pos++
			continue
		}
		buf = append(buf, b)
		r.pos++
	}
	return "", fmt.Errorf("fastjson: unterminated string")
}

// ReadUint32 reads a JSON number and converts to uint32.
func (r *Reader) ReadUint32() (uint32, error) {
	r.skipWhitespace()
	// Handle quoted numbers (proto3 accepts both)
	if r.pos < len(r.data) && r.data[r.pos] == '"' {
		s, err := r.readStringValue()
		if err != nil {
			return 0, err
		}
		n, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("fastjson: parsing uint32 from string %q: %w", s, err)
		}
		return uint32(n), nil
	}
	raw := r.readNumberBytes()
	n, err := strconv.ParseUint(string(raw), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("fastjson: parsing uint32 %q: %w", raw, err)
	}
	return uint32(n), nil
}

// ReadInt32 reads a JSON number and converts to int32.
func (r *Reader) ReadInt32() (int32, error) {
	r.skipWhitespace()
	if r.pos < len(r.data) && r.data[r.pos] == '"' {
		s, err := r.readStringValue()
		if err != nil {
			return 0, err
		}
		n, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("fastjson: parsing int32 from string %q: %w", s, err)
		}
		return int32(n), nil
	}
	raw := r.readNumberBytes()
	n, err := strconv.ParseInt(string(raw), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("fastjson: parsing int32 %q: %w", raw, err)
	}
	return int32(n), nil
}

// ReadInt64 reads a JSON value and converts to int64.
// Accepts both quoted strings and numbers per proto3 spec.
func (r *Reader) ReadInt64() (int64, error) {
	r.skipWhitespace()
	if r.pos < len(r.data) && r.data[r.pos] == '"' {
		s, err := r.readStringValue()
		if err != nil {
			return 0, err
		}
		return strconv.ParseInt(s, 10, 64)
	}
	raw := r.readNumberBytes()
	return strconv.ParseInt(string(raw), 10, 64)
}

// ReadUint64 reads a JSON value and converts to uint64.
func (r *Reader) ReadUint64() (uint64, error) {
	r.skipWhitespace()
	if r.pos < len(r.data) && r.data[r.pos] == '"' {
		s, err := r.readStringValue()
		if err != nil {
			return 0, err
		}
		return strconv.ParseUint(s, 10, 64)
	}
	raw := r.readNumberBytes()
	return strconv.ParseUint(string(raw), 10, 64)
}

// ReadBool reads a JSON boolean value.
func (r *Reader) ReadBool() (bool, error) {
	r.skipWhitespace()
	if r.pos+4 <= len(r.data) && string(r.data[r.pos:r.pos+4]) == "true" {
		r.pos += 4
		return true, nil
	}
	if r.pos+5 <= len(r.data) && string(r.data[r.pos:r.pos+5]) == "false" {
		r.pos += 5
		return false, nil
	}
	return false, fmt.Errorf("fastjson: expected bool at pos %d", r.pos)
}

// ReadNullOrRaw reads the next JSON value. If it's null, returns (nil, true, nil).
// Otherwise returns the raw JSON bytes and (raw, false, nil).
func (r *Reader) ReadNullOrRaw() ([]byte, bool, error) {
	r.skipWhitespace()
	if r.pos+4 <= len(r.data) && string(r.data[r.pos:r.pos+4]) == "null" {
		r.pos += 4
		return nil, true, nil
	}
	raw, err := r.readRawValue()
	if err != nil {
		return nil, false, err
	}
	return raw, false, nil
}

// ReadRaw reads the next complete JSON value as raw bytes.
func (r *Reader) ReadRaw() ([]byte, error) {
	r.skipWhitespace()
	return r.readRawValue()
}

// SkipValue skips the next JSON value.
func (r *Reader) SkipValue() error {
	r.skipWhitespace()
	_, err := r.readRawValue()
	return err
}

// readNumberBytes reads a JSON number as raw bytes without parsing.
func (r *Reader) readNumberBytes() []byte {
	start := r.pos
	for r.pos < len(r.data) {
		b := r.data[r.pos]
		if b >= '0' && b <= '9' || b == '-' || b == '+' || b == '.' || b == 'e' || b == 'E' {
			r.pos++
			continue
		}
		break
	}
	return r.data[start:r.pos]
}

// readRawValue reads a complete JSON value (object, array, string, number, bool, null)
// and returns it as raw bytes. This is a zero-copy slice of the input for most cases.
func (r *Reader) readRawValue() ([]byte, error) {
	if r.pos >= len(r.data) {
		return nil, fmt.Errorf("fastjson: unexpected end of input")
	}

	start := r.pos
	switch r.data[r.pos] {
	case '{':
		if err := r.skipObject(); err != nil {
			return nil, err
		}
	case '[':
		if err := r.skipArray(); err != nil {
			return nil, err
		}
	case '"':
		if _, err := r.readStringValue(); err != nil {
			return nil, err
		}
	case 't':
		if r.pos+4 > len(r.data) {
			return nil, fmt.Errorf("fastjson: unexpected end of 'true'")
		}
		r.pos += 4
	case 'f':
		if r.pos+5 > len(r.data) {
			return nil, fmt.Errorf("fastjson: unexpected end of 'false'")
		}
		r.pos += 5
	case 'n':
		if r.pos+4 > len(r.data) {
			return nil, fmt.Errorf("fastjson: unexpected end of 'null'")
		}
		r.pos += 4
	default:
		// number
		r.readNumberBytes()
		if r.pos == start {
			return nil, fmt.Errorf("fastjson: unexpected character %q at pos %d", r.data[r.pos], r.pos)
		}
	}
	return r.data[start:r.pos], nil
}

// skipObject skips an entire JSON object.
func (r *Reader) skipObject() error {
	if r.data[r.pos] != '{' {
		return fmt.Errorf("fastjson: expected '{' at pos %d", r.pos)
	}
	depth := 1
	r.pos++
	for r.pos < len(r.data) && depth > 0 {
		switch r.data[r.pos] {
		case '{':
			depth++
		case '}':
			depth--
		case '"':
			// skip string to avoid counting braces inside strings
			r.pos++
			for r.pos < len(r.data) {
				if r.data[r.pos] == '\\' {
					r.pos += 2
					continue
				}
				if r.data[r.pos] == '"' {
					break
				}
				r.pos++
			}
		}
		r.pos++
	}
	if depth != 0 {
		return fmt.Errorf("fastjson: unterminated object")
	}
	return nil
}

// skipArray skips an entire JSON array.
func (r *Reader) skipArray() error {
	if r.data[r.pos] != '[' {
		return fmt.Errorf("fastjson: expected '[' at pos %d", r.pos)
	}
	depth := 1
	r.pos++
	for r.pos < len(r.data) && depth > 0 {
		switch r.data[r.pos] {
		case '[':
			depth++
		case ']':
			depth--
		case '"':
			r.pos++
			for r.pos < len(r.data) {
				if r.data[r.pos] == '\\' {
					r.pos += 2
					continue
				}
				if r.data[r.pos] == '"' {
					break
				}
				r.pos++
			}
		}
		r.pos++
	}
	if depth != 0 {
		return fmt.Errorf("fastjson: unterminated array")
	}
	return nil
}
