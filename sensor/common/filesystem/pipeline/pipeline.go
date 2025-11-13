package pipeline

import (
	"context"

	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/store"
)

var (
	log = logging.LoggerForModule()
)

type Pipeline struct {
	detector detector.Detector
	stopper  concurrency.Stopper

	activityChan    chan *sensorAPI.FileActivity
	clusterEntities *clusterentities.Store
	nodeStore       store.NodeStore

	msgCtx context.Context
}

func NewFileSystemPipeline(detector detector.Detector, clusterEntities *clusterentities.Store, nodeStore store.NodeStore, activityChan chan *sensorAPI.FileActivity) *Pipeline {
	msgCtx := context.Background()

	p := &Pipeline{
		detector:        detector,
		activityChan:    activityChan,
		clusterEntities: clusterEntities,
		nodeStore:       nodeStore,
		stopper:         concurrency.NewStopper(),
		msgCtx:          msgCtx,
	}

	go p.run()
	return p
}

func (p *Pipeline) translate(fs *sensorAPI.FileActivity) *storage.FileAccess {

	access := &storage.FileAccess{
		Process:   p.getIndicator(fs.GetProcess()),
		Hostname:  fs.GetHostname(),
		Timestamp: fs.GetTimestamp(),
	}

	if fs.GetProcess().GetContainerId() == "" {
		access.Hostname = fs.GetHostname()
	}

	switch fs.GetFile().(type) {
	case *sensorAPI.FileActivity_Creation:
		access.File = &storage.FileAccess_File{
			MountedPath: fs.GetCreation().GetActivity().GetPath(),
			NodePath:    fs.GetCreation().GetActivity().GetHostPath(),
		}
		access.Operation = storage.FileAccess_CREATE
	case *sensorAPI.FileActivity_Unlink:
		access.File = &storage.FileAccess_File{
			MountedPath: fs.GetUnlink().GetActivity().GetPath(),
			NodePath:    fs.GetUnlink().GetActivity().GetHostPath(),
		}
		access.Operation = storage.FileAccess_UNLINK
	case *sensorAPI.FileActivity_Rename:
		access.File = &storage.FileAccess_File{
			MountedPath: fs.GetRename().GetOld().GetPath(),
			NodePath:    fs.GetRename().GetOld().GetHostPath(),
		}
		access.Moved = &storage.FileAccess_File{
			MountedPath: fs.GetRename().GetNew().GetPath(),
			NodePath:    fs.GetRename().GetNew().GetHostPath(),
		}
		access.Operation = storage.FileAccess_RENAME
	case *sensorAPI.FileActivity_Permission:
		access.File = &storage.FileAccess_File{
			MountedPath: fs.GetPermission().GetActivity().GetPath(),
			NodePath:    fs.GetPermission().GetActivity().GetHostPath(),
			Meta: &storage.FileAccess_FileMetadata{
				Mode: fs.GetPermission().GetMode(),
			},
		}
		access.Operation = storage.FileAccess_PERMISSION_CHANGE
	case *sensorAPI.FileActivity_Ownership:
		access.File = &storage.FileAccess_File{
			MountedPath: fs.GetOwnership().GetActivity().GetPath(),
			NodePath:    fs.GetOwnership().GetActivity().GetHostPath(),
			Meta: &storage.FileAccess_FileMetadata{
				Uid:      fs.GetOwnership().GetUid(),
				Gid:      fs.GetOwnership().GetGid(),
				Username: fs.GetOwnership().GetUsername(),
				Group:    fs.GetOwnership().GetGroup(),
			},
		}
		access.Operation = storage.FileAccess_OWNERSHIP_CHANGE
	case *sensorAPI.FileActivity_Write:
		access.File = &storage.FileAccess_File{
			MountedPath: fs.GetWrite().GetActivity().GetPath(),
			NodePath:    fs.GetWrite().GetActivity().GetHostPath(),
		}
		access.Operation = storage.FileAccess_WRITE
	case *sensorAPI.FileActivity_Open:
		access.File = &storage.FileAccess_File{
			MountedPath: fs.GetOpen().GetActivity().GetPath(),
			NodePath:    fs.GetOpen().GetActivity().GetHostPath(),
		}
		access.Operation = storage.FileAccess_OPEN
	default:
		log.Warn("Not implemented file activity type")
		return nil
	}

	return access
}

func (p *Pipeline) getIndicator(process *sensorAPI.ProcessSignal) *storage.ProcessIndicator {
	pi := &storage.ProcessIndicator{
		Id: uuid.NewV4().String(),
		Signal: &storage.ProcessSignal{
			Id:           process.GetId(),
			Uid:          process.GetUid(),
			Gid:          process.GetGid(),
			Time:         process.GetCreationTime(),
			Name:         process.GetName(),
			Args:         process.GetArgs(),
			ExecFilePath: process.GetExecFilePath(),
			Pid:          process.GetPid(),
			Scraped:      process.GetScraped(),
			ContainerId:  process.GetContainerId(),
		},
	}

	if process.GetContainerId() == "" {
		// Process is running on the host (not in a container)
		return pi
	}

	// TODO(ROX-30798): Enrich file system events with deployment details
	metadata, ok, _ := p.clusterEntities.LookupByContainerID(process.GetContainerId())
	if !ok {
		// unexpected - process should exist before file activity is
		// reported
		log.Warnf("Container ID: %s not found for file activity", process.GetContainerId())
	} else {
		pi.DeploymentId = metadata.DeploymentID
		pi.ContainerName = metadata.ContainerName
		pi.PodId = metadata.PodID
		pi.PodUid = metadata.PodUID
		pi.Namespace = metadata.Namespace
		pi.ContainerStartTime = protocompat.ConvertTimeToTimestampOrNil(metadata.StartTime)
		pi.ImageId = metadata.ImageID
	}

	return pi
}

func (p *Pipeline) Stop() {
	p.stopper.Client().Stop()
	<-p.stopper.Client().Stopped().Done()
}

func (p *Pipeline) run() {
	defer p.stopper.Flow().ReportStopped()
	for {
		select {
		case <-p.stopper.Flow().StopRequested():
			return
		case fs, ok := <-p.activityChan:
			if !ok {
				// Channel closed, no more messages
				return
			}
			event := p.translate(fs)

			// TODO: Send event to detector
			if event.GetProcess().GetContainerName() != "" {
				// Do deployment based detection but for now just log
				log.Infof("Container FS event = %+v", event)
			} else {
				node := p.nodeStore.GetNode(event.GetHostname())
				if node == nil {
					log.Warnf("Node %s not found in node store", event.GetHostname())
					continue
				}

				// Do node based detection but for now just log
				log.Infof("Node FS event on %s = %+v", node.GetName(), event)
			}
		}
	}
}
