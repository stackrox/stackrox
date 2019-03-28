package cluster

import (
	"fmt"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/stringutils"
)

// Validate validates a cluster object
func Validate(cluster *storage.Cluster) *errorhelpers.ErrorList {
	errorList := errorhelpers.NewErrorList("Cluster Validation")
	if cluster.GetName() == "" {
		errorList.AddString("Cluster name is required")
	}
	if _, err := reference.ParseAnyReference(cluster.GetMainImage()); err != nil {
		errorList.AddError(fmt.Errorf("invalid main image '%s': %s", cluster.GetMainImage(), err))
	}
	if cluster.GetCollectorImage() != "" {
		ref, err := reference.ParseAnyReference(cluster.GetCollectorImage())
		if err != nil {
			errorList.AddError(fmt.Errorf("invalid collector image '%s': %s", cluster.GetCollectorImage(), err))
		}
		namedTagged, ok := ref.(reference.NamedTagged)
		if ok {
			errorList.AddError(fmt.Errorf("collector image may not specify a tag.  Please "+
				"remove tag '%s' to continue", namedTagged.Tag()))
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
	return errorList
}
