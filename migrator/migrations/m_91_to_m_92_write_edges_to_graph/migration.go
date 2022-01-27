package m91tom92

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/dackboxhelpers"
	"github.com/stackrox/rox/migrator/migrations/m_91_to_m_92_write_edges_to_graph/sortedkeys"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
)

var (
	migration = types.Migration{
		StartingSeqNum: 91,
		VersionAfter:   storage.Version{SeqNum: 92},
		Run:            writeImageCVEEdgesToGraph,
	}

	readOpts  = gorocksdb.NewDefaultReadOptions()
	writeOpts = gorocksdb.NewDefaultWriteOptions()
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func writeImageCVEEdgesToGraph(databases *types.Databases) error {
	it := databases.RocksDB.NewIterator(readOpts)
	defer it.Close()

	wb := gorocksdb.NewWriteBatch()
	defer wb.Destroy()

	for it.Seek(imageCVEEdgePrefix); it.ValidForPrefix(imageCVEEdgePrefix); it.Next() {
		imageCVEEdgeKey := it.Key().Copy()
		id := rocksdbmigration.GetIDFromPrefixedKey(imageCVEEdgePrefix, imageCVEEdgeKey)
		edgeID, err := dackboxhelpers.FromString(string(id))
		if err != nil {
			return err
		}

		imageKey := getImageKey([]byte(edgeID.ParentID))
		fromKey := getGraphKey(imageKey)
		cveKey := getCVEKey([]byte(edgeID.ChildID))
		edgesInGraph, err := databases.RocksDB.Get(readOpts, fromKey)
		if err != nil {
			return err
		}

		var tos sortedkeys.SortedKeys
		if edgesInGraph.Exists() {
			ss, err := sortedkeys.Unmarshal(edgesInGraph.Data())
			if err != nil {
				return err
			}
			tos = ss
		}
		tos, _ = tos.Insert(cveKey)

		wb.Put(fromKey, tos.Marshal())
		if err := databases.RocksDB.Write(writeOpts, wb); err != nil {
			return errors.Wrap(err, "writing to RocksDB")
		}
	}
	return nil
}
