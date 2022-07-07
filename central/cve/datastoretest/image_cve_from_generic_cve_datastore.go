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

type imageCVEDataStoreFromGenericStore struct {
	genericStore genericCVEDataStore.DataStore
}

func isImageCVE(genericCVE *storage.CVE) bool {
	if genericCVE.GetType() == storage.CVE_IMAGE_CVE {
		return true
	}
	for _, cveType := range genericCVE.GetTypes() {
		if cveType == storage.CVE_IMAGE_CVE {
			return true
		}
	}
	return false
}

func (s *imageCVEDataStoreFromGenericStore) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return s.genericStore.Search(ctx, q)
}

func (s *imageCVEDataStoreFromGenericStore) SearchImageCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return s.genericStore.SearchCVEs(ctx, q)
}

func (s *imageCVEDataStoreFromGenericStore) SearchRawImageCVEs(ctx context.Context, q *v1.Query) ([]*storage.ImageCVE, error) {
	cves, error := s.genericStore.SearchRawCVEs(ctx, q)
	if error != nil {
		return nil, error
	}
	imageCVES := make([]*storage.ImageCVE, 0, len(cves))
	for ix := range cves {
		cve := cves[ix]
		if isImageCVE(cve) {
			imageCVES = append(imageCVES, utils.ProtoCVEToImageCVE(cve))
		}
	}
	return imageCVES, nil
}

func (s *imageCVEDataStoreFromGenericStore) Exists(ctx context.Context, id string) (bool, error) {
	return s.genericStore.Exists(ctx, id)
}

func (s *imageCVEDataStoreFromGenericStore) Get(ctx context.Context, id string) (*storage.ImageCVE, bool, error) {
	cve, found, err := s.genericStore.Get(ctx, id)
	if err != nil || !found {
		return nil, found, err
	}
	if !isImageCVE(cve) {
		return nil, false, nil
	}
	return utils.ProtoCVEToImageCVE(cve), true, nil
}

func (s *imageCVEDataStoreFromGenericStore) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.genericStore.Count(ctx, q)
}

func (s *imageCVEDataStoreFromGenericStore) GetBatch(ctx context.Context, id []string) ([]*storage.ImageCVE, error) {
	cves, err := s.genericStore.GetBatch(ctx, id)
	if err != nil {
		return nil, err
	}
	imageCVEs := make([]*storage.ImageCVE, 0, len(cves))
	for _, cve := range cves {
		if isImageCVE(cve) {
			imageCVEs = append(imageCVEs, utils.ProtoCVEToImageCVE(cve))
		}
	}
	return imageCVEs, nil
}

func (s *imageCVEDataStoreFromGenericStore) Suppress(ctx context.Context, start *types.Timestamp, duration *types.Duration, cves ...string) error {
	return s.genericStore.Suppress(ctx, start, duration, cves...)
}

func (s *imageCVEDataStoreFromGenericStore) Unsuppress(ctx context.Context, cves ...string) error {
	return s.genericStore.Unsuppress(ctx, cves...)
}

func (s *imageCVEDataStoreFromGenericStore) EnrichImageWithSuppressedCVEs(image *storage.Image) {
	s.genericStore.EnrichImageWithSuppressedCVEs(image)
}
