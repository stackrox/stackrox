package clusters

import (
	"bytes"
	"strconv"
	"text/template"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/env"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
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

// Command returns an orchestrator-specific command that the user can use with
// the downloaded deployment specification to deploy a sensor.
func (c Wrap) Command() (string, error) {
	dep, ok := deployers[c.Type]
	if !ok {
		return "", status.Errorf(codes.Unimplemented, "Cluster type %s is not currently implemented", c.Type.String())
	}

	return dep.Command(c)
}

// Deployment returns an orchestrator-specific configuration file that the user
// can use to deploy a sensor.
func (c Wrap) Deployment() (string, error) {
	dep, ok := deployers[c.Type]
	if !ok {
		return "", status.Errorf(codes.Unimplemented, "Cluster type %s is not currently implemented", c.Type.String())
	}

	return dep.Deployment(c)
}

type deployer interface {
	Deployment(Wrap) (string, error)
	Command(Wrap) (string, error)
}

var deployers = make(map[v1.ClusterType]deployer)

type basicDeployer struct {
	deploy    *template.Template
	cmd       *template.Template
	addFields func(Wrap, map[string]string)
}

func (d basicDeployer) Deployment(c Wrap) (string, error) {
	var b []byte
	buf := bytes.NewBuffer(b)
	err := d.deploy.Execute(buf, d.fields(c))
	if err != nil {
		log.Errorf("%s deployment template execution: %s", c.Type.String(), err)
		return "", err
	}
	return buf.String(), nil
}

func (d basicDeployer) Command(c Wrap) (string, error) {
	var b []byte
	buf := bytes.NewBuffer(b)
	err := d.cmd.Execute(buf, d.fields(c))
	if err != nil {
		log.Errorf("%s deployment template execution: %s", c.Type.String(), err)
		return "", err
	}
	return buf.String(), nil
}

func (d basicDeployer) fields(c Wrap) map[string]string {
	fields := map[string]string{
		"ImageEnv":              env.Image.EnvVar(),
		"Image":                 c.PreventImage,
		"PublicEndpointEnv":     env.CentralEndpoint.EnvVar(),
		"PublicEndpoint":        c.CentralApiEndpoint,
		"ClusterIDEnv":          env.ClusterID.EnvVar(),
		"ClusterID":             c.Id,
		"AdvertisedEndpointEnv": env.AdvertisedEndpoint.EnvVar(),
		"AdvertisedEndpoint":    env.AdvertisedEndpoint.Setting(),
		"DisableSwarmTLS":       strconv.FormatBool(c.DisableSwarmTls),
	}
	if d.addFields != nil {
		d.addFields(c, fields)
	}
	return fields
}
