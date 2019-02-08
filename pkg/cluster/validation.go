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
		errorList.AddError(fmt.Errorf("invalid image '%s': %s", cluster.GetMainImage(), err))
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
