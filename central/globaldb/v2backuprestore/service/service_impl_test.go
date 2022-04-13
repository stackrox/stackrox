package service

import (
	"testing"

	"github.com/stackrox/stackrox/pkg/grpc/testutils"
)

func TestAuthzWorks(t *testing.T) {
	testutils.AssertAuthzWorks(t, newService(nil))
}
