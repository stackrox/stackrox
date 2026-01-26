package index

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/stackrox/rox/compliance/node/index"
	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/common"
	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/vsock"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

const (
	mappingClientTimeout = 30 * time.Second
)

func RunDaemon(ctx context.Context, cfg *common.Config, client *vsock.Client) error {
	if err := applyRandomDelay(ctx, cfg.MaxInitialReportDelay); err != nil {
		return fmt.Errorf("delaying initial index: %w", err)
	}

	if err := RunSingle(ctx, cfg, client); err != nil {
		return fmt.Errorf("running initial index: %w", err)
	}

	ticker := time.NewTicker(cfg.IndexInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := RunSingle(ctx, cfg, client); err != nil {
				log.Errorf("Failed to run index: %v", err)
			}
		}
	}
}

func RunSingle(ctx context.Context, cfg *common.Config, client *vsock.Client) error {
	report, err := runIndexer(ctx, cfg)
	if err != nil {
		return fmt.Errorf("creating index report: %w", err)
	}
	if !report.GetSuccess() {
		return fmt.Errorf("failed index report: %s", report.GetErr())
	}
	if err := client.SendIndexReport(report); err != nil {
		return fmt.Errorf("sending index report: %w", err)
	}
	return nil
}

func runIndexer(ctx context.Context, cfg *common.Config) (*v4.IndexReport, error) {
	indexerCfg := index.NodeIndexerConfig{
		HostPath: cfg.IndexHostPath,
		// Client used to fetch the repo to cpe mapping json.
		Client: &http.Client{Transport: proxy.RoundTripper()},
		// URL where to get the repo to cpe mapping json from.
		// In ACS, we fetch it internally from the cluster (to prevent Collector from accessing the Internet):
		// "https://sensor.stackrox.svc:443/scanner/definitions?file=repo2cpe"
		Repo2CPEMappingURL: cfg.RepoToCPEMappingURL,
		Timeout:            mappingClientTimeout,
		// Disable package filtering.
		PackageDBFilter: "",
	}

	report, err := index.NewNodeIndexer(indexerCfg).IndexNode(ctx)
	if err != nil {
		return nil, err
	}
	return report, nil
}

func applyRandomDelay(ctx context.Context, maxDelay time.Duration) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if maxDelay <= 0 {
		return nil
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	delay := time.Duration(r.Int63n(maxDelay.Nanoseconds() + 1))

	log.Infof("Delaying initial index report by %s (use --max-initial-report-delay to control this).", delay)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}
