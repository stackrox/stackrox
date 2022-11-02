package common

import (
	"regexp"
	"strings"

	"github.com/facebookincubator/nvdtools/wfn"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/scanner/pkg/component"
)

var (
	numRegex = regexp.MustCompile(`[0-9].*$`)
)

// GenerateVersionKeys generates versions based on the given component.
func GenerateVersionKeys(c *component.Component) set.StringSet {
	return set.NewStringSet(c.Version, strings.ReplaceAll(c.Version, ".", `\.`))
}

// GenerateNameKeys generates names based on the given component.
func GenerateNameKeys(c *component.Component) set.StringSet {
	componentName := c.Name
	if componentName == "" {
		return set.NewStringSet()
	}
	return set.NewStringSet(
		componentName,
		strings.ReplaceAll(componentName, "_", "-"),
		strings.ReplaceAll(componentName, "-", "_"),
	)
}

// AddMutatedNameKeys adds mutated keys to teh given nameSet based on the given component.
func AddMutatedNameKeys(c *component.Component, nameSet set.StringSet) {
	nameSet.Add(strings.TrimRight(numRegex.ReplaceAllString(c.Name, ""), "-_"))
	for name := range nameSet {
		if idx := strings.Index(name, "-"); idx != -1 {
			nameSet.Add(name[:idx])
		}
	}
}

// GenerateAttributesFromSets returns the cross-product of the given attributes.
func GenerateAttributesFromSets(vendors, names, versions set.StringSet, targetSW string) []*wfn.Attributes {
	if vendors.Cardinality() == 0 {
		vendors.Add("")
	}
	attributes := make([]*wfn.Attributes, 0, vendors.Cardinality()*names.Cardinality()*versions.Cardinality())
	for vendor := range vendors {
		for name := range names {
			for version := range versions {
				attributes = append(attributes, &wfn.Attributes{
					Vendor:   strings.ToLower(vendor),
					Product:  strings.ToLower(name),
					Version:  strings.ToLower(version),
					TargetSW: targetSW,
				})
			}
		}
	}
	return attributes
}
