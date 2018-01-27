// Package central generates configurations for the Central service.
package central

import (
	"bytes"
	"text/template"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.New("pkg/central")
)

const (
	commandPrefix = `#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"
`
)

// Config configures the deployer for the central service.
type Config struct {
	Image      string
	PublicPort int
	Namespace  string
}

type deployer interface {
	Deployment(Config) (string, error)
	Command(Config) (string, error)
}

// Deployers contains all implementations for central deployment generators.
var Deployers = make(map[v1.ClusterType]deployer)

type basicDeployer struct {
	deploy *template.Template
	cmd    *template.Template
}

// Deployment returns an orchestrator-specific configuration file that the user
// can use to deploy a sensor.
func (d basicDeployer) Deployment(c Config) (string, error) {
	var b []byte
	buf := bytes.NewBuffer(b)
	err := d.deploy.Execute(buf, c)
	if err != nil {
		log.Errorf("Template execution failed: %s", err)
		return "", err
	}
	return buf.String(), nil
}

// Command returns an orchestrator-specific command that the user can use with
// the downloaded deployment specification to deploy a sensor.
func (d basicDeployer) Command(c Config) (string, error) {
	var b []byte
	buf := bytes.NewBuffer(b)
	err := d.cmd.Execute(buf, c)
	if err != nil {
		log.Errorf("Template execution failed: %s", err)
		return "", err
	}
	return buf.String(), nil
}
