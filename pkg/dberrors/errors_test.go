package dberrors

import (
	"testing"

	grpc_errors "github.com/stackrox/rox/pkg/grpc/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestErrNotFound(t *testing.T) {
	err := New("foo", "bar")
	s := grpc_errors.ErrToGrpcStatus(err)
	assert.NotNil(t, s)
	assert.Equal(t, codes.NotFound, s.Code())
}
