package pipeline

import (
	"context"
	"fmt"
	"time"

	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/detector"
	detectorMetrics "github.com/stackrox/rox/sensor/common/detector/metrics"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/common/processsignal"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

const (
	// Maximum number of file activities to buffer per process
	maxBufferedActivitiesPerProcess = 100
	// Maximum total number of buffered file activities across all processes
	maxTotalBufferedActivities = 10000
	// How long to keep buffered activities before dropping them
	bufferedActivityTTL = 5 * time.Minute
	// How often to run cleanup of expired buffers
	bufferCleanupInterval = 1 * time.Minute
)

var (
	log = logging.LoggerForModule()
)

type bufferedActivityEntry struct {
	activities []*sensorAPI.FileActivity
	timestamp  time.Time
}

type Pipeline struct {
	detector detector.Detector
	stopper  concurrency.Stopper

	activityChan    chan *sensorAPI.FileActivity
	clusterEntities *clusterentities.Store

	bufferedActivity      map[string]*bufferedActivityEntry
	totalBufferedActivity int
	activityMutex         sync.Mutex

	pubSubDispatcher common.PubSubDispatcher

	wg     sync.WaitGroup
	msgCtx context.Context
}

func NewFileSystemPipeline(detector detector.Detector, clusterEntities *clusterentities.Store, activityChan chan *sensorAPI.FileActivity, pubSubDispatcher common.PubSubDispatcher) *Pipeline {
	msgCtx := context.Background()

	p := &Pipeline{
		detector:              detector,
		activityChan:          activityChan,
		clusterEntities:       clusterEntities,
		pubSubDispatcher:      pubSubDispatcher,
		stopper:               concurrency.NewStopper(),
		msgCtx:                msgCtx,
		bufferedActivity:      make(map[string]*bufferedActivityEntry),
		totalBufferedActivity: 0,
	}

	if features.SensorInternalPubSub.Enabled() && pubSubDispatcher != nil {
		if err := pubSubDispatcher.RegisterConsumerToLane(
			pubsub.FileActivityEnrichedProcessConsumer,
			pubsub.EnrichedProcessIndicatorTopic,
			pubsub.EnrichedProcessIndicatorLane,
			p.processEnrichedIndicator,
		); err != nil {
			log.Errorf("Failed to register consumer for enriched process indicators, falling back to legacy mode: %v", err)
			p.pubSubDispatcher = nil
		} else {
			log.Debug("File system pipeline using pub/sub mode for process enrichment")
			p.wg.Add(1)
			go p.cleanupExpiredBuffers()
		}
	}

	if p.pubSubDispatcher == nil {
		log.Debug("File system pipeline using legacy mode (direct enrichment)")
	}

	p.wg.Add(1)
	go p.run()
	return p
}

func (p *Pipeline) Stop() {
	p.stopper.Client().Stop()
	p.wg.Wait()
}

func (p *Pipeline) translateWithIndicator(fs *sensorAPI.FileActivity, indicator *storage.ProcessIndicator) *storage.FileAccess {
	access := &storage.FileAccess{
		Process:   indicator,
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

func (p *Pipeline) buildIndicator(fs *sensorAPI.FileActivity) *storage.ProcessIndicator {
	process := fs.GetProcess()
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

	return &storage.ProcessIndicator{
		Id:     uuid.NewV4().String(),
		Signal: signal,
	}
}

func cacheKey(containerID, processSignalID string) string {
	return containerID + ":" + processSignalID
}

func (p *Pipeline) bufferActivity(fs *sensorAPI.FileActivity) {
	process := fs.GetProcess()
	if process == nil {
		return
	}

	key := cacheKey(process.GetContainerId(), process.GetId())
	p.activityMutex.Lock()
	defer p.activityMutex.Unlock()

	if p.totalBufferedActivity >= maxTotalBufferedActivities {
		log.Warnf("File activity buffer is full (%d activities), dropping file activity for process %s",
			maxTotalBufferedActivities, process.GetId())
		metrics.IncrementFileActivityBufferDrops()
		return
	}

	entry, exists := p.bufferedActivity[key]
	if !exists {
		entry = &bufferedActivityEntry{
			activities: make([]*sensorAPI.FileActivity, 0, 10),
			timestamp:  time.Now(),
		}
		p.bufferedActivity[key] = entry
	}

	if len(entry.activities) >= maxBufferedActivitiesPerProcess {
		log.Warnf("Too many buffered activities for process %s (limit: %d), dropping file activity",
			process.GetId(), maxBufferedActivitiesPerProcess)
		metrics.IncrementFileActivityBufferDrops()
		return
	}

	entry.activities = append(entry.activities, fs)
	p.totalBufferedActivity++
	metrics.SetFileActivityBufferSize(p.totalBufferedActivity)
}

func (p *Pipeline) popBufferedActivity(key string) []*sensorAPI.FileActivity {
	p.activityMutex.Lock()
	defer p.activityMutex.Unlock()

	entry := p.bufferedActivity[key]
	if entry == nil {
		return nil
	}

	activities := entry.activities
	p.totalBufferedActivity -= len(activities)
	delete(p.bufferedActivity, key)
	metrics.SetFileActivityBufferSize(p.totalBufferedActivity)
	return activities
}

func (p *Pipeline) processEnrichedIndicator(event pubsub.Event) error {
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
	buffered := p.popBufferedActivity(key)
	for _, fs := range buffered {
		access := p.translateWithIndicator(fs, indicator)
		if access != nil {
			p.detector.ProcessFileAccess(enrichedEvent.Context, access)
		}
	}

	return nil
}

func (p *Pipeline) processFileActivity(fs *sensorAPI.FileActivity) {
	process := fs.GetProcess()
	if process == nil {
		return
	}

	indicator := p.buildIndicator(fs)

	// Host processes (no container ID) bypass enrichment entirely.
	if process.GetContainerId() == "" {
		if access := p.translateWithIndicator(fs, indicator); access != nil {
			p.detector.ProcessFileAccess(p.msgCtx, access)
		}
		return
	}

	if features.SensorInternalPubSub.Enabled() && p.pubSubDispatcher != nil {
		// Buffer the file activity BEFORE publishing the unenriched event
		// to avoid a TOCTOU race where the enriched callback fires before
		// the activity is buffered.
		p.bufferActivity(fs)
		event := processsignal.NewUnenrichedProcessIndicatorEvent(p.msgCtx, indicator)
		if err := p.pubSubDispatcher.Publish(event); err != nil {
			log.Errorf("Failed to publish unenriched process indicator for file activity: %v, falling back to legacy enrichment", err)
			// Remove the buffered activity and fall through to legacy path.
			key := cacheKey(process.GetContainerId(), process.GetId())
			_ = p.popBufferedActivity(key)
		} else {
			return
		}
	}

	// Legacy path: enrich directly from cluster entities store.
	metadata, ok, _ := p.clusterEntities.LookupByContainerID(process.GetContainerId())
	if !ok {
		log.Warnf("Container ID: %s not found for file activity", process.GetContainerId())
	} else {
		processsignal.PopulateIndicatorFromContainer(indicator, metadata)
	}

	if access := p.translateWithIndicator(fs, indicator); access != nil {
		p.detector.ProcessFileAccess(p.msgCtx, access)
	}
}

func (p *Pipeline) cleanupExpiredBuffers() {
	defer p.wg.Done()
	ticker := time.NewTicker(bufferCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopper.Flow().StopRequested():
			return
		case <-ticker.C:
			p.pruneExpiredBuffers()
		}
	}
}

func (p *Pipeline) pruneExpiredBuffers() {
	p.activityMutex.Lock()
	defer p.activityMutex.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, entry := range p.bufferedActivity {
		if now.Sub(entry.timestamp) > bufferedActivityTTL {
			expiredKeys = append(expiredKeys, key)
			p.totalBufferedActivity -= len(entry.activities)
			metrics.IncrementFileActivityBufferDropsBy(len(entry.activities))
		}
	}

	for _, key := range expiredKeys {
		delete(p.bufferedActivity, key)
	}

	if p.totalBufferedActivity < 0 {
		log.Warnf("totalBufferedActivity went negative (%d), resetting to 0 (possible accounting bug)", p.totalBufferedActivity)
		p.totalBufferedActivity = 0
	}

	if len(expiredKeys) > 0 {
		log.Debugf("Pruned %d expired file activity buffers (TTL: %v)", len(expiredKeys), bufferedActivityTTL)
		metrics.SetFileActivityBufferSize(p.totalBufferedActivity)
	}
}

func (p *Pipeline) run() {
	defer p.wg.Done()
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
			detectorMetrics.FileAccessEventsReceived.Inc()
			p.processFileActivity(fs)
		}
	}
}
