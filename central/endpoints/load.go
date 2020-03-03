package endpoints

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

func loadFromFile(path string) (*Config, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "loading endpoints config from file %q", path)
	}

	var cfg Config
	if err := yaml.Unmarshal(bytes, &cfg); err != nil {
		return nil, errors.Wrapf(err, "unmarshaling endpoints config YAML from file %s", path)
	}
	return &cfg, nil
}
