package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/processwhitelist/index"
	"github.com/stackrox/rox/central/processwhitelist/search"
	"github.com/stackrox/rox/central/processwhitelist/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
)

type datastoreImpl struct {
	storage       store.Store
	indexer       index.Indexer
	searcher      search.Searcher
	whitelistLock *concurrency.KeyedMutex
}

func (ds *datastoreImpl) SearchRawProcessWhitelists(ctx context.Context, q *v1.Query) ([]*storage.ProcessWhitelist, error) {
	return ds.searcher.SearchRawProcessWhitelists(q)
}

func (ds *datastoreImpl) GetProcessWhitelist(ctx context.Context, key *storage.ProcessWhitelistKey) (*storage.ProcessWhitelist, error) {
	id, err := keyToID(key)
	if err != nil {
		return nil, err
	}
	return ds.storage.GetWhitelist(id)
}

func (ds *datastoreImpl) GetProcessWhitelists(ctx context.Context) ([]*storage.ProcessWhitelist, error) {
	return ds.storage.GetWhitelists()
}

func (ds *datastoreImpl) AddProcessWhitelist(ctx context.Context, whitelist *storage.ProcessWhitelist) (string, error) {
	id, err := keyToID(whitelist.GetKey())
	if err != nil {
		return "", err
	}
	ds.whitelistLock.Lock(id)
	defer ds.whitelistLock.Unlock(id)
	return ds.addProcessWhitelistUnlocked(id, whitelist)
}

func (ds *datastoreImpl) addProcessWhitelistUnlocked(id string, whitelist *storage.ProcessWhitelist) (string, error) {
	whitelist.Id = id
	if err := ds.storage.AddWhitelist(whitelist); err != nil {
		return id, errors.Wrapf(err, "inserting whitelist %q into store", whitelist.GetId())
	}
	if err := ds.indexer.AddWhitelist(whitelist); err != nil {
		err = errors.Wrapf(err, "inserting whitelist %q into index", whitelist.GetId())
		_, subErr := ds.storage.DeleteWhitelist(id)
		if subErr != nil {
			err = errors.Wrapf(err, "error rolling back process whitelist addition")
		}
		return id, err
	}
	return id, nil
}

func (ds *datastoreImpl) RemoveProcessWhitelist(ctx context.Context, key *storage.ProcessWhitelistKey) error {
	id, err := keyToID(key)
	if err != nil {
		return err
	}
	ds.whitelistLock.Lock(id)
	defer ds.whitelistLock.Unlock(id)
	if err := ds.indexer.DeleteWhitelist(id); err != nil {
		return errors.Wrap(err, "error removing whitelist from index")
	}
	if _, err := ds.storage.DeleteWhitelist(id); err != nil {
		return errors.Wrap(err, "error removing whitelist from store")
	}
	return nil
}

func (ds *datastoreImpl) getWhitelistForUpdate(id string) (*storage.ProcessWhitelist, error) {
	whitelist, err := ds.storage.GetWhitelist(id)
	if err != nil {
		return nil, err
	}
	if whitelist == nil {
		return nil, errors.Errorf("no process whitelist with id %q", id)
	}
	return whitelist, nil
}

func (ds *datastoreImpl) updateProcessWhitelistElementsUnlocked(whitelist *storage.ProcessWhitelist, addElements []*storage.WhitelistItem, removeElements []*storage.WhitelistItem, auto bool) (*storage.ProcessWhitelist, error) {
	whitelistMap := make(map[string]*storage.WhitelistElement, len(whitelist.Elements))
	for _, listItem := range whitelist.Elements {
		whitelistMap[listItem.GetElement().GetProcessName()] = listItem
	}

	for _, element := range addElements {
		existing, ok := whitelistMap[element.GetProcessName()]
		if !ok || existing.Auto {
			whitelistMap[element.GetProcessName()] = &storage.WhitelistElement{
				Element: &storage.WhitelistItem{
					Item: &storage.WhitelistItem_ProcessName{ProcessName: element.GetProcessName()},
				},
				Auto: auto,
			}
		}
	}

	for _, removeElement := range removeElements {
		delete(whitelistMap, removeElement.GetProcessName())
	}
	whitelist.Elements = make([]*storage.WhitelistElement, 0, len(whitelistMap))
	for _, process := range whitelistMap {
		whitelist.Elements = append(whitelist.Elements, process)
	}

	err := ds.storage.UpdateWhitelist(whitelist)
	if err != nil {
		return nil, err
	}
	err = ds.indexer.AddWhitelist(whitelist)
	if err != nil {
		return nil, err
	}

	return whitelist, nil
}

