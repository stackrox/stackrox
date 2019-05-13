package store

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper/crud/proto"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type store struct {
	crud proto.MessageCrud
}

func (s *store) UpsertWhitelistResults(results *storage.ProcessWhitelistResults) error {
	if results.GetDeploymentId() == "" {
		return errors.Errorf("UpsertWhitelistResults received empty deployment id: %+v", results)
	}
	return s.crud.Upsert(results)
}

func (s *store) GetWhitelistResults(deploymentID string) (*storage.ProcessWhitelistResults, error) {
	results, err := s.crud.Read(deploymentID)
	if err != nil {
		return nil, err
	}
	if results == nil {
		return nil, nil
	}
	return results.(*storage.ProcessWhitelistResults), nil
}

func (s *store) DeleteWhitelistResults(deploymentID string) error {
	return s.crud.Delete(deploymentID)
}
