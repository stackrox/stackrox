package broker

import (
	"context"
	"fmt"
	"iter"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	// initialTimeout is the timeout before receiving RepoScanStart.
	initialTimeout = 30 * time.Second

	// progressTimeout is the timeout between any streaming messages.
	progressTimeout = 2 * time.Minute
)

var log = logging.LoggerForModule()

// Broker coordinates repository scan requests and responses between Central and Sensor.
type Broker struct {
	connManager connection.Manager
	stopSig     concurrency.ReadOnlyErrorSignal

	// chans stores channels organized by cluster ID, then request ID.
	chans      map[string]map[string]chan *central.RepoScanResponse
	chansMutex sync.Mutex
}

// New creates a new Broker instance.
func New(connManager connection.Manager, stopSig concurrency.ReadOnlyErrorSignal) *Broker {
	return &Broker{
		connManager: connManager,
		stopSig:     stopSig,
		chans:       make(map[string]map[string]chan *central.RepoScanResponse),
	}
}

// StreamRepoScan sends a repository scan request and returns an iterator over responses.
// The iterator yields (response, nil) for each tag update, or (nil, err) on terminal error.
func (b *Broker) StreamRepoScan(ctx context.Context, clusterID string, req *central.RepoScanRequest) iter.Seq2[*central.RepoScanResponse, error] {
	return func(yield func(*central.RepoScanResponse, error) bool) {
		if clusterID == "" {
			yield(nil, errors.New("missing cluster ID"))
			return
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		concurrency.CancelContextOnSignal(ctx, cancel, b.stopSig)

		reqID := uuid.NewV4().String()
		req.RequestId = reqID
		// Create channel for this request.
		// TODO(jvdm): The channel size should be configurable.
		retC := make(chan *central.RepoScanResponse, 10)

		concurrency.WithLock(&b.chansMutex, func() {
			reqChans, ok := b.chans[clusterID]
			if !ok {
				reqChans = make(map[string]chan *central.RepoScanResponse)
				b.chans[clusterID] = reqChans
			}
			reqChans[reqID] = retC
		})

		defer concurrency.WithLock(&b.chansMutex, func() {
			reqChans, ok := b.chans[clusterID]
			if ok {
				// TODO(jvdm): Should we drain the channel?
				delete(reqChans, reqID)
			}
			if len(reqChans) == 0 {
				delete(b.chans, clusterID)
			}
		})

		// Send request to Sensor.
		if err := b.connManager.SendMessage(clusterID, &central.MsgToSensor{
			Msg: &central.MsgToSensor_RepoScanRequest{
				RepoScanRequest: req,
			},
		}); err != nil {
			yield(nil, fmt.Errorf("sending repo scan request to cluster %q: %w", clusterID, err))
			return
		}

		log.Infof("Sent repo scan request %q to cluster %q for repository %q",
			reqID, clusterID, req.GetRepository())

		started := false
		timeout := initialTimeout
		timeoutTicket := time.NewTicker(timeout)
		defer timeoutTicket.Stop()

		sendCancellation := func() {
			// Ignore errors, best effort cancellation.
			_ = b.connManager.SendMessage(clusterID, &central.MsgToSensor{
				Msg: &central.MsgToSensor_RepoScanCancellation{
					RepoScanCancellation: &central.RepoScanCancellation{
						RequestId: reqID,
					},
				},
			})
		}

		var resp *central.RepoScanResponse
		for {
			select {
			case <-ctx.Done():
				sendCancellation()
				yield(nil, fmt.Errorf("context done: %w", ctx.Err()))
				return
			case <-b.stopSig.Done():
				sendCancellation()
				yield(nil, fmt.Errorf("stop signal done: %w", b.stopSig.Err()))
				return
			case <-timeoutTicket.C:
				sendCancellation()
				if started {
					yield(nil, errors.Errorf("sensor didn't send any data in last %s (idle timeout)", progressTimeout))
				} else {
					yield(nil, errors.Errorf("sensor didn't send RepoScanStart in %s (initial timeout)", initialTimeout))
				}
				return
			case resp = <-retC:
				if resp == nil {
					yield(nil, errors.New("channel closed"))
					return
				}
				if !started {
					started = true
					timeout = progressTimeout
				}
				timeoutTicket.Reset(timeout)
			}
			if end := resp.GetEnd(); end != nil {
				if !end.GetSuccess() {
					yield(nil, fmt.Errorf("repo scan terminal error: %s", end.GetError()))
					return
				}
				return
			}
			if !yield(resp, nil) {
				return
			}
		}
	}
}

// OnScanResponse is called when a RepoScanResponse arrives from Sensor.
func (b *Broker) OnScanResponse(clusterID string, resp *central.RepoScanResponse) {
	reqID := resp.GetRequestId()
	retC := concurrency.WithLock1(&b.chansMutex, func() chan *central.RepoScanResponse {
		reqChans, ok := b.chans[clusterID]
		if !ok {
			log.Warnf("unknown cluster id %q for request %q: no active channels found", clusterID, reqID)
			return nil
		}
		c, ok := reqChans[reqID]
		if !ok {
			log.Warnf("unknown request id %q from cluster %q): no active channel found: "+
				"silently ignoring request id", reqID, clusterID)
			return nil
		}
		return c
	})
	if retC == nil {
		// TODO(jvdm): Add metrics to track this situation.
		return
	}
	select {
	case retC <- resp:
	default:
		// TODO(jvdm): Add metrics to track this situation.
		log.Errorf("dropping request %q (cluster: %q): response channel is full or closed", reqID, clusterID)
	}
}

// OnClusterDisconnect cancels all pending requests for the given cluster.
// This prevents resource leaks when a cluster disconnects while scans are in progress.
func (b *Broker) OnClusterDisconnect(clusterID string) {
	chans := concurrency.WithLock1(&b.chansMutex, func() map[string]chan *central.RepoScanResponse {
		reqChans, ok := b.chans[clusterID]
		if !ok {
			return nil
		}
		delete(b.chans, clusterID)
		return reqChans
	})

	for reqID, ch := range chans {
		if ch != nil {
			close(ch)
			log.Infof("Closed channel for request %s due to cluster %s disconnect", reqID, clusterID)
		}
	}
}
