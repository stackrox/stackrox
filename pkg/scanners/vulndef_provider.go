package scanners

import (
	"errors"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// VulnDefsInfoProvider provides functionality to obtain vulnerability definitions information.
type VulnDefsInfoProvider interface {
	GetVulnDefsInfo() (*v1.VulnDefinitionsInfo, error)
}

// NewVulnDefsInfoProvider returns new instance of NewVulnDefsInfoProvider.
func NewVulnDefsInfoProvider(scanners Set) VulnDefsInfoProvider {
	return &vulnDefsInfoProviderImpl{
		scanners: scanners,
	}
}

type vulnDefsInfoProviderImpl struct {
	scanners Set
}

func (p *vulnDefsInfoProviderImpl) GetVulnDefsInfo() (*v1.VulnDefinitionsInfo, error) {
	if len(p.scanners.GetAll()) == 0 {
		return nil, errors.New("no image integrations found")
	}

	for _, scanner := range p.scanners.GetAll() {
		info, err := scanner.GetScanner().GetVulnDefinitionsInfo()
		if err != nil {
			return nil, err
		}

		if info != nil {
			return info, nil
		}
	}
	return nil, errors.New("no vulnerability definitions information found")
}
