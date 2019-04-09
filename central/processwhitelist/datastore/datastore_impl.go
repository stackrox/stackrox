package datastore

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/processwhitelist/index"
	"github.com/stackrox/rox/central/processwhitelist/search"
	"github.com/stackrox/rox/central/processwhitelist/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func makeID(deploymentID, containerName string) string {
	return fmt.Sprintf("%s/%s", deploymentID, containerName)
}

func (ds *datastoreImpl) SearchRawProcessWhitelists(q *v1.Query) ([]*storage.ProcessWhitelist, error) {
	return ds.searcher.SearchRawProcessWhitelists(q)
}

func (ds *datastoreImpl) GetProcessWhitelist(id string) (*storage.ProcessWhitelist, error) {
	return ds.storage.GetWhitelist(id)
}

func (ds *datastoreImpl) GetProcessWhitelistByNames(deploymentID, containerName string) (*storage.ProcessWhitelist, error) {
	id := makeID(deploymentID, containerName)
	return ds.storage.GetWhitelist(id)
}

func (ds *datastoreImpl) GetProcessWhitelists() ([]*storage.ProcessWhitelist, error) {
	return ds.storage.GetWhitelists()
}

func (ds *datastoreImpl) AddProcessWhitelist(whitelist *storage.ProcessWhitelist) (string, error) {
	id := makeID(whitelist.GetDeploymentId(), whitelist.GetContainerName())
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

func (ds *datastoreImpl) RemoveProcessWhitelist(id string) error {
	if err := ds.indexer.DeleteWhitelist(id); err != nil {
		return errors.Wrap(err, "error removing whitelist from index")
	}
	if _, err := ds.storage.DeleteWhitelist(id); err != nil {
		return errors.Wrap(err, "error removing whitelist from store")
	}
	return nil
}
