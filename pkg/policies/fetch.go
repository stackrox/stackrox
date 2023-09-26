package policies

import (
	"bytes"
	"context"
	"os"
	"path"

	"github.com/golang/protobuf/jsonpb"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/utils"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

var (
	_ Fetcher = (*fetcherImpl)(nil)
)

// Fetcher fetches policies from a registry where they are stored as OCI artifacts.
type Fetcher interface {
	Fetch(ctx context.Context, registryConfig *types.Config, repository string) ([]*storage.Policy, error)
}

// NewFetcher creates a new policy fetcher
func NewFetcher() Fetcher {
	return &fetcherImpl{}
}

type fetcherImpl struct {
}

func (f *fetcherImpl) Fetch(ctx context.Context, registryConfig *types.Config, repository string) ([]*storage.Policy, error) {
	fs, err := file.New("")
	if err != nil {
		return nil, errors.Wrap(err, "creating ORAS file store")
	}
	defer utils.IgnoreError(fs.Close)

	ref := path.Join(registryConfig.RegistryHostname, repository)

	// Repository config.
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return nil, errors.Wrap(err, "creating repository")
	}
	// TODO(dhaus): For simplicity, assume that policies can be fetched without credentials.
	repo.Client = &auth.Client{
		// TODO(dhaus): For now only supporting secure, probably best anyways for things like policies.
		Client: retry.DefaultClient,
		Cache:  auth.DefaultCache,
		Credential: auth.StaticCredential(registryConfig.RegistryHostname, auth.Credential{
			Username: registryConfig.Username,
			Password: registryConfig.Password,
		}),
	}

	// Iterate through all tags for the given repository.
	// Would be nice to support filtering based on the artifact type, so that we only retrieve stuff that's actually
	// relevant, somehow artifact type gets lost from Docker Hub (probably me doing stuff wrong).
	var tags []string
	if err := repo.Tags(ctx, "", func(t []string) error {
		tags = append(tags, t...)
		return nil
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve tags in repository %s", ref)
	}

	var fetchErrors *multierror.Error
	var policyBytes []*bytes.Buffer
	// Go through all tags. First copy the remote manifest and its layers to local.
	// Then access the layers associated with the copied manifest which the file store has written to disk.
	for _, tag := range tags {
		md, err := oras.Copy(ctx, repo, tag, fs, tag, oras.DefaultCopyOptions)
		if err != nil {
			fetchErrors = multierror.Append(fetchErrors, err)
			continue
		}

		log.Infof("Manifest descriptor for tag %s: %+v", tag, md)

		// Layers are stored locally from the file store. For us this means working dir + manifest tag.
		contents, err := os.ReadFile(tag)
		if err != nil {
			fetchErrors = multierror.Append(fetchErrors, err)
			continue
		}
		policyBytes = append(policyBytes, bytes.NewBuffer(contents))
	}

	// Make sure we delete previously fetched files.
	defer deleteFiles(tags)

	if err := fetchErrors.ErrorOrNil(); err != nil {
		log.Errorw("Some errors during fetching of policies, might not be related but some might be missing ¯\\_(ツ)_/¯",
			logging.Err(err))
	}

	var policies []*storage.Policy
	var unmarshalErrors *multierror.Error
	for _, buf := range policyBytes {
		var policy storage.Policy
		if err := jsonpb.Unmarshal(buf, &policy); err != nil {
			unmarshalErrors = multierror.Append(unmarshalErrors, err)
			continue
		}
		policies = append(policies, &policy)
	}

	if err := unmarshalErrors.ErrorOrNil(); err != nil {
		return nil, err
	}

	return policies, nil
}

func deleteFiles(names []string) {
	for _, name := range names {
		if err := os.Remove(name); err != nil {
			log.Errorf("Failed deleting downloaded manifest contents %s", name)
		}
	}
}
