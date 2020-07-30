package m43tom44

import (
	"encoding/base64"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
)

const (
	batchSize = 500
)

var (
	bindingPrefix = []byte("rolebindings\x00")

	migration = types.Migration{
		StartingSeqNum: 43,
		VersionAfter:   storage.Version{SeqNum: 44},
		Run: func(databases *types.Databases) error {
			return runEnrichSubjects(databases.RocksDB)
		},
	}

	readOpts  = gorocksdb.NewDefaultReadOptions()
	writeOpts = gorocksdb.NewDefaultWriteOptions()
)

func createSubjectID(clusterID, subjectName string) string {
	clusterEncoded := base64.URLEncoding.EncodeToString([]byte(clusterID))
	subjectEncoded := base64.URLEncoding.EncodeToString([]byte(subjectName))
	return fmt.Sprintf("%s:%s", clusterEncoded, subjectEncoded)
}

func enrichSubjects(binding *storage.K8SRoleBinding) {
	for _, subject := range binding.GetSubjects() {
		subject.ClusterId = binding.GetClusterId()
		subject.ClusterName = binding.GetClusterName()
		subject.Id = createSubjectID(binding.GetClusterId(), subject.GetName())
	}
}

func runEnrichSubjects(db *gorocksdb.DB) error {
	log.WriteToStderr("Enriching subjects with k8s role bindings")

	it := db.NewIterator(readOpts)
	defer it.Close()

	wb := gorocksdb.NewWriteBatch()
	var totalRewrites int
	for it.Seek(bindingPrefix); it.ValidForPrefix(bindingPrefix); it.Next() {
		var roleBinding storage.K8SRoleBinding
		key := it.Key().Copy()
		if err := proto.Unmarshal(it.Value().Data(), &roleBinding); err != nil {
			return errors.Wrapf(err, "unmarshaling %s", key)
		}

		enrichSubjects(&roleBinding)

		newData, err := proto.Marshal(&roleBinding)
		if err != nil {
			return errors.Wrapf(err, "marshaling %s", key)
		}

		wb.Put(key, newData)
		totalRewrites++

		if totalRewrites%batchSize == 0 {
			if err := db.Write(writeOpts, wb); err != nil {
				return errors.Wrap(err, "writing to RocksDB")
			}
			wb.Clear()
		}
	}
	if err := db.Write(writeOpts, wb); err != nil {
		return errors.Wrap(err, "writing final batch to RocksDB")
	}
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
