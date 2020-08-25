package m45tom46

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/dackboxhelpers"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
)

const (
	batchSize = 500
)

var (
	imagePrefix             = []byte("imageBucket")
	imageToComponentsPrefix = []byte("image_to_comp")
	imageToCVEPrefix        = []byte("image_to_cve")
	componentsToCVEsPrefix  = []byte("comp_to_vuln")

	migration = types.Migration{
		StartingSeqNum: 45,
		VersionAfter:   storage.Version{SeqNum: 46},
		Run:            writeImageCVEEdges,
	}

	readOpts  = gorocksdb.NewDefaultReadOptions()
	writeOpts = gorocksdb.NewDefaultWriteOptions()
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func writeImageCVEEdges(databases *types.Databases) error {
	imageToComponents, err := getEdges(databases.RocksDB, imageToComponentsPrefix)
	if err != nil {
		return err
	}
	componentToCVEs, err := getEdges(databases.RocksDB, componentsToCVEsPrefix)
	if err != nil {
		return err
	}

	it := databases.RocksDB.NewIterator(readOpts)
	defer it.Close()

	wb := gorocksdb.NewWriteBatch()
	for it.Seek(imagePrefix); it.ValidForPrefix(imagePrefix); it.Next() {
		imageID := rocksdbmigration.GetIDFromPrefixedKey(imagePrefix, it.Key().Copy())

		componentIDs, ok := imageToComponents[string(imageID)]
		if !ok {
			continue
		}

		var image storage.Image
		if err := proto.Unmarshal(it.Value().Data(), &image); err != nil {
			return errors.Wrapf(err, "unmarshaling %s", imageID)
		}

		if image.GetScan().GetScanTime() == nil {
			continue
		}

		for _, componentID := range componentIDs {
			cveIDs, ok := componentToCVEs[componentID]
			if !ok {
				continue
			}

			for _, cveID := range cveIDs {
				imageCVEEdge := &storage.ImageCVEEdge{
					Id: dackboxhelpers.EdgeID{ParentID: string(imageID), ChildID: cveID}.ToString(),
					// We do not know when exactly a CVE was surfaced for an image. Therefore we tie it to the last scan time.
					FirstImageOccurrence: image.GetScan().GetScanTime(),
				}

				newKey := rocksdbmigration.GetPrefixedKey(imageToCVEPrefix, []byte(imageCVEEdge.GetId()))
				newData, err := proto.Marshal(imageCVEEdge)
				if err != nil {
					return errors.Wrapf(err, "marshaling %s", newKey)
				}
				wb.Put(newKey, newData)
			}

			if wb.Count() == batchSize {
				if err := databases.RocksDB.Write(writeOpts, wb); err != nil {
					return errors.Wrap(err, "writing to RocksDB")
				}
				wb.Clear()
			}
		}
	}
	if wb.Count() != 0 {
		if err := databases.RocksDB.Write(writeOpts, wb); err != nil {
			return errors.Wrap(err, "writing final batch to RocksDB")
		}
	}
	return nil
}

func getEdges(db *gorocksdb.DB, prefix []byte) (map[string][]string, error) {
	it := db.NewIterator(gorocksdb.NewDefaultReadOptions())
	defer it.Close()

	edges := make(map[string][]string)
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		id := rocksdbmigration.GetIDFromPrefixedKey(prefix, it.Key().Copy())
		edgeID, err := dackboxhelpers.FromString(string(id))
		if err != nil {
			return nil, err
		}
		edges[edgeID.ParentID] = append(edges[edgeID.ParentID], edgeID.ChildID)
	}
	return edges, nil
}
