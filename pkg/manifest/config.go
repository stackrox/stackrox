package manifest

import (
	"fmt"
	"github.com/stackrox/rox/pkg/utils"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"strings"
)

type Images struct {
	Stackrox string `yaml:"stackrox"`
	DB       string `yaml:"db"`
}

type Config struct {
	Namespace string `yaml:"namespace"`
	ScannerV4 bool   `yaml:"scannerV4"`
	Images    Images `yaml:"images"`
}

var DefaultConfig Config = Config{
	Namespace: "stackrox",
	ScannerV4: false,
	Images: Images{
		Stackrox: "localhost:5001/stackrox/stackrox:latest",
		DB:       "localhost:5001/stackrox/db:latest",
	},
}

func ReadConfig(filename string) (*Config, error) {
	if filename == "" {
		cfg := DefaultConfig
		return &cfg, nil
	}
	r, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer utils.IgnoreError(r.Close)
	return load(r)
}

func load(r io.Reader) (*Config, error) {
	yd := yaml.NewDecoder(r)
	yd.KnownFields(true)
	cfg := DefaultConfig
	if err := yd.Decode(&cfg); err != nil {
		msg := strings.TrimPrefix(err.Error(), `yaml: `)
		return nil, fmt.Errorf("malformed yaml: %v", msg)
	}
	return &cfg, nil
}
