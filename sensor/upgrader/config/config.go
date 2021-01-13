package config

import (
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/sensorupgrader"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"k8s.io/client-go/rest"
)

// UpgraderConfig contains (static) configuration that is relevant for the upgrader process.
type UpgraderConfig struct {
	ClusterID       string
	ProcessID       string
	CentralEndpoint string

	InCertRotationMode bool

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
	if c.Owner != nil && c.Owner.Namespace != common.Namespace {
		errs.AddStringf("owner %v is in disallowed namespace", c.Owner)
	}
	return errs.ToError()
}

// Create instantiates a new upgrader config using environment variables and well-known config files.
func Create() (*UpgraderConfig, error) {
	restConfig, err := loadKubeConfig()
	if err != nil {
		return nil, errors.Wrap(err, "obtaining Kubernetes API config")
	}

	// clusterID is optional and only required when fetching the bundle, not when used in standalone mode
	clusterID := os.Getenv(sensorupgrader.ClusterIDEnvVarName)

	centralEndpoint := os.Getenv(env.CentralEndpoint.EnvVar())
	if strings.HasPrefix(centralEndpoint, "ws://") || strings.HasPrefix(centralEndpoint, "wss://") {
		_, centralEndpoint = stringutils.Split2(centralEndpoint, "://")
	}
	cfg := &UpgraderConfig{
		ClusterID:          clusterID,
		ProcessID:          os.Getenv(upgradeProcessIDEnvVar),
		CentralEndpoint:    centralEndpoint,
		K8sRESTConfig:      restConfig,
		InCertRotationMode: env.UpgraderCertsOnly.BooleanSetting(),
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
