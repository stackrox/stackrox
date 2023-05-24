package scancomponent

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cvss"
)

// ScanComponent is the interface which encompasses potentially vulnerable components of entites
// (ex: image component or node component).
type ScanComponent interface {
	GetName() string
	GetVersion() string
	GetVulns() []cvss.VulnI
}

// NewFromImageComponent returns a instance of ScanComponent created from image component.
func NewFromImageComponent(comp *storage.EmbeddedImageScanComponent) ScanComponent {
	ret := &scanComponentImpl{
		name:    comp.GetName(),
		version: comp.GetVersion(),
	}
	for _, vuln := range comp.GetVulns() {
		ret.vulns = append(ret.vulns, cvss.NewFromEmbeddedVulnerability(vuln))
	}
	return ret
}

// NewFromNodeComponent returns a instance of ScanComponent created from node component.
func NewFromNodeComponent(comp *storage.EmbeddedNodeScanComponent) ScanComponent {
	ret := &scanComponentImpl{
		name:    comp.GetName(),
		version: comp.GetVersion(),
	}
	for _, vuln := range comp.GetVulnerabilities() {
		ret.vulns = append(ret.vulns, cvss.NewFromNodeVulnerability(vuln))
	}
	return ret
}

type scanComponentImpl struct {
	name    string
	version string
	vulns   []cvss.VulnI
}

func (c *scanComponentImpl) GetName() string {
	return c.name
}

func (c *scanComponentImpl) GetVersion() string {
	return c.version
}

func (c *scanComponentImpl) GetVulns() []cvss.VulnI {
	return c.vulns
}
