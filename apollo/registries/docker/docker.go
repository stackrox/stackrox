package docker

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"bitbucket.org/stack-rox/apollo/apollo/registries"
	registryTypes "bitbucket.org/stack-rox/apollo/apollo/registries/types"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/heroku/docker-registry-client/registry"
)

var (
	log = logging.New("registry/docker")
)

type dockerRegistry struct {
	protoRegistry *v1.Registry

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

func newRegistry(protoRegistry *v1.Registry) (*dockerRegistry, error) {
	username, ok := protoRegistry.Config["username"]
	if !ok {
		return nil, errors.New("Config parameter 'username' must be defined for docker registry plugin")
	}
	password, ok := protoRegistry.Config["password"]
	if !ok {
		return nil, errors.New("Config parameter 'password' must be defined for docker registry plugin")
	}

	url := protoRegistry.Endpoint
	if !strings.HasPrefix(protoRegistry.Endpoint, "http") {
		url = "https://" + protoRegistry.Endpoint
	}
	hub, err := registry.New(url, username, password)
	if err != nil {
		return nil, err
	}
	return &dockerRegistry{
		protoRegistry: protoRegistry,
		hub:           hub,
		registry:      protoRegistry.Endpoint,
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
	protoTS, err := ptypes.TimestampProto(compat.Created)
	if err != nil {
		log.Error(err)
	}
	return &v1.ImageLayer{
		Instruction: lineInstruction,
		Value:       line,
		Created:     protoTS,
		Author:      compat.Author,
	}
}

func compareProtoTimestamps(t1, t2 *timestamp.Timestamp) bool {
	if t1 == nil {
		return true
	}
	if t2 == nil {
		return false
	}
	if t1.Seconds < t2.Seconds {
		return true
	} else if t2.Seconds > t1.Seconds {
		return false
	}
	return t1.Nanos < t2.Nanos
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
		if compareProtoTimestamps(latest.Created, layer.Created) {
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

func (d *dockerRegistry) ProtoRegistry() *v1.Registry {
	return d.protoRegistry
}

// Test tests the current registry and makes sure that it is working properly
func (d *dockerRegistry) Test() error {
	_, err := d.hub.Repositories()
	return err
}

func init() {
	registries.Registry["docker"] = func(registry *v1.Registry) (registryTypes.ImageRegistry, error) {
		reg, err := newRegistry(registry)
		return reg, err
	}
}
