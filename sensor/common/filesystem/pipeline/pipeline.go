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
)

var (
	log = logging.LoggerForModule()
)

type Pipeline struct {
	detector detector.Detector
	stopper  concurrency.Stopper

	activityChan    chan *storage.FileAccess
	clusterEntities *clusterentities.Store

	msgCtx context.Context
}

func NewFileSystemPipeline(detector detector.Detector, clusterEntities *clusterentities.Store) *Pipeline {
	msgCtx := context.Background()

	p := &Pipeline{
		detector:        detector,
		activityChan:    make(chan *storage.FileAccess),
		clusterEntities: clusterEntities,
		stopper:         concurrency.NewStopper(),
		msgCtx:          msgCtx,
	}

	go p.run()
	return p
}

func (p *Pipeline) translate(fs *sensorAPI.FileActivity) *storage.FileAccess {

	activity := &storage.FileAccess{
		Process: p.getIndicator(fs.GetProcess()),
	}

	switch fs.GetFile().(type) {
	case *sensorAPI.FileActivity_Creation:
		activity.File = &storage.FileAccess_File{
			Path:     fs.GetCreation().GetActivity().GetPath(),
			HostPath: fs.GetCreation().GetActivity().GetHostPath(),
		}
		activity.Operation = storage.FileAccess_CREATE
	case *sensorAPI.FileActivity_Unlink:
		activity.File = &storage.FileAccess_File{
			Path:     fs.GetUnlink().GetActivity().GetPath(),
			HostPath: fs.GetUnlink().GetActivity().GetHostPath(),
		}
		activity.Operation = storage.FileAccess_UNLINK
	case *sensorAPI.FileActivity_Rename:
		activity.File = &storage.FileAccess_File{
			Path:     fs.GetRename().GetOld().GetPath(),
			HostPath: fs.GetRename().GetOld().GetHostPath(),
		}
		activity.Moved = &storage.FileAccess_File{
			Path:     fs.GetRename().GetNew().GetPath(),
			HostPath: fs.GetRename().GetNew().GetHostPath(),
		}
		activity.Operation = storage.FileAccess_RENAME
	case *sensorAPI.FileActivity_Permission:
		activity.File = &storage.FileAccess_File{
			Path:     fs.GetPermission().GetActivity().GetPath(),
			HostPath: fs.GetPermission().GetActivity().GetHostPath(),
			Meta: &storage.FileAccess_FileMetadata{
				Mode: fs.GetPermission().GetMode(),
			},
		}
		activity.Operation = storage.FileAccess_PERMISSION_CHANGE
	case *sensorAPI.FileActivity_Ownership:
		activity.File = &storage.FileAccess_File{
			Path:     fs.GetOwnership().GetActivity().GetPath(),
			HostPath: fs.GetOwnership().GetActivity().GetHostPath(),
			Meta: &storage.FileAccess_FileMetadata{
				Uid:      fs.GetOwnership().GetUid(),
				Gid:      fs.GetOwnership().GetGid(),
				Username: fs.GetOwnership().GetUsername(),
				Group:    fs.GetOwnership().GetGroup(),
			},
		}
		activity.Operation = storage.FileAccess_OWNERSHIP_CHANGE
	case *sensorAPI.FileActivity_Write:
		activity.File = &storage.FileAccess_File{
			Path:     fs.GetWrite().GetActivity().GetPath(),
			HostPath: fs.GetWrite().GetActivity().GetHostPath(),
		}
		activity.Operation = storage.FileAccess_WRITE
	case *sensorAPI.FileActivity_Open:
		activity.File = &storage.FileAccess_File{
			Path:     fs.GetOpen().GetActivity().GetPath(),
			HostPath: fs.GetOpen().GetActivity().GetHostPath(),
		}
		activity.Operation = storage.FileAccess_OPEN
	default:
		log.Warn("Not implemented file activity type")
		return nil
	}

	return activity
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

	if process.GetContainerId() != "" {
		// TODO(ROX-30798): Enrich file system events with deployment details
		metadata, ok, _ := p.clusterEntities.LookupByContainerID(process.GetContainerId())
		if !ok {
			// unexpected - process should exist before file activity is
			// reported
			log.Debug("Container ID:", process.GetContainerId(), "not found for file activity")
		} else {
			pi.DeploymentId = metadata.DeploymentID
			pi.ContainerName = metadata.ContainerName
			pi.PodId = metadata.PodID
			pi.PodUid = metadata.PodUID
			pi.Namespace = metadata.Namespace
			pi.ContainerStartTime = protocompat.ConvertTimeToTimestampOrNil(metadata.StartTime)
			pi.ImageId = metadata.ImageID
		}
	}
	// TODO(ROX-31434): populate node info otherwise

	return pi
}

func (p *Pipeline) Process(fs *sensorAPI.FileActivity) {

	activity := p.translate(fs)

	if activity != nil {
		p.activityChan <- activity
	}
}

func (p *Pipeline) run() {
	for {
		select {
		case <-p.stopper.Flow().StopRequested():
			return
		case event := <-p.activityChan:
			// p.detector.ProcessFilesystem(p.msgCtx, event)
			log.Infof("event= %+v", event)
		}
	}
}
