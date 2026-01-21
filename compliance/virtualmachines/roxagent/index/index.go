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
	mappingClientTimeout  = 30 * time.Second
	maxInitialReportDelay = 20 * time.Minute
)

func RunDaemon(ctx context.Context, cfg *common.Config, client *vsock.Client) error {
	if err := RunSingleWithInitialDelay(ctx, cfg, client); err != nil {
		log.Errorf("Failed to run initial index: %v", err)
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

// RunSingleWithInitialDelay applies a randomized startup delay before sending the first
// index report unless explicitly bypassed via the SendNow flag.
func RunSingleWithInitialDelay(ctx context.Context, cfg *common.Config, client *vsock.Client) error {
	if err := maybeDelayInitialReport(ctx, cfg.SendNow); err != nil {
		return fmt.Errorf("initial delay: %w", err)
	}
	return RunSingle(ctx, cfg, client)
}

func maybeDelayInitialReport(ctx context.Context, sendNow bool) error {
	if sendNow {
		log.Infof("Bypassing randomized initial delay (--now flag provided).")
		return nil
	}

	delay := time.Duration(rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(int64(maxInitialReportDelay) + 1))
	log.Infof("Delaying initial index report by %s (use --now to send immediately).", delay)
	if delay == 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
