package postgres

//import (
//	"context"
//	"fmt"
//	"os"
//	"testing"
//
//	"github.com/gogo/protobuf/types"
//	"github.com/jackc/pgx/v4/pgxpool"
//	jsoniter "github.com/json-iterator/go"
//	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/common"
//	"github.com/stackrox/rox/generated/storage"
//	"github.com/stackrox/rox/pkg/batcher"
//	"github.com/stackrox/rox/pkg/postgres"
//	"github.com/stackrox/rox/pkg/timestamp"
//	"github.com/stackrox/rox/pkg/uuid"
//)
//
//const createTable = `
//create table if not exists networkflows_flat (
//	id varchar primary key,
//	lastSeen TIMESTAMP,
//	dstPort integer,
//	l4Protocol integer,
//	srcEntity_type integer,
//	srcEntity_id varchar,
//	srcEntity_deployment_name varchar,
//	srcEntity_deployment_namespace varchar,
//	srcEntity_deployment_listenPorts json,
//	srcEntity_externalSource_name varchar,
//	srcEntity_externalSource_default bool,
//	srcEntity_externalSource_cidr varchar,
//	dstEntity_type integer,
//	dstEntity_id varchar,
//	dstEntity_deployment_name varchar,
//	dstEntity_deployment_namespace varchar,
//	dstEntity_deployment_listenPorts json,
//	dstEntity_externalSource_name varchar,
//	dstEntity_externalSource_default bool,
//	dstEntity_externalSource_cidr varchar
//)
//`
//
//const (
//	flatBatchInsertTemplate = "insert into networkflows_flat (id, lastSeen, dstPort, l4Protocol, srcEntity_type, srcEntity_id, srcEntity_deployment_name, srcEntity_deployment_namespace, srcEntity_deployment_listenPorts, srcEntity_externalSource_name, srcEntity_externalSource_default, srcEntity_externalSource_cidr, dstEntity_type, dstEntity_id, dstEntity_deployment_name, dstEntity_deployment_namespace, dstEntity_deployment_listenPorts, dstEntity_externalSource_name, dstEntity_externalSource_default, dstEntity_externalSource_cidr) values %s on conflict(id) do nothing"
//)
//
//func flatInsert(conn *pgxpool.Conn, flows []*storage.NetworkFlow) {
//	numElems := 20
//	batch := batcher.New(len(flows), 60000/numElems)
//	for start, end, ok := batch.Next(); ok; start, end, ok = batch.Next() {
//		var placeholderStr string
//		data := make([]interface{}, 0, numElems*len(flows))
//		for i, obj := range flows[start:end] {
//			if i != 0 {
//				placeholderStr += ", "
//			}
//			placeholderStr += postgres.GetValues(i*numElems+1, (i+1)*numElems+1)
//			id := common.GetIDString(obj.GetProps())
//			props := obj.GetProps()
//			src := obj.GetProps().GetSrcEntity()
//			dst := obj.GetProps().GetDstEntity()
//
//			srcListenPorts, err := jsoniter.MarshalToString(src.GetDeployment().GetListenPorts())
//			if err != nil {
//				panic(err)
//			}
//
//			dstListenPorts, err := jsoniter.MarshalToString(dst.GetDeployment().GetListenPorts())
//			if err != nil {
//				panic(err)
//			}
//
//			data = append(data,
//				id,
//				obj.GetLastSeenTimestamp().String(),
//				props.GetDstPort(),
//				props.GetL4Protocol(),
//				src.GetType(),
//				src.GetId(),
//				src.GetDeployment().GetName(),
//				src.GetDeployment().GetNamespace(),
//				srcListenPorts,
//				src.GetExternalSource().GetName(),
//				src.GetExternalSource().GetDefault(),
//				src.GetExternalSource().GetCidr(),
//				dst.GetType(),
//				dst.GetId(),
//				dst.GetDeployment().GetName(),
//				dst.GetDeployment().GetNamespace(),
//				dstListenPorts,
//				dst.GetExternalSource().GetName(),
//				dst.GetExternalSource().GetDefault(),
//				dst.GetExternalSource().GetCidr(),
//			)
//		}
//		if _, err := conn.Exec(context.Background(), fmt.Sprintf(flatBatchInsertTemplate, placeholderStr), data...); err != nil {
//			panic(err)
//		}
//	}
//}
//
//func setup() *pgxpool.Pool {
//	config, err := pgxpool.ParseConfig("pool_min_conns=100 pool_max_conns=100 host=localhost database=postgres port=5432 user=connorgorman sslmode=disable statement_timeout=60000")
//	if err != nil {
//		panic(err)
//	}
//
//	db, err := pgxpool.ConnectConfig(context.Background(), config)
//	if err != nil {
//		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
//		os.Exit(1)
//	}
//	return db
//}
//
//func getFlows() []*storage.NetworkFlow {
//	numFlows := 1000000
//	flows := make([]*storage.NetworkFlow, 0, numFlows)
//	for i := 0; i < numFlows; i++ {
//		flows = append(flows, &storage.NetworkFlow{
//			Props:                &storage.NetworkFlowProperties{
//				SrcEntity:            &storage.NetworkEntityInfo{
//					Type:                 1,
//					Id:                   uuid.NewV4().String(),
//					Desc: &storage.NetworkEntityInfo_Deployment_{
//						Deployment: &storage.NetworkEntityInfo_Deployment{
//							Name:                 "central",
//							Namespace:            "stackrox",
//							//ListenPorts:          nil,
//						},
//					},
//				},
//				DstEntity:            &storage.NetworkEntityInfo{
//					Type:                 1,
//					Id:                   uuid.NewV4().String(),
//					Desc: &storage.NetworkEntityInfo_Deployment_{
//						Deployment: &storage.NetworkEntityInfo_Deployment{
//							Name:                 "central",
//							Namespace:            "stackrox",
//							//ListenPorts:          nil,
//						},
//					},
//				},
//				DstPort:              21477,
//				L4Protocol:           1,
//			},
//			LastSeenTimestamp:    types.TimestampNow(),
//		})
//	}
//	return flows
//}
//
//func BenchmarkFlatInsert(b *testing.B) {
//	db := setup()
//	defer db.Close()
//
//	conn, err := db.Acquire(context.Background())
//	if err != nil {
//		panic(err)
//	}
//	if _, err := conn.Exec(context.Background(), createTable); err != nil {
//		panic(err)
//	}
//
//	if _, err := db.Exec(context.Background(), `create index if not exists networkflows_flat_dst_entity on networkflows_flat using hash (srcEntity_id)`); err != nil {
//		panic(err)
//	}
//	if _, err := db.Exec(context.Background(), `create index if not exists networkflows_flat_src_entity on networkflows_flat using hash (dstEntity_id)`); err != nil {
//		panic(err)
//	}
//
//	defer conn.Release()
//
//	flows := getFlows()
//	b.ResetTimer()
//	for i := 0; i < b.N; i++ {
//		flatInsert(conn, flows)
//	}
//}
//
//func BenchmarkJSONInsert(b *testing.B) {
//	db := setup()
//	defer db.Close()
//
//	conn, err := db.Acquire(context.Background())
//	if err != nil {
//		panic(err)
//	}
//	conn.Release()
//
//	NewClusterStore(db)
//	store := &flowStoreImpl{db: db, clusterID: "clusterid"}
//
//	flows := getFlows()
//	b.ResetTimer()
//	for i := 0; i < b.N; i++ {
//		err := store.UpsertFlows(flows, timestamp.Now())
//		if err != nil {
//			panic(err)
//		}
//	}
//}
