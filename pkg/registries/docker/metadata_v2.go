package docker

import (
	"runtime"

	"github.com/docker/distribution/manifest/manifestlist"
	manifestV2 "github.com/docker/distribution/manifest/schema2"
	"github.com/heroku/docker-registry-client/registry"
	godigest "github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
)

func handleManifestLists(r *Registry, remote, ref string, manifests []manifestlist.ManifestDescriptor) (*storage.ImageMetadata, error) {
	if len(manifests) == 0 {
		return nil, errors.Errorf("no valid manifests found for %s:%s", remote, ref)
	}
	if len(manifests) == 1 {
		return handleManifests(r, manifests[0].MediaType, remote, manifests[0].Digest.String())
	}
	var amdManifest manifestlist.ManifestDescriptor
	var foundAMD bool
	for _, m := range manifests {
		if m.Platform.OS != "linux" {
			continue
		}
		// Matching platform for GOARCH takes priority so return immediately
		if m.Platform.Architecture == runtime.GOARCH {
			return handleManifests(r, m.MediaType, remote, m.Digest.String())
		}
		if m.Platform.Architecture == "amd64" {
			foundAMD = true
			amdManifest = m
		}
	}
	if foundAMD {
		return handleManifests(r, amdManifest.MediaType, remote, amdManifest.Digest.String())
	}
	return nil, errors.Errorf("no manifest in list matched linux and amd64 or %s architectures: %q", runtime.GOARCH, ref)
}

// HandleV2ManifestList takes in a v2 manifest list ref and returns the image metadata
func HandleV2ManifestList(r *Registry, remote, ref string) (*storage.ImageMetadata, error) {
	manifestList, err := r.Client.ManifestList(remote, ref)
	if err != nil {
		return nil, err
	}
	return handleManifestLists(r, remote, ref, manifestList.Manifests)
}

// HandleV2Manifest takes in a v2 ref and returns the image metadata
func HandleV2Manifest(r *Registry, remote, ref string) (*storage.ImageMetadata, error) {
	metadata, _, err := handleV2ManifestWithPlatform(r, remote, ref)
	return metadata, err
}

// handleV2ManifestWithPlatform returns the image metadata together with the
// platform (os/architecture) read from the image config.
func handleV2ManifestWithPlatform(r *Registry, remote, ref string) (*storage.ImageMetadata, *v1.Platform, error) {
	metadata, dig, err := r.Client.ManifestV2WithDigest(remote, ref)
	if err != nil {
		return nil, nil, err
	}
	layers := make([]string, 0, len(metadata.Layers))
	for _, layer := range metadata.Layers {
		layers = append(layers, layer.Digest.String())
	}

	var v1Metadata *storage.V1Metadata
	var platform *v1.Platform
	if metadata.Config.Digest != "" {
		v1Metadata, platform, err = r.handleV1ManifestLayer(remote, metadata.Config.Digest)
		if err != nil {
			return nil, nil, err
		}
	}
	return &storage.ImageMetadata{
		V1: v1Metadata,
		V2: &storage.V2Metadata{
			Digest: digestOrRef(ref, dig),
		},
		LayerShas: layers,
	}, platform, nil
}

// HandleOCIImageIndex handles fetching data if the media type is OCI image index.
func HandleOCIImageIndex(r *Registry, remote, ref string) (*storage.ImageMetadata, error) {
	index, err := r.Client.ImageIndex(remote, ref)
	if err != nil {
		return nil, err
	}
	return handleManifestLists(r, remote, ref, index.Manifests)
}

// HandleOCIManifest handles fetching data if the media type is OCI
func HandleOCIManifest(r *Registry, remote, ref string) (*storage.ImageMetadata, error) {
	metadata, _, err := handleOCIManifestWithPlatform(r, remote, ref)
	return metadata, err
}

// handleOCIManifestWithPlatform returns the image metadata together with the
// platform (os/architecture) read from the image config.
func handleOCIManifestWithPlatform(r *Registry, remote, ref string) (*storage.ImageMetadata, *v1.Platform, error) {
	metadata, dig, err := r.Client.ManifestOCIWithDigest(remote, ref)
	if err != nil {
		return nil, nil, err
	}
	layers := make([]string, 0, len(metadata.Layers))
	for _, layer := range metadata.Layers {
		layers = append(layers, layer.Digest.String())
	}

	var v1Metadata *storage.V1Metadata
	var platform *v1.Platform
	if metadata.Config.Digest != "" {
		v1Metadata, platform, err = r.handleV1ManifestLayer(remote, metadata.Config.Digest)
		if err != nil {
			return nil, nil, err
		}
	}
	return &storage.ImageMetadata{
		V1: v1Metadata,
		V2: &storage.V2Metadata{
			Digest: digestOrRef(ref, dig),
		},
		LayerShas: layers,
	}, platform, nil
}

