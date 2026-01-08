package watcher

import (
	"context"
	"fmt"
	"iter"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/baseimage/broker"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/baseimage/reposcan"
	"github.com/stackrox/rox/pkg/baseimage/tagfetcher"
	"github.com/stackrox/rox/pkg/delegatedregistry"
)

// DelegatedScanner delegates scanning to a secured cluster.
type DelegatedScanner struct {
	delegator delegatedregistry.Delegator
	broker    *broker.Broker
	clusterID string
}

// NewDelegatedScanner creates a DelegatedScanner.
func NewDelegatedScanner(delegator delegatedregistry.Delegator, broker *broker.Broker, clusterID string) *DelegatedScanner {
	return &DelegatedScanner{
		delegator: delegator,
		broker:    broker,
		clusterID: clusterID,
	}
}

// Name implements reposcan.Scanner.
func (c *DelegatedScanner) Name() string {
	return "delegated"
}

// ScanRepository implements reposcan.Scanner.
func (c *DelegatedScanner) ScanRepository(
	ctx context.Context,
	repo *storage.BaseImageRepository,
	req reposcan.ScanRequest,
) iter.Seq2[reposcan.TagEvent, error] {
	return func(yield func(reposcan.TagEvent, error) bool) {
		for resp, err := range c.broker.StreamRepoScan(ctx, c.clusterID, repoScanRequest(repo, req)) {
			if err != nil {
				yield(reposcan.TagEvent{}, errors.Wrap(err, "streaming repo scan"))
				return
			}
			if update := resp.GetUpdate(); update != nil {
				event := tagEvent(update)
				if !yield(event, nil) {
					return
				}
				continue
			}
		}
	}
}

func repoScanRequest(repo *storage.BaseImageRepository, req reposcan.ScanRequest) *central.RepoScanRequest {
	// Build the proto request.
	protoReq := &central.RepoScanRequest{
		Repository:    repo.GetRepositoryPath(),
		TagPattern:    req.Pattern,
		TagsToIgnore:  make([]string, 0, len(req.SkipTags)),
		TagsToRecheck: make(map[string]*central.TagMetadata, len(req.CheckTags)),
	}

	// Convert SkipTags to slice.
	for tag := range req.SkipTags {
		protoReq.TagsToIgnore = append(protoReq.TagsToIgnore, tag)
	}

	// Convert CheckTags to proto format.
	for tag, tagInfo := range req.CheckTags {
		protoMeta := &central.TagMetadata{
			ManifestDigest: tagInfo.GetManifestDigest(),
			// Note: LayerDigests are not in storage.BaseImageTag yet (will be added in future)
			// For now, we only check manifest digest for change detection
		}
		if tagInfo.GetCreated() != nil {
			protoMeta.Created = tagInfo.GetCreated()
		}
		protoReq.TagsToRecheck[tag] = protoMeta
	}
	return protoReq
}

// tagEvent converts a RepoScanResponse.Update to a TagEvent.
func tagEvent(update *central.RepoScanResponse_Update) reposcan.TagEvent {
	tag := update.GetTag()
	if errMsg := update.GetError(); errMsg != "" {
		return reposcan.TagEvent{
			Tag:   tag,
			Type:  reposcan.TagEventError,
			Error: fmt.Errorf("sensor error: %s", errMsg),
		}
	}
	if protoMeta := update.GetMetadata(); protoMeta != nil {
		metadata := &tagfetcher.TagMetadata{
			Tag:            tag,
			ManifestDigest: protoMeta.GetManifestDigest(),
			LayerDigests:   protoMeta.GetLayerDigests(),
		}
		if protoMeta.Created != nil {
			created := protoMeta.Created.AsTime()
			metadata.Created = &created
		}
		return reposcan.TagEvent{
			Tag:      tag,
			Type:     reposcan.TagEventMetadata,
			Metadata: metadata,
		}
	}
	// Shouldn't reach here, but handle gracefully.
	return reposcan.TagEvent{
		Tag:   tag,
		Type:  reposcan.TagEventError,
		Error: fmt.Errorf("update message has no metadata or error"),
	}
}
