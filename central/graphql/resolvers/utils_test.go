package resolvers

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestPaginationWrapper(t *testing.T) {
	stuff := []int{1, 2}
	result, _ := paginationWrapper{pv: &v1.QueryPagination{Offset: 0, Limit: 2}}.paginate(stuff, nil)
	rs := result.([]int)
	assert.Equal(t, []int{1, 2}, rs)

	stuff = []int{1, 2, 3}
	result, _ = paginationWrapper{pv: &v1.QueryPagination{Offset: 1, Limit: 2}}.paginate(stuff, nil)
	rs = result.([]int)
	assert.Equal(t, []int{2, 3}, rs)

	stuff = []int{1, 2, 3}
	result, _ = paginationWrapper{pv: &v1.QueryPagination{Offset: 2, Limit: 2}}.paginate(stuff, nil)
	rs = result.([]int)
	assert.Equal(t, []int{3}, rs)

	stuff = []int{1, 2}
	result, _ = paginationWrapper{pv: &v1.QueryPagination{Offset: 2, Limit: 2}}.paginate(stuff, nil)
	rs = result.([]int)
	assert.Equal(t, ([]int)(nil), rs)

	stuff = []int{}
	result, _ = paginationWrapper{pv: &v1.QueryPagination{Offset: 2, Limit: 2}}.paginate(stuff, nil)
	rs = result.([]int)
	assert.Equal(t, []int{}, rs)
}
