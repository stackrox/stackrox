package reposcanhandler

import (
	"context"
	"errors"
	"iter"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/baseimage/reposcan"
	"github.com/stackrox/rox/pkg/baseimage/tagfetcher"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stretchr/testify/suite"
)

func TestHandler(t *testing.T) {
	suite.Run(t, &HandlerTestSuite{})
}

type HandlerTestSuite struct {
	suite.Suite
}

// mockScanner implements reposcan.Scanner for testing.
type mockScanner struct {
	name       string
	scanFunc   func(ctx context.Context, repo *storage.BaseImageRepository, req reposcan.ScanRequest) iter.Seq2[reposcan.TagEvent, error]
	scanCalled bool
}

func (m *mockScanner) Name() string {
	return m.name
}

func (m *mockScanner) ScanRepository(ctx context.Context, repo *storage.BaseImageRepository, req reposcan.ScanRequest) iter.Seq2[reposcan.TagEvent, error] {
	m.scanCalled = true
	if m.scanFunc != nil {
		return m.scanFunc(ctx, repo, req)
	}
	// Default: return empty sequence.
	return func(yield func(reposcan.TagEvent, error) bool) {}
}

// TestAccepts verifies the handler accepts the correct message types.
func (s *HandlerTestSuite) TestAccepts() {
	h := NewHandler(nil)

	// Should accept RepoScanRequest.
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_RepoScanRequest{
			RepoScanRequest: &central.RepoScanRequest{},
		},
	}
	s.True(h.Accepts(msg))

	// Should accept RepoScanCancellation.
	msg = &central.MsgToSensor{
		Msg: &central.MsgToSensor_RepoScanCancellation{
			RepoScanCancellation: &central.RepoScanCancellation{},
		},
	}
	s.True(h.Accepts(msg))

	// Should not accept other message types.
	msg = &central.MsgToSensor{
		Msg: &central.MsgToSensor_ClusterConfig{
			ClusterConfig: &central.ClusterConfig{},
		},
	}
	s.False(h.Accepts(msg))
}

// TestProcessRequestSuccess verifies successful scan with metadata.
func (s *HandlerTestSuite) TestProcessRequestSuccess() {
	// Mock scanner that yields 2 successful tags.
	scanner := &mockScanner{
		name: "test-scanner",
		scanFunc: func(ctx context.Context, repo *storage.BaseImageRepository, req reposcan.ScanRequest) iter.Seq2[reposcan.TagEvent, error] {
			s.Equal("docker.io/library/nginx", repo.GetRepositoryPath())
			s.Equal("1.2*", req.Pattern)

			return func(yield func(reposcan.TagEvent, error) bool) {
				// Yield 2 successful tags.
				createdTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				if !yield(reposcan.TagEvent{
					Tag:  "1.20",
					Type: reposcan.TagEventMetadata,
					Metadata: &tagfetcher.TagMetadata{
						ManifestDigest: "sha256:abc123",
						Created:        &createdTime,
						LayerDigests:   []string{"sha256:layer1", "sha256:layer2"},
					},
				}, nil) {
					return
				}

				yield(reposcan.TagEvent{
					Tag:  "1.21",
					Type: reposcan.TagEventMetadata,
					Metadata: &tagfetcher.TagMetadata{
						ManifestDigest: "sha256:def456",
						Created:        &createdTime,
						LayerDigests:   []string{"sha256:layer3"},
					},
				}, nil)
			}
		},
	}

	h := NewHandler(scanner)

	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_RepoScanRequest{
			RepoScanRequest: &central.RepoScanRequest{
				RequestId:  "req-2",
				Repository: "docker.io/library/nginx",
				TagPattern: "1.2*",
			},
		},
	}

	err := h.ProcessMessage(context.Background(), msg)
	s.NoError(err)

	// Collect responses: Start, 2 Updates, End.
	responses := collectResponses(s.T(), h.ResponsesC(), 4, 500*time.Millisecond)
	s.Len(responses, 4)

	// Response 1: Start.
	s.Equal("req-2", responses[0].GetRequestId())
	s.NotNil(responses[0].GetStart())

	// Response 2: Update for tag 1.20.
	s.Equal("req-2", responses[1].GetRequestId())
	update1 := responses[1].GetUpdate()
	s.NotNil(update1)
	s.Equal("1.20", update1.GetTag())
	metadata1 := update1.GetMetadata()
	s.NotNil(metadata1)
	s.Equal("sha256:abc123", metadata1.GetManifestDigest())
	s.Len(metadata1.GetLayerDigests(), 2)

	// Response 3: Update for tag 1.21.
	s.Equal("req-2", responses[2].GetRequestId())
	update2 := responses[2].GetUpdate()
	s.NotNil(update2)
	s.Equal("1.21", update2.GetTag())

	// Response 4: End with success.
	s.Equal("req-2", responses[3].GetRequestId())
	end := responses[3].GetEnd()
	s.NotNil(end)
	s.True(end.GetSuccess())
	s.Equal(int32(2), end.GetSuccessfulCount())
	s.Equal(int32(0), end.GetFailedCount())

	s.True(scanner.scanCalled)
}

