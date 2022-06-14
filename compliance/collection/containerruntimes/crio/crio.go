package crio

import (
	"net"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/collection/containerruntimes/common"
	collectionUtils "github.com/stackrox/rox/compliance/collection/utils"
	"github.com/stackrox/rox/generated/internalapi/compliance"
)

const (
	crioConfHostPath = "/etc/crio/crio.conf"
)

// GetContainerRuntimeData retrieves CRI-O specific information about the container runtime config.
func GetContainerRuntimeData() (*compliance.ContainerRuntimeInfo, error) {
	data, err := collectionUtils.ReadHostFile(crioConfHostPath)
	if err != nil {
		return nil, err
	}

	crioCfg, err := parseCRIOConfig(data)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse CRI-O config file %s", crioConfHostPath)
	}

	cri := &compliance.ContainerRuntimeInfo{
		InsecureRegistries: &compliance.InsecureRegistriesConfig{},
	}

	for _, insecureRegistry := range crioCfg.Image.InsecureRegistries {
		if _, _, err := net.ParseCIDR(insecureRegistry); err == nil {
			cri.InsecureRegistries.InsecureCidrs = append(cri.InsecureRegistries.InsecureCidrs, insecureRegistry)
		} else {
			cri.InsecureRegistries.InsecureRegistries = append(cri.InsecureRegistries.InsecureRegistries, insecureRegistry)
		}
	}

	common.AugmentInsecureRegistriesConfig(cri.InsecureRegistries)

	return cri, nil
}
