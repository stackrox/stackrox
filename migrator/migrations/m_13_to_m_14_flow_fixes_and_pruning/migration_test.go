package m13to14

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	uuid "github.com/satori/go.uuid"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/badgerhelpers"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func flow(srcID, dstID string, dstPort uint32, l4Proto storage.L4Protocol) *storage.NetworkFlow {
	f := &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_INTERNET,
				Id:   srcID,
			},
			DstEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_INTERNET,
				Id:   dstID,
			},
			DstPort:    dstPort,
			L4Protocol: l4Proto,
		},
		LastSeenTimestamp: types.TimestampNow(),
	}
	if f.GetProps().GetSrcEntity().GetId() != "" {
		f.Props.SrcEntity.Type = storage.NetworkEntityInfo_DEPLOYMENT
	}
	if f.GetProps().GetDstEntity().GetId() != "" {
		f.Props.DstEntity.Type = storage.NetworkEntityInfo_DEPLOYMENT
	}
	return f
}

var (
	clusterIDs = []string{
		uuid.NewV4().String(),
		uuid.NewV4().String(),
	}

	miscDeployments = []*storage.ListDeployment{
		{
			Id:        uuid.NewV4().String(),
			Name:      "foo",
			Namespace: "bar",
			ClusterId: clusterIDs[0],
		},
		{
			Id:        uuid.NewV4().String(),
			Name:      "baz",
			Namespace: "qux",
			ClusterId: clusterIDs[0],
		},
		{
			Id:        uuid.NewV4().String(),
			Name:      "something",
			Namespace: "kube-system",
			ClusterId: clusterIDs[1],
		},
		{
			Id:        uuid.NewV4().String(),
			Name:      "else",
			Namespace: "yans",
			ClusterId: clusterIDs[1],
		},
	}

	kubeDNSDeployments = []*storage.ListDeployment{
		{
			Id:        uuid.NewV4().String(),
			Name:      "kube-dns",
			Namespace: "kube-system",
			ClusterId: clusterIDs[0],
		},
		{
			Id:        uuid.NewV4().String(),
			Name:      "kube-dns",
			Namespace: "kube-system",
			ClusterId: clusterIDs[1],
		},
	}

	flowsToPrune = map[string][]*storage.NetworkFlow{
		clusterIDs[0]: {
			// internet to kube DNS on any UDP port
			flow("", kubeDNSDeployments[0].GetId(), 4096, storage.L4Protocol_L4_PROTOCOL_UDP),
			// UDP with ephemeral-looking target port
			flow(miscDeployments[0].GetId(), miscDeployments[1].GetId(), 32769, storage.L4Protocol_L4_PROTOCOL_UDP),
			// flow between existing and non-existing deployment
			flow(miscDeployments[0].GetId(), uuid.NewV4().String(), 80, storage.L4Protocol_L4_PROTOCOL_TCP),
		},
		clusterIDs[1]: {
			// UDP with ephemeral-looking target port
			flow(miscDeployments[2].GetId(), "", 32768, storage.L4Protocol_L4_PROTOCOL_UDP),
			// internet to kube DNS on any UDP port
			flow("", kubeDNSDeployments[1].GetId(), 1, storage.L4Protocol_L4_PROTOCOL_UDP),
			// flow between non-existing deployments
			flow(uuid.NewV4().String(), uuid.NewV4().String(), 443, storage.L4Protocol_L4_PROTOCOL_TCP),
		},
	}

	flowsToRetain = map[string][]*storage.NetworkFlow{
		clusterIDs[0]: {
			// non kube-dns target and port does not strongly look ephemeral
			flow("", miscDeployments[1].GetId(), 4096, storage.L4Protocol_L4_PROTOCOL_UDP),
			// kube-dns target, but TCP connection
			flow("", kubeDNSDeployments[0].GetId(), 4096, storage.L4Protocol_L4_PROTOCOL_TCP),
		},
		clusterIDs[1]: {
			// kube-dns target and UDP, but flow is not coming from the internet
			flow(miscDeployments[2].GetId(), kubeDNSDeployments[1].GetId(), 4096, storage.L4Protocol_L4_PROTOCOL_UDP),
			// ephemeral-looking port, but TCP connection
			flow(miscDeployments[2].GetId(), miscDeployments[3].GetId(), 32768, storage.L4Protocol_L4_PROTOCOL_TCP),
		},
	}
)

func generateRandomFlows(count int) []*storage.NetworkFlow {
	flows := make([]*storage.NetworkFlow, 0, count)
	for i := 0; i < count; i++ {
		flows = append(flows, &storage.NetworkFlow{
			Props: &storage.NetworkFlowProperties{
				SrcEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   uuid.NewV4().String(),
				},
				DstEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   uuid.NewV4().String(),
				},
				DstPort:    443,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: types.TimestampNow(),
		})
	}
	return flows
}

