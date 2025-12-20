package reposcanhandler

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/baseimage/reposcan"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

var log = logging.LoggerForModule()

// Handler processes repository scan requests from Central and streams responses.
type Handler struct {
	responsesC chan *message.ExpiringMessage
	stopSig    concurrency.ErrorSignal
	scanner    reposcan.Scanner

	// cancels stores cancel functions for in-progress requests.
	cancels      map[string]context.CancelFunc
	cancelsMutex sync.Mutex
}

// NewHandler creates a new repository scan handler.
func NewHandler(scanner reposcan.Scanner) *Handler {
	return &Handler{
		// Buffer the response channel to avoid blocking senders when the
		// downstream gRPC stream is temporarily slow.
		responsesC: make(chan *message.ExpiringMessage, 100),
		stopSig:    concurrency.NewErrorSignal(),
		scanner:    scanner,
		cancels:    make(map[string]context.CancelFunc),
	}
}

// Name returns the name of this component.
func (h *Handler) Name() string {
	return "reposcanhandler.Handler"
}

// Accepts returns true if the message is a RepoScanRequest or RepoScanCancellation.
func (h *Handler) Accepts(msg *central.MsgToSensor) bool {
	return msg.GetRepoScanRequest() != nil || msg.GetRepoScanCancellation() != nil
}

// ProcessMessage handles incoming repository scan requests and cancellations.
func (h *Handler) ProcessMessage(_ context.Context, msg *central.MsgToSensor) error {
	switch {
	case msg.GetRepoScanRequest() != nil:
		return h.processRequest(msg.GetRepoScanRequest())
	case msg.GetRepoScanCancellation() != nil:
		return h.processCancellation(msg.GetRepoScanCancellation())
	}
	return nil
}

func (h *Handler) processRequest(req *central.RepoScanRequest) error {
	log.Infof("OnScanResponse RepoScanRequest: requestID=%s repository=%s tagPattern=%s",
		req.GetRequestId(), req.GetRepository(), req.GetTagPattern())

	go h.dispatchRequest(req)
	return nil
}

func (h *Handler) processCancellation(req *central.RepoScanCancellation) error {
	requestID := req.GetRequestId()
	log.Infof("OnScanResponse RepoScanCancellation: requestID=%s", requestID)
	cancel := concurrency.WithLock1(&h.cancelsMutex, func() context.CancelFunc {
		cancel, ok := h.cancels[requestID]
		if !ok {
			return nil
		}
		log.Infof("Cancelling in-progress request %s", requestID)
		delete(h.cancels, requestID)
		return cancel
	})
	if cancel != nil {
		cancel()
	}
	return nil
}

func (h *Handler) addCancel(requestID string, cancel context.CancelFunc) {
	h.cancelsMutex.Lock()
	defer h.cancelsMutex.Unlock()
	h.cancels[requestID] = cancel
}

// dispatchRequest handles a single repository scan request with streaming.
func (h *Handler) dispatchRequest(req *central.RepoScanRequest) {
	requestID := req.GetRequestId()

	// Create cancellable context for this request.
	ctx, cancel := context.WithCancel(concurrency.AsContext(&h.stopSig))
	concurrency.WithLock(&h.cancelsMutex, func() {
		h.cancels[requestID] = cancel
	})

	repo, scanReq := scanRequest(req)

	// Send start message to signal processing began.
	h.sendStart(ctx, requestID)

	// Scan repository and stream events.
	var successCount, failedCount int32
	for event, err := range h.scanner.ScanRepository(ctx, repo, scanReq) {
		// Handle fatal errors from the iterator.
		if err != nil {
			log.Errorf("Request %s: scan failed: %v", requestID, err)
			h.sendEnd(ctx, requestID, &central.RepoScanResponse_End{
				Success:         false,
				Error:           err.Error(),
				SuccessfulCount: successCount,
				FailedCount:     failedCount,
			})
			return
		}

		// Convert tag event to proto response.
		switch event.Type {
		case reposcan.TagEventMetadata:
			h.sendUpdate(ctx, requestID, &central.RepoScanResponse_Update{
				Tag:     event.Tag,
				Outcome: &central.RepoScanResponse_Update_Metadata{Metadata: tagMetadata(event)},
			})
			successCount++

		case reposcan.TagEventError:
			h.sendUpdate(ctx, requestID, &central.RepoScanResponse_Update{
				Tag:     event.Tag,
				Outcome: &central.RepoScanResponse_Update_Error{Error: event.Error.Error()},
			})
			failedCount++

		case reposcan.TagEventDeleted:
			h.sendUpdate(ctx, requestID, &central.RepoScanResponse_Update{
				Tag:     event.Tag,
				Outcome: &central.RepoScanResponse_Update_Deleted{Deleted: true},
			})
			log.Infof("Request %s: tag %s was deleted", requestID, event.Tag)
		}
	}

	// Send end message.
	log.Infof("Request %s: completed with %d successful, %d failed",
		requestID, successCount, failedCount)
	h.sendEnd(ctx, requestID, &central.RepoScanResponse_End{
		Success:         true,
		SuccessfulCount: successCount,
		FailedCount:     failedCount,
	})
}

