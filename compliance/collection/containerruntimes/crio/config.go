package crio

import "github.com/BurntSushi/toml"

type crioConfigFile struct {
	Crio crioConfig `toml:"crio"`
}

type crioConfig struct {
	Image crioImageConfig `toml:"image"`
}

type crioImageConfig struct {
	InsecureRegistries []string `toml:"insecure_registries"`
}

// parseCRIOConfig parses the relevant fragment of the CRI-O config.
func parseCRIOConfig(data []byte) (*crioConfig, error) {
	var file crioConfigFile
	if err := toml.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	return &file.Crio, nil
}
