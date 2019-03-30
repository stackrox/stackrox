package service

import (
	"testing"

	"github.com/stackrox/rox/pkg/grpc/testutils"
)

func TestServiceAuthz_Lockdown(t *testing.T) {
	t.Parallel()

	testutils.AssertAuthzWorks(t, newService(true))
}

func TestServiceAuthz_NonLockdown(t *testing.T) {
	t.Parallel()

	testutils.AssertAuthzWorks(t, newService(false))
}
