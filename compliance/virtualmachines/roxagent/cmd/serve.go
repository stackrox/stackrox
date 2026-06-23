package cmd

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	mdvsock "github.com/mdlayher/vsock"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/compliance/node/index"
	roxvsock "github.com/stackrox/rox/compliance/virtualmachines/roxagent/vsock"
	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/vsockserver"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/httputil/proxy"
)

const mappingClientTimeout = 30 * time.Second

// ServeCmd creates the "serve" subcommand that scans packages,
// caches a VMReport, and listens on a VSOCK port for Sensor to pull it.
func ServeCmd(ctx context.Context) *cobra.Command {
	var (
		port           uint32
		hostPath       string
		repoCPEURL     string
		rescanInterval time.Duration
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Scan packages and serve the report over VSOCK (pull mode).",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runServe(ctx, port, hostPath, repoCPEURL, rescanInterval)
		},
	}
	cmd.Flags().Uint32Var(&port, "port", 818, "VSOCK port to listen on.")
	cmd.Flags().StringVar(&hostPath, "host-path", "/", "Root path for RPM/DNF scanning.")
	cmd.Flags().StringVar(&repoCPEURL, "repo-cpe-url", repoToCPEMappingURL, "URL for the repository-to-CPE mapping.")
	cmd.Flags().DurationVar(&rescanInterval, "rescan-interval", 4*time.Hour, "How often to rescan packages.")
	return cmd
}

func runServe(ctx context.Context, port uint32, hostPath, repoCPEURL string, rescanEvery time.Duration) error {
	srv := vsockserver.NewServer()

	report, err := scan(ctx, hostPath, repoCPEURL)
	if err != nil {
		return fmt.Errorf("initial scan: %w", err)
	}
	srv.SetReport(buildVMReport(report, hostPath))
	log.Infof("Initial scan complete, %d packages", len(report.GetContents().GetPackages()))

	ln, err := mdvsock.Listen(port, nil)
	if err != nil {
		return fmt.Errorf("vsock listen on port %d: %w", port, err)
	}
	defer func() { _ = ln.Close() }()
	log.Infof("Listening on VSOCK port %d (pull mode)", port)

	go srv.Serve(ctx, ln)

	ticker := time.NewTicker(rescanEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			r, err := scan(ctx, hostPath, repoCPEURL)
			if err != nil {
				log.Errorf("Rescan failed: %v", err)
				continue
			}
			srv.SetReport(buildVMReport(r, hostPath))
			log.Infof("Rescan complete, %d packages", len(r.GetContents().GetPackages()))
		}
	}
}

func scan(ctx context.Context, hostPath, repoCPEURL string) (*v4.IndexReport, error) {
	cfg := index.NodeIndexerConfig{
		HostPath:           hostPath,
		Client:             &http.Client{Transport: proxy.RoundTripper()},
		Repo2CPEMappingURL: repoCPEURL,
		Timeout:            mappingClientTimeout,
		PackageDBFilter:    "",
	}
	report, err := index.NewNodeIndexer(cfg).IndexNode(ctx)
	if err != nil {
		return nil, err
	}
	if !report.GetSuccess() {
		return nil, fmt.Errorf("index report failed: %s", report.GetErr())
	}
	return report, nil
}

func buildVMReport(ir *v4.IndexReport, hostPath string) *v1.VMReport {
	return &v1.VMReport{
		IndexReport: &v1.IndexReport{
			VsockCid: vsockCIDString(),
			IndexV4:  ir,
		},
		DiscoveredData: roxvsock.DiscoverVMData(hostPath),
	}
}

func vsockCIDString() string {
	cid, err := mdvsock.ContextID()
	if err != nil {
		log.Errorf("Failed to get VSOCK CID: %v", err)
		return "0"
	}
	return strconv.FormatUint(uint64(cid), 10)
}
