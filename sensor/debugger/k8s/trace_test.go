package k8s

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkWrite(b *testing.B) {
	dir := b.TempDir()
	data := []byte("abc")
	writer, err := NewTraceWriter(path.Join(dir, "test"))
	assert.NoError(b, err)
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, err := writer.Write(data)
		assert.NoError(b, err)
	}
}

func TestWriter(t *testing.T) {
	writer, err := NewTraceWriter("")
	assert.Error(t, err)
	assert.Nil(t, writer)

	dir := t.TempDir()
	filePath := path.Join(dir, "test")

	writer, err = NewTraceWriter(filePath)
	assert.NoError(t, err)
	assert.NotNil(t, writer)

	n, err := writer.Write([]byte("abc"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)

	n, err = writer.Write([]byte("1337"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)

	err = writer.Close()
	assert.NoError(t, err)
	err = writer.Close()
	assert.Error(t, err)

	n, err = writer.Write([]byte("fail"))
	assert.Error(t, err)
	assert.Equal(t, 0, n)

	data, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, data, []byte("abc\n1337\n"))
}
