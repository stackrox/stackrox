package bundle

import (
	"bytes"
	"io"
	"net/http"

	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/upgrader/upgradectx"
)

type fetcher struct {
	ctx *upgradectx.UpgradeContext
}

func (f *fetcher) FetchBundle() (Contents, error) {
	// If we are not in standalone mode which means we should be fetching a bundle,
	// and cluster id is empty, panic.
	if f.ctx.ClusterID() == "" {
		log.Panic("Cluster id is empty, unable to fetch bundle for upgrade")
	}

	resByID := &v1.ResourceByID{
		Id: f.ctx.ClusterID(),
	}
	var buf bytes.Buffer
	if err := new(jsonpb.Marshaler).Marshal(&buf, resByID); err != nil {
		return nil, utils.ShouldErr(err)
	}

	req, err := http.NewRequest(http.MethodPost, "/api/extensions/clusters/zip", &buf)
	if err != nil {
		return nil, utils.ShouldErr(err)
	}

	resp, err := f.ctx.DoCentralHTTPRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "making HTTP request to central for cluster bundle download")
	}
	defer utils.IgnoreError(resp.Body.Close)
	if err := httputil.ResponseToError(resp); err != nil {
		return nil, errors.Wrap(err, "making HTTP request to central for cluster bundle download")
	}

	bundleContents, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading cluster bundle HTTP response body")
	}

	return ContentsFromZIPData(bytes.NewReader(bundleContents), int64(len(bundleContents)))
}
