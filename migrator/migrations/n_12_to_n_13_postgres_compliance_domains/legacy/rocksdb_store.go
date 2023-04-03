// This file was originally generated with
// //go:generate cp ../../../../central/compliance/datastore/internal/store/rocksdb/rocksdb_store.go .

package legacy

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/rocksdb"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
	"github.com/tecbot/gorocksdb"
)

var (
	readOptions  = generic.DefaultReadOptions()
	writeOptions = generic.DefaultWriteOptions()

	resultsBucketName = []byte("compliance-run-results")

	domainKey = dbhelper.GetBucketKey(resultsBucketName, []byte("domain"))
)

// New returns a compliance results store that is backed by RocksDB.
func New(db *rocksdb.RocksDB) (Store, error) {
	return &rocksdbStore{
		db: db,
	}, nil
}

type rocksdbStore struct {
	db *rocksdb.RocksDB
}

func (r *rocksdbStore) Walk(_ context.Context, fn func(obj *storage.ComplianceDomain) error) error {
	iterator := r.db.NewIterator(readOptions)
	defer iterator.Close()
	// Runs are sorted by time so we must iterate over each key to see if it has the correct run ID.
	for iterator.Seek(domainKey); iterator.ValidForPrefix(domainKey); iterator.Next() {
		domain, err := unmarshalDomain(iterator)
		if err != nil {
			return err
		}
		if err = fn(domain); err != nil {
			return err
		}
	}
	return nil
}

func (r *rocksdbStore) UpsertMany(ctx context.Context, objs []*storage.ComplianceDomain) error {
	for _, obj := range objs {
		if err := r.StoreComplianceDomain(ctx, obj); err != nil {
			return err
		}
	}
	return nil
}

func unmarshalDomain(iterator *gorocksdb.Iterator) (*storage.ComplianceDomain, error) {
	bytes := iterator.Value().Data()
	if len(bytes) == 0 {
		return nil, errors.New("compliance domain data is empty")
	}
	var domain storage.ComplianceDomain
	if err := domain.Unmarshal(bytes); err != nil {
		return nil, errors.Wrap(err, "unmarshalling compliance domain")
	}
	return &domain, nil
}

func getDomainKey(clusterID, domainID string) []byte {
	// Store externalized domain under the key "compliance-run-results\x00domain:CLUSTER:DOMAIN_ID.
	// Note the lack of a standard ID as all standard results for the same cluster will have the same domain.
	return []byte(fmt.Sprintf("%s:%s:%s", string(domainKey), clusterID, domainID))
}

func (r *rocksdbStore) StoreComplianceDomain(_ context.Context, domain *storage.ComplianceDomain) error {
	serializedDomain, err := domain.Marshal()
	if err != nil {
		return errors.Wrap(err, "serializing domain")
	}

	domainKey := getDomainKey(domain.GetCluster().GetId(), domain.GetId())
	err = r.db.Put(writeOptions, domainKey, serializedDomain)
	return errors.Wrap(err, "storing domain")
}
