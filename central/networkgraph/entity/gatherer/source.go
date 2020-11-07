package gatherer

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/license/manager"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/license"
	"github.com/stackrox/rox/pkg/networkgraph/defaultexternalsrcs"
	"github.com/stackrox/rox/pkg/urlfmt"
)

// DefaultNetworksRemoteSource wraps network graph external sources remote data and checksum URLs.
type DefaultNetworksRemoteSource struct {
	checksumURL string
	dataURL     string
}

// NewDefaultNetworksRemoteSource return an instance of DefaultNetworksRemoteSource.
func NewDefaultNetworksRemoteSource(licenseMgr manager.LicenseManager) (*DefaultNetworksRemoteSource, error) {
	source := &DefaultNetworksRemoteSource{}
	params := license.IDAsURLParam(licenseMgr.GetActiveLicense().GetMetadata().GetId())

	latestPrefixURL, err := urlfmt.FullyQualifiedURL(defaultexternalsrcs.RemoteBaseURL, params, defaultexternalsrcs.LatestPrefixFileName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain URL for latest provider networks remote directory")
	}

	prefixBytes, err := httputil.HTTPGet(latestPrefixURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain latest provider networks remote directory name")
	}

	source.checksumURL, err = urlfmt.FullyQualifiedURL(defaultexternalsrcs.RemoteBaseURL, params, string(prefixBytes), defaultexternalsrcs.ChecksumFileName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain URL for latest provider networks remote checksum")
	}

	source.dataURL, err = urlfmt.FullyQualifiedURL(defaultexternalsrcs.RemoteBaseURL, params, string(prefixBytes), defaultexternalsrcs.DataFileName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain URL for latest provider networks remote data")
	}

	return source, nil
}

// DataURL returns external sources data URL as string.
func (s *DefaultNetworksRemoteSource) DataURL() string {
	return s.dataURL
}

// ChecksumURL returns external sources checksum URL as string.
func (s *DefaultNetworksRemoteSource) ChecksumURL() string {
	return s.checksumURL
}
