// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package zapcore

import (
	"fmt"
	"sync"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/internal/bufferpool"
)

const (
	defaultConsoleFieldOrder = "TLNC"

	fieldTime   rune = 'T'
	fieldLevel  rune = 'L'
	fieldName   rune = 'N'
	fieldCaller rune = 'C'
)

var _sliceEncoderPool = sync.Pool{
	New: func() interface{} {
		return &sliceArrayEncoder{elems: make([]interface{}, 0, 2)}
	},
}

func getSliceEncoder() *sliceArrayEncoder {
	return _sliceEncoderPool.Get().(*sliceArrayEncoder)
}

func putSliceEncoder(e *sliceArrayEncoder) {
	e.elems = e.elems[:0]
	_sliceEncoderPool.Put(e)
}

type preludeField struct {
	handler  preludeFieldHandler
	addColon bool
}

type preludeFieldHandler func(c consoleEncoder, ent Entry, fields []Field, enc PrimitiveArrayEncoder)

func timePreludeFieldHandler(c consoleEncoder, ent Entry, fields []Field, enc PrimitiveArrayEncoder) {
	c.EncodeTime(ent.Time, enc)
}

func levelPreludeFieldHandler(c consoleEncoder, ent Entry, fields []Field, enc PrimitiveArrayEncoder) {
	c.EncodeLevel(ent.Level, enc)
}

func namePreludeFieldHandler(c consoleEncoder, ent Entry, fields []Field, enc PrimitiveArrayEncoder) {
	if ent.LoggerName != "" {
		c.EncodeName(ent.LoggerName, enc)
	}
}

func callerPreludeFieldHandler(c consoleEncoder, ent Entry, fields []Field, enc PrimitiveArrayEncoder) {
	if ent.Caller.Defined {
		c.EncodeCaller(ent.Caller, enc)
	}
}

type consoleEncoder struct {
	*jsonEncoder

	preludeFields []preludeField
}

// NewConsoleEncoder creates an encoder whose output is designed for human -
// rather than machine - consumption. It serializes the core log entry data
// (message, level, timestamp, etc.) in a plain-text format and leaves the
// structured context as JSON.
//
// Note that although the console encoder doesn't use the keys specified in the
// encoder configuration, it will omit any element whose key is set to the empty
// string.
func NewConsoleEncoder(cfg EncoderConfig) Encoder {
	if len(cfg.ConsoleSeparator) == 0 {
		// Use a default delimiter of '\t' for backwards compatibility
		cfg.ConsoleSeparator = "\t"
	}
	if len(cfg.ConsoleFieldOrder) == 0 {
		cfg.ConsoleFieldOrder = defaultConsoleFieldOrder
	}
	if cfg.EncodeName == nil {
		cfg.EncodeName = FullNameEncoder
	}

	ce := consoleEncoder{
		jsonEncoder: newJSONEncoder(cfg, true),
	}

	for _, field := range ce.ConsoleFieldOrder {
		switch field {
		case fieldTime:
			if ce.TimeKey != "" && ce.EncodeTime != nil {
				ce.preludeFields = append(ce.preludeFields, preludeField{
					handler: timePreludeFieldHandler,
				})
			}
		case fieldLevel:
			if ce.LevelKey != "" && ce.EncodeLevel != nil {
				ce.preludeFields = append(ce.preludeFields, preludeField{
					handler: levelPreludeFieldHandler,
				})
			}
		case fieldName:
			if ce.NameKey != "" {
				ce.preludeFields = append(ce.preludeFields, preludeField{
					handler: namePreludeFieldHandler,
				})
			}
		case fieldCaller:
			if ce.CallerKey != "" && ce.EncodeCaller != nil {
				ce.preludeFields = append(ce.preludeFields, preludeField{
					handler: callerPreludeFieldHandler,
				})
			}
		case ':':
			if l := len(ce.preludeFields); l > 0 {
				ce.preludeFields[l-1].addColon = true
			}
		}
	}

	return ce
}

func (c consoleEncoder) Clone() Encoder {
	clone := consoleEncoder{
		jsonEncoder:   c.jsonEncoder.Clone().(*jsonEncoder),
		preludeFields: c.preludeFields,
	}
	copy(c.preludeFields, clone.preludeFields)
	return clone
}

func (c consoleEncoder) EncodeEntry(ent Entry, fields []Field) (*buffer.Buffer, error) {
	line := bufferpool.Get()

	// We don't want the entry's metadata to be quoted and escaped (if it's
	// encoded as strings), which means that we can't use the JSON encoder. The
	// simplest option is to use the memory encoder and fmt.Fprint.
	//
	// If this ever becomes a performance bottleneck, we can implement
	// ArrayEncoder for our plain-text format.
	arr := getSliceEncoder()

	for _, field := range c.preludeFields {
		field.handler(c, ent, fields, arr)
	}

	for i := range arr.elems {
		if i > 0 {
			line.AppendString(c.ConsoleSeparator)
		}
		fmt.Fprint(line, arr.elems[i])
		if c.preludeFields[i].addColon {
			line.AppendString(":")
		}
	}
	putSliceEncoder(arr)

	// Add the message itself.
	if c.MessageKey != "" {
		c.addSeparatorIfNecessary(line)
		line.AppendString(ent.Message)
	}

	// Add any structured context.
	c.writeContext(line, fields)

	// If there's no stacktrace key, honor that; this allows users to force
	// single-line output.
	if ent.Stack != "" && c.StacktraceKey != "" {
		line.AppendByte('\n')
		line.AppendString(ent.Stack)
	}

	if c.LineEnding != "" {
		line.AppendString(c.LineEnding)
	} else {
		line.AppendString(DefaultLineEnding)
	}
	return line, nil
}

func (c consoleEncoder) writeContext(line *buffer.Buffer, extra []Field) {
	context := c.jsonEncoder.Clone().(*jsonEncoder)
	defer context.buf.Free()

	addFields(context, extra)
	context.closeOpenNamespaces()
	if context.buf.Len() == 0 {
		return
	}

	c.addSeparatorIfNecessary(line)
	line.AppendByte('{')
	line.Write(context.buf.Bytes())
	line.AppendByte('}')
}

func (c consoleEncoder) addSeparatorIfNecessary(line *buffer.Buffer) {
	if line.Len() > 0 {
		line.AppendString(c.ConsoleSeparator)
	}
}
