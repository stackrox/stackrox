package resolvers

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestPagination(t *testing.T) {
	stuff := []int{1, 2}
	qp := &v1.QueryPagination{}
	qp.SetOffset(0)
	qp.SetLimit(2)
	result, _ := paginate(qp, stuff, nil)
	assert.Equal(t, []int{1, 2}, result)

	stuff = []int{1, 2, 3}
	qp2 := &v1.QueryPagination{}
	qp2.SetOffset(1)
	qp2.SetLimit(2)
	result, _ = paginate(qp2, stuff, nil)
	assert.Equal(t, []int{2, 3}, result)

	stuff = []int{1, 2, 3}
	qp3 := &v1.QueryPagination{}
	qp3.SetOffset(2)
	qp3.SetLimit(2)
	result, _ = paginate(qp3, stuff, nil)
	assert.Equal(t, []int{3}, result)

	stuff = []int{1, 2}
	qp4 := &v1.QueryPagination{}
	qp4.SetOffset(2)
	qp4.SetLimit(2)
	result, _ = paginate(qp4, stuff, nil)
	assert.Nil(t, result)

	stuff = []int{}
	qp5 := &v1.QueryPagination{}
	qp5.SetOffset(2)
	qp5.SetLimit(2)
	result, _ = paginate(qp5, stuff, nil)
	assert.Equal(t, []int{}, result)
}
