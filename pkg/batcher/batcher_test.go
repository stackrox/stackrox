package batcher

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ordering struct {
	start, end int
	valid      bool
}

func TestBatcher(t *testing.T) {
	var cases = []struct {
		totalSize int
		batchSize int
		ordering  []ordering
	}{
		{
			totalSize: 0,
			batchSize: 0,
			ordering: []ordering{
				{
					valid: false,
				},
			},
		},
		{
			totalSize: 0,
			batchSize: 50,
			ordering: []ordering{
				{
					start: 0,
					end:   0,
					valid: false,
				},
			},
		},
		{
			totalSize: 1,
			batchSize: 50,
			ordering: []ordering{
				{
					start: 0,
					end:   1,
					valid: true,
				},
				{
					valid: false,
				},
			},
		},
		{
			totalSize: 100,
			batchSize: 50,
			ordering: []ordering{
				{
					start: 0,
					end:   50,
					valid: true,
				},
				{
					start: 50,
					end:   100,
					valid: true,
				},
				{
					valid: false,
				},
			},
		},
		{
			totalSize: 3,
			batchSize: 2,
			ordering: []ordering{
				{
					start: 0,
					end:   2,
					valid: true,
				},
				{
					start: 2,
					end:   3,
					valid: true,
				},
				{
					valid: false,
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%d-%d", c.totalSize, c.batchSize), func(t *testing.T) {
			b := New(c.totalSize, c.batchSize)
			for _, o := range c.ordering {
				start, end, ok := b.Next()
				assert.Equal(t, o.start, start)
				assert.Equal(t, o.end, end)
				assert.Equal(t, o.valid, ok)
			}
		})
	}
}
