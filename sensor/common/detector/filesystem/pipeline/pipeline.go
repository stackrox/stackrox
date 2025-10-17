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

	activityChan    chan *storage.FileActivity
	clusterEntities *clusterentities.Store

	msgCtx context.Context
}

func NewFileSystemPipeline(detector detector.Detector, clusterEntities *clusterentities.Store) *Pipeline {
	msgCtx := context.Background()

	p := &Pipeline{
		detector:        detector,
		activityChan:    make(chan *storage.FileActivity),
		clusterEntities: clusterEntities,
		stopper:         concurrency.NewStopper(),
		msgCtx:          msgCtx,
	}

	go p.run()
	return p
}

func (p *Pipeline) Process(fs *sensorAPI.FileActivity) {
	psignal := fs.GetProcess()

	pi := &storage.ProcessIndicator{
		Id: uuid.NewV4().String(),
		Signal: &storage.ProcessSignal{
			Id:           psignal.GetId(),
			Uid:          psignal.GetUid(),
			Gid:          psignal.GetGid(),
			Time:         psignal.GetCreationTime(),
			Name:         psignal.GetName(),
			Args:         psignal.GetArgs(),
			ExecFilePath: psignal.GetExecFilePath(),
			Pid:          psignal.GetPid(),
			Scraped:      psignal.GetScraped(),
			ContainerId:  psignal.GetContainerId(),
		},
	}

	if psignal.GetContainerId() != "" {
		// TODO(ROX-30798): Enrich file system events with deployment details
		metadata, ok, _ := p.clusterEntities.LookupByContainerID(psignal.GetContainerId())
		if !ok {
			// unexpected - process should exist before file activity is
			// reported
			log.Debug("Container ID:", psignal.GetContainerId(), "not found for file activity")
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
	// TODO: populate node info otherwise

	activity := &storage.FileActivity{
		Process: pi,
	}

	switch fs.GetFile().(type) {
	case *sensorAPI.FileActivity_Creation:
		activity.File = &storage.FileActivity_File{
			Path:     fs.GetCreation().GetActivity().GetPath(),
			HostPath: fs.GetCreation().GetActivity().GetHostPath(),
		}
		activity.Operation = storage.FileActivity_CREATE
	case *sensorAPI.FileActivity_Unlink:
		activity.File = &storage.FileActivity_File{
			Path:     fs.GetUnlink().GetActivity().GetPath(),
			HostPath: fs.GetUnlink().GetActivity().GetHostPath(),
		}
		activity.Operation = storage.FileActivity_UNLINK
	case *sensorAPI.FileActivity_Rename:
		activity.File = &storage.FileActivity_File{
			// Not sure if GetNew or GetOld should be used here.
			Path:     fs.GetRename().GetNew().GetPath(),
			HostPath: fs.GetRename().GetNew().GetHostPath(),
		}
		activity.Operation = storage.FileActivity_RENAME
	case *sensorAPI.FileActivity_Permission:
		activity.File = &storage.FileActivity_File{
			Path:     fs.GetPermission().GetActivity().GetPath(),
			HostPath: fs.GetPermission().GetActivity().GetHostPath(),
		}
		activity.Operation = storage.FileActivity_PERMISSION_CHANGE
	case *sensorAPI.FileActivity_Ownership:
		activity.File = &storage.FileActivity_File{
			Path:     fs.GetOwnership().GetActivity().GetPath(),
			HostPath: fs.GetOwnership().GetActivity().GetHostPath(),
		}
		activity.Operation = storage.FileActivity_OWNERSHIP_CHANGE
	case *sensorAPI.FileActivity_Write:
		activity.File = &storage.FileActivity_File{
			Path:     fs.GetWrite().GetActivity().GetPath(),
			HostPath: fs.GetWrite().GetActivity().GetHostPath(),
		}
		activity.Operation = storage.FileActivity_WRITE
	case *sensorAPI.FileActivity_Open:
		activity.File = &storage.FileActivity_File{
			Path:     fs.GetOpen().GetActivity().GetPath(),
			HostPath: fs.GetOpen().GetActivity().GetHostPath(),
		}
		activity.Operation = storage.FileActivity_OPEN
	default:
		log.Warn("Not implemented file activity type")
		return
	}

	p.activityChan <- activity
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
