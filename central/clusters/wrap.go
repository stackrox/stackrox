package clusters

import (
	"bytes"
	"text/template"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log = logging.LoggerForModule()
)

const (
	commandPrefix = `#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
`
)

// Wrap adds additional functionality to a v1.Cluster.
type Wrap v1.Cluster

// NewDeployer takes in a cluster and returns the cluster implementation
func NewDeployer(c *v1.Cluster) (Deployer, error) {
	dep, ok := deployers[c.Type]
	if !ok {
		return nil, status.Errorf(codes.Unimplemented, "Cluster type %s is not currently implemented", c.Type.String())
	}
	return dep, nil
}

// Deployer is the interface that defines how to get the specific files per orchestrator
type Deployer interface {
	Render(Wrap) ([]*v1.File, error)
}

var deployers = make(map[v1.ClusterType]Deployer)

func executeTemplate(temp *template.Template, fields map[string]string) (string, error) {
	var b []byte
	buf := bytes.NewBuffer(b)
	err := temp.Execute(buf, fields)
	if err != nil {
		log.Errorf("template execution: %s", err)
		return "", err
	}
	return buf.String(), nil
}

func fieldsFromWrap(c Wrap) map[string]string {
	fields := map[string]string{
		"ImageEnv":              env.Image.EnvVar(),
		"Image":                 c.PreventImage,
		"PublicEndpointEnv":     env.CentralEndpoint.EnvVar(),
		"PublicEndpoint":        c.CentralApiEndpoint,
		"ClusterIDEnv":          env.ClusterID.EnvVar(),
		"ClusterID":             c.Id,
		"AdvertisedEndpointEnv": env.AdvertisedEndpoint.EnvVar(),
		"AdvertisedEndpoint":    env.AdvertisedEndpoint.Setting(),
	}
	return fields
}
