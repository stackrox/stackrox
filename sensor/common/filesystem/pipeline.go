package filesystem

import (
	"context"

	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/detector"
)

type Pipeline struct {
	detector detector.Detector
	stopper  concurrency.Stopper

	activity        chan *storage.FileActivity
	clusterEntities *clusterentities.Store

	msgCtx context.Context
}

func NewFileSystemPipeline(detector detector.Detector, clusterEntities *clusterentities.Store) *Pipeline {
	msgCtx := context.Background()

	p := &Pipeline{
		detector:        detector,
		activity:        make(chan *storage.FileActivity),
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
			Id:           psignal.Id,
			Uid:          psignal.Uid,
			Gid:          psignal.Gid,
			Time:         psignal.CreationTime,
			Name:         psignal.Name,
			Args:         psignal.Args,
			ExecFilePath: psignal.ExecFilePath,
			Pid:          psignal.Pid,
			Scraped:      psignal.Scraped,
			ContainerId:  psignal.ContainerId,
		},
	}

	if psignal.GetContainerId() != "" {
		metadata, ok, _ := p.clusterEntities.LookupByContainerID(psignal.GetContainerId())
		if !ok {
			// unexpected - process should exist before file activity is
			// reported
			log.Debug("Container ID:", psignal.GetContainerId(), "not found for file activity")
		}

		pi.DeploymentId = metadata.DeploymentID
		pi.ContainerName = metadata.ContainerName
		pi.PodId = metadata.PodID
		pi.PodUid = metadata.PodUID
		pi.Namespace = metadata.Namespace
		pi.ContainerStartTime = protocompat.ConvertTimeToTimestampOrNil(metadata.StartTime)
		pi.ImageId = metadata.ImageID
	} else {
		// TODO: populate node info
	}

	activity := &storage.FileActivity{
		Process: pi,
	}

	if fs.GetOpen() != nil {
		activity.File = &storage.FileActivity_File{
			Path:     fs.GetOpen().GetActivity().GetPath(),
			HostPath: fs.GetOpen().GetActivity().GetHostPath(),
		}
		activity.Type = storage.FileActivity_OPEN
	} else if fs.GetWrite() != nil {
		activity.File = &storage.FileActivity_File{
			Path:     fs.GetWrite().GetActivity().GetPath(),
			HostPath: fs.GetWrite().GetActivity().GetHostPath(),
		}
		activity.Type = storage.FileActivity_WRITE
	} else {
		log.Warn("Not implemented file activity type")
		return
	}

	log.Info("sending")
	p.activity <- activity
	log.Info("sent")
}

func (p *Pipeline) run() {
	for {
		select {
		case <-p.stopper.Flow().StopRequested():
			return
		case event := <-p.activity:
			p.detector.ProcessFilesystem(p.msgCtx, event)
		}
	}
}
