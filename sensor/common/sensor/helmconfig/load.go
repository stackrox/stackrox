package helmconfig

import (
	"bytes"
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
)

const (
	configFile = "/run/secrets/stackrox.io/helm-cluster-config/config.yaml"
)

// Load loads the cluster configuration for Helm-managed cluster from its canonical location.
func Load() (*central.HelmManagedConfigInit, error) {
	contents, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, errors.Wrap(err, "loading cluster config file")
	}

	contentsJSON, err := yaml.YAMLToJSON(contents)
	if err != nil {
		return nil, errors.Wrap(err, "converting cluster config YAML to JSON")
	}

	var config central.HelmManagedConfigInit
	if err := jsonpb.Unmarshal(bytes.NewReader(contentsJSON), &config); err != nil {
		return nil, errors.Wrap(err, "unmarshaling config proto")
	}

	return &config, nil
}
