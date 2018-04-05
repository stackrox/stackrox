package docker

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	"bitbucket.org/stack-rox/apollo/pkg/urlfmt"
	manifestV1 "github.com/docker/distribution/manifest/schema1"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/opencontainers/go-digest"
)

var (
	log = logging.LoggerForModule()
)

// Registry is the basic docker registry implementation
type Registry struct {
	cfg                   Config
	protoImageIntegration *v1.ImageIntegration

	getClientOnce sync.Once
	clientObj     client

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

type client interface {
	Manifest(repository, reference string) (*manifestV1.SignedManifest, error)
	ManifestDigest(repository, reference string) (digest.Digest, error)
	SignedManifest(repository, reference string) (*manifestV1.SignedManifest, error)
	Repositories() ([]string, error)
	Ping() error
}

type nilClient struct {
	error error
}

func (n nilClient) Manifest(repository, reference string) (*manifestV1.SignedManifest, error) {
	return nil, n.error
}

func (n nilClient) ManifestDigest(repository, reference string) (digest.Digest, error) {
	return digest.Digest(""), n.error
}

func (n nilClient) SignedManifest(repository, reference string) (*manifestV1.SignedManifest, error) {
	return nil, n.error
}

func (n nilClient) Repositories() ([]string, error) {
	return nil, n.error
}

func (n nilClient) Ping() error {
	return n.error
}

func (c Config) validate() error {
	username := c.Username
	password := c.Password
	hasUsername := username != ""
	hasPassword := password != ""

	if hasUsername != hasPassword {
		if !hasUsername {
			return errors.New("Config parameter 'username' must be defined for all non Docker Hub registries")
		}
		return errors.New("Config parameter 'password' must be defined for all non Docker Hub registries")
	}
	if c.Endpoint == "" {
		return errors.New("Config parameter 'endpoint' must be defined")
	}

	if (!hasUsername && !hasPassword) && !strings.Contains(c.Endpoint, "docker.io") {
		return errors.New("Config parameters 'username' and 'password' must be defined for all non Docker Hub registries")
	}
	return nil
}

// NewDockerRegistry creates a new instantiation of the docker registry
// TODO(cgorman) AP-386 - properly put the base docker registry into another pkg
func NewDockerRegistry(cfg Config, integration *v1.ImageIntegration) (*Registry, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	url, err := urlfmt.FormatURL(cfg.Endpoint, true, false)
	if err != nil {
		return nil, err
	}
	// if the registry endpoint contains docker.io then the image will be docker.io/namespace/repo:tag
	registry := urlfmt.GetServerFromURL(url)
	if strings.Contains(cfg.Endpoint, "docker.io") {
		registry = "docker.io"
	}

	return &Registry{
		url:      url,
		registry: registry,

		cfg: cfg,
		protoImageIntegration: integration,
	}, nil
}

func newRegistry(integration *v1.ImageIntegration) (*Registry, error) {
	cfg := Config{
		Endpoint: integration.Config["endpoint"],
		Username: integration.Config["username"],
		Password: integration.Config["password"],
	}
	return NewDockerRegistry(cfg, integration)
}

func (d *Registry) client() (c client) {
	d.getClientOnce.Do(func() {
		reg, err := registry.New(d.url, d.cfg.Username, d.cfg.Password)
		if err != nil {
			d.clientObj = nilClient{err}
			return
		}
		d.clientObj = reg
	})
	return d.clientObj
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
	for instruction := range registries.DockerfileInstructionSet {
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
	metadata, err := d.client().(*registry.Registry).ManifestV2(image.GetName().GetRemote(), image.GetName().GetTag())
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

// Metadata returns the metadata via this registries implementation
func (d *Registry) Metadata(image *v1.Image) (*v1.ImageMetadata, error) {
	log.Infof("Getting metadata for image %s", image.GetName().GetFullName())
	if image == nil {
		return nil, nil
	}

	// fetch the digest from registry because it is more correct than from the orchestrator
	digest, err := d.client().ManifestDigest(image.GetName().GetRemote(), image.GetName().GetTag())
	if err != nil {
		return nil, err
	}
	image.Name.Sha = digest.String()
	manifest, err := d.client().SignedManifest(image.GetName().GetRemote(), image.GetName().GetTag())
	if err != nil {
		manifest, err = d.client().Manifest(image.GetName().GetRemote(), image.GetName().GetTag())
		if err != nil {
			return nil, err
		}
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
	fsLayers := make([]string, 0, len(manifest.FSLayers))
	for _, fsLayer := range manifest.FSLayers {
		fsLayers = append(fsLayers, fsLayer.BlobSum.String())
	}
	imageMetadata := &v1.ImageMetadata{
		Created:  latest.Created,
		Author:   latest.Author,
		Layers:   layers,
		FsLayers: fsLayers,
		V2:       d.getV2Metadata(image),
	}
	return imageMetadata, nil
}

// Test tests the current registry and makes sure that it is working properly
func (d *Registry) Test() error {
	return d.client().Ping()
}

func init() {
	f := func(integration *v1.ImageIntegration) (registries.ImageRegistry, error) {
		reg, err := newRegistry(integration)
		return reg, err
	}
	registries.Registry["dtr"] = f
	registries.Registry["docker"] = f
	registries.Registry["artifactory"] = f
}
