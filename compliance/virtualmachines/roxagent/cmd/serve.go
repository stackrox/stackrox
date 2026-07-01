package cmd

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/compliance/node/index"
	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/discovery"
	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/vsockserver"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/httputil/proxy"
)

// Set via -ldflags at build time.
var agentVersion = "development" //XDef:STABLE_MAIN_VERSION

const mappingClientTimeout = 30 * time.Second

// ServeCmd returns the "serve" cobra subcommand for pull-mode operation.
func ServeCmd(ctx context.Context) *cobra.Command {
	var (
		port           uint32
		hostPath       string
		repoCPEURL     string
		rescanInterval time.Duration
	)
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Scan and serve report over VSOCK (pull mode).",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runServe(ctx, port, hostPath, repoCPEURL, rescanInterval)
		},
	}
	cmd.Flags().Uint32Var(&port, "port", 818, "VSOCK port to listen on")
	cmd.Flags().StringVar(&hostPath, "host-path", "/", "Root filesystem path for indexing")
	cmd.Flags().StringVar(&repoCPEURL, "repo-cpe-url", repoToCPEMappingURL, "Repository to CPE mapping URL")
	cmd.Flags().DurationVar(&rescanInterval, "rescan-interval", 4*time.Hour, "Interval between rescans")
	return cmd
}

func runServe(ctx context.Context, port uint32, hostPath, repoCPEURL string, rescanEvery time.Duration) error {
	if rescanEvery <= 0 {
		return errors.New("rescan-interval must be greater than 0")
	}

	httpClient := &http.Client{Transport: proxy.RoundTripper()}
	cache := &vsockserver.ReportCache{}

	report, err := scanWithDiagnostics(ctx, hostPath, repoCPEURL, httpClient)
	if err != nil {
		return fmt.Errorf("initial scan: %w", err)
	}
	cache.SetReport(report, discoverFacts(hostPath))
	log.Infof("Initial scan complete, report cached. Num packages: %d", len(report.GetContents().GetPackages()))

	handler := vsockserver.NewHandler(cache, agentVersion)

	// TLS is mandatory: sensor always dials with TLS, so a plaintext agent is
	// unreachable. The KubeVirt CA (served by virt-handler on CID 2, port 1)
	// is a prerequisite for the pull-mode architecture to function — without it
	// the agent cannot verify connecting clients (virt-handler's client cert).
	refresher := vsockserver.NewCARefresher()
	if err := refresher.TryInitialFetch(); err != nil {
		return fmt.Errorf("KubeVirt CA required for VSOCK TLS: %w", err)
	}
	serverCert, err := selfSignedCert()
	if err != nil {
		return fmt.Errorf("generating server certificate: %w", err)
	}
	tlsCfg := refresher.TLSConfig()
	tlsCfg.Certificates = []tls.Certificate{serverCert}
	go refresher.RunRefreshLoop(ctx)
	log.Info("TLS enabled with KubeVirt CA")

	srv := vsockserver.NewServer(handler, tlsCfg)

	ln, err := vsock.Listen(port, nil)
	if err != nil {
		return fmt.Errorf("listening on VSOCK port %d: %w", port, err)
	}
	log.Infof("Listening on VSOCK port %d (pull mode)", port)

	go srv.Serve(ctx, ln)

	ticker := time.NewTicker(rescanEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			log.Info("Starting periodic rescan")
			r, err := scanWithDiagnostics(ctx, hostPath, repoCPEURL, httpClient)
			if err != nil {
				log.Errorf("Rescan failed: %v", err)
				continue
			}
			cache.SetReport(r, discoverFacts(hostPath))
			log.Infof("Rescan complete, report updated. Num packages: %d", len(r.GetContents().GetPackages()))
		}
	}
}

// scanWithDiagnostics runs the node indexer and surrounds it with filesystem
// and report diagnostics logging. This mirrors the diagnostics roxagent logs
// in push mode, so scan issues (e.g. "0 packages" or "0 repositories") can be
// triaged from agent logs regardless of transport mode.
func scanWithDiagnostics(ctx context.Context, hostPath, repoCPEURL string, httpClient *http.Client) (*v4.IndexReport, error) {
	// This may slow the indexing process down by 1-2 seconds, but the diagnostics are invaluable for debugging.
	logFilesystemDiagnostics(hostPath)

	report, err := scan(ctx, hostPath, repoCPEURL, httpClient)
	if err != nil {
		return nil, err
	}

	logIndexReportDiagnostics(report)
	return report, nil
}

func scan(ctx context.Context, hostPath, repoCPEURL string, httpClient *http.Client) (*v4.IndexReport, error) {
	cfg := index.NodeIndexerConfig{
		HostPath:           hostPath,
		Client:             httpClient,
		Repo2CPEMappingURL: repoCPEURL,
		Timeout:            mappingClientTimeout,
		PackageDBFilter:    "",
	}
	return index.NewNodeIndexer(cfg).IndexNode(ctx)
}

func discoverFacts(hostPath string) map[string]string {
	d := discovery.DiscoverVMData(hostPath)
	return map[string]string{
		"detected_os":         d.GetDetectedOs().String(),
		"os_version":          d.GetOsVersion(),
		"activation_status":   d.GetActivationStatus().String(),
		"dnf_metadata_status": d.GetDnfMetadataStatus().String(),
		"dnf_status":          formatDnfStatusFlags(d.GetDnfStatus()),
	}
}

func formatDnfStatusFlags(flags []v1.DnfStatusFlag) string {
	if len(flags) == 0 {
		return "none"
	}
	names := make([]string, 0, len(flags))
	for _, f := range flags {
		names = append(names, f.String())
	}
	slices.Sort(names)
	return strings.Join(names, ", ")
}

// selfSignedCert generates a self-signed ECDSA TLS certificate.
//
// This cert exists solely to satisfy TLS protocol requirements: a server MUST
// present a certificate so the key exchange can establish an encrypted channel.
// No party in the connection path validates this cert's identity or expiry —
// virt-handler connects with InsecureSkipVerify: true and no VerifyPeerCertificate
// callback (see kubevirt/kubevirt pkg/virt-handler/rest/console.go, VSOCKHandler).
// Authentication is handled in the opposite direction: the agent verifies
// virt-handler's client cert against the KubeVirt CA via RequireAndVerifyClientCert.
//
// The cert is ephemeral: regenerated on every agent start, never persisted.
func selfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generating ECDSA key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generating serial: %w", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		NotBefore:             time.Now().Add(-1 * time.Minute),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("creating certificate: %w", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}, nil
}
