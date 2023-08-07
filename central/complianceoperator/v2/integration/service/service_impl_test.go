package service

import (
	"testing"

	"github.com/stackrox/rox/pkg/grpc/testutils"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}