// TestProcessRequestWithErrors verifies handling of per-tag errors.
func (s *HandlerTestSuite) TestProcessRequestWithErrors() {
	scanner := &mockScanner{
		name: "test-scanner",
		scanFunc: func(ctx context.Context, repo *storage.BaseImageRepository, req reposcan.ScanRequest) iter.Seq2[reposcan.TagEvent, error] {
			return func(yield func(reposcan.TagEvent, error) bool) {
				// Yield 1 successful tag.
				createdTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				if !yield(reposcan.TagEvent{
					Tag:  "latest",
					Type: reposcan.TagEventMetadata,
					Metadata: &tagfetcher.TagMetadata{
						ManifestDigest: "sha256:abc123",
						Created:        &createdTime,
					},
				}, nil) {
					return
				}

				// Yield 1 tag with error.
				yield(reposcan.TagEvent{
					Tag:   "broken",
					Type:  reposcan.TagEventError,
					Error: errors.New("manifest not found"),
				}, nil)
			}
		},
	}

	h := NewHandler(scanner)

	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_RepoScanRequest{
			RepoScanRequest: &central.RepoScanRequest{
				RequestId:  "req-3",
				Repository: "docker.io/library/test",
				TagPattern: "*",
			},
		},
	}

	err := h.ProcessMessage(context.Background(), msg)
	s.NoError(err)

	// Collect responses: Start, Update (success), Update (error), End.
	responses := collectResponses(s.T(), h.ResponsesC(), 4, 500*time.Millisecond)
	s.Len(responses, 4)

	// Response 1: Start.
	s.NotNil(responses[0].GetStart())

	// Response 2: Update for successful tag.
	update1 := responses[1].GetUpdate()
	s.Equal("latest", update1.GetTag())
	s.NotNil(update1.GetMetadata())

	// Response 3: Update for failed tag.
	update2 := responses[2].GetUpdate()
	s.Equal("broken", update2.GetTag())
	s.Contains(update2.GetError(), "manifest not found")

	// Response 4: End with partial success.
	end := responses[3].GetEnd()
	s.True(end.GetSuccess()) // Scan completed even with per-tag errors.
	s.Equal(int32(1), end.GetSuccessfulCount())
	s.Equal(int32(1), end.GetFailedCount())
}

// TestProcessRequestFatalError verifies handling of fatal scan errors.
func (s *HandlerTestSuite) TestProcessRequestFatalError() {
	scanner := &mockScanner{
		name: "test-scanner",
		scanFunc: func(ctx context.Context, repo *storage.BaseImageRepository, req reposcan.ScanRequest) iter.Seq2[reposcan.TagEvent, error] {
			return func(yield func(reposcan.TagEvent, error) bool) {
				// Yield fatal error (can't list tags).
				yield(reposcan.TagEvent{}, errors.New("repository not found"))
			}
		},
	}

	h := NewHandler(scanner)

	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_RepoScanRequest{
			RepoScanRequest: &central.RepoScanRequest{
				RequestId:  "req-4",
				Repository: "docker.io/invalid/repo",
				TagPattern: "*",
			},
		},
	}

	err := h.ProcessMessage(context.Background(), msg)
	s.NoError(err)

	// Collect responses: Start, End (with fatal error).
	responses := collectResponses(s.T(), h.ResponsesC(), 2, 500*time.Millisecond)
	s.Len(responses, 2)

	// Response 1: Start.
	s.NotNil(responses[0].GetStart())

	// Response 2: End with fatal error.
	end := responses[1].GetEnd()
	s.False(end.GetSuccess())
	s.Contains(end.GetError(), "repository not found")
}

