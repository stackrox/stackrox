package service

import (
	"testing"

	"github.com/stackrox/rox/pkg/grpc/testutils"
)

func TestServiceAuthz(t *testing.T) {
	t.Parallel()

	testutils.AssertAuthzWorks(t, newService())
}
