package datastore

import (
	"context"

	"github.com/stackrox/rox/central/cve/image/info/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sliceutils"
)

type datastoreImpl struct {
	storage store.Store
}

func (ds *datastoreImpl) SearchRawImageCVEInfos(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEInfo, error) {
	infos := make([]*storage.ImageCVEInfo, 0)
	err := ds.storage.WalkByQuery(ctx, q, func(cve *storage.ImageCVEInfo) error {
		infos = append(infos, cve)
		return nil
	})
	return infos, err
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	return ds.storage.Exists(ctx, id)
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ImageCVEInfo, bool, error) {
	return ds.storage.Get(ctx, id)
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.storage.Count(ctx, q)
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.ImageCVEInfo, error) {
	infos, _, err := ds.storage.GetMany(ctx, ids)
	return infos, err
}

func (ds *datastoreImpl) Upsert(ctx context.Context, info *storage.ImageCVEInfo) error {
	existing, found, err := ds.Get(ctx, info.GetId())
	if err != nil {
		return err
	}
	if found {
		info = updateTimestamps(existing, info)
	}

	return ds.storage.Upsert(ctx, info)
}

func (ds *datastoreImpl) UpsertMany(ctx context.Context, infos []*storage.ImageCVEInfo) error {
	// Create a list of ids to look up
	ids := sliceutils.Map[*storage.ImageCVEInfo, string](infos, func(info *storage.ImageCVEInfo) string {
		return info.GetId()
	})
	existing, err := ds.GetBatch(ctx, ids)
	if err != nil {
		return err
	}
	newInfoMap := make(map[string]*storage.ImageCVEInfo)
	oldInfoMap := make(map[string]*storage.ImageCVEInfo)
	// Populate both maps separately - can't use index-based loop because
	// existing may not be in the same order as infos
	for _, info := range infos {
		newInfoMap[info.GetId()] = info
	}
	for _, info := range existing {
		oldInfoMap[info.GetId()] = info
	}
	// Create our list that we're going to actually upsert
	toUpsert := make([]*storage.ImageCVEInfo, 0)
	for k, v := range newInfoMap {
		newValue := updateTimestamps(oldInfoMap[k], v)
		toUpsert = append(toUpsert, newValue)
	}
	return ds.storage.UpsertMany(ctx, toUpsert)
}

func updateTimestamps(old, new *storage.ImageCVEInfo) *storage.ImageCVEInfo {
	if old == nil {
		return new
	}
	// Update timestamps to use the earlier of the two timestamps, where applicable.
	// Only use old's value if new is zero, or if old is non-zero and earlier than new.
	if protocompat.IsZeroTimestamp(new.GetFirstSystemOccurrence()) {
		new.FirstSystemOccurrence = old.GetFirstSystemOccurrence()
	} else if !protocompat.IsZeroTimestamp(old.GetFirstSystemOccurrence()) && protocompat.CompareTimestamps(old.GetFirstSystemOccurrence(), new.GetFirstSystemOccurrence()) < 0 {
		new.FirstSystemOccurrence = old.GetFirstSystemOccurrence()
	}

	if protocompat.IsZeroTimestamp(new.GetFixAvailableTimestamp()) {
		new.FixAvailableTimestamp = old.GetFixAvailableTimestamp()
	} else if !protocompat.IsZeroTimestamp(old.GetFixAvailableTimestamp()) && protocompat.CompareTimestamps(old.GetFixAvailableTimestamp(), new.GetFixAvailableTimestamp()) < 0 {
		new.FixAvailableTimestamp = old.GetFixAvailableTimestamp()
	}
	return new
}
