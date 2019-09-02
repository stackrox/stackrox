package config

import (
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/upgrader/k8sobjects"
	"k8s.io/client-go/rest"
)

// UpgraderConfig contains (static) configuration that is relevant for the upgrader process.
type UpgraderConfig struct {
	ClusterID       string
	ProcessID       string
	CentralEndpoint string

	K8sRESTConfig *rest.Config

	Owner *k8sobjects.ObjectRef
}

// Validate checks if this upgrader config is complete and well-formed. It does *not* check whether the values stored
// in this config actually work in practice.
func (c *UpgraderConfig) Validate() error {
	errs := errorhelpers.NewErrorList("validating upgrader config")
	if c.ProcessID != "" {
		if _, err := uuid.FromString(c.ProcessID); err != nil {
			errs.AddWrap(err, "upgrade process ID must be a valid UUID")
		}
	}
	if c.CentralEndpoint != "" {
		if _, _, _, err := netutil.ParseEndpoint(c.CentralEndpoint); err != nil {
			errs.AddWrapf(err, "central endpoint %q is invalid", c.CentralEndpoint)
		}
	}
	if c.K8sRESTConfig == nil {
		errs.AddString("kubernetes REST config not present")
	}
	return errs.ToError()
}

// Create instantiates a new upgrader config using environment variables and well-known config files.
func Create() (*UpgraderConfig, error) {
	restConfig, err := loadKubeConfig()
	if err != nil {
		return nil, errors.Wrap(err, "obtaining Kubernetes API config")
	}

	cfg := &UpgraderConfig{
		ClusterID:       env.ClusterID.Setting(),
		ProcessID:       os.Getenv(upgradeProcessIDEnvVar),
		CentralEndpoint: os.Getenv(env.CentralEndpoint.EnvVar()),
		K8sRESTConfig:   restConfig,
	}

	if ownerRefStr := os.Getenv(upgraderOwnerEnvVar); ownerRefStr != "" {
		owner, err := k8sobjects.ParseRef(ownerRefStr)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid owner reference string %q", ownerRefStr)
		}
		cfg.Owner = &owner
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
