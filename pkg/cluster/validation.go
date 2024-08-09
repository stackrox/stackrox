package cluster

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/distribution/reference"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/urlfmt"
)

const (
	// maxClusterNameLength is inspired by Subdomain vaildation as in RFC 1123
	maxClusterNameLength int = 253
	// Taken from RFC 1123 Subdomain Regex
	// (source: https://github.com/kubernetes/apimachinery/blob/d794766488ac2892197a7cc8d0b4b46b0edbda80/pkg/util/validation/validation.go#L205-L224)
	// and modified by: (1) allowing underscores, (2) spaces, and (3) uppercase characters.
	labelFmt          string = "[a-zA-Z0-9]([-_a-zA-Z0-9]*[a-zA-Z0-9])?"
	subdomainFmt      string = labelFmt + "([\\. ]" + labelFmt + ")*"
	clusterNameErrMsg string = "cluster name must consist of alphanumeric characters, spaces, '-', or '_', and must start and end with an alphanumeric character"
)

var clusterNameRegexp = regexp.MustCompile("^" + subdomainFmt + "$")

// Validate validates a cluster object
func Validate(cluster *storage.Cluster) *errorhelpers.ErrorList {
	errorList := ValidatePartial(cluster)

	// Here go all other server-side checks
	if _, err := reference.ParseAnyReference(cluster.GetMainImage()); err != nil {
		errorList.AddError(errors.Wrapf(err, "invalid main image '%s'", cluster.GetMainImage()))
	}

	return errorList
}

// ValidatePartial partially validates a cluster object.
// Some fields are allowed to be left empty for the server to complete with default values.
func ValidatePartial(cluster *storage.Cluster) *errorhelpers.ErrorList {
	errorList := errorhelpers.NewErrorList("Cluster Validation")
	if cluster.GetName() == "" {
		errorList.AddString("Cluster name is required")
	}
	if cluster.GetMainImage() != "" {
		if _, err := reference.ParseAnyReference(cluster.GetMainImage()); err != nil {
			errorList.AddError(errors.Wrapf(err, "invalid main image '%s'", cluster.GetMainImage()))
		}
	}
	if cluster.GetCollectorImage() != "" {
		ref, err := reference.ParseAnyReference(cluster.GetCollectorImage())
		if err != nil {
			errorList.AddError(errors.Wrapf(err, "invalid collector image '%s'", cluster.GetCollectorImage()))
		}

		if cluster.GetHelmConfig() == nil {
			namedTagged, ok := ref.(reference.NamedTagged)
			if ok {
				errorList.AddStringf("collector image may not specify a tag.  Please "+
					"remove tag '%s' to continue", namedTagged.Tag())
			}
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

// IsNameValid  is added to avoid problems with rendering of helm templates.
// The cluster names may be cast to various types in Helm and that
// may lead to the name being changed - e.g., a number may be represented
// in scientific notation.
// Because the fix in the Helm chart is tricky, we reject all names that
// could be parsable to a number or any other Helm construct.
func IsNameValid(name string) error {
	if len(name) == 0 {
		return errors.New("cluster name cannot be empty")
	}
	if len(name) > maxClusterNameLength {
		return fmt.Errorf("cluster name cannot be longer than %d characters", maxClusterNameLength)
	}
	if !clusterNameRegexp.MatchString(name) {
		return errors.New(clusterNameErrMsg)
	}
	if _, err := strconv.ParseFloat(name, 64); err == nil {
		return errors.New("cluster name cannot be a number")
	}
	if _, err := strconv.ParseInt(name, 0, 64); err == nil {
		return errors.New("cluster name cannot be a number")
	}
	if name == "true" || name == "false" {
		return errors.New("cluster name cannot be a representation of a boolean value")
	}
	if name == "null" {
		return errors.New("cluster name cannot be a representation of a null value")
	}
	return nil
}
