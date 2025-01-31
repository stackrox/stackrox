package k8s

import (
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
