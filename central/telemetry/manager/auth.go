package manager

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pkg/errors"
	licenseManager "github.com/stackrox/rox/central/license/manager"
	licenseproto "github.com/stackrox/rox/generated/shared/license"
)

// createAuthToken returns an authentication token for the license server.
func createAuthToken(licenseMD *licenseproto.License_Metadata, currTime time.Time, licenseMgr licenseManager.LicenseManager) (string, error) {
	authTokenPrefix := fmt.Sprintf("%s:%d", licenseMD.GetId(), currTime.Unix())
	signature, err := licenseMgr.SignWithLicenseKeyHash(licenseMD.GetId(), []byte(authTokenPrefix))
	if err != nil {
		return "", errors.Wrap(err, "could not sign token with license key")
	}
	authToken := fmt.Sprintf("%s:%s", authTokenPrefix, hex.EncodeToString(signature))
	return base64.RawStdEncoding.EncodeToString([]byte(authToken)), nil
}
