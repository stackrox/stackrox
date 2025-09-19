package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/quay/zlog"
	"github.com/rs/zerolog"
	"github.com/stackrox/rox/compliance/node/index"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// This program demonstrates how to create a `v4.NodeIndex` object having access to a mounted filesystem,
// so that it can be later fed to scanner v4 matcher.
func main() {
	cfg := index.NodeIndexerConfig{
		// Where to start searching for the rpm db - usually /usr/share/rpm or /var/lib/rpm
		HostPath: "/usr/share/rpm",
		// Client used to fetch the mapping json
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
		// URL where to get the mapping json from
		// In ACS, we fetch it internally from the cluster (to prevent Collector from accessing the Internet):
		// "https://sensor.stackrox.svc:443/scanner/definitions?file=repo2cpe"
		Repo2CPEMappingURL: "https://security.access.redhat.com/data/metrics/repository-to-cpe.json",
		Timeout:            10 * time.Second,
	}

	// Enable debug logs from scanner-indexer (as clair-core uses a different logger)
	l := zerolog.New(os.Stderr).
		Level(zerolog.DebugLevel)
	zlog.Set(&l)

	report, err := index.NewNodeIndexer(cfg).IndexNode(context.Background())
	if err != nil {
		log.Errorf("creating node index: %v", err)
	}
	fmt.Printf("Node indexer report:\n%v", report)
}
