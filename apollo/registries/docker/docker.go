package docker

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"bitbucket.org/stack-rox/apollo/apollo/registries"
	registryTypes "bitbucket.org/stack-rox/apollo/apollo/registries/types"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/heroku/docker-registry-client/registry"
)

const pluginName = "docker"

type dockerRegistry struct {
	config   map[string]string
	endpoint string
	hub      *registry.Registry
	registry string
}

type v1Config struct {
	Cmd []string `json:"Cmd"`
}

// Parse out the layer JSON
type v1Compatibility struct {
	Created time.Time `json:"created"`
	Author  string    `json:"author"`
	Config  v1Config  `json:"container_config"`
}

func newRegistry(endpoint string, config map[string]string) (registryTypes.ImageRegistry, error) {
	username, ok := config["username"]
	if !ok {
		return nil, errors.New("Config parameter 'username' must be defined for docker registry plugin")
	}
	password, ok := config["password"]
	if !ok {
		return nil, errors.New("Config parameter 'password' must be defined for docker registry plugin")
	}

	url := endpoint
	if !strings.HasPrefix(endpoint, "http") {
		url = "https://" + endpoint
	}
	hub, err := registry.New(url, username, password)
	if err != nil {
		return nil, err
	}
	return &dockerRegistry{
		config:   config,
		endpoint: endpoint,
		hub:      hub,
		registry: endpoint,
	}, nil
}

var instructions = []string{
	"ADD",
	"ARG",
	"CMD",
	"COPY",
	"ENTRYPOINT",
	"ENV",
	"EXPOSE",
	"FROM",
	"HEALTHCHECK",
	"LABEL",
	"MAINTAINER",
	"ONBUILD",
	"RUN",
	"SHELL",
	"STOPSIGNAL",
	"USER",
	"VOLUME",
	"WORKDIR",
}

var scrubPrefixes = []string{
	"/bin/sh -c #(nop)",
	"/bin/sh -c ",
}

func scrubDockerfileLines(compat v1Compatibility) *v1.ImageLayer {
	line := strings.Join(compat.Config.Cmd, " ")
	for _, scrubPrefix := range scrubPrefixes {
		line = strings.TrimPrefix(line, scrubPrefix)
	}
	line = strings.Join(strings.Fields(line), " ")
	var lineInstruction string
	for _, instruction := range instructions {
		if strings.HasPrefix(line, instruction) {
			lineInstruction = instruction
			line = strings.TrimPrefix(line, instruction+" ")
			break
		}
	}
	if lineInstruction == "" {
		lineInstruction = "RUN"
	}
	return &v1.ImageLayer{
		Instruction: lineInstruction,
		Value:       line,
		Created:     compat.Created.UnixNano(),
		Author:      compat.Author,
	}
}

// Metadata returns the metadata via this registries implementation
func (d *dockerRegistry) Metadata(image *v1.Image) (*v1.ImageMetadata, error) {
	manifest, err := d.hub.Manifest(image.Remote, image.Tag)
	if err != nil {
		return nil, err
	}

	// Get the latest layer and author
	var latest v1.ImageLayer
	var layers []*v1.ImageLayer
	for _, layer := range manifest.History {
		var compat v1Compatibility
		if err := json.Unmarshal([]byte(layer.V1Compatibility), &compat); err != nil {
			return nil, err
		}
		layer := scrubDockerfileLines(compat)
		if layer.Created > latest.Created {
			latest = *layer
		}
		layers = append(layers, layer)
	}
	imageMetadata := &v1.ImageMetadata{
		Created: latest.Created,
		Author:  latest.Author,
		Layers:  layers,
	}
	return imageMetadata, nil
}

func (d *dockerRegistry) Config() map[string]string {
	return d.config
}

func (d *dockerRegistry) Endpoint() string {
	return d.endpoint
}

func (d *dockerRegistry) Type() string {
	return pluginName
}

// Test tests the current registry and makes sure that it is working properly
func (d *dockerRegistry) Test() error {
	_, err := d.hub.Repositories()
	return err
}

func init() {
	registries.Registry["docker"] = newRegistry
}
