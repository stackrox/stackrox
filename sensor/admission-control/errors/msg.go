package errors

import (
	"fmt"

	"github.com/stackrox/stackrox/generated/storage"
)

// ImageScanUnavailableMsg return message indicating inability to handle policies requiring image scans.
func ImageScanUnavailableMsg(policy *storage.Policy) string {
	return fmt.Sprintf("Policy %q (%s) depends on the existence of image scans. To enforce this policy "+
		"please enable the 'Contact image scanners' option", policy.GetName(), policy.GetId())
}
