package writer

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestWriteLogsErrorReturnError(t *testing.T) {
	err := WriteLogs(errorWriter{}, logs())
	assert.NotNil(t, err)
}

func TestWriteLogsNoLogsWritesNothing(t *testing.T) {
	var b bytes.Buffer
	err := WriteLogs(&b, []*storage.LogImbue{})
	assert.Nil(t, err)
	assert.Equal(t, "[]", b.String())
}

func TestWriteLogsLogSurroundedByBraces(t *testing.T) {
	var b bytes.Buffer
	err := WriteLogs(&b, logs())
	assert.Nil(t, err)
	s := b.String()
	assert.True(t, strings.HasPrefix(s, "["))
	assert.True(t, strings.HasSuffix(s, "]"))
}

func TestWriteLogsLogsOnSeparateLines(t *testing.T) {
	var b bytes.Buffer
	err := WriteLogs(&b, logs())
	assert.Nil(t, err)
	s := b.String()
	assert.True(t, strings.Contains(s, "\n"))
}

func TestWriteNonJSONLogAddsErrorToOutput(t *testing.T) {
	var b bytes.Buffer
	input := "not a json"
	err := WriteLogs(&b, []*storage.LogImbue{{Log: []byte(input)}})
	assert.Nil(t, err)
	s := b.String()
	assert.True(t, strings.Contains(s, "encodingError"))
	assert.True(t, strings.Contains(s, input))
}

type errorWriter struct{}

func (w errorWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("i'm an error, how about you")
}

func logs() []*storage.LogImbue {
	return []*storage.LogImbue{
		{
			Log: []byte(`{"json": "object"}`),
		},
		{
			Log: []byte(`{"weirdstring" : "**&&^^%%$$"}`),
		},
		{
			Log: []byte(`[{},{"hey": "hehehehey"}]`),
		},
	}
}
