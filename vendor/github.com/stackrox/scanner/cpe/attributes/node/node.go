package node

import (
	"github.com/facebookincubator/nvdtools/wfn"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/scanner/cpe/attributes/common"
	"github.com/stackrox/scanner/pkg/component"
)

// GetNodeAttributes returns the attributes from the given NPM component.
func GetNodeAttributes(c *component.Component) []*wfn.Attributes {
	vendorSet := set.NewStringSet()
	nameSet := common.GenerateNameKeys(c)
	versionSet := common.GenerateVersionKeys(c)

	return common.GenerateAttributesFromSets(vendorSet, nameSet, versionSet, `node\.js`)
}
