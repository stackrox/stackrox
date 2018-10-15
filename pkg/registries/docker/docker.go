package docker

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	manifestV1 "github.com/docker/distribution/manifest/schema1"
	ptypes "github.com/gogo/protobuf/types"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/urlfmt"
)

var (
	log = logging.LoggerForModule()
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, func(integration *v1.ImageIntegration) (types.ImageRegistry, error)) {
	return "docker", func(integration *v1.ImageIntegration) (types.ImageRegistry, error) {
		reg, err := newRegistry(integration)
		return reg, err
	}
}

// Registry is the basic docker registry implementation
type Registry struct {
	cfg                   Config
	protoImageIntegration *v1.ImageIntegration

	client *registry.Registry

	url      string
	registry string // This is the registry portion of the image
}

// Config is the basic config for the docker registry
type Config struct {
	// Endpoint defines the Docker Registry URL
	Endpoint string
	// Username defines the Username for the Docker Registry
	Username string
	// Password defines the password for the Docker Registry
	Password string
	// Insecure defines if the registry should be insecure
	Insecure bool
}

type v1Config struct {
	Cmd []string `json:"Cmd"`
}

// Parse out the layer JSON
type v1Compatibility struct {
	ID      string    `json:"id"`
	Created time.Time `json:"created"`
	Author  string    `json:"author"`
	Config  v1Config  `json:"container_config"`
}

// NewDockerRegistry creates a new instantiation of the docker registry
// TODO(cgorman) AP-386 - properly put the base docker registry into another pkg
func NewDockerRegistry(cfg Config, integration *v1.ImageIntegration) (*Registry, error) {
	url, err := urlfmt.FormatURL(cfg.Endpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if err != nil {
		return nil, err
	}
	// if the registryServer endpoint contains docker.io then the image will be docker.io/namespace/repo:tag
	registryServer := urlfmt.GetServerFromURL(url)
	if strings.Contains(cfg.Endpoint, "docker.io") {
		registryServer = "docker.io"
	}

	var client *registry.Registry
	if cfg.Insecure {
		client, err = registry.NewInsecure(url, cfg.Username, cfg.Password)
	} else {
		client, err = registry.New(url, cfg.Username, cfg.Password)
	}
	if err != nil {
		return nil, err
	}

	return &Registry{
		url:                   url,
		registry:              registryServer,
		client:                client,
		cfg:                   cfg,
		protoImageIntegration: integration,
	}, nil
}

func newRegistry(integration *v1.ImageIntegration) (*Registry, error) {
	dockerConfig, ok := integration.IntegrationConfig.(*v1.ImageIntegration_Docker)
	if !ok {
		return nil, fmt.Errorf("Docker configuration required")
	}
	cfg := Config{
		Endpoint: dockerConfig.Docker.GetEndpoint(),
		Username: dockerConfig.Docker.GetUsername(),
		Password: dockerConfig.Docker.GetPassword(),
		Insecure: dockerConfig.Docker.GetInsecure(),
	}
	return NewDockerRegistry(cfg, integration)
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
	for instruction := range types.DockerfileInstructionSet {
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

func (d *Registry) getV2Metadata(image *v1.Image) *v1.V2Metadata {
	metadata, err := d.client.ManifestV2(image.GetName().GetRemote(), utils.Reference(image))
	if err != nil {
		return nil
	}
	layers := make([]string, 0, len(metadata.Layers))
	for _, layer := range metadata.Layers {
		layers = append(layers, layer.Digest.String())
	}
	return &v1.V2Metadata{
		Digest: metadata.Config.Digest.String(),
		Layers: layers,
	}
}

// Match decides if the image is contained within this registry
func (d *Registry) Match(image *v1.Image) bool {
	return d.registry == image.GetName().GetRegistry()
}

// Global returns whether or not this registry is available from all clusters
func (d *Registry) Global() bool {
	return len(d.protoImageIntegration.GetClusters()) == 0
}

func (d *Registry) populateV1DataFromManifest(manifest *manifestV1.SignedManifest) (*v1.ImageMetadata, error) {
	// Get the latest layer and author
	var latest v1.ImageLayer
	var layers []*v1.ImageLayer
	for _, layer := range manifest.History {
		var compat v1Compatibility
		if err := json.Unmarshal([]byte(layer.V1Compatibility), &compat); err != nil {
			return nil, fmt.Errorf("Failed unmarshalling v1 capability: %s", err)
		}
		layer := scrubDockerfileLines(compat)
		if compareProtoTimestamps(latest.Created, layer.Created) {
			latest = *layer
		}
		layers = append(layers, layer)
	}
	fsLayers := make([]string, 0, len(manifest.FSLayers))
	for _, fsLayer := range manifest.FSLayers {
		fsLayers = append(fsLayers, fsLayer.BlobSum.String())
	}
	return &v1.ImageMetadata{
		Created:  latest.Created,
		Author:   latest.Author,
		Layers:   layers,
		FsLayers: fsLayers,
	}, nil
}

func (d *Registry) populateManifestV1(image *v1.Image) (*v1.ImageMetadata, error) {
	// First try to use the SHA given (which is probably V2, but for backwards compatibility reasons on old images we will try)
	signedManifest, err := d.client.SignedManifest(image.GetName().GetRemote(), image.GetId())
	if err == nil {
		return d.populateV1DataFromManifest(signedManifest)
	}

	manifest, err := d.client.Manifest(image.GetName().GetRemote(), image.GetId())
	if err == nil {
		return d.populateV1DataFromManifest(manifest)
	}

	// Now try to use the tag, which is less specific than the SHA
	// TODO(cgorman) is it possible to get the tags by the sha?
	signedManifest, err = d.client.SignedManifest(image.GetName().GetRemote(), image.GetName().GetTag())
	if err == nil {
		return d.populateV1DataFromManifest(signedManifest)
	}

	manifest, err = d.client.Manifest(image.GetName().GetRemote(), image.GetName().GetTag())
	if err == nil {
		return d.populateV1DataFromManifest(manifest)
	}

	return nil, fmt.Errorf("Failed to get the manifest: %s", err)
}

// Metadata returns the metadata via this registries implementation
func (d *Registry) Metadata(image *v1.Image) (*v1.ImageMetadata, error) {
	log.Infof("Getting metadata for image %s", image.GetName().GetFullName())
	if image == nil {
		return nil, nil
	}

	// fetch the digest from registry because it is more correct than from the orchestrator
	digest, err := d.client.ManifestDigest(image.GetName().GetRemote(), utils.Reference(image))
	if err != nil {
		return nil, fmt.Errorf("Failed to get the manifest digest : %s", err)
	}

	// metadata will be populated as an empty struct if there is a failure
	metadata, err := d.populateManifestV1(image)
	if err != nil {
		log.Error(err)
		metadata = &v1.ImageMetadata{}
	}
	metadata.RegistrySha = digest.String()
	metadata.V2 = d.getV2Metadata(image)
	return metadata, nil
}

// Test tests the current registry and makes sure that it is working properly
func (d *Registry) Test() error {
	return d.client.Ping()
}

// Config returns the configuration of the docker registry
func (d *Registry) Config() *types.Config {
	return &types.Config{
		Username:         d.cfg.Username,
		Password:         d.cfg.Password,
		Insecure:         d.cfg.Insecure,
		URL:              d.url,
		RegistryHostname: d.registry,
	}
}
