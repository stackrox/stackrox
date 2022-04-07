package sac

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/default-authz-plugin/pkg/payload"
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/client"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	clusterContext = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster),
		))
)

// ScopeRequestTrackerImpl is the implementation of the interface which tracks pending scope requests.  It is only
// exported because the tests live in a different package
type ScopeRequestTrackerImpl struct {
	client           client.Client
	clusterDataStore datastore.DataStore
	runnerChannel    chan struct{}
	principal        *payload.Principal

	requestSetLock sync.Mutex
	// Keep a Set for de-duplication
	requestList []sac.ScopeRequest
}

// NewRequestTracker creates a new empty ScopeRequestTracker.
func NewRequestTracker(client client.Client, clusterDatastore datastore.DataStore, principal *payload.Principal) sac.ScopeRequestTracker {
	tracker := &ScopeRequestTrackerImpl{
		client:           client,
		clusterDataStore: clusterDatastore,
		runnerChannel:    make(chan struct{}, 1),
		principal:        principal,
	}
	tracker.runnerChannel <- struct{}{}
	return tracker
}

// AddRequested adds requested scopes to the list of pending permission checks
func (srt *ScopeRequestTrackerImpl) AddRequested(scopes ...sac.ScopeRequest) {
	srt.requestSetLock.Lock()
	defer srt.requestSetLock.Unlock()

	srt.requestList = append(srt.requestList, scopes...)
}

// getAndClearPendingRequests copies, clears, and returns the list of pending permission checks
func (srt *ScopeRequestTrackerImpl) getAndClearPendingRequests() []sac.ScopeRequest {
	srt.requestSetLock.Lock()
	defer srt.requestSetLock.Unlock()

	requestList := srt.requestList
	srt.requestList = nil
	return requestList
}

// PerformChecks performs all pending checks, clear the pending scopes list, and update the relevant scopes
func (srt *ScopeRequestTrackerImpl) PerformChecks(ctx context.Context) error {
	select {
	case <-srt.runnerChannel:
		defer func() { srt.runnerChannel <- struct{}{} }()

		requestList := srt.getAndClearPendingRequests()
		if len(requestList) <= 0 {
			return nil
		}

		accessScopeList := make([]payload.AccessScope, 0, len(requestList))
		// Multiple ScopeCheckerCores can have the same scope if a ScopeCheckerCore has expired and been removed from the
		// scope tree.  The old SCC still needs to be updated because a goroutine may still be using it.
		requestMap := make(map[payload.AccessScope][]sac.ScopeRequest, len(requestList))
		for _, node := range requestList {
			accessScope := node.GetAccessScope()
			clusterID := accessScope.Attributes.Cluster.ID
			if clusterID != "" {
				// This won't loop infinitely because the clusterContext never invokes the auth plugin client
				clusterName, _, err := srt.clusterDataStore.GetClusterName(clusterContext, clusterID)
				if err != nil {
					return err
				}
				accessScope.Attributes.Cluster.Name = clusterName
			}
			accessScopeList = append(accessScopeList, accessScope)
			requestMap[accessScope] = append(requestMap[accessScope], node)
		}

		allowed, denied, err := srt.client.ForUser(ctx, *srt.principal, accessScopeList...)
		if err != nil {
			// Retry failed requests.  There should be some kind of limit to this or it'll just grow.
			srt.AddRequested(requestList...)
			return err
		}
		for _, allow := range allowed {
			for _, scc := range requestMap[allow] {
				scc.SetState(sac.Allow)
			}
		}
		for _, deny := range denied {
			for _, scc := range requestMap[deny] {
				scc.SetState(sac.Deny)
			}

		}
		return nil
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "PerformChecks() for principal")
	}
}
