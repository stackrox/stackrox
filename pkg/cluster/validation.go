package cluster

import (
	stdErrors "errors"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/urlfmt"
)

// Validate validates a cluster object
func Validate(cluster *storage.Cluster) error {
	errs := ValidatePartial(cluster)

	// Here go all other server-side checks
	if _, err := reference.ParseAnyReference(cluster.GetMainImage()); err != nil {
		errs = stdErrors.Join(err, errors.Wrapf(err, "invalid main image %q", cluster.GetMainImage()))
	}

	return errs
}

// ValidatePartial partially validates a cluster object.
// Some fields are allowed to be left empty for the server to complete with default values.
func ValidatePartial(cluster *storage.Cluster) error {
	var validationErrs error
	if cluster.GetName() == "" {
		validationErrs = stdErrors.Join(validationErrs, errox.InvalidArgs.New("Cluster name is required"))
	}
	if cluster.GetMainImage() != "" {
		if _, err := reference.ParseAnyReference(cluster.GetMainImage()); err != nil {
			validationErrs = stdErrors.Join(validationErrs,
				errors.Wrapf(err, "invalid main image %q", cluster.GetMainImage()))
		}
	}
	if cluster.GetCollectorImage() != "" {
		ref, err := reference.ParseAnyReference(cluster.GetCollectorImage())
		if err != nil {
			validationErrs = stdErrors.Join(validationErrs,
				errors.Wrapf(err, "invalid collector image %q", cluster.GetCollectorImage()))
		}

		if cluster.GetHelmConfig() == nil {
			namedTagged, ok := ref.(reference.NamedTagged)
			if ok {
				validationErrs = stdErrors.Join(validationErrs, errox.InvalidArgs.Newf(
					"collector image may not specify a tag.  Please "+
						"remove tag %q to continue", namedTagged.Tag()))
			}
		}
	}
	if cluster.GetCentralApiEndpoint() == "" {
		validationErrs = stdErrors.Join(validationErrs, errox.InvalidArgs.New("Central API Endpoint is required"))
	} else if !strings.Contains(cluster.GetCentralApiEndpoint(), ":") {
		validationErrs = stdErrors.Join(validationErrs,
			errox.InvalidArgs.New("Central API Endpoint must have port specified"))
	}

	if stringutils.ContainsWhitespace(cluster.GetCentralApiEndpoint()) {
		validationErrs = stdErrors.Join(validationErrs, errox.InvalidArgs.New("Central API endpoint cannot contain whitespace"))
	}

	if cluster.GetAdmissionControllerEvents() && cluster.Type == storage.ClusterType_OPENSHIFT_CLUSTER {
		validationErrs = stdErrors.Join(validationErrs,
			errox.InvalidArgs.New("OpenShift 3.x compatibility mode does not support admission controller webhooks on port-forward and exec"))
	}
	if !cluster.GetDynamicConfig().GetDisableAuditLogs() && cluster.Type != storage.ClusterType_OPENSHIFT4_CLUSTER {
		// Note: this will not fail server-side validation, because on those paths, normalization (which forces it to
		// true for incompatible clusters) happens prior to validation.
		validationErrs = stdErrors.Join(validationErrs,
			errox.InvalidArgs.New("Audit log collection is only supported on OpenShift 4.x clusters"))
	}
	centralEndpoint := urlfmt.FormatURL(cluster.GetCentralApiEndpoint(), urlfmt.NONE, urlfmt.NoTrailingSlash)
	_, _, _, err := netutil.ParseEndpoint(centralEndpoint)
	if err != nil {
		validationErrs = stdErrors.Join(validationErrs, errors.Wrap(err, "Central API Endpoint must be a valid endpoint"))
	}
	cluster.CentralApiEndpoint = centralEndpoint

	return errors.Wrap(validationErrs, "cluster validation")
}
