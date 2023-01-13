package helmconfig

import (
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/jsonutil"
)

const (
	configFile      = "/run/secrets/stackrox.io/helm-cluster-config/config.yaml"
	clusterNameFile = "/run/secrets/stackrox.io/helm-effective-cluster-name/cluster-name"
)

// Load loads the cluster configuration for Helm-managed cluster from its canonical location.
func Load() (*central.HelmManagedConfigInit, error) {
	contents, err := os.ReadFile(configFile)
	if err != nil {
		return nil, errors.Wrap(err, "loading cluster config file")
	}
	return load(contents)
}

func load(data []byte) (*central.HelmManagedConfigInit, error) {
	contentsJSON, err := yaml.YAMLToJSON(data)
	if err != nil {
		return nil, errors.Wrap(err, "converting cluster config YAML to JSON")
	}

	var config central.HelmManagedConfigInit
	if err := jsonutil.JSONBytesToProto(contentsJSON, &config); err != nil {
		return nil, errors.Wrap(err, "unmarshaling config proto")
	}

	return &config, nil
}

// getEffectiveClusterName returns the cluster name which is currently used within central.
func getEffectiveClusterName() (string, error) {
	name, err := os.ReadFile(clusterNameFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(name)), nil
}
