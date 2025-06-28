package manifest

import (
	"fmt"
	"github.com/stackrox/rox/pkg/utils"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"strings"
)

const (
	localStackroxImage = "localhost:5001/stackrox/stackrox:latest"
	localDbImage       = "localhost:5001/stackrox/db:latest"
)

type Config struct {
	Action               string `yaml:"action"`
	Namespace            string `yaml:"namespace"`
	ScannerV4            bool   `yaml:"scannerV4"`
	DevMode              bool   `yaml:"devMode"`
	ApplyNetworkPolicies bool   `yaml:"applyNetworkPolicies"`
	CertPath             string `yaml:"certPath"`
	Images               Images `yaml:"images"`
}

type Images struct {
	AdmissionControl string `yaml:"admissionControl"`
	Sensor           string `yaml:"sensor"`
	ConfigController string `yaml:"configController"`
	Central          string `yaml:"central"`
	CentralDB        string `yaml:"centralDb"`
	Scanner          string `yaml:"scanner"`
	ScannerDB        string `yaml:"scannerDb"`
	ScannerV4        string `yaml:"scannerv4"`
	ScannerV4DB      string `yaml:"scannerv4Db"`
	Collector        string `yaml:"collector"`
}

var DefaultConfig Config = Config{
	Namespace:            "stackrox",
	ScannerV4:            false,
	DevMode:              false,
	ApplyNetworkPolicies: false,
	CertPath:             "./certs",
	Images: Images{
		AdmissionControl: localStackroxImage,
		Sensor:           localStackroxImage,
		ConfigController: localStackroxImage,
		Central:          localStackroxImage,
		Scanner:          localStackroxImage,
		ScannerV4:        localStackroxImage,
		Collector:        localStackroxImage,
		CentralDB:        localDbImage,
		ScannerDB:        localDbImage,
		ScannerV4DB:      localDbImage,
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
