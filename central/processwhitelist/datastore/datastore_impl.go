package datastore

import (
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

func (ds *datastoreImpl) SearchRawProcessWhitelists(q *v1.Query) ([]*storage.ProcessWhitelist, error) {
	return ds.searcher.SearchRawProcessWhitelists(q)
}

func (ds *datastoreImpl) GetProcessWhitelist(key *storage.ProcessWhitelistKey) (*storage.ProcessWhitelist, error) {
	id, err := keyToID(key)
	if err != nil {
		return nil, err
	}
	return ds.storage.GetWhitelist(id)
}

func (ds *datastoreImpl) GetProcessWhitelists() ([]*storage.ProcessWhitelist, error) {
	return ds.storage.GetWhitelists()
}

func (ds *datastoreImpl) AddProcessWhitelist(whitelist *storage.ProcessWhitelist) (string, error) {
	id, err := keyToID(whitelist.GetKey())
	if err != nil {
		return "", err
	}
	whitelist.Id = id
	ds.whitelistLock.Lock(id)
	defer ds.whitelistLock.Unlock(id)
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

func (ds *datastoreImpl) RemoveProcessWhitelist(key *storage.ProcessWhitelistKey) error {
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

func (ds *datastoreImpl) UpdateProcessWhitelist(key *storage.ProcessWhitelistKey, addNames []string, removeNames []string) (*storage.ProcessWhitelist, error) {
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

	whitelistMap := make(map[string]*storage.WhitelistElement, len(whitelist.Elements))
	for _, element := range whitelist.Elements {
		whitelistMap[element.GetProcessName()] = element
	}

	for _, addName := range addNames {
		existing, ok := whitelistMap[addName]
		if !ok || existing.Auto {
			whitelistMap[addName] = &storage.WhitelistElement{Element: &storage.WhitelistElement_ProcessName{ProcessName: addName}, Auto: false}
		}
	}

	for _, removeName := range removeNames {
		delete(whitelistMap, removeName)
	}
	whitelist.Elements = make([]*storage.WhitelistElement, 0, len(whitelistMap))
	for _, process := range whitelistMap {
		whitelist.Elements = append(whitelist.Elements, process)
	}

	err = ds.storage.UpdateWhitelist(whitelist)
	if err != nil {
		return nil, err
	}
	err = ds.indexer.AddWhitelist(whitelist)
	if err != nil {
		return nil, err
	}

	return whitelist, nil
}

func (ds *datastoreImpl) UserLockProcessWhitelist(key *storage.ProcessWhitelistKey, locked bool) (*storage.ProcessWhitelist, error) {
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

func (ds *datastoreImpl) RoxLockProcessWhitelist(key *storage.ProcessWhitelistKey, locked bool) (*storage.ProcessWhitelist, error) {
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