// handleManifestsAllPlatforms returns metadata for every linux platform described by
// the manifest at ref. For a manifest list / OCI image index it fans out to each
// platform's manifest; for a single-platform image it returns one entry.
func handleManifestsAllPlatforms(r *Registry, manifestType, remote, ref string) ([]*types.PlatformMetadata, error) {
	switch manifestType {
	case manifestlist.MediaTypeManifestList:
		list, err := r.Client.ManifestList(remote, ref)
		if err != nil {
			return nil, err
		}
		return handleManifestListAllPlatforms(r, remote, ref, list.Manifests)
	case registry.MediaTypeImageIndex:
		index, err := r.Client.ImageIndex(remote, ref)
		if err != nil {
			return nil, err
		}
		return handleManifestListAllPlatforms(r, remote, ref, index.Manifests)
	default:
		pm, err := handleSingleManifestPlatform(r, manifestType, remote, ref)
		if err != nil {
			return nil, err
		}
		return []*types.PlatformMetadata{pm}, nil
	}
}

// handleManifestListAllPlatforms resolves each linux platform in a manifest list to its
// own metadata. os/architecture come from the list descriptor, which is authoritative.
// Non-linux platforms are skipped (base image matching is linux-only). A single failed
// platform is skipped; an error is only returned if no linux platform could be resolved.
func handleManifestListAllPlatforms(r *Registry, remote, ref string, manifests []manifestlist.ManifestDescriptor) ([]*types.PlatformMetadata, error) {
	if len(manifests) == 0 {
		return nil, errors.Errorf("no valid manifests found for %s:%s", remote, ref)
	}
	results := make([]*types.PlatformMetadata, 0, len(manifests))
	var errs []error
	for _, m := range manifests {
		if m.Platform.OS != "linux" {
			continue
		}
		md, err := handleManifests(r, m.MediaType, remote, m.Digest.String())
		if err != nil {
			// Log here as well as collecting: when at least one platform succeeds the
			// error is otherwise dropped, hiding partial failures from investigation.
			log.Warnf("Failed to resolve metadata for %s:%s platform %s/%s/%s: %v",
				remote, ref, m.Platform.OS, m.Platform.Architecture, m.Platform.Variant, err)
			errs = append(errs, errors.Wrapf(err, "platform %s/%s/%s", m.Platform.OS, m.Platform.Architecture, m.Platform.Variant))
			continue
		}
		results = append(results, &types.PlatformMetadata{
			OS:           m.Platform.OS,
			Architecture: m.Platform.Architecture,
			Variant:      m.Platform.Variant,
			Metadata:     md,
		})
	}
	if len(results) == 0 {
		return nil, errors.Errorf("no linux manifests resolved for %s:%s: %v", remote, ref, errs)
	}
	return results, nil
}

// handleSingleManifestPlatform resolves a non-list manifest, deriving os/architecture
// from the image config when available (v2/OCI). Legacy v1 manifests carry no config
// platform, so os/architecture are left empty.
func handleSingleManifestPlatform(r *Registry, manifestType, remote, ref string) (*types.PlatformMetadata, error) {
	switch manifestType {
	case manifestV2.MediaTypeManifest:
		md, platform, err := handleV2ManifestWithPlatform(r, remote, ref)
		if err != nil {
			return nil, err
		}
		return platformMetadata(platform, md), nil
	case registry.MediaTypeImageManifest:
		md, platform, err := handleOCIManifestWithPlatform(r, remote, ref)
		if err != nil {
			return nil, err
		}
		return platformMetadata(platform, md), nil
	default:
		md, err := handleManifests(r, manifestType, remote, ref)
		if err != nil {
			return nil, err
		}
		return &types.PlatformMetadata{Metadata: md}, nil
	}
}

func platformMetadata(platform *v1.Platform, md *storage.ImageMetadata) *types.PlatformMetadata {
	pm := &types.PlatformMetadata{Metadata: md}
	if platform != nil {
		pm.OS = platform.OS
		pm.Architecture = platform.Architecture
		pm.Variant = platform.Variant
	}
	return pm
}

// digestOrRef returns digest if populated and ref is NOT a valid digest, otherwise returns ref.
func digestOrRef(ref string, digest godigest.Digest) string {
	if digest == "" {
		// If no digest, return the ref as is.
		return ref
	}

	if _, err := godigest.Parse(ref); err != nil {
		// If ref itself is not a digest, then return digest instead.
		return string(digest)
	}

	return ref
}
