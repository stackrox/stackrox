package python

import (
	"fmt"
	"strings"

	"github.com/facebookincubator/nvdtools/wfn"
	log "github.com/sirupsen/logrus"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/scanner/cpe/attributes/common"
	"github.com/stackrox/scanner/pkg/component"
)

var (
	// Core packages like python and pip are installed via package manager
	blocklistedPkgs = []string{"python", "docker", "pip"}

	// predisposedKeywords are the keywords that are likely to be used to specify
	// a version of a product that is not the core product. jira-plugin should not be resolved as jira for example
	// postgres-client is not postgres
	predisposedKeywords = []string{
		"plugin",
		"client",
		"python",
		"integration",
	}

	summaryExcludedKeywords = []string{
		"client",
		"plugin",
	}
)

func predisposed(c *component.Component) bool {
	for _, keyword := range predisposedKeywords {
		if strings.Contains(c.Name, keyword) {
			return true
		}
	}
	return false
}

func ignored(c *component.Component) bool {
	for _, excluded := range summaryExcludedKeywords {
		// Ignore all clients, plugins, etc if they don't have the substring python so
		// we should capture things like python-dns or pydns and then these will be predisposed and not split
		// py is more prone to false positives as normal words have py, but it's proven to be fairly effective
		nameLower := strings.ToLower(c.Name)
		if strings.Contains(nameLower, excluded) {
			continue
		}
		if strings.Contains(nameLower, "py") {
			continue
		}
		if strings.Contains(strings.ToLower(c.PythonPkgMetadata.Description), excluded) {
			log.Debugf("Python pkg ignored: %q - description %q contained %q", c.Name, c.PythonPkgMetadata.Description, excluded)
			return true
		}
		if strings.Contains(strings.ToLower(c.PythonPkgMetadata.Summary), excluded) {
			log.Debugf("Python pkg ignored: %q - summary %q contained %q", c.Name, c.PythonPkgMetadata.Summary, excluded)
			return true
		}
	}
	return false
}

func parseAuthorEmailAsVendor(email string) string {
	startIdx := strings.Index(email, "@")
	if startIdx != -1 && startIdx != len(email)-1 {
		shortened := email[startIdx+1:]
		endIdx := strings.Index(shortened, ".")
		if endIdx == -1 {
			return shortened
		}
		return shortened[:endIdx]
	}
	return ""
}

// GetPythonAttributes returns the Python-related attributes for the given component.
func GetPythonAttributes(c *component.Component) []*wfn.Attributes {
	python := c.PythonPkgMetadata
	if python == nil {
		return nil
	}
	if ignored(c) {
		return nil
	}

	vendorSet := set.NewStringSet()
	versionSet := common.GenerateVersionKeys(c)
	nameSet := common.GenerateNameKeys(c)

	// Post filtering
	if vendorSet.Cardinality() != 0 && !predisposed(c) {
		common.AddMutatedNameKeys(c, nameSet)
	}
	for _, blocklisted := range blocklistedPkgs {
		nameSet.Remove(blocklisted)
	}

	if python.Homepage != "" {
		url := strings.TrimPrefix(python.Homepage, "http://")
		url = strings.TrimPrefix(url, "https://")
		url = strings.TrimPrefix(url, "www.")
		if idx := strings.Index(url, "."); idx != -1 {
			vendorSet.Add(url[:idx])
		}
	}
	if python.AuthorEmail != "" {
		if vendor := parseAuthorEmailAsVendor(python.AuthorEmail); vendor != "" {
			vendorSet.Add(vendor)
		}
	}
	if strings.HasPrefix(python.DownloadURL, "https://pypi.org/project/") {
		project := strings.TrimPrefix(python.DownloadURL, "https://pypi.org/project/")
		project = strings.TrimSuffix(project, "/")
		vendorSet.Add(strings.ToLower(fmt.Sprintf("%s_project", project)))
	}
	vendorSet.Add("python")
	// purposefully add an empty vendor. This will be evaluated in the python validator
	vendorSet.Add("")
	return common.GenerateAttributesFromSets(vendorSet, nameSet, versionSet, "")
}
