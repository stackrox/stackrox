package dotnetcoreruntime

import (
	"strings"

	"github.com/facebookincubator/nvdtools/wfn"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/scanner/cpe/attributes/common"
	"github.com/stackrox/scanner/pkg/component"
)

// GetDotNetCoreRuntimeAttributes returns the [ASP].NET attributes for the given component.
func GetDotNetCoreRuntimeAttributes(c *component.Component) []*wfn.Attributes {
	vendorSet := set.NewStringSet("microsoft")

	nameSet := set.NewStringSet(c.Name, escapePeriod(c.Name))
	versionSet := set.NewStringSet(c.Version, escapePeriod(c.Version))
	return common.GenerateAttributesFromSets(vendorSet, nameSet, versionSet, "")
}

func escapePeriod(str string) string {
	return strings.ReplaceAll(str, ".", `\.`)
}
