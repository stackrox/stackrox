package manager

import (
	"testing"

	licenseproto "github.com/stackrox/rox/generated/shared/license"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func TestIsStackRoxLicense(t *testing.T) {
	t.Parallel()

	roxLicensees := []string{
		"mi@stackrox.com",
		"support@stackrox.com",
		"circleci-rox@stackrox-ci.iam.gserviceaccount.com",
	}

	for _, roxLicensee := range roxLicensees {
		md := &licenseproto.License_Metadata{
			LicensedForId: roxLicensee,
		}

		assert.Truef(t, isStackRoxLicense(md), "Expected licensee %s to be indicative of a StackRox license", roxLicensee)
	}

	nonRoxLicenses := []string{
		"mi@example.com",
		"123456789",
		uuid.NewV4().String(),
		"fakerox@someproject.iam.gserviceaccount.com",
	}

	for _, nonRoxLicensee := range nonRoxLicenses {
		md := &licenseproto.License_Metadata{
			LicensedForId: nonRoxLicensee,
		}

		assert.Falsef(t, isStackRoxLicense(md), "Expected licensee %s to be indicative of a non-StackRox license", nonRoxLicensee)
	}
}
