// Package central generates configurations for the Central service.
package central

import (
	"bytes"
	"text/template"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

const (
	commandPrefix = `#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
`
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
