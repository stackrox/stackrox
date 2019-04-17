package service

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/testutils"
)

func TestServiceAuthz_Lockdown(t *testing.T) {
	t.Parallel()

	var licenseStatus v1.Metadata_LicenseStatus
	testutils.AssertAuthzWorks(t, newService(&licenseStatus, nil))
}

func TestServiceAuthz_NonLockdown(t *testing.T) {
	t.Parallel()

	var licenseStatus v1.Metadata_LicenseStatus
	testutils.AssertAuthzWorks(t, newService(&licenseStatus, nil))
}
