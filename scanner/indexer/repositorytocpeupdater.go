package indexer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/quay/claircore/rhel"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/scannerv4/repositorytocpe"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/internal/httputil"
)

// RepositoryToCPEUpdater wraps the httputil.Updater and provides repository-to-CPE mapping
// functionality with periodic refresh support.
type RepositoryToCPEUpdater struct {
	updater *httputil.Updater
	client  *http.Client
}

// NewUpdater creates a new RepositoryToCPEUpdater for the repository-to-CPE mapping.
// It optionally loads initial data from a file and sets up periodic refresh from the URL.
func NewUpdater(ctx context.Context, client *http.Client, url, filePath string) (*RepositoryToCPEUpdater, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "scanner/repositorytocpe.NewUpdater")

	var initValue *repositorytocpe.MappingFile

	// If a file is configured, load it as the initial value.
	if filePath != "" {
		f, err := os.Open(filePath)
		if err == nil {
			initValue = &repositorytocpe.MappingFile{}
			if err := json.NewDecoder(f).Decode(initValue); err != nil {
				zlog.Warn(ctx).Err(err).Msg("failed to decode initial repo-to-CPE mapping file")
				initValue = nil
			}
			defer utils.IgnoreError(f.Close)
		} else {
			zlog.Warn(ctx).Err(err).Str("file", filePath).Msg("failed to open repo-to-CPE mapping file")
		}
	}

	if initValue == nil {
		initValue = &repositorytocpe.MappingFile{}
	}

	// Use default URL if not specified.
	if url == "" {
		url = rhel.DefaultRepo2CPEMappingURL
	}

	updater := httputil.NewUpdater(url, initValue)

	u := &RepositoryToCPEUpdater{
		updater: updater,
		client:  client,
	}

	// Trigger initial fetch if no file data was loaded.
	if len(initValue.Data) == 0 {
		if _, err := updater.Get(ctx, client); err != nil {
			zlog.Warn(ctx).Err(err).Msg("failed to fetch initial repo-to-CPE mapping")
		}
	}

	return u, nil
}

// Get returns the current repository-to-CPE mapping.
// This may trigger a refresh if the periodic update interval has elapsed.
func (u *RepositoryToCPEUpdater) Get(ctx context.Context) (*repositorytocpe.MappingFile, error) {
	v, err := u.updater.Get(ctx, u.client)
	if err != nil && v == nil {
		return nil, err
	}

	mf, ok := v.(*repositorytocpe.MappingFile)
	if !ok || mf == nil {
		return nil, errors.New("unable to get repository-to-CPE mapping")
	}

	return mf, nil
}
