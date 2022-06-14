package service

import (
	"testing"

	"github.com/stackrox/rox/pkg/grpc/testutils"
)

func TestAuthzWorks(t *testing.T) {
	testutils.AssertAuthzWorks(t, New(nil, nil))
}
