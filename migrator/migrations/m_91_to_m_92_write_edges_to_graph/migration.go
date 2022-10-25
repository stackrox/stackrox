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

const (
	batchSize = 1000
)

var (
	migration = types.Migration{
		StartingSeqNum: 91,
		VersionAfter:   &storage.Version{SeqNum: 92},
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

	connections := make(map[string]sortedkeys.SortedKeys)
	for it.Seek(imageCVEEdgePrefix); it.ValidForPrefix(imageCVEEdgePrefix); it.Next() {
		imageCVEEdgeKey := it.Key().Copy()
		id := rocksdbmigration.GetIDFromPrefixedKey(imageCVEEdgePrefix, imageCVEEdgeKey)
		edgeID, err := dackboxhelpers.FromString(string(id))
		if err != nil {
			return err
		}

		imageKey := getImageKey([]byte(edgeID.ParentID))
		imageKeyString := string(imageKey)
		cveKey := getCVEKey([]byte(edgeID.ChildID))

		// Read from the DB only if we do not have the latest snapshot of connections for that image in the map.
		if _, ok := connections[imageKeyString]; !ok {
			edgesInGraph, err := databases.RocksDB.Get(readOpts, getGraphKey(imageKey))
			if err != nil {
				return err
			}

			tos, err := sortedkeys.Unmarshal(edgesInGraph.Data())
			if err != nil {
				return err
			}
			connections[imageKeyString] = tos
		}
		connections[imageKeyString], _ = connections[imageKeyString].Insert(cveKey)
	}

	for from, tos := range connections {
		wb.Put(getGraphKey([]byte(from)), tos.Marshal())
		if wb.Count() == batchSize {
			if err := databases.RocksDB.Write(writeOpts, wb); err != nil {
				return errors.Wrap(err, "writing to RocksDB")
			}
			wb.Clear()
		}
	}
	if wb.Count() != 0 {
		if err := databases.RocksDB.Write(writeOpts, wb); err != nil {
			return errors.Wrap(err, "writing final batch to RocksDB")
		}
	}
	return nil
}
