package clusters

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/zip"
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

func executeTemplate(temp *template.Template, fields map[string]interface{}) (string, error) {
	var b []byte
	buf := bytes.NewBuffer(b)
	err := temp.Execute(buf, fields)
	if err != nil {
		log.Errorf("template execution: %s", err)
		return "", err
	}
	return buf.String(), nil
}

func generateCollectorImage(preventImage, tag string) string {
	img := types.Wrapper{Image: utils.GenerateImageFromString(preventImage)}
	registry := img.GetName().GetRegistry()
	if registry == "stackrox.io" {
		registry = "collector.stackrox.io"
	}
	remote := img.Namespace() + "/collector"
	// This handles the case where there is no namespace. e.g. stackrox.io/collector:latest
	if img.Repo() == "" {
		remote = "collector"
	}
	return fmt.Sprintf("%s/%s:%s", registry, remote, tag)
}

func fieldsFromWrap(c Wrap) map[string]interface{} {
	fields := map[string]interface{}{
		"ImageEnv":              env.Image.EnvVar(),
		"Image":                 c.PreventImage,
		"PublicEndpointEnv":     env.CentralEndpoint.EnvVar(),
		"PublicEndpoint":        c.CentralApiEndpoint,
		"ClusterIDEnv":          env.ClusterID.EnvVar(),
		"ClusterID":             c.Id,
		"AdvertisedEndpointEnv": env.AdvertisedEndpoint.EnvVar(),
		"AdvertisedEndpoint":    env.AdvertisedEndpoint.Setting(),
		"RuntimeSupport":        c.RuntimeSupport,
		"CollectorImage":        generateCollectorImage(c.PreventImage, version.GetCollectorVersion()),
	}
	return fields
}

func renderFilenames(filenames []string, c map[string]interface{}) ([]*v1.File, error) {
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