// TestProcessCancellation verifies cancellation of in-progress scans.
func (s *HandlerTestSuite) TestProcessCancellation() {
	// Scanner that blocks indefinitely, simulating a slow external call.
	cancelledCh := make(chan struct{})
	blockedCh := make(chan struct{})
	scanner := &mockScanner{
		name: "test-scanner",
		scanFunc: func(ctx context.Context, repo *storage.BaseImageRepository, req reposcan.ScanRequest) iter.Seq2[reposcan.TagEvent, error] {
			return func(yield func(reposcan.TagEvent, error) bool) {
				// Yield first tag successfully.
				createdTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				if !yield(reposcan.TagEvent{
					Tag:  "tag1",
					Type: reposcan.TagEventMetadata,
					Metadata: &tagfetcher.TagMetadata{
						ManifestDigest: "sha256:abc123",
						Created:        &createdTime,
					},
				}, nil) {
					return
				}

				// Signal that we're about to block.
				close(blockedCh)

				// Simulate blocking on slow external call that respects context.
				// In real scanner, this would be HTTP request to registry.
				<-ctx.Done()
				close(cancelledCh)
				// Don't yield any more events - just return.
				// This simulates the external call being cancelled.
			}
		},
	}

	h := NewHandler(scanner)

	// Send request.
	requestMsg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_RepoScanRequest{
			RepoScanRequest: &central.RepoScanRequest{
				RequestId:  "req-5",
				Repository: "docker.io/library/test",
				TagPattern: "*",
			},
		},
	}

	err := h.ProcessMessage(context.Background(), requestMsg)
	s.NoError(err)

	// Collect Start and first Update.
	responses := collectResponses(s.T(), h.ResponsesC(), 2, 500*time.Millisecond)
	s.Len(responses, 2)
	s.NotNil(responses[0].GetStart())
	update := responses[1].GetUpdate()
	s.NotNil(update)
	s.Equal("tag1", update.GetTag())

	// Wait for scanner to be blocked on ctx.Done().
	select {
	case <-blockedCh:
		// Scanner is now blocked.
	case <-time.After(500 * time.Millisecond):
		s.Fail("scanner didn't block")
	}

	// Send cancellation.
	cancelMsg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_RepoScanCancellation{
			RepoScanCancellation: &central.RepoScanCancellation{
				RequestId: "req-5",
			},
		},
	}

	err = h.ProcessMessage(context.Background(), cancelMsg)
	s.NoError(err)

	// Wait for scanner to detect cancellation.
	select {
	case <-cancelledCh:
		// Scanner was cancelled.
	case <-time.After(500 * time.Millisecond):
		s.Fail("scanner didn't detect cancellation")
	}

	// No End message is sent after cancellation - the handler just returns.
	// Central has already stopped waiting for responses after sending cancellation.
	// Verify no more messages are sent.
	select {
	case msg := <-h.ResponsesC():
		s.Failf("unexpected message after cancellation", "got: %+v", msg)
	case <-time.After(100 * time.Millisecond):
		// Expected - no more messages.
	}
}

// TestStopCancelsAllRequests verifies Stop cancels all pending requests.
func (s *HandlerTestSuite) TestStopCancelsAllRequests() {
	// Scanner that blocks until context is cancelled.
	blockCh1 := make(chan struct{})
	blockCh2 := make(chan struct{})
	scanner := &mockScanner{
		name: "test-scanner",
		scanFunc: func(ctx context.Context, repo *storage.BaseImageRepository, req reposcan.ScanRequest) iter.Seq2[reposcan.TagEvent, error] {
			return func(yield func(reposcan.TagEvent, error) bool) {
				// Block until cancelled.
				<-ctx.Done()
				if repo.GetRepositoryPath() == "repo1" {
					close(blockCh1)
				} else {
					close(blockCh2)
				}
			}
		},
	}

	h := NewHandler(scanner)

	// Send 2 requests.
	req1 := &central.MsgToSensor{
		Msg: &central.MsgToSensor_RepoScanRequest{
			RepoScanRequest: &central.RepoScanRequest{
				RequestId:  "req-6",
				Repository: "repo1",
				TagPattern: "*",
			},
		},
	}
	req2 := &central.MsgToSensor{
		Msg: &central.MsgToSensor_RepoScanRequest{
			RepoScanRequest: &central.RepoScanRequest{
				RequestId:  "req-7",
				Repository: "repo2",
				TagPattern: "*",
			},
		},
	}

	s.NoError(h.ProcessMessage(context.Background(), req1))
	s.NoError(h.ProcessMessage(context.Background(), req2))

	// Collect Start messages.
	_ = collectResponses(s.T(), h.ResponsesC(), 2, 500*time.Millisecond)

	// Stop handler - should cancel both requests.
	h.Stop()

	// Wait for both scans to be cancelled.
	select {
	case <-blockCh1:
	case <-time.After(500 * time.Millisecond):
		s.Fail("req1 was not cancelled")
	}

	select {
	case <-blockCh2:
	case <-time.After(500 * time.Millisecond):
		s.Fail("req2 was not cancelled")
	}
}

