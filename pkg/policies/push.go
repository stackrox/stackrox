package policies

import (
	"bytes"
	"context"
	"os"
	"path"

	"github.com/golang/protobuf/jsonpb"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries/types"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

const (
	artifactType = "application/vnd.stackrox.policy"
	mediaType    = "text/json"
)

var (
	_ Pusher = (*pusherImpl)(nil)

	log = logging.LoggerForModule()
)

// Pusher pushes policies to a registry as OCI artifacts.
type Pusher interface {
	Push(ctx context.Context, policy *storage.Policy, registryConfig *types.Config, repository string) (string, error)
}

// NewPusher creates a new policy pusher.
func NewPusher() Pusher {
	return &pusherImpl{}
}

type pusherImpl struct {
}

func (p *pusherImpl) Push(ctx context.Context, policy *storage.Policy,
	registryConfig *types.Config, repository string) (string, error) {
	log.Infof("Received policy %+v to push to registry %s", policy, registryConfig.RegistryHostname)

	m := jsonpb.Marshaler{
		Indent: "    ",
	}
	buf := &bytes.Buffer{}
	// TODO(dhaus): Theoretically need to strip away things like exclusion scope etc. but later.
	if err := m.Marshal(buf, policy); err != nil {
		return "", errors.Wrap(err, "marshalling policy")
	}

	// Need to write the file on disk due to ugly API.
	dir := os.TempDir()
	policyFilePath := path.Join(dir, policy.GetId())
	if err := os.WriteFile(policyFilePath, buf.Bytes(), 0644); err != nil {
		return "", errors.Wrap(err, "writing temp file with policy")
	}

	// DAG stores holding the layer contents and manifest contents.
	fs, err := file.New(dir)
	if err != nil {
		return "", errors.Wrap(err, "failed to create file store")
	}

	// Add the file to the file store and create digest for the layer.
	desc, err := fs.Add(ctx, policy.GetId(), mediaType, policyFilePath)
	if err != nil {
		return "", errors.Wrap(err, "add file to file store")
	}

	// Create the options for the manifest, including which layers to include in it.
	packOpts := oras.PackManifestOptions{
		Layers: []v1.Descriptor{desc},
	}

	// Remote config for the repository, including auth stuff.
	repo, err := remote.NewRepository(path.Join(registryConfig.RegistryHostname, repository))
	if err != nil {
		return "", errors.Wrap(err, "creating repository")
	}
	repo.Client = &auth.Client{
		// TODO(dhaus): For now only supporting secure, probably best anyways for things like policies.
		Client: retry.DefaultClient,
		Cache:  auth.DefaultCache,
		Credential: auth.StaticCredential(registryConfig.RegistryHostname, auth.Credential{
			Username: registryConfig.Username,
			Password: registryConfig.Password,
		}),
	}

	// Pack manifest creates an OCI manifest with the appropriate artifact type as well as the file content
	// as layer.
	root, err := oras.PackManifest(ctx, fs, oras.PackManifestVersion1_1_RC4, artifactType, packOpts)
	if err != nil {
		return "", errors.Wrap(err, "packing manifest")
	}
	// Tag the manifest in the DAG, so copying can just copy the local manifest remotely including the tag.
	if err := fs.Tag(ctx, root, policy.GetId()); err != nil {
		return "", errors.Wrap(err, "tagging manifest")
	}

	// Copy the contents from local to remote.
	if _, err := oras.Copy(ctx, fs, policy.GetId(), repo, policy.GetId(), oras.DefaultCopyOptions); err != nil {
		return "", errors.Wrap(err, "copying manifest to remote")
	}

	return policy.GetId(), nil
}
