package endpoints

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/tlsconfig"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()

	allowMisdirectedRequestsEnv = env.RegisterBooleanSetting("ROX_ALLOW_MISDIRECTED_REQUESTS", false)
)

func loadAllConfigs() ([]EndpointConfig, error) {
	cfg := &defaultConfig

	if ok, err := fileutils.Exists(endpointsConfigPath); err != nil {
		log.Errorf("Could not determine if endpoints config file %s exists: %v", endpointsConfigPath, err)
	} else if ok {
		loadedCfg, err := loadFromFile(endpointsConfigPath)
		if err != nil {
			return nil, errors.Wrapf(err, "could not load endpoints configuration file %s: %v", endpointsConfigPath, err)
		}
		cfg = loadedCfg
	}

	var allEndpointCfgs []EndpointConfig

	if !cfg.DisableDefault {
		allEndpointCfgs = append(allEndpointCfgs, defaultEndpoint)
	}

	allEndpointCfgs = append(allEndpointCfgs, ParseLegacySpec(env.PlaintextEndpoints.Setting(), &plaintextTLSConfig)...)
	allEndpointCfgs = append(allEndpointCfgs, ParseLegacySpec(env.SecureEndpoints.Setting(), &defaultTLSConfig)...)

	allEndpointCfgs = append(allEndpointCfgs, cfg.Endpoints...)

	return allEndpointCfgs, nil
}

// InstantiateAll loads and instantiates all endpoint configurations. It considers the endpoint config from the config
// file, as well as all the ROX_PLAINTEXT_ENDPOINTS and the ROX_SECURE_ENDPOINTS environment variables.
func InstantiateAll(tlsMgr tlsconfig.Manager) ([]*grpc.EndpointConfig, error) {
	endpointCfgs, err := loadAllConfigs()
	if err != nil {
		return nil, errors.Wrap(err, "loading endpoint configurations")
	}

	instantiatedCfgs := make([]*grpc.EndpointConfig, 0, len(endpointCfgs))

	for _, endpointCfg := range endpointCfgs {
		instantiatedCfg, err := endpointCfg.Instantiate(tlsMgr)
		if err != nil {
			if endpointCfg.Optional {
				log.Warnf("Error instantiating optional endpoint listening at %q: %v", endpointCfg.Listen, err)
			} else {
				return nil, errors.Wrapf(err, "instantiating required endpoint listening at %q", endpointCfg.Listen)
			}
		} else {
			instantiatedCfg.DenyMisdirectedRequests = !allowMisdirectedRequestsEnv.BooleanSetting()
			instantiatedCfgs = append(instantiatedCfgs, instantiatedCfg)
		}
	}

	return instantiatedCfgs, nil
}
