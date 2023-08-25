package m104tom105

import (
	"sort"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
)

const (
	batchSize = 500
)

var (
	activeComponentsPrefix = []byte("active_components")

	migration = types.Migration{
		StartingSeqNum: 104,
		VersionAfter:   &storage.Version{SeqNum: 105},
		Run: func(databases *types.Databases) error {
			return updateActiveComponents(databases.RocksDB)
		},
	}

	readOpts  = gorocksdb.NewDefaultReadOptions()
	writeOpts = gorocksdb.NewDefaultWriteOptions()
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func convertActiveContextsMapToSlice(contextMap map[string]*storage.ActiveComponent_ActiveContext) []*storage.ActiveComponent_ActiveContext {
	contexts := make([]*storage.ActiveComponent_ActiveContext, 0, len(contextMap))
	for _, ctx := range contextMap {
		contexts = append(contexts, ctx)
	}
	sort.SliceStable(contexts, func(i, j int) bool {
		return contexts[i].ContainerName < contexts[j].ContainerName
	})
	return contexts
}

func decomposeID(id string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return "", "", errors.Errorf("invalid active component id: %q", id)
	}
	return parts[0], parts[1], nil
}

func updateActiveComponents(db *gorocksdb.DB) error {
	it := db.NewIterator(readOpts)
	defer it.Close()

	wb := gorocksdb.NewWriteBatch()
	for it.Seek(activeComponentsPrefix); it.ValidForPrefix(activeComponentsPrefix); it.Next() {
		var activeComponent storage.ActiveComponent
		if err := proto.Unmarshal(it.Value().Data(), &activeComponent); err != nil {
			return errors.Wrap(err, "unable to marshal active component")
		}

		deploymentID, componentID, err := decomposeID(activeComponent.GetId())
		if err != nil {
			return errors.Wrap(err, "unable to decompose active component ID")
		}
		activeComponent.DeploymentId = deploymentID
		activeComponent.ComponentId = componentID
		activeComponent.ActiveContextsSlice = convertActiveContextsMapToSlice(activeComponent.GetDEPRECATEDActiveContexts())
		activeComponent.DEPRECATEDActiveContexts = nil

		data, err := proto.Marshal(&activeComponent)
		if err != nil {
			return errors.Wrap(err, "unable to marshal active component")
		}

		wb.Put(it.Key().Copy(), data)

		if wb.Count() == batchSize {
			if err := db.Write(writeOpts, wb); err != nil {
				return errors.Wrap(err, "writing to RocksDB")
			}
			wb.Clear()
		}
	}

	if wb.Count() != 0 {
		if err := db.Write(writeOpts, wb); err != nil {
			return errors.Wrap(err, "writing final batch to RocksDB")
		}
	}
	return nil
}
