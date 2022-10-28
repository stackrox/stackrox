package datastoretest

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/cve/converter/utils"
	genericCVEDataStore "github.com/stackrox/rox/central/cve/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

type nodeCVEDataStoreFromGenericStore struct {
	genericStore genericCVEDataStore.DataStore
}

func isNodeCVE(genericCVE *storage.CVE) bool {
	if genericCVE.GetType() == storage.CVE_NODE_CVE {
		return true
	}
	for _, cveType := range genericCVE.GetTypes() {
		if cveType == storage.CVE_NODE_CVE {
			return true
		}
	}
	return false
}

func (s *nodeCVEDataStoreFromGenericStore) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return s.genericStore.Search(ctx, q)
}

func (s *nodeCVEDataStoreFromGenericStore) SearchNodeCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return s.genericStore.SearchCVEs(ctx, q)
}

func (s *nodeCVEDataStoreFromGenericStore) SearchRawCVEs(ctx context.Context, q *v1.Query) ([]*storage.NodeCVE, error) {
	cves, err := s.genericStore.SearchRawCVEs(ctx, q)
	if err != nil {
		return nil, err
	}
	nodeCVEs := make([]*storage.NodeCVE, 0, len(cves))
	for _, cve := range cves {
		if isNodeCVE(cve) {
			nodeCVEs = append(nodeCVEs, utils.ProtoCVEToNodeCVE(cve))
		}
	}
	return nodeCVEs, nil
}

func (s *nodeCVEDataStoreFromGenericStore) Exists(ctx context.Context, id string) (bool, error) {
	return s.genericStore.Exists(ctx, id)
}

func (s *nodeCVEDataStoreFromGenericStore) Get(ctx context.Context, id string) (*storage.NodeCVE, bool, error) {
	cve, found, err := s.genericStore.Get(ctx, id)
	if err != nil || !found {
		return nil, found, err
	}
	if !isNodeCVE(cve) {
		return nil, false, nil
	}
	return utils.ProtoCVEToNodeCVE(cve), true, nil
}

func (s *nodeCVEDataStoreFromGenericStore) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.genericStore.Count(ctx, q)
}

func (s *nodeCVEDataStoreFromGenericStore) GetBatch(ctx context.Context, id []string) ([]*storage.NodeCVE, error) {
	cves, err := s.genericStore.GetBatch(ctx, id)
	if err != nil {
		return nil, err
	}
	nodeCVEs := make([]*storage.NodeCVE, 0, len(cves))
	for _, cve := range cves {
		if isNodeCVE(cve) {
			nodeCVEs = append(nodeCVEs, utils.ProtoCVEToNodeCVE(cve))
		}
	}
	return nodeCVEs, nil
}

func (s *nodeCVEDataStoreFromGenericStore) Suppress(ctx context.Context, start *types.Timestamp, duration *types.Duration, cves ...string) error {
	return s.genericStore.Suppress(ctx, start, duration, cves...)
}

func (s *nodeCVEDataStoreFromGenericStore) Unsuppress(ctx context.Context, cves ...string) error {
	return s.genericStore.Unsuppress(ctx, cves...)
}

func (s *nodeCVEDataStoreFromGenericStore) EnrichNodeWithSuppressedCVEs(node *storage.Node) {
	s.genericStore.EnrichNodeWithSuppressedCVEs(node)
}
