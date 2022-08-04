package loaders

import (
	"context"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	legacyImageCVEDataStore "github.com/stackrox/rox/central/cve/datastore"
	distroctx "github.com/stackrox/rox/central/graphql/resolvers/distroctx"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cvss"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var cveLoaderType = reflect.TypeOf(storage.CVE{})

func init() {
	if features.PostgresDatastore.Enabled() {
		return
	}
	// TODO: [ROX-11257, ROX-11258, ROX-11259] Replace this cve loader.
	RegisterTypeFactory(reflect.TypeOf(storage.CVE{}), func() interface{} {
		return NewCVELoader(legacyImageCVEDataStore.Singleton())
	})
}

// NewCVELoader creates a new loader for cve data.
func NewCVELoader(ds legacyImageCVEDataStore.DataStore) CVELoader {
	return &cveLoaderImpl{
		loaded: make(map[string]*storage.CVE),
		ds:     ds,
	}
}

// GetCVELoader returns the CVELoader from the context if it exists.
func GetCVELoader(ctx context.Context) (CVELoader, error) {
	loader, err := GetLoader(ctx, cveLoaderType)
	if err != nil {
		return nil, err
	}
	return loader.(CVELoader), nil
}

// CVELoader loads cve data, and stores already loaded cves for other ops in the same context to use.
type CVELoader interface {
	FromIDs(ctx context.Context, ids []string) ([]*storage.CVE, error)
	FromID(ctx context.Context, id string) (*storage.CVE, error)
	FromQuery(ctx context.Context, query *v1.Query) ([]*storage.CVE, error)
	GetIDs(ctx context.Context, query *v1.Query) ([]string, error)
	CountFromQuery(ctx context.Context, query *v1.Query) (int32, error)
	CountAll(ctx context.Context) (int32, error)
}

// cveLoaderImpl implements the CVEDataLoader interface.
type cveLoaderImpl struct {
	lock   sync.RWMutex
	loaded map[string]*storage.CVE

	ds legacyImageCVEDataStore.DataStore
}

func enrich(distro string, value *storage.CVE) {
	if specifics, ok := value.DistroSpecifics[distro]; ok {
		value.Cvss = specifics.GetCvss()
		value.CvssV2 = specifics.GetCvssV2()
		value.CvssV3 = specifics.GetCvssV3()
		value.Severity = specifics.GetSeverity()
	} else {
		value.Severity = cvss.VulnToSeverity(cvss.NewFromCVE(value))
	}
}

// FromIDs loads a set of cves from a set of ids.
func (idl *cveLoaderImpl) FromIDs(ctx context.Context, ids []string) ([]*storage.CVE, error) {
	cves, err := idl.load(ctx, ids)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

// FromID loads an cve from an ID.
func (idl *cveLoaderImpl) FromID(ctx context.Context, id string) (*storage.CVE, error) {
	cves, err := idl.load(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	return cves[0], nil
}

// FromQuery loads a set of cves that match a query.
func (idl *cveLoaderImpl) FromQuery(ctx context.Context, query *v1.Query) ([]*storage.CVE, error) {
	results, err := idl.ds.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return idl.FromIDs(ctx, search.ResultsToIDs(results))
}

func (idl *cveLoaderImpl) GetIDs(ctx context.Context, query *v1.Query) ([]string, error) {
	results, err := idl.ds.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return search.ResultsToIDs(results), nil
}

func (idl *cveLoaderImpl) CountFromQuery(ctx context.Context, query *v1.Query) (int32, error) {
	count, err := idl.ds.Count(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

func (idl *cveLoaderImpl) CountAll(ctx context.Context) (int32, error) {
	count, err := idl.ds.Count(ctx, search.EmptyQuery())
	return int32(count), err
}

func (idl *cveLoaderImpl) load(ctx context.Context, ids []string) ([]*storage.CVE, error) {
	distro := distroctx.FromContext(ctx)
	cves, missing := idl.readAll(ids)
	if len(missing) > 0 {
		var err error
		cves, err = idl.ds.GetBatch(ctx, collectMissing(ids, missing))
		if err != nil {
			return nil, err
		}
		for _, cve := range cves {
			enrich(distro, cve)
		}
		idl.setAll(cves)
		cves, missing = idl.readAll(ids)
	}
	if len(missing) > 0 {
		missingIDs := make([]string, 0, len(missing))
		for _, m := range missing {
			missingIDs = append(missingIDs, ids[m])
		}
		return nil, errors.Errorf("not all cves could be found: %s", strings.Join(missingIDs, ","))
	}
	return cves, nil
}

func (idl *cveLoaderImpl) setAll(cves []*storage.CVE) {
	idl.lock.Lock()
	defer idl.lock.Unlock()

	for _, cve := range cves {
		idl.loaded[cve.GetId()] = cve
	}
}

func (idl *cveLoaderImpl) readAll(ids []string) (cves []*storage.CVE, missing []int) {
	idl.lock.RLock()
	defer idl.lock.RUnlock()

	for idx, id := range ids {
		cve, isLoaded := idl.loaded[id]
		if !isLoaded {
			missing = append(missing, idx)
		} else {
			cves = append(cves, cve)
		}
	}
	return
}
