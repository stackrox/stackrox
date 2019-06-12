package pagination

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestPaginationCalculation(t *testing.T) {
	type expected struct {
		first, last int
	}

	cases := []struct {
		name                   string
		length, constMaxLength int
		pagination             *v1.Pagination
		expected               expected
	}{
		{
			name:           "less than max length",
			length:         10,
			constMaxLength: 20,
			expected: expected{
				first: 0,
				last:  10,
			},
		},
		{
			name:           "greater than max length",
			length:         30,
			constMaxLength: 20,
			expected: expected{
				first: 0,
				last:  20,
			},
		},
		{
			name:   "pagination regular case no offset",
			length: 30,
			pagination: &v1.Pagination{
				Offset: 0,
				Limit:  5,
			},
			expected: expected{
				first: 0,
				last:  5,
			},
		},
		{
			name:   "pagination regular case with offset",
			length: 30,
			pagination: &v1.Pagination{
				Offset: 5,
				Limit:  10,
			},
			expected: expected{
				first: 5,
				last:  15,
			},
		},
		{
			name:   "pagination offset > length",
			length: 10,
			pagination: &v1.Pagination{
				Offset: 15,
				Limit:  10,
			},
			expected: expected{
				first: 0,
				last:  0,
			},
		},
		{
			name:   "pagination offset + limit > length",
			length: 10,
			pagination: &v1.Pagination{
				Offset: 5,
				Limit:  10,
			},
			expected: expected{
				first: 5,
				last:  10,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			first, last := CalculatePaginationIndices(c.length, c.constMaxLength, c.pagination)
			assert.Equal(t, c.expected.first, first)
			assert.Equal(t, c.expected.last, last)
		})
	}
}
