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

	connections := make(map[string]sortedkeys.SortedKeys)
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

		if edgesInGraph.Exists() {
			// Read from the DB only if we do not have the latest snapshot of connections for that image in the map.
			if _, ok := connections[string(imageKey)]; !ok {
				tos, err := sortedkeys.Unmarshal(edgesInGraph.Data())
				if err != nil {
					return err
				}
				connections[string(imageKey)] = connections[string(imageKey)].Union(tos)
			}
		}
		connections[string(imageKey)], _ = connections[string(imageKey)].Insert(cveKey)

		for _, to := range connections {
			wb.Put(fromKey, to.Marshal())
		}

		if wb.Count() == batchSize {
			if err := databases.RocksDB.Write(writeOpts, wb); err != nil {
				return errors.Wrap(err, "writing to RocksDB")
			}
			wb.Clear()
			// We have written the current snapshot to DB so flush the map before moving to next batch,
			// instead of keeping all the connections until the end of the migrations. When we encounter the same
			// image again for some image-cve edge, we have record in the DB, which is pulled out.
			connections = make(map[string]sortedkeys.SortedKeys)
		}
	}
	if wb.Count() != 0 {
		if err := databases.RocksDB.Write(writeOpts, wb); err != nil {
			return errors.Wrap(err, "writing final batch to RocksDB")
		}
	}
	return nil
}