func makeIDOld(props *storage.NetworkFlowProperties) string {
	return fmt.Sprintf("%x:%s:%x:%s:%x:%x", props.GetSrcEntity().GetType(), props.GetSrcEntity().GetId(), props.GetDstEntity().GetType(), props.GetDstEntity().GetId(), props.GetDstPort(), props.GetL4Protocol())
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

type migrationTestSuite struct {
	suite.Suite

	boltDB   *bbolt.DB
	badgerDB *badger.DB
}

func (suite *migrationTestSuite) SetupTest() {
	boltDB, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	suite.Require().NoError(err, "Failed to make BoltDB")

	badgerDB, err := badgerhelpers.NewTemp(testutils.DBFileName(suite))
	suite.Require().NoError(err, "Failed to make Badger DB")

	var allDeployments []*storage.ListDeployment
	allDeployments = append(allDeployments, miscDeployments...)
	allDeployments = append(allDeployments, kubeDNSDeployments...)

	suite.Require().NoError(boltDB.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(listDeploymentsBucketName))
		if err != nil {
			return err
		}
		for _, deployment := range allDeployments {
			bytes, err := proto.Marshal(deployment)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(deployment.GetId()), bytes); err != nil {
				return err
			}
		}
		return nil
	}))

	suite.boltDB, suite.badgerDB = boltDB, badgerDB
}

func (suite *migrationTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.boltDB)
	_ = suite.badgerDB.Close()
}

func (suite *migrationTestSuite) upsertFlowsOld(flowsMap map[string][]*storage.NetworkFlow) {
	suite.Require().NoError(suite.badgerDB.Update(func(tx *badger.Txn) error {
		for clusterID, flows := range flowsMap {
			for _, flow := range flows {
				bytes, err := proto.Marshal(flow)
				if err != nil {
					return err
				}
				key := []byte(fmt.Sprintf("%s\x00%s\x00%s", oldGlobalPrefix, clusterID, makeIDOld(flow.GetProps())))
				if err := tx.Set(key, bytes); err != nil {
					return err
				}
			}
		}
		return nil
	}))
}

func (suite *migrationTestSuite) upsertLastUpdateTSOld(clusterID string, ts *types.Timestamp) {
	tsBytes, err := proto.Marshal(ts)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.badgerDB.Update(func(tx *badger.Txn) error {
		key := []byte(fmt.Sprintf("%s\x00%s\x00%s", oldGlobalPrefix, clusterID, string(updatedTSKey)))
		return tx.Set(key, tsBytes)
	}))
}

func (suite *migrationTestSuite) readLastUpdateTSNew(clusterID string) *types.Timestamp {
	var result *types.Timestamp
	suite.Require().NoError(suite.badgerDB.View(func(tx *badger.Txn) error {
		key := []byte(fmt.Sprintf("%s\x00%s\x00%s", newGlobalPrefix, clusterID, string(updatedTSKey)))
		item, err := tx.Get(key)
		if err != nil {
			return err
		}
		result = &types.Timestamp{}
		return badgerhelpers.UnmarshalProtoValue(item, result)
	}))
	return result
}

func (suite *migrationTestSuite) readFlowsOld() map[string][]*storage.NetworkFlow {
	result := make(map[string][]*storage.NetworkFlow)
	prefix := []byte(fmt.Sprintf("%s\x00", oldGlobalPrefix))
	suite.Require().NoError(suite.badgerDB.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = prefix
		it := tx.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			keyParts := bytes.SplitN(item.Key(), []byte("\x00"), 3)
			clusterID := string(keyParts[1])
			if bytes.Equal(keyParts[2], updatedTSKey) {
				continue
			}
			var flow storage.NetworkFlow
			if err := item.Value(func(v []byte) error { return proto.Unmarshal(v, &flow) }); err != nil {
				return err
			}
			result[clusterID] = append(result[clusterID], &flow)
		}
		return nil
	}))
	return result
}

func (suite *migrationTestSuite) readFlowsNew() map[string][]*storage.NetworkFlow {
	result := make(map[string][]*storage.NetworkFlow)
	prefix := []byte(fmt.Sprintf("%s\x00", newGlobalPrefix))
	suite.Require().NoError(suite.badgerDB.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = prefix
		it := tx.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			keyParts := bytes.SplitN(item.Key(), []byte("\x00"), 3)
			clusterID := string(keyParts[1])
			if bytes.Equal(keyParts[2], updatedTSKey) {
				continue
			}
			var flow storage.NetworkFlow
			if err := item.Value(func(v []byte) error { return proto.Unmarshal(v, &flow) }); err != nil {
				return err
			}
			result[clusterID] = append(result[clusterID], &flow)
		}
		return nil
	}))
	return result
}

func (suite *migrationTestSuite) TestMigration() {
	suite.upsertFlowsOld(flowsToPrune)
	suite.upsertFlowsOld(flowsToRetain)
	// Write a bunch of flows that will be pruned to provoke transaction splitting.
	for i := 0; i < 10; i++ {
		flows := generateRandomFlows(10000)
		flowMap := map[string][]*storage.NetworkFlow{
			clusterIDs[i%2]: flows,
		}
		suite.upsertFlowsOld(flowMap)
	}

	clusterUpdateTSs := make(map[string]*types.Timestamp)
	for _, clusterID := range clusterIDs {
		ts := types.TimestampNow()
		clusterUpdateTSs[clusterID] = ts
		suite.upsertLastUpdateTSOld(clusterID, ts)
	}

	suite.NoError(migrateAndPruneNetworkFlows(suite.boltDB, suite.badgerDB))

	flowsAfterPruning := suite.readFlowsNew()
	for _, clusterID := range clusterIDs {
		suite.ElementsMatch(flowsAfterPruning[clusterID], flowsToRetain[clusterID])
		suite.Equal(clusterUpdateTSs[clusterID], suite.readLastUpdateTSNew(clusterID))
	}

	oldFlowsAfterPruning := suite.readFlowsOld()
	suite.Empty(oldFlowsAfterPruning)
}
