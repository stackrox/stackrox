package pipeline

import (
	"context"
	"fmt"
	"time"

	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/processsignal"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

var (
	log = logging.LoggerForModule()
)

type Pipeline struct {
	detector detector.Detector
	stopper  concurrency.Stopper

	activityChan    chan *sensorAPI.FileActivity
	clusterEntities *clusterentities.Store

	// processCache avoids duplicate enrichment by reusing enriched indicators from the process pipeline.
	// Key format: "containerID:processSignalID"
	processCache expiringcache.Cache[string, *storage.ProcessIndicator]

	pubSubDispatcher common.PubSubDispatcher

	msgCtx context.Context
}

func NewFileSystemPipeline(detector detector.Detector, clusterEntities *clusterentities.Store, activityChan chan *sensorAPI.FileActivity, pubSubDispatcher common.PubSubDispatcher) *Pipeline {
	msgCtx := context.Background()

	p := &Pipeline{
		detector:         detector,
		activityChan:     activityChan,
		clusterEntities:  clusterEntities,
		pubSubDispatcher: pubSubDispatcher,
		stopper:          concurrency.NewStopper(),
		msgCtx:           msgCtx,
	}

	if features.SensorInternalPubSub.Enabled() && pubSubDispatcher != nil {
		log.Info("File system pipeline using pub/sub mode for process enrichment")
		p.processCache = expiringcache.NewExpiringCache[string, *storage.ProcessIndicator](5 * time.Minute)

		if err := pubSubDispatcher.RegisterConsumerToLane(pubsub.EnrichedProcessIndicatorTopic, pubsub.EnrichedProcessIndicatorLane, p.cacheEnrichedIndicator); err != nil {
			log.Errorf("Failed to register consumer for enriched process indicators in file system pipeline: %v", err)
		}
	} else {
		log.Info("File system pipeline using legacy mode (direct enrichment)")
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

	switch fs.GetFile().(type) {
	case *sensorAPI.FileActivity_Creation:
		access.File = &storage.FileAccess_File{
			EffectivePath: fs.GetCreation().GetActivity().GetPath(),
			ActualPath:    fs.GetCreation().GetActivity().GetHostPath(),
		}
		access.Operation = storage.FileAccess_CREATE
	case *sensorAPI.FileActivity_Unlink:
		access.File = &storage.FileAccess_File{
			EffectivePath: fs.GetUnlink().GetActivity().GetPath(),
			ActualPath:    fs.GetUnlink().GetActivity().GetHostPath(),
		}
		access.Operation = storage.FileAccess_UNLINK
	case *sensorAPI.FileActivity_Rename:
		access.File = &storage.FileAccess_File{
			EffectivePath: fs.GetRename().GetOld().GetPath(),
			ActualPath:    fs.GetRename().GetOld().GetHostPath(),
		}
		access.Moved = &storage.FileAccess_File{
			EffectivePath: fs.GetRename().GetNew().GetPath(),
			ActualPath:    fs.GetRename().GetNew().GetHostPath(),
		}
		access.Operation = storage.FileAccess_RENAME
	case *sensorAPI.FileActivity_Permission:
		access.File = &storage.FileAccess_File{
			EffectivePath: fs.GetPermission().GetActivity().GetPath(),
			ActualPath:    fs.GetPermission().GetActivity().GetHostPath(),
			Meta: &storage.FileAccess_FileMetadata{
				Mode: fs.GetPermission().GetMode(),
			},
		}
		access.Operation = storage.FileAccess_PERMISSION_CHANGE
	case *sensorAPI.FileActivity_Ownership:
		access.File = &storage.FileAccess_File{
			EffectivePath: fs.GetOwnership().GetActivity().GetPath(),
			ActualPath:    fs.GetOwnership().GetActivity().GetHostPath(),
			Meta: &storage.FileAccess_FileMetadata{
				Uid:      fs.GetOwnership().GetUid(),
				Gid:      fs.GetOwnership().GetGid(),
				Username: fs.GetOwnership().GetUsername(),
				Group:    fs.GetOwnership().GetGroup(),
			},
		}
		access.Operation = storage.FileAccess_OWNERSHIP_CHANGE
	case *sensorAPI.FileActivity_Open:
		access.File = &storage.FileAccess_File{
			EffectivePath: fs.GetOpen().GetActivity().GetPath(),
			ActualPath:    fs.GetOpen().GetActivity().GetHostPath(),
		}
		access.Operation = storage.FileAccess_OPEN
	default:
		log.Warn("Not implemented file activity type")
		return nil
	}

	return access
}

func cacheKey(containerID, processSignalID string) string {
	return fmt.Sprintf("%s:%s", containerID, processSignalID)
}

func (p *Pipeline) cacheEnrichedIndicator(event pubsub.Event) error {
	enrichedEvent, ok := event.(*processsignal.EnrichedProcessIndicatorEvent)
	if !ok {
		log.Errorf("File system pipeline received unexpected event type: %T", event)
		return fmt.Errorf("unexpected event type: %T", event)
	}

	indicator := enrichedEvent.Indicator
	if indicator == nil || indicator.GetSignal() == nil {
		return nil
	}

	key := cacheKey(indicator.GetSignal().GetContainerId(), indicator.GetSignal().GetId())
	p.processCache.Add(key, indicator)

	return nil
}

func (p *Pipeline) getIndicator(process *sensorAPI.ProcessSignal) *storage.ProcessIndicator {
	if p.processCache != nil && process.GetContainerId() != "" {
		key := cacheKey(process.GetContainerId(), process.GetId())
		if cachedIndicator, ok := p.processCache.Get(key); ok {
			log.Debugf("Using cached enriched indicator for process %s in container %s", process.GetId(), process.GetContainerId())
			return cachedIndicator
		}
	}

	signal := &storage.ProcessSignal{
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
		LineageInfo:  make([]*storage.ProcessSignal_LineageInfo, 0, len(process.GetLineageInfo())),
	}

	for _, lineage := range process.GetLineageInfo() {
		signal.LineageInfo = append(signal.LineageInfo,
			&storage.ProcessSignal_LineageInfo{
				ParentUid:          lineage.GetParentUid(),
				ParentExecFilePath: lineage.GetParentExecFilePath(),
			},
		)
	}

	pi := &storage.ProcessIndicator{
		Id:     uuid.NewV4().String(),
		Signal: signal,
	}

	if process.GetContainerId() == "" {
		// Process is running on the host (not in a container)
		return pi
	}

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
			p.detector.ProcessFileAccess(p.msgCtx, event)
		}
	}
}
