package timeutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeChunks(t *testing.T) {
	tr := TimeRange{}
	assert.True(t, tr.Done())

	from := time.Now()
	to := from.Add(150 * time.Minute)

	chunk := 1 * time.Hour

	chunks := []*TimeRange{}

	tr = TimeRange{from, to}
	for !tr.Done() {
		chunks = append(chunks, tr.Next(chunk))
	}

	expected := []*TimeRange{
		{from, from.Add(chunk)},                // full chunk.
		{from.Add(chunk), from.Add(2 * chunk)}, // full chunk.
		{from.Add(2 * chunk), to},              // half chunk.
	}
	assert.Equal(t, expected, chunks)
}
