package util

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/roxctl/common/zipdownload"
)

const (
	// WarningSlimCollectorModeWithoutKernelSupport contains the warning text message, which shall be emitted during cluster generation
	// or get-bundle for an existing cluster in case slim collector mode is enabled and kernel support is not available for Central.
	WarningSlimCollectorModeWithoutKernelSupport = `WARNING: The deployment bundle will reference a slim collector image, but it appears that central cannot provide kernel modules or eBPF probes.
Newly spawned collector pods will have to retrieve a matching kernel module or eBPF probe in order to function.
When central is deployed in offline mode, a matching kernel support package needs to be uploaded to central using

	roxctl collector support-packages upload.`
)

// GetBundle downloads the sensor bundle for the cluster with the given ID to the specified output directory.
func GetBundle(id, outputDir string, createUpgraderSA bool, timeout time.Duration, slimCollector bool) error {
	path := "/api/extensions/clusters/zip"
	body, err := json.Marshal(&apiparams.ClusterZip{ID: id, CreateUpgraderSA: &createUpgraderSA, SlimCollector: &slimCollector})
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
