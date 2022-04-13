package service

import (
	"testing"

	"github.com/stackrox/stackrox/pkg/grpc/testutils"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &service{})
}
