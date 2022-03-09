package cluster

import (
	"fmt"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/urlfmt"
)

// Validate validates a cluster object
func Validate(cluster *storage.Cluster) *errorhelpers.ErrorList {
	errorList := ValidatePartial(cluster)

	// Here go all other server-side checks
	if _, err := reference.ParseAnyReference(cluster.GetMainImage()); err != nil {
		errorList.AddError(errors.Wrapf(err, "invalid main image '%s'", cluster.GetMainImage()))
	}

	return errorList
}

// IsManagerManualOrUnknown returns whether a given manager is Manual or Unknown
func IsManagerManualOrUnknown(manager storage.ManagerType) bool {
	return manager == storage.ManagerType_MANAGER_TYPE_MANUAL || manager == storage.ManagerType_MANAGER_TYPE_UNKNOWN
}

// ValidatePartial partially validates a cluster object.
// Some fields are allowed to be left empty for the server to complete with default values.
func ValidatePartial(cluster *storage.Cluster) *errorhelpers.ErrorList {
	errorList := errorhelpers.NewErrorList("Cluster Validation")
	if cluster.GetName() == "" {
		errorList.AddString("Cluster name is required")
	}
	if cluster.GetMainImage() != "" {
		if imageWithoutTag, err := utils.DropImageTagAndDigest(cluster.GetMainImage()); err != nil {
			errorList.AddError(err)
		} else if imageWithoutTag != cluster.GetMainImage() && IsManagerManualOrUnknown(cluster.GetManagedBy()) {
			errorList.AddString("main image should not contain tags or digests")
		}
	}
	if cluster.GetCollectorImage() != "" {
		if imageWithoutTag, err := utils.DropImageTagAndDigest(cluster.GetCollectorImage()); err != nil {
			errorList.AddError(err)
		} else if imageWithoutTag != cluster.GetCollectorImage() && IsManagerManualOrUnknown(cluster.GetManagedBy()) {
			errorList.AddString("collector image should not contain tags or digests")
		}
	}
	if cluster.GetCentralApiEndpoint() == "" {
		errorList.AddString("Central API Endpoint is required")
	} else if !strings.Contains(cluster.GetCentralApiEndpoint(), ":") {
		errorList.AddString("Central API Endpoint must have port specified")
	}

	if stringutils.ContainsWhitespace(cluster.GetCentralApiEndpoint()) {
		errorList.AddString("Central API endpoint cannot contain whitespace")
	}

	if cluster.GetAdmissionControllerEvents() && cluster.Type == storage.ClusterType_OPENSHIFT_CLUSTER {
		errorList.AddString("OpenShift 3.x compatibility mode does not support admission controller webhooks on port-forward and exec.")
	}
	if !cluster.GetDynamicConfig().GetDisableAuditLogs() && cluster.Type != storage.ClusterType_OPENSHIFT4_CLUSTER {
		// Note: this will not fail server-side validation, because on those paths, normalization (which forces it to
		// true for incompatible clusters) happens prior to validation.
		errorList.AddString("Audit log collection is only supported on OpenShift 4.x clusters")
	}
	centralEndpoint := urlfmt.FormatURL(cluster.GetCentralApiEndpoint(), urlfmt.NONE, urlfmt.NoTrailingSlash)
	_, _, _, err := netutil.ParseEndpoint(centralEndpoint)
	if err != nil {
		errorList.AddString(fmt.Sprintf("Central API Endpoint must be a valid endpoint. Error: %s", err))
	}
	cluster.CentralApiEndpoint = centralEndpoint

	return errorList
}
