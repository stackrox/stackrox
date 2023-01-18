package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	metrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/processlisteningonport/store"
	"github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
)

// NewFullStore augments the generated store with GetProcessListeningOnPort functions.
func NewFullStore(db *pgxpool.Pool) store.Store {
	return &fullStoreImpl{
		Store: New(db),
		db:    db,
	}
}

// FullStoreWrap augments the wrapped store with GetProcessListeningOnPort functions.
func FullStoreWrap(wrapped store.Store) store.Store {
	return &fullStoreImpl{
		Store: wrapped,
	}
}

type fullStoreImpl struct {
	Store
	db *pgxpool.Pool
}

// NewFullTestStore is used for testing.
func NewFullTestStore(_ testing.TB, store store.Store) store.Store {
	return &fullStoreImpl{
		Store: store,
	}
}

// SQL query to join process_listening_on_port together with
// process_indicators. Used to provide information for queries like 'give
// me all PLOP by this deployment'.
// XXX: Verify the query plan to make sure needed indexes are in use.
const getByDeploymentStmt = "SELECT plop.id, plop.serialized, " +
	"proc.podid, proc.containername, " +
	"proc.signal_containerid, " +
	"proc.signal_name, proc.signal_args, proc.signal_execfilepath, " +
	"proc.clusterid, proc.serialized as procSerialized " +
	"FROM process_listening_on_ports plop " +
	"JOIN process_indicators proc " +
	"ON plop.processindicatorid = proc.id " +
	"WHERE proc.deploymentid = $1 AND plop.closed = false"

// Manually written function to get PLOP joined with ProcessIndicators
func (s *fullStoreImpl) GetProcessListeningOnPort(
	ctx context.Context,
	deploymentID string,
) ([]*storage.ProcessListeningOnPort, error) {
	defer metrics.SetPostgresOperationDurationTime(
		time.Now(),
		ops.GetProcessListeningOnPort,
		"ProcessListeningOnPortStorage",
	)

	return pgutils.Retry2(func() ([]*storage.ProcessListeningOnPort, error) {
		return s.retryableGetPLOP(ctx, deploymentID)
	})
}

func (s *fullStoreImpl) retryableGetPLOP(
	ctx context.Context,
	deploymentID string,
) ([]*storage.ProcessListeningOnPort, error) {
	var rows pgx.Rows
	var err error

	rows, err = s.db.Query(ctx, getByDeploymentStmt, deploymentID)

	if err != nil {
		log.Warnf("%s: %s", getByDeploymentStmt, err)
		return nil, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()

	return s.readRows(rows)
}

// Manual converting of raw data from SQL query to ProcessListeningOnPort (not
// ProcessListeningOnPortStorage) object enriched with ProcessIndicator info.
func (s *fullStoreImpl) readRows(
	rows pgx.Rows,
) ([]*storage.ProcessListeningOnPort, error) {
	var plops []*storage.ProcessListeningOnPort

	for rows.Next() {
		var id string
		var serialized []byte
		var podID string
		var containerName string
		var signalContainerID string
		var signalName string
		var signalArgs string
		var signalExecFilePath string
		var clusterID string
		var procSerialized []byte

		// We're getting ProcessIndicator directly from the SQL query, PLOP
		// parts have to be extra deserialized.
		if err := rows.Scan(
			&id, &serialized,
			&podID, &containerName,
			&signalContainerID,
			&signalName, &signalArgs, &signalExecFilePath,
			&clusterID, &procSerialized); err != nil {
			return nil, pgutils.ErrNilIfNoRows(err)
		}

		var msg storage.ProcessListeningOnPortStorage
		if err := proto.Unmarshal(serialized, &msg); err != nil {
			return nil, err
		}

		var procMsg storage.ProcessIndicator
		if err := proto.Unmarshal(procSerialized, &procMsg); err != nil {
			return nil, err
		}

		plop := &storage.ProcessListeningOnPort{
			Endpoint: &storage.ProcessListeningOnPort_Endpoint{
				Port:     msg.Port,
				Protocol: msg.Protocol,
			},
			DeploymentId:  procMsg.DeploymentId,
			PodId:         podID,
			PodUid:        procMsg.PodUid,
			ContainerName: containerName,
			Signal: &storage.ProcessSignal{
				Id:           procMsg.Signal.Id,
				ContainerId:  signalContainerID,
				Time:         procMsg.Signal.Time,
				Name:         signalName,
				Args:         signalArgs,
				ExecFilePath: signalExecFilePath,
				Pid:          procMsg.Signal.Pid,
				Uid:          procMsg.Signal.Uid,
				Gid:          procMsg.Signal.Gid,
				Lineage:      procMsg.Signal.Lineage,
				Scraped:      procMsg.Signal.Scraped,
				LineageInfo:  procMsg.Signal.LineageInfo,
			},
			ClusterId:          clusterID,
			Namespace:          procMsg.Namespace,
			ContainerStartTime: procMsg.ContainerStartTime,
			ImageId:            procMsg.ImageId,
		}

		plops = append(plops, plop)
	}

	log.Debugf("Read returned %+v plops", len(plops))
	return plops, nil
}
