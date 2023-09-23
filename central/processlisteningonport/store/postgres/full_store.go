package postgres

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/jackc/pgx/v4"
	metrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/processlisteningonport/store"
	"github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
)

// NewFullStore augments the generated store with GetProcessListeningOnPort functions.
func NewFullStore(db postgres.DB) store.Store {
	return &fullStoreImpl{
		Store: New(db),
		db:    db,
	}
}

type fullStoreImpl struct {
	Store
	db postgres.DB
}

// SQL query to join process_listening_on_port together with
// process_indicators. Used to provide information for queries like 'give
// me all PLOP by this deployment'.
// XXX: Verify the query plan to make sure needed indexes are in use.
const getByDeploymentStmt = "SELECT plop.id, plop.serialized, " +
	"proc.serialized as proc_serialized " +
	"FROM listening_endpoints plop " +
	"LEFT OUTER JOIN process_indicators proc " +
	"ON plop.processindicatorid = proc.id " +
	"WHERE plop.deploymentid = $1 AND plop.closed = false"

//const getPodsStmt = "SELECT id from pods"
const getPodsStmt = "SELECT name from pods where deploymentid = $1"

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
	var podRows pgx.Rows
	var podErr error

	rows, err = s.db.Query(ctx, getByDeploymentStmt, deploymentID)

	if err != nil {
		// Do not be alarmed if the error is simply NoRows
		err = pgutils.ErrNilIfNoRows(err)
		if err != nil {
			log.Warnf("%s: %s", getByDeploymentStmt, err)
		}
		return nil, err
	}
	defer rows.Close()

	podRows, podErr = s.db.Query(ctx, getPodsStmt, deploymentID)
	if podErr != nil {
		// Do not be alarmed if the error is simply NoRows
		podErr = pgutils.ErrNilIfNoRows(podErr)
		if err != nil {
			log.Warnf("%s: %s", getPodsStmt, podErr)
		}
		return nil, podErr
	}
	defer podRows.Close()

	podMap, podErr := s.readPodRows(podRows)
	if err != nil {
		return nil, podErr
	}

	results, err := s.readRows(rows, podMap)
	if err != nil {
		return nil, err
	}

	return results, rows.Err()
}

func (s *fullStoreImpl) readPodRows(rows pgx.Rows) (map[string]bool, error) {
	podMap := make(map[string]bool)
	log.Infof("In readPdRows")
	for rows.Next() {
		var podID string
		log.Infof("Reading pod row")
		if err := rows.Scan(&podID); err != nil {
			return nil, pgutils.ErrNilIfNoRows(err)
		}

		podMap[podID] = true
	}

	return podMap, nil
}

// Manual converting of raw data from SQL query to ProcessListeningOnPort (not
// ProcessListeningOnPortStorage) object enriched with ProcessIndicator info.
func (s *fullStoreImpl) readRows(
	rows pgx.Rows,
	podMap map[string]bool,
) ([]*storage.ProcessListeningOnPort, error) {
	var plops []*storage.ProcessListeningOnPort

	for rows.Next() {
		var id string
		var serialized []byte
		var procSerialized []byte
		var podID string
		var containerName string
		var name string
		var args string
		var execFilePath string

		// We're getting ProcessIndicator directly from the SQL query, PLOP
		// parts have to be extra deserialized.
		if err := rows.Scan(&id, &serialized, &procSerialized); err != nil {
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

		if procMsg.GetPodId() != "" {
			podID = procMsg.GetPodId()
			containerName = procMsg.GetContainerName()
			name = procMsg.GetSignal().GetName()
			args = procMsg.GetSignal().GetArgs()
			execFilePath = procMsg.GetSignal().GetExecFilePath()
		} else {
			podID = msg.GetProcess().GetPodId()
			containerName = msg.GetProcess().GetContainerName()
			name = msg.GetProcess().GetProcessName()
			args = msg.GetProcess().GetProcessArgs()
			execFilePath = msg.GetProcess().GetProcessExecFilePath()
		}

		// If we don't have any of this information from either the process indicator side or
		// processes listening on ports side, the process indicator has been deleted and the
		// port has been closed. Central just hasn't gotten the message yet.
		if podID == "" && containerName == "" && name == "" && args == "" && execFilePath == "" {
			log.Infof("Unable to get podID")
			continue
		}

		_, podExists := podMap[podID]
		log.Infof("podMap= %+v", podMap)
		log.Infof("podID= %+v", podID)
		log.Infof("podMap[podID]= %+v", podMap[podID])
		log.Infof("podExists= %+v", podExists)

		// If the pod of the listening endpoint is not active, don't report the endpoint
		if !podExists {
			continue
		}

		plop := &storage.ProcessListeningOnPort{
			Endpoint: &storage.ProcessListeningOnPort_Endpoint{
				Port:     msg.GetPort(),
				Protocol: msg.GetProtocol(),
			},
			DeploymentId:  msg.GetDeploymentId(),
			PodId:         podID,
			PodUid:        procMsg.GetPodUid(),
			ContainerName: containerName,
			Signal: &storage.ProcessSignal{
				Id:           procMsg.GetSignal().GetId(),
				ContainerId:  procMsg.GetSignal().GetContainerId(),
				Time:         procMsg.GetSignal().GetTime(),
				Name:         name,
				Args:         args,
				ExecFilePath: execFilePath,
				Pid:          procMsg.GetSignal().GetPid(),
				Uid:          procMsg.GetSignal().GetUid(),
				Gid:          procMsg.GetSignal().GetGid(),
				Lineage:      procMsg.GetSignal().GetLineage(),
				Scraped:      procMsg.GetSignal().GetScraped(),
				LineageInfo:  procMsg.GetSignal().GetLineageInfo(),
			},
			ClusterId:          procMsg.GetClusterId(),
			Namespace:          procMsg.GetNamespace(),
			ContainerStartTime: procMsg.GetContainerStartTime(),
			ImageId:            procMsg.GetImageId(),
		}

		plops = append(plops, plop)
	}

	log.Debugf("Read returned %+v plops", len(plops))
	return plops, nil
}
