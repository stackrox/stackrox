package reportgenerator

import (
	"context"
	"path"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
)

const imageMetadataBatchSize = 200

type baseImageNameCache struct {
	byImageID           map[string]string
	configuredRepos     map[string][]string
	configuredReposRead bool
}

func reportImageID(name, digest string) string {
	if digest == "" {
		return ""
	}
	if features.FlattenImageData.Enabled() {
		return imageUtils.NewImageV2ID(&storage.ImageName{FullName: name}, digest)
	}
	return digest
}

// BaseImageInfo lives in serialized image metadata, not in a report-queryable column.
// Fetch and cache it per image so selecting base images does not multiply CVE rows.
func (rg *reportGeneratorImpl) withBaseImageNames(ctx context.Context, rows []*ImageCVEQueryResponse, cache *baseImageNameCache) error {
	if len(rows) == 0 || rg.imageDatastore == nil {
		return nil
	}
	if cache.byImageID == nil {
		cache.byImageID = make(map[string]string)
	}

	unknownIDs := set.NewStringSet()
	for _, row := range rows {
		id := reportImageID(row.GetImage(), row.GetImageSHA())
		if id == "" {
			continue
		}
		if _, cached := cache.byImageID[id]; !cached {
			cache.byImageID[id] = "" // Also cache images that are no longer available.
			unknownIDs.Add(id)
		}
	}

	for batch := range slices.Chunk(unknownIDs.AsSlice(), imageMetadataBatchSize) {
		images, err := rg.imageDatastore.GetManyImageMetadata(ctx, batch)
		if err != nil {
			return errors.Wrap(err, "fetching base image metadata")
		}
		for _, image := range images {
			infos := image.GetBaseImageInfo()
			if !cache.configuredReposRead && needsConfiguredBaseImageReference(infos) {
				rg.loadConfiguredBaseImageRepositories(ctx, cache)
			}
			id := reportImageID(image.GetName().GetFullName(), image.GetId())
			if _, requested := cache.byImageID[id]; requested {
				cache.byImageID[id] = formatBaseImageNames(infos, cache.configuredRepos)
			}
		}
	}

	for _, row := range rows {
		row.BaseImage = cache.byImageID[reportImageID(row.GetImage(), row.GetImageSHA())]
	}
	return nil
}

func (rg *reportGeneratorImpl) loadConfiguredBaseImageRepositories(ctx context.Context, cache *baseImageNameCache) {
	cache.configuredReposRead = true
	if rg.baseImageRepositoryDatastore == nil {
		return
	}
	repos, err := rg.baseImageRepositoryDatastore.ListRepositories(ctx)
	if err != nil {
		log.Warnw("Could not fetch configured base image references; using a deterministic name", logging.Err(err))
		return
	}
	cache.configuredRepos = make(map[string][]string, len(repos))
	for _, repo := range repos {
		cache.configuredRepos[repo.GetRepositoryPath()] = append(cache.configuredRepos[repo.GetRepositoryPath()], repo.GetTagPattern())
	}
}

func needsConfiguredBaseImageReference(infos []*storage.BaseImageInfo) bool {
	seen := make(map[string]string, len(infos))
	for _, info := range infos {
		digest, name := info.GetBaseImageDigest(), info.GetBaseImageFullName()
		if digest == "" || name == "" {
			continue
		}
		if existing, ok := seen[digest]; ok && existing != name {
			return true
		}
		seen[digest] = name
	}
	return false
}

func matchesConfiguredBaseImageReference(name string, configuredRepos map[string][]string) bool {
	separator := strings.LastIndexByte(name, ':')
	if separator < 0 {
		return false
	}
	for _, pattern := range configuredRepos[name[:separator]] {
		matches, err := path.Match(pattern, name[separator+1:])
		if err == nil && matches {
			return true
		}
	}
	return false
}

func formatBaseImageNames(infos []*storage.BaseImageInfo, configuredRepos map[string][]string) string {
	byDigest := make(map[string][]string)
	for _, info := range infos {
		digest, name := info.GetBaseImageDigest(), info.GetBaseImageFullName()
		if digest != "" && name != "" {
			byDigest[digest] = append(byDigest[digest], name)
		}
	}

	names := make([]string, 0, len(byDigest))
	for _, aliases := range byDigest {
		slices.Sort(aliases)
		chosen := aliases[0]
		for _, alias := range aliases {
			if matchesConfiguredBaseImageReference(alias, configuredRepos) {
				chosen = alias
				break
			}
		}
		names = append(names, chosen)
	}
	slices.Sort(names)
	return strings.Join(names, "; ")
}
