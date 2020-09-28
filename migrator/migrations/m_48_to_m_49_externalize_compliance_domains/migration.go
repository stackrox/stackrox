package m48tom49

import (
	"fmt"
	"strings"

	uuid "github.com/satori/go.uuid"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
	"github.com/tecbot/gorocksdb"
)

var (
	migration = types.Migration{
		StartingSeqNum: 48,
		VersionAfter:   storage.Version{SeqNum: 49},
		Run:            migrateComplianceDomains,
	}

	defaultWriteOptions = generic.DefaultWriteOptions()
	defaultReadOptions  = generic.DefaultReadOptions()
)

func init() {
	migrations.MustRegisterMigration(migration)
}

var (
	resultsBucketName = []byte("compliance-run-results")

	resultsKey  = rocksdbmigration.GetPrefixedKey(resultsBucketName, []byte("results"))
	domainKey   = rocksdbmigration.GetPrefixedKey(resultsBucketName, []byte("domain"))
	metadataKey = rocksdbmigration.GetPrefixedKey(resultsBucketName, []byte("metadata"))
)

func migrateComplianceDomains(databases *types.Databases) error {
	iterator := databases.RocksDB.NewIterator(defaultReadOptions)
	defer iterator.Close()
	for iterator.Seek(metadataKey); iterator.ValidForPrefix(metadataKey); iterator.Next() {
		mdSlice := iterator.Value()
		mdBytes := mdSlice.Data()

		mdKeySlice := iterator.Key()
		mdKey := mdKeySlice.Data()

		var metadata storage.ComplianceRunMetadata
		if err := metadata.Unmarshal(mdBytes); err != nil {
			log.WriteToStderrf("Error unmarshalling metadata for %s: %s", string(mdKey), err.Error())
		}

		if metadata.GetDomainId() != "" {
			continue
		}

		migrateDomain(databases.RocksDB, mdKey, &metadata)
	}

	return nil
}

// migrateDomain returns the generated domain ID to enable convenient testing
func migrateDomain(rocksDB *gorocksdb.DB, mdKey []byte, metadata *storage.ComplianceRunMetadata) []byte {
	runKey := string(append([]byte{}, mdKey...))
	runKey = strings.Replace(runKey, string(metadataKey), string(resultsKey), 1)
	runSlice, err := rocksDB.Get(defaultReadOptions, []byte(runKey))
	if err != nil {
		return nil
	}
	defer runSlice.Free()
	if !runSlice.Exists() {
		log.WriteToStderrf("no run data found for key %s", string(runKey))
		return nil
	}
	runBytes := runSlice.Data()

	var run storage.ComplianceRunResults
	if err := run.Unmarshal(runBytes); err != nil {
		log.WriteToStderrf("Error unmarshalling run for %s: %s", string(runKey), err.Error())
	}
	if run.GetDomain() == nil || run.GetDomain().GetId() != "" {
		return nil
	}

	domain := run.GetDomain()
	domain.Id = uuid.NewV4().String()
	run.Domain = nil
	run.GetRunMetadata().DomainId = domain.GetId()
	metadata.DomainId = domain.GetId()

	mdBytes, err := metadata.Marshal()
	if err != nil {
		log.WriteToStderrf("Error marshalling metadata for %s: %s", string(metadataKey), err.Error())
		return nil
	}

	runBytes, err = run.Marshal()
	if err != nil {
		log.WriteToStderrf("Error marshalling run for %s: %s", string(runKey), err.Error())
		return nil
	}

	domainBytes, err := domain.Marshal()
	if err != nil {
		log.WriteToStderrf("Error marshalling domain for %s: %s", string(domainKey), err.Error())
		return nil
	}

	domKey := getDomainKey(domain.GetCluster().GetId(), domain.GetId())

	batch := gorocksdb.NewWriteBatch()
	defer batch.Clear()

	batch.Put(mdKey, mdBytes)
	batch.Put([]byte(runKey), runBytes)
	batch.Put([]byte(domKey), domainBytes)
	if err := rocksDB.Write(defaultWriteOptions, batch); err != nil {
		log.WriteToStderrf("Error writing externalized domain batch: %s", err.Error())
	}

	return domKey
}

func getDomainKey(clusterID, domainID string) []byte {
	// Store externalized domain under the key "compliance-run-results\x00domain:CLUSTER:DOMAIN_ID.
	// Note the lack of a standard ID as all standard results for the same cluster will have the same domain.
	return []byte(fmt.Sprintf("%s:%s:%s", string(domainKey), clusterID, domainID))
}
