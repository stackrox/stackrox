package util

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/roxctl/common/zipdownload"
)

// GetBundle downloads the sensor bundle for the cluster with the given ID to the specified output directory.
func GetBundle(id, outputDir string, createUpgraderSA bool, timeout time.Duration) error {
	path := "/api/extensions/clusters/zip"
	body, err := json.Marshal(&apiparams.ClusterZip{ID: id, CreateUpgraderSA: &createUpgraderSA})
	if err != nil {
		return err
	}
	return zipdownload.GetZip(zipdownload.GetZipOptions{
		Path:       path,
		Method:     http.MethodPost,
		Body:       body,
		Timeout:    timeout,
		BundleType: "sensor",
		ExpandZip:  true,
		OutputDir:  outputDir,
	})
}
