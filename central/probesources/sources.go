package probesources

import (
	"context"
	"net/http"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/stackrox/rox/central/clusters"
	probeUploadManager "github.com/stackrox/rox/central/probeupload/manager"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/kocache"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/probeupload"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	httpTimeout = 30 * time.Second
)

var (
	log          = logging.LoggerForModule()
	instance     probeSources
	instanceInit sync.Once
)

//go:generate mockgen-wrapper

// ProbeSources interface provides the availability of the probes packages.
type ProbeSources interface {
	AnyAvailable(ctx context.Context) (bool, error)
	CopyAsSlice() []probeupload.ProbeSource
}

// probeSources contains the list of activated probe sources.
type probeSources struct {
	probeSources []probeupload.ProbeSource
}

// CopyAsSlice retrieves the activated probe sources as a slice backed by newly allocated memory.
func (s *probeSources) CopyAsSlice() []probeupload.ProbeSource {
	probeSources := make([]probeupload.ProbeSource, len(s.probeSources))
	copy(probeSources, s.probeSources)
	return probeSources
}

// AnyAvailable implements a simple heuristic for the availability of kernel probes.
// It returns true if any of the activated probe sources is available in the sense
// that it does support the transmitting of (some) kernel probes.
func (s *probeSources) AnyAvailable(ctx context.Context) (bool, error) {
	var finalErr error

	for _, source := range s.probeSources {
		isAvailable, err := source.IsAvailable(ctx)
		if err != nil {
			log.Warnf("Failed to check availability of kernel probe source %T: %v", source, err)
			finalErr = multierror.Append(finalErr, err)
		}
		if isAvailable {
			return true, nil
		}
	}

	return false, finalErr
}

func (s *probeSources) initializeStandardSources(probeUploadManager probeUploadManager.Manager) {
	s.probeSources = make([]probeupload.ProbeSource, 0, 2)
	s.probeSources = append(s.probeSources, probeUploadManager)
	if env.OfflineModeEnv.BooleanSetting() {
		return
	}
	baseURL := clusters.CollectorModuleDownloadBaseURL.Setting()
	if baseURL == "" {
		return
	}

	httpClient := &http.Client{
		Transport: proxy.RoundTripper(),
		Timeout:   httpTimeout,
	}
	s.probeSources = append(s.probeSources, kocache.New(context.Background(), httpClient, baseURL))
}

// Singleton returns the singleton instance for the ProbeSources entity.
func Singleton() ProbeSources {
	instanceInit.Do(func() {
		instance.initializeStandardSources(probeUploadManager.Singleton())
	})
	return &instance
}
