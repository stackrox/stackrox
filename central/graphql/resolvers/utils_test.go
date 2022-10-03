package resolvers

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestPagination(t *testing.T) {
	stuff := []int{1, 2}
	result, _ := paginate(&v1.QueryPagination{Offset: 0, Limit: 2}, stuff, nil)
	assert.Equal(t, []int{1, 2}, result)

	stuff = []int{1, 2, 3}
	result, _ = paginate(&v1.QueryPagination{Offset: 1, Limit: 2}, stuff, nil)
	assert.Equal(t, []int{2, 3}, result)

	stuff = []int{1, 2, 3}
	result, _ = paginate(&v1.QueryPagination{Offset: 2, Limit: 2}, stuff, nil)
	assert.Equal(t, []int{3}, result)

	stuff = []int{1, 2}
	result, _ = paginate(&v1.QueryPagination{Offset: 2, Limit: 2}, stuff, nil)
	assert.Equal(t, ([]int)(nil), result)

	stuff = []int{}
	result, _ = paginate(&v1.QueryPagination{Offset: 2, Limit: 2}, stuff, nil)
	assert.Equal(t, []int{}, result)
}
