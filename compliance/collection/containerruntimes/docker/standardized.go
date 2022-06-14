package docker

import (
	"github.com/stackrox/rox/generated/internalapi/compliance"
	internalTypes "github.com/stackrox/rox/pkg/docker/types"
)

func toStandardizedInfo(dockerData *internalTypes.Data) *compliance.ContainerRuntimeInfo {
	insecureRegs := &compliance.InsecureRegistriesConfig{}

	if regCfg := dockerData.Info.RegistryConfig; regCfg != nil {
		for _, cidr := range regCfg.InsecureRegistryCIDRs {
			insecureRegs.InsecureCidrs = append(insecureRegs.InsecureCidrs, cidr.String())
		}
		for _, idxCfg := range regCfg.IndexConfigs {
			if !idxCfg.Secure {
				insecureRegs.InsecureRegistries = append(insecureRegs.InsecureRegistries, idxCfg.Name)
			}
		}
	}
	return &compliance.ContainerRuntimeInfo{
		InsecureRegistries: insecureRegs,
	}
}
