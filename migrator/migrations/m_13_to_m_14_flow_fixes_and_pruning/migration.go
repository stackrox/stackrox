package m13to14

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/badgerhelpers"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"go.etcd.io/bbolt"
)

var migration = types.Migration{
	StartingSeqNum: 13,
	VersionAfter:   storage.Version{SeqNum: 14},
	Run:            migrateAndPruneNetworkFlows,
}

const (
	oldGlobalPrefix           = "networkFlows"
	newGlobalPrefix           = "networkFlows2"
	listDeploymentsBucketName = "deployments_list"
)

var (
	oldEntityTypeMap map[string]storage.NetworkEntityInfo_Type
	oldL4ProtoMap    map[string]storage.L4Protocol

	updatedTSKey = []byte("\x00")
)

func init() {
	oldEntityTypeMap = make(map[string]storage.NetworkEntityInfo_Type)
	for num := range storage.NetworkEntityInfo_Type_name {
		entityType := storage.NetworkEntityInfo_Type(num)
		oldEntityTypeMap[fmt.Sprintf("%x", entityType)] = entityType
	}

	oldL4ProtoMap = make(map[string]storage.L4Protocol)
	for num := range storage.L4Protocol_name {
		l4Proto := storage.L4Protocol(num)
		oldL4ProtoMap[fmt.Sprintf("%x", l4Proto)] = l4Proto
	}
}

func readDeploymentInfo(db *bbolt.DB) (deploymentIDs map[string]bool, kubeDNSIDs map[string]string, err error) {
	deploymentIDs = make(map[string]bool)
	kubeDNSIDs = make(map[string]string)
	err = db.View(func(tx *bbolt.Tx) error {
		listDeploymentsBucket := tx.Bucket([]byte(listDeploymentsBucketName))
		if listDeploymentsBucket == nil {
			return nil
		}
		return listDeploymentsBucket.ForEach(func(k, v []byte) error {
			deploymentIDs[string(k)] = true
			var deployment storage.ListDeployment
			if err := proto.Unmarshal(v, &deployment); err != nil {
				return err
			}
			if deployment.GetNamespace() == "kube-system" && deployment.GetName() == "kube-dns" {
				kubeDNSIDs[deployment.GetClusterId()] = deployment.GetId()
			}
			return nil
		})
	})
	return
}

func parseIDOld(key []byte) (*storage.NetworkFlowProperties, error) {
	parts := strings.Split(string(key), ":")
	if len(parts) != 6 {
		return nil, errors.Errorf("expected 6 parts in key, got %d", len(parts))
	}
	srcType, ok := oldEntityTypeMap[parts[0]]
	if !ok {
		return nil, errors.Errorf("unknown source entity type %s", parts[0])
	}
	dstType, ok := oldEntityTypeMap[parts[2]]
	if !ok {
		return nil, errors.Errorf("unknown dest entity type %s", parts[2])
	}
	portNum, err := strconv.ParseUint(parts[4], 16, 32)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse destination port number")
	}
	l4Proto, ok := oldL4ProtoMap[parts[5]]
	if !ok {
		return nil, errors.Errorf("unknown L4 protocol %s", parts[5])
	}
	result := &storage.NetworkFlowProperties{
		SrcEntity: &storage.NetworkEntityInfo{
			Type: srcType,
			Id:   parts[1],
		},
		DstEntity: &storage.NetworkEntityInfo{
			Type: dstType,
			Id:   parts[3],
		},
		DstPort:    uint32(portNum),
		L4Protocol: l4Proto,
	}
	return result, nil
}

func makeIDNew(props *storage.NetworkFlowProperties) string {
	return fmt.Sprintf("%x:%s:%x:%s:%x:%x", int32(props.GetSrcEntity().GetType()), props.GetSrcEntity().GetId(), int32(props.GetDstEntity().GetType()), props.GetDstEntity().GetId(), props.GetDstPort(), int32(props.GetL4Protocol()))
}

func isLikelyEphemeralPort(port uint16) bool {
	return port >= 32768
}

func init() {
	migrations.MustRegisterMigration(migration)
}

func migrateFlow(props *storage.NetworkFlowProperties, clusterID string, deploymentIDs map[string]bool, kubeDNSIDs map[string]string) bool {
	// Prune UDP flows with a target port that strongly looks like an ephemeral port.
	if props.GetL4Protocol() == storage.L4Protocol_L4_PROTOCOL_UDP && isLikelyEphemeralPort(uint16(props.GetDstPort())) {
		return false
	}

	// Prune UDP flows from internet to kube-dns.
	if props.GetSrcEntity().GetType() == storage.NetworkEntityInfo_INTERNET && props.GetL4Protocol() == storage.L4Protocol_L4_PROTOCOL_UDP {
		if props.GetDstEntity().GetType() == storage.NetworkEntityInfo_DEPLOYMENT && props.GetDstEntity().GetId() == kubeDNSIDs[clusterID] {
			return false
		}
	}

	// Prune flows referencing non-existing deployments.
	if props.GetSrcEntity().GetType() == storage.NetworkEntityInfo_DEPLOYMENT && !deploymentIDs[props.GetSrcEntity().GetId()] {
		return false
	}
	if props.GetDstEntity().GetType() == storage.NetworkEntityInfo_DEPLOYMENT && !deploymentIDs[props.GetDstEntity().GetId()] {
		return false
	}
	return true
}

func migrateAndPruneNetworkFlows(boltDB *bbolt.DB, badgerDB *badger.DB) error {
	deploymentIDs, kubeDNSIDs, err := readDeploymentInfo(boltDB)
	if err != nil {
		return err
	}

	prefix := []byte(fmt.Sprintf("%s\x00", oldGlobalPrefix))

	err = badgerhelpers.ErrAgain
	for err == badgerhelpers.ErrAgain {
		// This transaction is safe to be split into several transactions, since we delete keys as soon as we have
		// processed them. There is a chance that we migrate an entry and then complete the transaction without deleting
		// the old key, but this does not matter.
		err = badgerhelpers.SplitUpdate(badgerDB, func(tx *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.Prefix = prefix
			it := tx.NewIterator(opts)
			defer it.Close()

			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				item := it.Item()

				keyParts := bytes.SplitN(item.Key(), []byte("\x00"), 3)
				clusterID := string(keyParts[1])
				flowID := keyParts[2]

				if bytes.Equal(flowID, updatedTSKey) {
					newKey := []byte(fmt.Sprintf("%s\x00%s\x00%s", newGlobalPrefix, clusterID, string(updatedTSKey)))
					value, err := item.ValueCopy(nil)
					if err != nil {
						return err
					}
					if err := tx.Set(newKey, value); err != nil {
						return err
					}
					if err := tx.Delete(item.KeyCopy(nil)); err != nil {
						return err
					}
					continue
				}

				props, err := parseIDOld(flowID)
				if err != nil {
					return err
				}

				if migrateFlow(props, clusterID, deploymentIDs, kubeDNSIDs) {
					newKey := []byte(fmt.Sprintf("%s\x00%s\x00%s", newGlobalPrefix, clusterID, makeIDNew(props)))
					value, err := item.ValueCopy(nil)
					if err != nil {
						return err
					}
					if err := tx.Set(newKey, value); err != nil {
						return err
					}
				}
				if err := tx.Delete(item.KeyCopy(nil)); err != nil {
					return err
				}
			}
			return nil
		})
		if err == badgerhelpers.ErrAgain {
			fmt.Println("Tx was too big!")
		}
	}
	return err
}