func (h *Handler) sendStart(ctx context.Context, requestID string) {
	h.sendResponse(ctx, &central.RepoScanResponse{
		RequestId: requestID,
		Payload: &central.RepoScanResponse_Start_{
			Start: &central.RepoScanResponse_Start{},
		},
	})
}

func (h *Handler) sendUpdate(ctx context.Context, requestID string, update *central.RepoScanResponse_Update) {
	h.sendResponse(ctx, &central.RepoScanResponse{
		RequestId: requestID,
		Payload:   &central.RepoScanResponse_Update_{Update: update},
	})
}

func (h *Handler) sendEnd(ctx context.Context, requestID string, end *central.RepoScanResponse_End) {
	h.sendResponse(ctx, &central.RepoScanResponse{
		RequestId: requestID,
		Payload:   &central.RepoScanResponse_End_{End: end},
	})
}

func (h *Handler) sendResponse(ctx context.Context, resp *central.RepoScanResponse) {
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_RepoScanResponse{
			RepoScanResponse: resp,
		},
	}
	select {
	case h.responsesC <- message.New(msg):
		return
	case <-ctx.Done():
		log.Errorf("Sending response %s: context cancelled: %v",
			resp.GetRequestId(), ctx.Err().Error())
	}
}

// Capabilities returns the capabilities of this component.
func (h *Handler) Capabilities() []centralsensor.SensorCapability {
	return nil
}

// ResponsesC returns the channel for sending messages back to Central.
func (h *Handler) ResponsesC() <-chan *message.ExpiringMessage {
	return h.responsesC
}

// Start starts the handler.
func (h *Handler) Start() error {
	return nil
}

// Stop stops the handler and cancels all pending requests.
func (h *Handler) Stop() {
	h.cancelsMutex.Lock()
	defer h.cancelsMutex.Unlock()

	for reqID, cancel := range h.cancels {
		log.Infof("Stopping: cancelling in-progress request %s", reqID)
		cancel()
		delete(h.cancels, reqID)
	}

	h.stopSig.Signal()
}

// Notify implements the SensorComponent interface.
// This handler does not require notification of connectivity state changes.
func (h *Handler) Notify(_ common.SensorComponentEvent) {}

func tagMetadata(event reposcan.TagEvent) *central.TagMetadata {
	metadata := &central.TagMetadata{
		ManifestDigest: event.Metadata.ManifestDigest,
		LayerDigests:   event.Metadata.LayerDigests,
	}
	if event.Metadata.Created != nil {
		metadata.Created = protocompat.ConvertTimeToTimestampOrNil(event.Metadata.Created)
	}
	return metadata
}

func scanRequest(req *central.RepoScanRequest) (*storage.BaseImageRepository, reposcan.ScanRequest) {
	// Build repository and scan request from the proto request.
	repo := &storage.BaseImageRepository{
		RepositoryPath: req.GetRepository(),
		TagPattern:     req.GetTagPattern(),
	}

	// Convert proto request to scan request.
	scanReq := reposcan.ScanRequest{
		Pattern:   req.GetTagPattern(),
		CheckTags: make(map[string]*storage.BaseImageTag),
		SkipTags:  make(map[string]struct{}),
	}

	// Tags to recheck have cached digests.
	for tag, cached := range req.GetTagsToRecheck() {
		scanReq.CheckTags[tag] = &storage.BaseImageTag{
			Tag:            tag,
			ManifestDigest: cached.GetManifestDigest(),
		}
	}

	// Tags to ignore are skipped entirely.
	for _, tag := range req.GetTagsToIgnore() {
		scanReq.SkipTags[tag] = struct{}{}
	}

	return repo, scanReq
}
