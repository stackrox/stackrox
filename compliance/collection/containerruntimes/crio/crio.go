package crio

import (
	"net"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/collection/containerruntimes/common"
	complianceUtils "github.com/stackrox/rox/compliance/utils"
	"github.com/stackrox/rox/generated/internalapi/compliance"
)

const (
	crioConfHostPath = "/etc/crio/crio.conf"
)

// GetContainerRuntimeData retrieves CRI-O specific information about the container runtime config.
func GetContainerRuntimeData() (*compliance.ContainerRuntimeInfo, error) {
	data, err := complianceUtils.ReadHostFile(crioConfHostPath)
	if err != nil {
		return nil, err
	}

	crioCfg, err := parseCRIOConfig(data)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse CRI-O config file %s", crioConfHostPath)
	}

	cri := &compliance.ContainerRuntimeInfo{}
	cri.SetInsecureRegistries(&compliance.InsecureRegistriesConfig{})

	for _, insecureRegistry := range crioCfg.Image.InsecureRegistries {
		if _, _, err := net.ParseCIDR(insecureRegistry); err == nil {
			cri.GetInsecureRegistries().SetInsecureCidrs(append(cri.GetInsecureRegistries().GetInsecureCidrs(), insecureRegistry))
		} else {
			cri.GetInsecureRegistries().SetInsecureRegistries(append(cri.GetInsecureRegistries().GetInsecureRegistries(), insecureRegistry))
		}
	}

	common.AugmentInsecureRegistriesConfig(cri.GetInsecureRegistries())

	return cri, nil
}
