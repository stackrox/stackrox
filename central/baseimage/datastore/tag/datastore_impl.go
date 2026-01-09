package tag

import (
	"context"
	"slices"
	"time"

	tagStore "github.com/stackrox/rox/central/baseimage/store/tag/postgres"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoconv"
)

const (
	resourceType = "BaseImageTag"
)

type datastoreImpl struct {
	store tagStore.Store
}

func (d *datastoreImpl) UpsertMany(ctx context.Context, tags []*storage.BaseImageTag) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "UpsertMany")
	return d.store.UpsertMany(ctx, tags)
}

func (d *datastoreImpl) DeleteMany(ctx context.Context, ids []string) error {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "DeleteMany")
	return d.store.DeleteMany(ctx, ids)
}

func (d *datastoreImpl) ListTagsByRepository(ctx context.Context, repositoryID string) ([]*storage.BaseImageTag, error) {
	defer metrics.SetDatastoreFunctionDuration(time.Now(), resourceType, "ListTagsByRepository")

	// Walk all tags, filter by repository, sort by created timestamp descending.
	var tags []*storage.BaseImageTag

	walkFn := func() error {
		tags = tags[:0]
		return d.store.Walk(ctx, func(tag *storage.BaseImageTag) error {
			if tag.GetBaseImageRepositoryId() == repositoryID {
				tags = append(tags, tag)
			}
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(ctx, walkFn); err != nil {
		return nil, err
	}

	// Sort by created timestamp descending (newest first).
	// Nil timestamps are not expected, and we sort them last.
	slices.SortFunc(tags, func(a, b *storage.BaseImageTag) int {
		aTime := protoconv.ConvertTimestampToTimeOrDefault(a.GetCreated(), time.Time{})
		bTime := protoconv.ConvertTimestampToTimeOrDefault(b.GetCreated(), time.Time{})

		// Compare in descending order (newer first)
		if aTime.After(bTime) {
			return -1
		}
		if aTime.Before(bTime) {
			return 1
		}
		return 0
	})

	return tags, nil
}
