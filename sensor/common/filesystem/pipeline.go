package filesystem

import (
	"context"

	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/channelmultiplexer"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/trace"
)

type Pipeline struct {
	detector detector.Detector
	stopper  concurrency.Stopper

	activity chan *storage.FileActivity
	cm       *channelmultiplexer.ChannelMultiplexer[*storage.FileActivity]

	msgCtx context.Context
}

func NewFileSystemPipeline(detector detector.Detector) *Pipeline {
	msgCtx, _ := context.WithCancelCause(trace.Background())

	p := &Pipeline{
		detector: detector,
		activity: make(chan *storage.FileActivity),
		stopper:  concurrency.NewStopper(),
		msgCtx:   msgCtx,
	}

	go p.run()
	return p
}

func (p *Pipeline) Process(fs *sensorAPI.FileActivity) {
	psignal := fs.GetProcess()

	activity := &storage.FileActivity{
		Process: &storage.ProcessIndicator{
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
			},
		},
	}

	if fs.GetOpen() != nil {
		activity.File = &storage.FileActivity_File{
			Path:            fs.GetOpen().GetActivity().GetPath(),
			HostPath:        fs.GetOpen().GetActivity().GetHostPath(),
			IsExternalMount: fs.GetOpen().GetActivity().GetIsExternalMount(),
		}
		activity.Type = storage.FileActivity_OPEN
	} else if fs.GetWrite() != nil {
		activity.File = &storage.FileActivity_File{
			Path:            fs.GetWrite().GetActivity().GetPath(),
			HostPath:        fs.GetWrite().GetActivity().GetHostPath(),
			IsExternalMount: fs.GetWrite().GetActivity().GetIsExternalMount(),
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