func (ds *datastoreImpl) UpdateProcessWhitelistElements(ctx context.Context, key *storage.ProcessWhitelistKey, addElements []*storage.WhitelistItem, removeElements []*storage.WhitelistItem, auto bool) (*storage.ProcessWhitelist, error) {
	id, err := keyToID(key)
	if err != nil {
		return nil, err
	}

	ds.whitelistLock.Lock(id)
	defer ds.whitelistLock.Unlock(id)

	whitelist, err := ds.getWhitelistForUpdate(id)
	if err != nil {
		return nil, err
	}
	return ds.updateProcessWhitelistElementsUnlocked(whitelist, addElements, removeElements, auto)
}

func (ds *datastoreImpl) UpsertProcessWhitelist(ctx context.Context, key *storage.ProcessWhitelistKey, addElements []*storage.WhitelistItem, auto bool) (*storage.ProcessWhitelist, error) {
	id, err := keyToID(key)
	if err != nil {
		return nil, err
	}

	ds.whitelistLock.Lock(id)
	defer ds.whitelistLock.Unlock(id)

	whitelist, err := ds.GetProcessWhitelist(ctx, key)
	if err != nil {
		return nil, err
	}

	if whitelist != nil {
		return ds.updateProcessWhitelistElementsUnlocked(whitelist, addElements, nil, auto)
	}

	timestamp := types.TimestampNow()
	var elements []*storage.WhitelistElement
	for _, element := range addElements {
		elements = append(elements, &storage.WhitelistElement{Element: &storage.WhitelistItem{Item: &storage.WhitelistItem_ProcessName{ProcessName: element.GetProcessName()}}, Auto: auto})
	}
	whitelist = &storage.ProcessWhitelist{
		Id:         id,
		Key:        key,
		Elements:   elements,
		Created:    timestamp,
		LastUpdate: timestamp,
	}
	_, err = ds.addProcessWhitelistUnlocked(id, whitelist)
	if err != nil {
		return nil, err
	}
	return whitelist, nil
}

func (ds *datastoreImpl) UserLockProcessWhitelist(ctx context.Context, key *storage.ProcessWhitelistKey, locked bool) (*storage.ProcessWhitelist, error) {
	id, err := keyToID(key)
	if err != nil {
		return nil, err
	}
	ds.whitelistLock.Lock(id)
	defer ds.whitelistLock.Unlock(id)

	whitelist, err := ds.getWhitelistForUpdate(id)
	if err != nil {
		return nil, err
	}

	if locked && whitelist.GetUserLockedTimestamp() == nil {
		whitelist.UserLockedTimestamp = types.TimestampNow()
		err = ds.storage.UpdateWhitelist(whitelist)
	} else if !locked && whitelist.GetUserLockedTimestamp() != nil {
		whitelist.UserLockedTimestamp = nil
		err = ds.storage.UpdateWhitelist(whitelist)
	}
	if err != nil {
		return nil, err
	}
	return whitelist, nil
}

func (ds *datastoreImpl) RoxLockProcessWhitelist(ctx context.Context, key *storage.ProcessWhitelistKey, locked bool) (*storage.ProcessWhitelist, error) {
	id, err := keyToID(key)
	if err != nil {
		return nil, err
	}
	ds.whitelistLock.Lock(id)
	defer ds.whitelistLock.Unlock(id)

	whitelist, err := ds.getWhitelistForUpdate(id)
	if err != nil {
		return nil, err
	}

	if locked && whitelist.GetStackRoxLockedTimestamp() == nil {
		whitelist.StackRoxLockedTimestamp = types.TimestampNow()
		err = ds.storage.UpdateWhitelist(whitelist)
	} else if !locked && whitelist.GetStackRoxLockedTimestamp() != nil {
		whitelist.StackRoxLockedTimestamp = nil
		err = ds.storage.UpdateWhitelist(whitelist)
	}
	if err != nil {
		return nil, err
	}
	return whitelist, nil
}
