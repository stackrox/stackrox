package ruby

import (
	"github.com/facebookincubator/nvdtools/wfn"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/scanner/cpe/attributes/common"
	"github.com/stackrox/scanner/pkg/component"
)

// GetRubyAttributes gets the Ruby-related attributes from the given component.
func GetRubyAttributes(c *component.Component) []*wfn.Attributes {
	vendorSet := set.NewStringSet()
	nameSet := common.GenerateNameKeys(c)
	versionSet := common.GenerateVersionKeys(c)
	return common.GenerateAttributesFromSets(vendorSet, nameSet, versionSet, "")
}
