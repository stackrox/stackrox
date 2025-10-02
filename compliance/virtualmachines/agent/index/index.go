package index

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/stackrox/rox/compliance/node/index"
	"github.com/stackrox/rox/compliance/virtualmachines/agent/config"
	"github.com/stackrox/rox/compliance/virtualmachines/agent/vsock"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

const (
	mappingClientTimeout = 30 * time.Second
)

func RunDaemon(ctx context.Context, cfg *config.AgentConfig, client *vsock.Client) error {
	// Create the initial index report immediately.
	if err := RunSingle(ctx, cfg, client); err != nil {
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

func RunSingle(ctx context.Context, cfg *config.AgentConfig, client *vsock.Client) error {
	report, err := runIndexer(ctx, cfg)
	if err != nil {
		return fmt.Errorf("creating index report: %w", err)
	}
	if cfg.Verbose {
		reportJson, err := jsonutil.ProtoToJSON(report)
		if err != nil {
			log.Errorf("failed to convert index report %q to json", report.GetHashId())
		} else {
			fmt.Println(reportJson)
		}
	}
	if !report.GetSuccess() {
		return fmt.Errorf("failed index report: %s", report.GetErr())
	}
	if err := client.SendIndexReport(report); err != nil {
		return fmt.Errorf("sending index report: %w", err)
	}
	return nil
}

func runIndexer(ctx context.Context, cfg *config.AgentConfig) (*v4.IndexReport, error) {
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
	// This is currently needed because claircore 1.5.40 enumerates
	// repositories as integers, but we reference repos by name in the
	// environments.
	report = fixupIndexReport(report)
	return report, nil
}

func fixupIndexReport(report *v4.IndexReport) *v4.IndexReport {
	for _, repo := range report.GetContents().GetRepositories() {
		repo.Id = repo.GetName()
	}
	return report
}