// TestTagsToRecheckConverted verifies tags_to_recheck is converted correctly.
func (s *HandlerTestSuite) TestTagsToRecheckConverted() {
	scanner := &mockScanner{
		name: "test-scanner",
		scanFunc: func(ctx context.Context, repo *storage.BaseImageRepository, req reposcan.ScanRequest) iter.Seq2[reposcan.TagEvent, error] {
			// Verify CheckTags was populated from tags_to_recheck.
			s.Len(req.CheckTags, 2)
			s.Contains(req.CheckTags, "v1.0")
			s.Contains(req.CheckTags, "v2.0")
			s.Equal("sha256:old1", req.CheckTags["v1.0"].GetManifestDigest())
			s.Equal("sha256:old2", req.CheckTags["v2.0"].GetManifestDigest())

			return func(yield func(reposcan.TagEvent, error) bool) {}
		},
	}

	h := NewHandler(scanner)

	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_RepoScanRequest{
			RepoScanRequest: &central.RepoScanRequest{
				RequestId:  "req-8",
				Repository: "docker.io/library/test",
				TagPattern: "*",
				TagsToRecheck: map[string]*central.TagMetadata{
					"v1.0": {ManifestDigest: "sha256:old1"},
					"v2.0": {ManifestDigest: "sha256:old2"},
				},
			},
		},
	}

	s.NoError(h.ProcessMessage(context.Background(), msg))

	// Wait for Start.
	_ = collectResponses(s.T(), h.ResponsesC(), 1, 500*time.Millisecond)
}

// TestTagsToIgnoreConverted verifies tags_to_ignore is converted correctly.
func (s *HandlerTestSuite) TestTagsToIgnoreConverted() {
	scanner := &mockScanner{
		name: "test-scanner",
		scanFunc: func(ctx context.Context, repo *storage.BaseImageRepository, req reposcan.ScanRequest) iter.Seq2[reposcan.TagEvent, error] {
			// Verify SkipTags was populated from tags_to_ignore.
			s.Len(req.SkipTags, 3)
			_, ok := req.SkipTags["old1"]
			s.True(ok)
			_, ok = req.SkipTags["old2"]
			s.True(ok)
			_, ok = req.SkipTags["old3"]
			s.True(ok)

			return func(yield func(reposcan.TagEvent, error) bool) {}
		},
	}

	h := NewHandler(scanner)

	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_RepoScanRequest{
			RepoScanRequest: &central.RepoScanRequest{
				RequestId:    "req-9",
				Repository:   "docker.io/library/test",
				TagPattern:   "*",
				TagsToIgnore: []string{"old1", "old2", "old3"},
			},
		},
	}

	s.NoError(h.ProcessMessage(context.Background(), msg))

	// Wait for Start.
	_ = collectResponses(s.T(), h.ResponsesC(), 1, 500*time.Millisecond)
}

// collectResponses collects N responses from the channel with timeout.
func collectResponses(t *testing.T, ch <-chan *message.ExpiringMessage, n int, timeout time.Duration) []*central.RepoScanResponse {
	var responses []*central.RepoScanResponse
	deadline := time.After(timeout)
	for i := 0; i < n; i++ {
		select {
		case msg := <-ch:
			// ExpiringMessage embeds *MsgFromSensor, so we can call methods directly.
			responses = append(responses, msg.GetRepoScanResponse())
		case <-deadline:
			t.Fatalf("timeout waiting for response %d/%d", i+1, n)
		}
	}
	return responses
}
