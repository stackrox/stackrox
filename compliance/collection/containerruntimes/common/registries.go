package common

import (
	"github.com/BurntSushi/toml"
	"github.com/containers/image/v5/pkg/sysregistriesv2"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/collection/utils"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log = logging.LoggerForModule()
)

type tomlConfig struct {
	sysregistriesv2.V1RegistriesConf
	sysregistriesv2.V2RegistriesConf
}

// This function was taken from containers/image and adapted.
func loadRegistryConf(configHostPath string) (*sysregistriesv2.V2RegistriesConf, error) {
	var config tomlConfig

	configBytes, err := utils.ReadHostFile(configHostPath)
	if err != nil {
		return nil, err
	}

	if err := toml.Unmarshal(configBytes, &config); err != nil {
		return nil, errors.Wrapf(err, "unmarshaling TOML data in config file %s", configHostPath)
	}

	if config.V1RegistriesConf.Nonempty() {
		if config.V2RegistriesConf.Nonempty() {
			return nil, errors.Errorf("registries config file %s contains both V1 and V2 data", configHostPath)
		}
		v2Cfg, err := config.V1RegistriesConf.ConvertToV2()
		return v2Cfg, errors.Wrapf(err, "converting V1 registries config in file %s", configHostPath)
	}
	return &config.V2RegistriesConf, nil
}

// AugmentInsecureRegistriesConfig augments the information about insecure registries with information from the
// /etc/containers/registries.conf file.
func AugmentInsecureRegistriesConfig(info *compliance.InsecureRegistriesConfig) {
	cfg, err := loadRegistryConf("/etc/containers/registries.conf")
	if err != nil {
		log.Warnf("Failed to load common container registries config: %v", err)
		return
	}

	registriesSet := set.NewStringSet(info.GetInsecureRegistries()...)
	for _, registry := range cfg.Registries {
		if !registry.Insecure {
			continue
		}
		location := registry.Prefix
		if location == "" {
			location = registry.Location
		}
		if location == "" {
			location = "<unknown registry>"
		}

		if registriesSet.Add(location) {
			info.InsecureRegistries = append(info.InsecureRegistries, location)
		}
	}
}
