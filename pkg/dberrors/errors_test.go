package dberrors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrNotFound(t *testing.T) {
	err := New("foo", "bar")
	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, s.Code())
	assert.Equal(t, err.Error(), "foo 'bar' not found")
}
