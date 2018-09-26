// Package central generates configurations for the Central service.
package central

import (
	"bytes"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/zip"
)

var (
	log = logging.LoggerForModule()
)

// ExternalPersistence holds the data for a volume that is already created (e.g. docker volume, PV, etc)
type ExternalPersistence struct {
	Name         string
	MountPath    string
	StorageClass string
}

// HostPathPersistence describes the parameters for a bind mount
type HostPathPersistence struct {
	Name              string
	HostPath          string
	MountPath         string
	NodeSelectorKey   string
	NodeSelectorValue string
}

// CommonConfig contains the common config between orchestrators that cannot be placed at the top level
// Image is an example as it can be parameterized per orchestrator with different defaults so it cannot be placed
// at the top level
type CommonConfig struct {
	PreventImage  string
	ClairifyImage string
}

// K8sConfig contains k8s fields
type K8sConfig struct {
	CommonConfig

	ImagePullSecret string
	Namespace       string
	Registry        string

	// These variables are not prompted for by Cobra, but are set based on
	// provided inputs for use in templating.
	PreventImageTag  string
	ClairifyImageTag string
}

// SwarmConfig contains swarm fields
type SwarmConfig struct {
	CommonConfig

	NetworkMode string
	PublicPort  int
}

// Config configures the deployer for the central service.
type Config struct {
	ClusterType v1.ClusterType
	K8sConfig   *K8sConfig
	SwarmConfig *SwarmConfig

	External *ExternalPersistence
	HostPath *HostPathPersistence
	Features []features.FeatureFlag
}

type deployer interface {
	Render(Config) ([]*v1.File, error)
}

// Deployers contains all implementations for central deployment generators.
var Deployers = make(map[v1.ClusterType]deployer)

func executeTemplate(temp *template.Template, c Config) (string, error) {
	var b []byte
	buf := bytes.NewBuffer(b)
	err := temp.Execute(buf, c)
	if err != nil {
		log.Errorf("Template execution failed: %s", err)
		return "", err
	}
	return buf.String(), nil
}

func renderFilenames(filenames []string, c Config) ([]*v1.File, error) {
	var files []*v1.File
	for _, f := range filenames {
		t, err := templates.ReadFileAndTemplate(f)
		if err != nil {
			return nil, err
		}
		d, err := executeTemplate(t, c)
		if err != nil {
			return nil, err
		}
		files = append(files, zip.NewFile(filepath.Base(f), d, strings.HasSuffix(f, ".sh")))
	}
	return files, nil
}
